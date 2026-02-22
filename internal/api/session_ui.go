package api

import (
	"bytes"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"engram/internal/auth"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type uiLoginFormPayload struct {
	Username  string
	Password  string
	CSRFToken string
	NextPath  string
}

type loginPageViewModel struct {
	ErrorMessage string
	CSRFToken    string
	NextPath     string
}

type dashboardPageViewModel struct {
	Username  string
	Role      string
	CSRFToken string
}

var sessionLoginPageTemplate = template.Must(template.New("session-login-page").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Engram Vault Login</title>
</head>
<body>
  <h1>Engram Vault Login</h1>
  {{if .ErrorMessage}}<p>{{.ErrorMessage}}</p>{{end}}
  <form method="post" action="/login">
    <label>Username <input type="text" name="username" /></label>
    <label>Password <input type="password" name="password" /></label>
    <input type="hidden" name="csrf_token" value="{{.CSRFToken}}" />
    <input type="hidden" name="next_path" value="{{.NextPath}}" />
    <button type="submit">Sign in</button>
  </form>
</body>
</html>`))

var sessionDashboardTemplate = template.Must(template.New("session-dashboard-page").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Engram Vault Test UI</title>
</head>
<body>
  <h1>Engram Vault Test UI</h1>
  <p>{{.Username}}</p>
  <p>{{.Role}}</p>
  <form method="post" action="/logout">
    <input type="hidden" name="csrf_token" value="{{.CSRFToken}}" />
    <button type="submit">Logout</button>
  </form>
</body>
</html>`))

// MountSessionUIRoutes registers the login and dashboard routes used by the migration UI.
func MountSessionUIRoutes(router chi.Router, dependencies SessionAuthDependencies) {
	deps := newSessionAuthDependencies(dependencies)
	router.Get("/", deps.handleHomeRedirect)
	router.Get("/login", deps.handleLoginPage)
	router.Post("/login", deps.handleLoginSubmit)
	router.Post("/logout", deps.handleLogoutSubmit)
	router.Get("/ui", deps.handleUIDashboard)
}

func (dependencies sessionAuthDependencies) handleHomeRedirect(writer http.ResponseWriter, request *http.Request) {
	if dependencies.isAuthenticated(request) {
		http.Redirect(writer, request, "/ui", http.StatusSeeOther)
		return
	}
	http.Redirect(writer, request, "/login", http.StatusSeeOther)
}

func (dependencies sessionAuthDependencies) handleLoginPage(writer http.ResponseWriter, request *http.Request) {
	redirectTarget := safeNextPath(request.URL.Query().Get("next"))
	if dependencies.isAuthenticated(request) {
		if redirectTarget == "" {
			redirectTarget = "/ui"
		}
		http.Redirect(writer, request, redirectTarget, http.StatusSeeOther)
		return
	}
	dependencies.renderLoginPage(writer, request, http.StatusOK, "", redirectTarget)
}

func (dependencies sessionAuthDependencies) handleLoginSubmit(writer http.ResponseWriter, request *http.Request) {
	if !dependencies.validateLoginDependencies(writer) {
		return
	}
	payload, ok := parseUILoginFormPayload(writer, request)
	if !ok {
		return
	}
	if !dependencies.validateUILoginCSRF(writer, request, payload.CSRFToken) {
		return
	}
	record, err := dependencies.lookupUserByUsername(request.Context(), payload.Username)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	if record == nil || !record.IsActive || !dependencies.verifyPassword(payload.Password, record.PasswordHash) {
		dependencies.renderLoginPage(
			writer,
			request,
			http.StatusUnauthorized,
			"Invalid username or password.",
			payload.NextPath,
		)
		return
	}
	dependencies.writeUILoginSuccess(writer, request, record, payload.NextPath)
}

func parseUILoginFormPayload(writer http.ResponseWriter, request *http.Request) (uiLoginFormPayload, bool) {
	if err := request.ParseForm(); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid form body"})
		return uiLoginFormPayload{}, false
	}
	return uiLoginFormPayload{
		Username:  strings.TrimSpace(request.FormValue("username")),
		Password:  request.FormValue("password"),
		CSRFToken: request.FormValue("csrf_token"),
		NextPath:  request.FormValue("next_path"),
	}, true
}

func (dependencies sessionAuthDependencies) validateUILoginCSRF(
	writer http.ResponseWriter,
	request *http.Request,
	csrfToken string,
) bool {
	state, err := dependencies.manager.DecodeRequest(request)
	if err != nil || strings.TrimSpace(state.CSRFToken) == "" || csrfToken != state.CSRFToken {
		writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "Invalid CSRF token"})
		return false
	}
	return true
}

func (dependencies sessionAuthDependencies) writeUILoginSuccess(
	writer http.ResponseWriter,
	request *http.Request,
	record *models.UserAuthRecord,
	nextPath string,
) {
	nextCSRFToken, err := dependencies.generateCSRFToken()
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "failed to generate csrf token"})
		return
	}
	if !dependencies.writeSessionCookie(writer, buildAuthenticatedSessionState(record, nextCSRFToken)) {
		return
	}
	redirectTarget := safeNextPath(nextPath)
	if redirectTarget == "" {
		redirectTarget = "/ui"
	}
	http.Redirect(writer, request, redirectTarget, http.StatusSeeOther)
}

func (dependencies sessionAuthDependencies) handleLogoutSubmit(writer http.ResponseWriter, request *http.Request) {
	if !dependencies.validateCoreDependencies(writer) {
		return
	}
	if err := request.ParseForm(); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid form body"})
		return
	}
	if !dependencies.validateLogoutCSRF(writer, request, request.FormValue("csrf_token")) {
		return
	}
	dependencies.clearSessionCookie(writer)
	http.Redirect(writer, request, "/login", http.StatusSeeOther)
}

func (dependencies sessionAuthDependencies) handleUIDashboard(writer http.ResponseWriter, request *http.Request) {
	record, ok := dependencies.resolveSessionUser(request)
	if !ok {
		http.Redirect(writer, request, "/login", http.StatusSeeOther)
		return
	}
	state, ok := dependencies.ensureCSRFSessionState(writer, request)
	if !ok {
		return
	}
	renderTemplate(
		writer,
		http.StatusOK,
		sessionDashboardTemplate,
		dashboardPageViewModel{
			Username:  record.Username,
			Role:      string(record.Role),
			CSRFToken: state.CSRFToken,
		},
	)
}

func (dependencies sessionAuthDependencies) resolveSessionUser(request *http.Request) (*models.UserAuthRecord, bool) {
	state, err := dependencies.manager.DecodeRequest(request)
	if err != nil {
		return nil, false
	}
	userID, username, role, ok := decodeSessionUser(state.User)
	if !ok {
		return nil, false
	}
	if dependencies.lookupUserByID == nil {
		parsedRole, parseErr := models.ParseUserRole(role)
		if parseErr != nil {
			return nil, false
		}
		return &models.UserAuthRecord{UserID: userID, Username: username, Role: parsedRole, IsActive: true}, true
	}
	record, lookupErr := dependencies.lookupUserByID(request.Context(), userID)
	if lookupErr != nil || record == nil || !record.IsActive {
		return nil, false
	}
	return record, true
}

func decodeSessionUser(user *auth.SessionUser) (uuid.UUID, string, string, bool) {
	if user == nil {
		return uuid.Nil, "", "", false
	}
	userID, err := uuid.Parse(strings.TrimSpace(user.UserID))
	if err != nil {
		return uuid.Nil, "", "", false
	}
	username := strings.TrimSpace(user.Username)
	role := strings.TrimSpace(user.Role)
	if username == "" || role == "" {
		return uuid.Nil, "", "", false
	}
	return userID, username, role, true
}

func (dependencies sessionAuthDependencies) isAuthenticated(request *http.Request) bool {
	_, ok := dependencies.resolveSessionUser(request)
	return ok
}

func (dependencies sessionAuthDependencies) renderLoginPage(
	writer http.ResponseWriter,
	request *http.Request,
	statusCode int,
	errorMessage string,
	nextPath string,
) {
	state, ok := dependencies.ensureCSRFSessionState(writer, request)
	if !ok {
		return
	}
	renderTemplate(
		writer,
		statusCode,
		sessionLoginPageTemplate,
		loginPageViewModel{
			ErrorMessage: errorMessage,
			CSRFToken:    state.CSRFToken,
			NextPath:     safeNextPath(nextPath),
		},
	)
}

func renderTemplate(writer http.ResponseWriter, statusCode int, tmpl *template.Template, data any) {
	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "failed to render template"})
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(statusCode)
	_, _ = writer.Write(body.Bytes())
}

func safeNextPath(rawPath string) string {
	candidate := strings.TrimSpace(rawPath)
	if candidate == "" {
		return ""
	}
	if decoded, err := url.QueryUnescape(candidate); err == nil {
		candidate = strings.TrimSpace(decoded)
	}
	if !strings.HasPrefix(candidate, "/") {
		return ""
	}
	if strings.HasPrefix(candidate, "//") {
		return ""
	}
	return candidate
}

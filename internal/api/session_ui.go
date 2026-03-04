package api

import (
	"bytes"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"strconv"
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
	OIDCLoginURL string
	OIDCEnabled  bool
}

type dashboardPageViewModel struct {
	Username  string
	Role      string
	CSRFToken string
}

type adminPageViewModel struct {
	Username  string
	Role      string
	CSRFToken string
}

type loginPageRenderRequest struct {
	StatusCode   int
	ErrorMessage string
	NextPath     string
}

type uiLoginSuccessRequest struct {
	Record     *models.UserAuthRecord
	AttemptKey string
	NextPath   string
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
  {{if .OIDCEnabled}}
  <p><a href="{{.OIDCLoginURL}}">Sign in with OIDC</a></p>
  {{end}}
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

var sessionAdminTemplate = template.Must(template.New("session-admin-page").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Engram Vault Admin</title>
</head>
<body>
  <h1>Engram Vault Admin</h1>
  <p>{{.Username}}</p>
  <p>{{.Role}}</p>
  <p>Migration admin surface enabled.</p>
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
	router.Get("/login/oidc", deps.handleOIDCStart)
	router.Get("/login/oidc/callback", deps.handleOIDCCallback)
	router.Post("/login", deps.handleLoginSubmit)
	router.Post("/logout", deps.handleLogoutSubmit)
	router.Get("/ui", deps.handleLegacyWorkspaceRedirect)
	router.Get("/ui/admin", deps.handleLegacyAdminRedirect)
	router.Get("/admin/memory", deps.handleLegacyAdminMemoryRedirect)
	router.Get("/projects/transfer", deps.handleLegacyProjectTransferRedirect)
	router.Get("/app", deps.handleAppRootRedirect)
	router.Get("/app/workspace", deps.handleUIDashboard)
	router.Get("/app/admin/sessions", deps.handleUIAdmin)
	router.Get("/app/admin/engrams", deps.handleUIAdmin)
	router.Get("/app/admin/collections", deps.handleUIAdmin)
	router.Get("/app/admin/curation", deps.handleUIAdmin)
	router.Get("/app/admin/contradictions", deps.handleUIAdmin)
	router.Get("/app/admin/tokens", deps.handleUIAdmin)
	router.Get("/app/admin/observability", deps.handleUIAdmin)
	router.Get("/app/projects/transfer/export", deps.handleUIDashboard)
	router.Get("/app/projects/transfer/import", deps.handleUIDashboard)
	router.Get("/app/*", deps.handleAppRoute)
	router.Post("/ui/admin/mcp-tokens/create", deps.handleUIAdminCreateMCPToken)
	router.Post("/ui/admin/mcp-tokens/{token_id}/revoke", deps.handleUIAdminRevokeMCPToken)
}

func (dependencies sessionAuthDependencies) handleHomeRedirect(writer http.ResponseWriter, request *http.Request) {
	if dependencies.isAuthenticated(request) {
		http.Redirect(writer, request, "/app/workspace", http.StatusSeeOther)
		return
	}
	http.Redirect(writer, request, "/login", http.StatusSeeOther)
}

func (dependencies sessionAuthDependencies) handleAppRootRedirect(writer http.ResponseWriter, request *http.Request) {
	if dependencies.isAuthenticated(request) {
		http.Redirect(writer, request, "/app/workspace", http.StatusSeeOther)
		return
	}
	http.Redirect(writer, request, "/login?next=/app/workspace", http.StatusSeeOther)
}

func (dependencies sessionAuthDependencies) handleLegacyWorkspaceRedirect(
	writer http.ResponseWriter,
	request *http.Request,
) {
	http.Redirect(writer, request, "/app/workspace", http.StatusSeeOther)
}

func (dependencies sessionAuthDependencies) handleLegacyAdminRedirect(writer http.ResponseWriter, request *http.Request) {
	http.Redirect(writer, request, "/app/admin/sessions", http.StatusSeeOther)
}

func (dependencies sessionAuthDependencies) handleLegacyAdminMemoryRedirect(
	writer http.ResponseWriter,
	request *http.Request,
) {
	http.Redirect(writer, request, "/app/admin/engrams", http.StatusSeeOther)
}

func (dependencies sessionAuthDependencies) handleLegacyProjectTransferRedirect(
	writer http.ResponseWriter,
	request *http.Request,
) {
	http.Redirect(writer, request, "/app/projects/transfer/export", http.StatusSeeOther)
}

func (dependencies sessionAuthDependencies) handleLoginPage(writer http.ResponseWriter, request *http.Request) {
	redirectTarget := safeNextPath(request.URL.Query().Get("next"))
	if dependencies.isAuthenticated(request) {
		if redirectTarget == "" {
			redirectTarget = "/app/workspace"
		}
		http.Redirect(writer, request, redirectTarget, http.StatusSeeOther)
		return
	}
	dependencies.renderLoginPage(
		writer,
		request,
		loginPageRenderRequest{
			StatusCode: http.StatusOK,
			NextPath:   redirectTarget,
		},
	)
}

func (dependencies sessionAuthDependencies) handleLoginSubmit(writer http.ResponseWriter, request *http.Request) {
	if !dependencies.validateLoginDependencies(writer) {
		return
	}
	payload, ok := parseUILoginFormPayload(writer, request)
	if !ok {
		return
	}
	attemptKey, ok := dependencies.validateUILoginPreconditions(writer, request, payload)
	if !ok {
		return
	}
	record, authenticated := dependencies.authenticateUILoginRecord(writer, request, payload)
	if !authenticated {
		dependencies.handleFailedUILogin(writer, request, payload, attemptKey)
		return
	}
	dependencies.handleSuccessfulUILogin(
		writer,
		request,
		uiLoginSuccessRequest{
			Record:     record,
			AttemptKey: attemptKey,
			NextPath:   payload.NextPath,
		},
	)
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
	if !isValidSessionCSRF(state, csrfToken, err) {
		writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "Invalid CSRF token"})
		return false
	}
	return true
}

func (dependencies sessionAuthDependencies) validateUILoginPreconditions(
	writer http.ResponseWriter,
	request *http.Request,
	payload uiLoginFormPayload,
) (string, bool) {
	if !dependencies.validateUILoginCSRF(writer, request, payload.CSRFToken) {
		dependencies.logAuditEventRequest(
			request,
			sessionAuditEvent{
				eventType: "login_csrf_rejected",
				success:   false,
				username:  payload.Username,
			},
		)
		return "", false
	}
	attemptKey := loginAttemptKey(request, payload.Username)
	if dependencies.loginAttemptGuard == nil {
		return attemptKey, true
	}
	allowed, retryAfterSeconds := dependencies.loginAttemptGuard.Check(attemptKey)
	if allowed {
		return attemptKey, true
	}
	detail := fmt.Sprintf("Too many login attempts. Retry in %d seconds.", retryAfterSeconds)
	dependencies.logAuditEventRequest(
		request,
		sessionAuditEvent{
			eventType: "login_rate_limited",
			success:   false,
			username:  payload.Username,
			detail:    fmt.Sprintf("retry_in_seconds=%d", retryAfterSeconds),
		},
	)
	writeJSON(writer, http.StatusTooManyRequests, map[string]string{"detail": detail})
	return "", false
}

func (dependencies sessionAuthDependencies) authenticateUILoginRecord(
	writer http.ResponseWriter,
	request *http.Request,
	payload uiLoginFormPayload,
) (*models.UserAuthRecord, bool) {
	record, err := dependencies.lookupUserByUsername(request.Context(), payload.Username)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return nil, false
	}
	if !isValidLoginRecord(record, payload.Password, dependencies.verifyPassword) {
		return nil, false
	}
	return record, true
}

func (dependencies sessionAuthDependencies) handleFailedUILogin(
	writer http.ResponseWriter,
	request *http.Request,
	payload uiLoginFormPayload,
	attemptKey string,
) {
	if dependencies.loginAttemptGuard != nil {
		dependencies.loginAttemptGuard.RegisterFailure(attemptKey)
	}
	dependencies.logAuditEventRequest(
		request,
		sessionAuditEvent{
			eventType: "login_failed",
			success:   false,
			username:  payload.Username,
		},
	)
	dependencies.renderLoginPage(writer, request, loginPageRenderRequest{
		StatusCode:   http.StatusUnauthorized,
		ErrorMessage: "Invalid username or password.",
		NextPath:     payload.NextPath,
	})
}

func (dependencies sessionAuthDependencies) handleSuccessfulUILogin(
	writer http.ResponseWriter,
	request *http.Request,
	result uiLoginSuccessRequest,
) {
	if dependencies.loginAttemptGuard != nil {
		dependencies.loginAttemptGuard.RegisterSuccess(result.AttemptKey)
	}
	dependencies.logAuditEventRequest(
		request,
		sessionAuditEvent{
			eventType: "login_success",
			success:   true,
			username:  result.Record.Username,
		},
	)
	dependencies.writeUILoginSuccess(writer, request, result.Record, result.NextPath)
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
		redirectTarget = "/app/workspace"
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
	record, _ := dependencies.resolveSessionUser(request)
	username := ""
	if record != nil {
		username = record.Username
	}
	if !dependencies.validateLogoutCSRF(writer, request, request.FormValue("csrf_token")) {
		dependencies.logAuditEventRequest(
			request,
			sessionAuditEvent{
				eventType: "logout_csrf_rejected",
				success:   false,
				username:  username,
			},
		)
		return
	}
	dependencies.logAuditEventRequest(
		request,
		sessionAuditEvent{
			eventType: "logout_success",
			success:   true,
			username:  username,
		},
	)
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

func (dependencies sessionAuthDependencies) handleAppRoute(writer http.ResponseWriter, request *http.Request) {
	if strings.HasPrefix(request.URL.Path, "/app/admin/") {
		dependencies.handleUIAdmin(writer, request)
		return
	}
	dependencies.handleUIDashboard(writer, request)
}

func (dependencies sessionAuthDependencies) handleUIAdmin(writer http.ResponseWriter, request *http.Request) {
	record, state, ok := dependencies.resolveSessionUserAndState(writer, request)
	if !ok {
		return
	}
	if !isAdminRole(record.Role) {
		writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "Admin role required"})
		return
	}
	renderTemplate(
		writer,
		http.StatusOK,
		sessionAdminTemplate,
		adminPageViewModel{
			Username:  record.Username,
			Role:      string(record.Role),
			CSRFToken: state.CSRFToken,
		},
	)
}

func (dependencies sessionAuthDependencies) handleUIAdminCreateMCPToken(writer http.ResponseWriter, request *http.Request) {
	record, _, ok := dependencies.resolveSessionUserAndState(writer, request)
	if !ok {
		return
	}
	if !isAdminRole(record.Role) {
		writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "Admin role required"})
		return
	}
	if !dependencies.hasMCPTokenCreateDependencies() {
		writeSessionUserDependenciesError(writer)
		return
	}
	if err := request.ParseForm(); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid form body"})
		return
	}
	if !dependencies.validateLogoutCSRF(writer, request, request.FormValue("csrf_token")) {
		return
	}
	createRequest, ok := decodeMCPTokenCreateFormRequest(writer, request)
	if !ok {
		return
	}
	created, err := dependencies.createTokenForOwner(
		request.Context(),
		record.UserID,
		createRequest,
		dependencies.mcpTokenPepper,
	)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	dependencies.logAuditEventRequest(
		request,
		sessionAuditEvent{
			eventType: "mcp_token_created",
			success:   true,
			username:  record.Username,
			metadata: map[string]any{
				"token_id":            created.TokenID.String(),
				"scope":               string(created.Scope),
				"name":                created.Name,
				"allowed_tools":       created.AllowedTools,
				"allowed_project_ids": created.AllowedProjectIDs,
				"expires_at":          created.ExpiresAt,
			},
		},
	)
	http.Redirect(writer, request, "/app/admin/tokens", http.StatusSeeOther)
}

func (dependencies sessionAuthDependencies) handleUIAdminRevokeMCPToken(writer http.ResponseWriter, request *http.Request) {
	record, _, ok := dependencies.resolveSessionUserAndState(writer, request)
	if !ok {
		return
	}
	if !isAdminRole(record.Role) {
		writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "Admin role required"})
		return
	}
	if dependencies.revokeTokenForOwner == nil {
		writeSessionUserDependenciesError(writer)
		return
	}
	input, ok := dependencies.parseUIAdminRevokeMCPTokenInput(writer, request)
	if !ok {
		return
	}
	revoked, err := dependencies.revokeTokenForOwner(request.Context(), input.tokenID, record.UserID)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	if revoked == nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "Token not found"})
		return
	}
	dependencies.logAuditEventRequest(
		request,
		sessionAuditEvent{
			eventType: "mcp_token_revoked",
			success:   true,
			username:  record.Username,
			metadata: map[string]any{
				"token_id": input.tokenID.String(),
				"reason":   optionalTrimmedReason(input.reason),
			},
		},
	)
	http.Redirect(writer, request, "/app/admin/tokens", http.StatusSeeOther)
}

type uiAdminRevokeMCPTokenInput struct {
	tokenID uuid.UUID
	reason  *string
}

func (dependencies sessionAuthDependencies) parseUIAdminRevokeMCPTokenInput(
	writer http.ResponseWriter,
	request *http.Request,
) (uiAdminRevokeMCPTokenInput, bool) {
	if err := request.ParseForm(); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid form body"})
		return uiAdminRevokeMCPTokenInput{}, false
	}
	if !dependencies.validateLogoutCSRF(writer, request, request.FormValue("csrf_token")) {
		return uiAdminRevokeMCPTokenInput{}, false
	}
	tokenID, ok := parsePathUUID(writer, request, "token_id")
	if !ok {
		return uiAdminRevokeMCPTokenInput{}, false
	}
	return uiAdminRevokeMCPTokenInput{
		tokenID: tokenID,
		reason:  optionalTrimmedString(request.FormValue("reason")),
	}, true
}

func decodeMCPTokenCreateFormRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (models.MCPTokenCreateRequest, bool) {
	name := strings.TrimSpace(request.FormValue("name"))
	if !isValidMCPTokenName(name) {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid token name"})
		return models.MCPTokenCreateRequest{}, false
	}
	scope := resolveMCPTokenScope(request.FormValue("scope"))
	if _, err := models.ParseMCPTokenScope(scope); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid scope"})
		return models.MCPTokenCreateRequest{}, false
	}
	expiresInDays := defaultMCPTokenExpiryDays
	if rawValue := strings.TrimSpace(request.FormValue("expires_in_days")); rawValue != "" {
		parsed, err := strconv.Atoi(rawValue)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid expires_in_days"})
			return models.MCPTokenCreateRequest{}, false
		}
		expiresInDays = parsed
	}
	if expiresInDays < 1 || expiresInDays > maxMCPTokenExpiryDays {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid expires_in_days"})
		return models.MCPTokenCreateRequest{}, false
	}
	return models.MCPTokenCreateRequest{
		Name:              name,
		Scope:             scope,
		AllowedTools:      splitAndNormalizeCSVValues(request.FormValue("allowed_tools")),
		AllowedProjectIDs: splitAndNormalizeCSVValues(request.FormValue("allowed_project_ids")),
		ExpiresInDays:     expiresInDays,
	}, true
}

func splitAndNormalizeCSVValues(raw string) []string {
	parts := strings.Split(raw, ",")
	if len(parts) == 0 {
		return []string{}
	}
	normalized := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func (dependencies sessionAuthDependencies) resolveSessionUserAndState(
	writer http.ResponseWriter,
	request *http.Request,
) (*models.UserAuthRecord, auth.SessionState, bool) {
	record, ok := dependencies.resolveSessionUser(request)
	if !ok {
		http.Redirect(writer, request, "/login", http.StatusSeeOther)
		return nil, auth.SessionState{}, false
	}
	state, ok := dependencies.ensureCSRFSessionState(writer, request)
	if !ok {
		return nil, auth.SessionState{}, false
	}
	return record, state, true
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
	if !isActiveUserRecord(record, lookupErr) {
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
	renderRequest loginPageRenderRequest,
) {
	state, ok := dependencies.ensureCSRFSessionState(writer, request)
	if !ok {
		return
	}
	renderTemplate(
		writer,
		renderRequest.StatusCode,
		sessionLoginPageTemplate,
		loginPageViewModel{
			ErrorMessage: renderRequest.ErrorMessage,
			CSRFToken:    state.CSRFToken,
			NextPath:     safeNextPath(renderRequest.NextPath),
			OIDCLoginURL: buildOIDCLoginURL(renderRequest.NextPath),
			OIDCEnabled:  dependencies.hasOIDCProvider(),
		},
	)
}

func buildOIDCLoginURL(nextPath string) string {
	sanitizedPath := safeNextPath(nextPath)
	if sanitizedPath == "" {
		return "/login/oidc"
	}
	return "/login/oidc?next=" + url.QueryEscape(sanitizedPath)
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

func loginAttemptKey(request *http.Request, username string) string {
	return strings.ToLower(strings.TrimSpace(username)) + ":" + requestClientIP(request)
}

func requestClientIP(request *http.Request) string {
	if request == nil {
		return "unknown"
	}
	remoteAddress := strings.TrimSpace(request.RemoteAddr)
	if remoteAddress == "" {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil && strings.TrimSpace(host) != "" {
		return host
	}
	return remoteAddress
}

func isValidSessionCSRF(state auth.SessionState, csrfToken string, decodeErr error) bool {
	if decodeErr != nil {
		return false
	}
	expectedToken := strings.TrimSpace(state.CSRFToken)
	if expectedToken == "" {
		return false
	}
	return csrfToken == expectedToken
}

func isValidLoginRecord(
	record *models.UserAuthRecord,
	password string,
	verifyPassword func(password, encodedHash string) bool,
) bool {
	if !isActiveUserRecord(record, nil) {
		return false
	}
	return verifyPassword != nil && verifyPassword(password, record.PasswordHash)
}

func isActiveUserRecord(record *models.UserAuthRecord, err error) bool {
	if err != nil || record == nil {
		return false
	}
	return record.IsActive
}

func isAdminRole(role models.UserRole) bool {
	return role == models.UserRoleAdmin
}

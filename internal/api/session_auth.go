package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"engram/internal/auth"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
)

// SessionUserByUsernameLookup resolves an auth-capable user record by username.
type SessionUserByUsernameLookup func(ctx context.Context, username string) (*models.UserAuthRecord, error)

// SessionAuthDependencies captures required collaborators for session auth routes.
type SessionAuthDependencies struct {
	SessionManager       *auth.SessionManager
	LookupUserByUsername SessionUserByUsernameLookup
	LookupUserByID       SessionUserLookup
	VerifyPassword       func(password, encodedHash string) bool
	GenerateCSRFToken    func() (string, error)
	CookieSecure         bool
}

type sessionAuthDependencies struct {
	manager              *auth.SessionManager
	lookupUserByUsername SessionUserByUsernameLookup
	lookupUserByID       SessionUserLookup
	verifyPassword       func(password, encodedHash string) bool
	generateCSRFToken    func() (string, error)
	cookieSecure         bool
}

type sessionLoginRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	CSRFToken string `json:"csrf_token"`
}

type sessionLogoutRequest struct {
	CSRFToken string `json:"csrf_token"`
}

func newSessionAuthDependencies(dependencies SessionAuthDependencies) sessionAuthDependencies {
	return sessionAuthDependencies{
		manager:              dependencies.SessionManager,
		lookupUserByUsername: dependencies.LookupUserByUsername,
		lookupUserByID:       dependencies.LookupUserByID,
		verifyPassword:       dependencies.VerifyPassword,
		generateCSRFToken:    dependencies.GenerateCSRFToken,
		cookieSecure:         dependencies.CookieSecure,
	}
}

// MountSessionAuthRoutes registers session login/logout/csrf/me endpoints.
func MountSessionAuthRoutes(router chi.Router, dependencies SessionAuthDependencies) {
	deps := newSessionAuthDependencies(dependencies)
	router.Route("/api/v1/session", func(session chi.Router) {
		session.Get("/csrf", deps.handleCSRF)
		session.Post("/login", deps.handleLogin)
		session.Post("/logout", deps.handleLogout)
	})

	router.Get("/api/v1/me", deps.handleCurrentUser)
}

func (dependencies sessionAuthDependencies) handleCSRF(writer http.ResponseWriter, request *http.Request) {
	state, ok := dependencies.ensureCSRFSessionState(writer, request)
	if !ok {
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"csrf_token": state.CSRFToken})
}

func (dependencies sessionAuthDependencies) handleLogin(writer http.ResponseWriter, request *http.Request) {
	if !dependencies.validateLoginDependencies(writer) {
		return
	}
	payload, ok := dependencies.decodeLoginRequest(writer, request)
	if !ok {
		return
	}
	if !dependencies.validateLoginCSRF(writer, request, payload.CSRFToken) {
		return
	}
	record, ok := dependencies.authenticateLogin(writer, request, payload.Username, payload.Password)
	if !ok {
		return
	}
	dependencies.writeLoginSuccess(writer, record)
}

func (dependencies sessionAuthDependencies) ensureCSRFSessionState(
	writer http.ResponseWriter,
	request *http.Request,
) (auth.SessionState, bool) {
	if !dependencies.validateCoreDependencies(writer) {
		return auth.SessionState{}, false
	}
	state, _ := dependencies.manager.DecodeRequest(request)
	if strings.TrimSpace(state.CSRFToken) == "" {
		token, err := dependencies.generateCSRFToken()
		if err != nil {
			writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "failed to generate csrf token"})
			return auth.SessionState{}, false
		}
		state.CSRFToken = token
	}
	if !dependencies.writeSessionCookie(writer, state) {
		return auth.SessionState{}, false
	}
	return state, true
}

func (dependencies sessionAuthDependencies) validateLoginDependencies(writer http.ResponseWriter) bool {
	if !dependencies.validateCoreDependencies(writer) || dependencies.lookupUserByUsername == nil || dependencies.verifyPassword == nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "session auth dependencies are not configured"})
		return false
	}
	return true
}

func (dependencies sessionAuthDependencies) decodeLoginRequest(writer http.ResponseWriter, request *http.Request) (sessionLoginRequest, bool) {
	var payload sessionLoginRequest
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return sessionLoginRequest{}, false
	}
	payload.Username = strings.TrimSpace(payload.Username)
	if payload.Username == "" || payload.Password == "" {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "username and password are required"})
		return sessionLoginRequest{}, false
	}
	return payload, true
}

func (dependencies sessionAuthDependencies) validateLoginCSRF(writer http.ResponseWriter, request *http.Request, csrfToken string) bool {
	state, err := dependencies.manager.DecodeRequest(request)
	if err != nil || strings.TrimSpace(state.CSRFToken) == "" || csrfToken != state.CSRFToken {
		writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "invalid csrf token"})
		return false
	}
	return true
}

func (dependencies sessionAuthDependencies) authenticateLogin(
	writer http.ResponseWriter,
	request *http.Request,
	username string,
	password string,
) (*models.UserAuthRecord, bool) {
	record, err := dependencies.lookupUserByUsername(request.Context(), username)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return nil, false
	}
	if record == nil || !record.IsActive || !dependencies.verifyPassword(password, record.PasswordHash) {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"detail": "invalid credentials"})
		return nil, false
	}
	return record, true
}

func buildAuthenticatedSessionState(record *models.UserAuthRecord, csrfToken string) auth.SessionState {
	return auth.SessionState{
		User: &auth.SessionUser{
			UserID:   record.UserID.String(),
			Username: record.Username,
			Role:     string(record.Role),
		},
		CSRFToken: csrfToken,
	}
}

func (dependencies sessionAuthDependencies) writeLoginSuccess(
	writer http.ResponseWriter,
	record *models.UserAuthRecord,
) {
	nextCSRFToken, err := dependencies.generateCSRFToken()
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "failed to generate csrf token"})
		return
	}
	nextState := buildAuthenticatedSessionState(record, nextCSRFToken)
	if !dependencies.writeSessionCookie(writer, nextState) {
		return
	}
	writeJSON(
		writer,
		http.StatusOK,
		map[string]any{
			"user_id":    record.UserID,
			"username":   record.Username,
			"role":       record.Role,
			"is_active":  record.IsActive,
			"csrf_token": nextCSRFToken,
		},
	)
}

func (dependencies sessionAuthDependencies) handleLogout(writer http.ResponseWriter, request *http.Request) {
	if !dependencies.validateCoreDependencies(writer) {
		return
	}
	var payload sessionLogoutRequest
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return
	}
	if !dependencies.validateLogoutCSRF(writer, request, payload.CSRFToken) {
		return
	}
	dependencies.clearSessionCookie(writer)
	writeJSON(writer, http.StatusOK, map[string]bool{"logged_out": true})
}

func (dependencies sessionAuthDependencies) validateLogoutCSRF(
	writer http.ResponseWriter,
	request *http.Request,
	csrfToken string,
) bool {
	state, err := dependencies.manager.DecodeRequest(request)
	if err == nil && strings.TrimSpace(state.CSRFToken) != "" && csrfToken != state.CSRFToken {
		writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "invalid csrf token"})
		return false
	}
	return true
}

func (dependencies sessionAuthDependencies) clearSessionCookie(writer http.ResponseWriter) {
	http.SetCookie(writer, &http.Cookie{
		Name:     dependencies.manager.CookieName(),
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Secure:   dependencies.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (dependencies sessionAuthDependencies) handleCurrentUser(writer http.ResponseWriter, request *http.Request) {
	if dependencies.lookupUserByID == nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "session auth dependencies are not configured"})
		return
	}
	actor, ok := AdminActorFromContext(request.Context())
	if !ok {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"detail": "authentication required"})
		return
	}
	record, err := dependencies.lookupUserByID(request.Context(), actor.UserID)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	if record == nil || !record.IsActive {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"detail": "authentication required"})
		return
	}
	writeJSON(
		writer,
		http.StatusOK,
		map[string]any{
			"user_id":   record.UserID,
			"username":  record.Username,
			"role":      record.Role,
			"is_active": record.IsActive,
		},
	)
}

func (dependencies sessionAuthDependencies) validateCoreDependencies(writer http.ResponseWriter) bool {
	if dependencies.manager == nil || dependencies.generateCSRFToken == nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "session auth dependencies are not configured"})
		return false
	}
	return true
}

func (dependencies sessionAuthDependencies) writeSessionCookie(writer http.ResponseWriter, state auth.SessionState) bool {
	token, err := dependencies.manager.Encode(state)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "failed to encode session"})
		return false
	}
	ttl := dependencies.manager.CookieTTL()
	cookie := &http.Cookie{
		Name:     dependencies.manager.CookieName(),
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   dependencies.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
	if ttl > 0 {
		cookie.MaxAge = int(ttl.Seconds())
		cookie.Expires = time.Now().UTC().Add(ttl)
	}
	http.SetCookie(writer, cookie)
	return true
}

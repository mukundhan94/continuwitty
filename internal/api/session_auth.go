package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"engram/internal/auth"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// SessionUserByUsernameLookup resolves an auth-capable user record by username.
type SessionUserByUsernameLookup func(ctx context.Context, username string) (*models.UserAuthRecord, error)

// SessionLoginAttemptGuard provides login attempt rate-limit checks.
type SessionLoginAttemptGuard interface {
	Check(key string) (bool, int)
	RegisterSuccess(key string)
	RegisterFailure(key string)
}

// SessionAuditLogger writes auth-related audit events.
type SessionAuditLogger func(
	request *http.Request,
	eventType string,
	success bool,
	username string,
	detail string,
	metadata map[string]any,
)

// SessionAuthDependencies captures required collaborators for session auth routes.
type SessionAuthDependencies struct {
	SessionManager           *auth.SessionManager
	LookupUserByUsername     SessionUserByUsernameLookup
	LookupUserByID           SessionUserLookup
	VerifyPassword           func(password, encodedHash string) bool
	GenerateCSRFToken        func() (string, error)
	CookieSecure             bool
	LoginAttemptGuard        SessionLoginAttemptGuard
	LogAuditEvent            SessionAuditLogger
	ListUsers                func(ctx context.Context, limit, offset int) ([]models.UserRecord, error)
	CreateUser               func(ctx context.Context, input SessionUserCreateInput) (*models.UserRecord, error)
	UpdateUser               func(ctx context.Context, userID uuid.UUID, input SessionUserUpdateInput) (*models.UserRecord, error)
	HashPassword             func(password string) (string, error)
	ResolveProjectIDForWrite func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		projectID string,
	) (SessionProjectResolution, error)
	CreateEngram func(
		ctx context.Context,
		payload models.MemoryEngramCreate,
		ownerUserID uuid.UUID,
	) (*models.EngramCreateResponse, error)
	ListEngrams func(
		ctx context.Context,
		projectID *string,
		limit int,
		offset int,
		actorUserID uuid.UUID,
	) ([]models.EngramSummary, error)
	QueryEngrams func(
		ctx context.Context,
		request models.EngramQueryRequest,
		actorUserID uuid.UUID,
	) ([]models.EngramQueryResult, error)
	GetRehydrationBundle func(
		ctx context.Context,
		engramID uuid.UUID,
		actorUserID uuid.UUID,
	) (*models.RehydrationBundle, error)
	GetEngramSources func(
		ctx context.Context,
		engramID uuid.UUID,
		limit int,
		actorUserID uuid.UUID,
	) ([]models.EngramSourceRecord, error)
}

type sessionAuthDependencies struct {
	manager                  *auth.SessionManager
	lookupUserByUsername     SessionUserByUsernameLookup
	lookupUserByID           SessionUserLookup
	verifyPassword           func(password, encodedHash string) bool
	generateCSRFToken        func() (string, error)
	cookieSecure             bool
	loginAttemptGuard        SessionLoginAttemptGuard
	logAuditEvent            SessionAuditLogger
	listUsers                func(ctx context.Context, limit, offset int) ([]models.UserRecord, error)
	createUser               func(ctx context.Context, input SessionUserCreateInput) (*models.UserRecord, error)
	updateUser               func(ctx context.Context, userID uuid.UUID, input SessionUserUpdateInput) (*models.UserRecord, error)
	hashPassword             func(password string) (string, error)
	resolveProjectIDForWrite func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		projectID string,
	) (SessionProjectResolution, error)
	createEngram func(
		ctx context.Context,
		payload models.MemoryEngramCreate,
		ownerUserID uuid.UUID,
	) (*models.EngramCreateResponse, error)
	listEngrams func(
		ctx context.Context,
		projectID *string,
		limit int,
		offset int,
		actorUserID uuid.UUID,
	) ([]models.EngramSummary, error)
	queryEngrams func(
		ctx context.Context,
		request models.EngramQueryRequest,
		actorUserID uuid.UUID,
	) ([]models.EngramQueryResult, error)
	getRehydrationBundle func(
		ctx context.Context,
		engramID uuid.UUID,
		actorUserID uuid.UUID,
	) (*models.RehydrationBundle, error)
	getEngramSources func(
		ctx context.Context,
		engramID uuid.UUID,
		limit int,
		actorUserID uuid.UUID,
	) ([]models.EngramSourceRecord, error)
}

type sessionLoginRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	CSRFToken string `json:"csrf_token"`
}

type sessionLogoutRequest struct {
	CSRFToken string `json:"csrf_token"`
}

type sessionAuditEvent struct {
	eventType string
	success   bool
	username  string
	detail    string
	metadata  map[string]any
}

func newSessionAuthDependencies(dependencies SessionAuthDependencies) sessionAuthDependencies {
	return sessionAuthDependencies{
		manager:                  dependencies.SessionManager,
		lookupUserByUsername:     dependencies.LookupUserByUsername,
		lookupUserByID:           dependencies.LookupUserByID,
		verifyPassword:           dependencies.VerifyPassword,
		generateCSRFToken:        dependencies.GenerateCSRFToken,
		cookieSecure:             dependencies.CookieSecure,
		loginAttemptGuard:        dependencies.LoginAttemptGuard,
		logAuditEvent:            dependencies.LogAuditEvent,
		listUsers:                dependencies.ListUsers,
		createUser:               dependencies.CreateUser,
		updateUser:               dependencies.UpdateUser,
		hashPassword:             dependencies.HashPassword,
		resolveProjectIDForWrite: dependencies.ResolveProjectIDForWrite,
		createEngram:             dependencies.CreateEngram,
		listEngrams:              dependencies.ListEngrams,
		queryEngrams:             dependencies.QueryEngrams,
		getRehydrationBundle:     dependencies.GetRehydrationBundle,
		getEngramSources:         dependencies.GetEngramSources,
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
	router.Get("/api/v1/users", deps.handleListUsers)
	router.Post("/api/v1/users", deps.handleCreateUser)
	router.Patch("/api/v1/users/{user_id}", deps.handleUpdateUser)
	router.Post("/api/v1/engrams", deps.handleCreateEngram)
	router.Get("/api/v1/engrams", deps.handleListEngrams)
	router.Post("/api/v1/engrams/query", deps.handleQueryEngrams)
	router.Get("/api/v1/engrams/{engram_id}/sources", deps.handleListEngramSources)
	router.Get("/api/v1/engrams/{engram_id}/rehydrate", deps.handleRehydrateEngram)
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
	if !dependencies.validateCoreDependencies(writer) || !dependencies.hasLoginAuthenticator() {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "session auth dependencies are not configured"})
		return false
	}
	return true
}

func (dependencies sessionAuthDependencies) hasLoginAuthenticator() bool {
	return dependencies.lookupUserByUsername != nil && dependencies.verifyPassword != nil
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
	if !dependencies.validateCSRFToken(request, csrfToken, true) {
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
	if !isAuthenticatedLoginRecord(record, password, dependencies.verifyPassword) {
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
	if !dependencies.validateCSRFToken(request, csrfToken, false) {
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
	record, ok := dependencies.requireAuthenticatedAPIActor(writer, request)
	if !ok {
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

func (dependencies sessionAuthDependencies) logAuditEventRequest(request *http.Request, event sessionAuditEvent) {
	if dependencies.logAuditEvent == nil {
		return
	}
	dependencies.logAuditEvent(
		request,
		event.eventType,
		event.success,
		event.username,
		event.detail,
		event.metadata,
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

func isAuthenticatedLoginRecord(
	record *models.UserAuthRecord,
	password string,
	verifyPassword func(password, encodedHash string) bool,
) bool {
	if record == nil {
		return false
	}
	if !record.IsActive {
		return false
	}
	if verifyPassword == nil {
		return false
	}
	return verifyPassword(password, record.PasswordHash)
}

func (dependencies sessionAuthDependencies) validateCSRFToken(
	request *http.Request,
	csrfToken string,
	required bool,
) bool {
	state, err := dependencies.manager.DecodeRequest(request)
	if err != nil {
		return !required
	}
	expectedToken := strings.TrimSpace(state.CSRFToken)
	if expectedToken == "" {
		return !required
	}
	return csrfToken == expectedToken
}

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
	OIDCProvider             auth.OIDCLoginProvider
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
	ShareEngram func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		engramID uuid.UUID,
	) (*models.EngramVisibilityRecord, error)
	UnshareEngram func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		engramID uuid.UUID,
	) (*models.EngramVisibilityRecord, error)
	CreateEngramLink func(
		ctx context.Context,
		input SessionEngramLinkCreateInput,
	) (*models.EngramLinkRecord, error)
	ListEngramLinks func(
		ctx context.Context,
		input SessionEngramLinkListInput,
	) ([]models.EngramLinkRecord, error)
	UpdateEngramLink func(
		ctx context.Context,
		input SessionEngramLinkUpdateInput,
	) (*models.EngramLinkRecord, error)
	ArchiveEngramLink func(
		ctx context.Context,
		input SessionEngramLinkArchiveInput,
	) (*models.EngramLinkRecord, error)
	SuggestEngramLinks func(
		ctx context.Context,
		input SessionEngramLinkSuggestInput,
	) ([]models.EngramLinkSuggestion, error)
	TraceEngramLinks func(
		ctx context.Context,
		input SessionEngramTraceInput,
	) ([]models.EngramLinkTraversalStep, error)
	HygieneEngramLinks func(
		ctx context.Context,
		input SessionEngramLinkHygieneInput,
	) ([]models.EngramLinkHygieneRecommendation, error)
	CreateTokenForOwner func(
		ctx context.Context,
		ownerUserID uuid.UUID,
		payload models.MCPTokenCreateRequest,
		pepper string,
	) (*models.MCPTokenCreateResponse, error)
	ListTokenSummaries func(
		ctx context.Context,
		ownerUserID uuid.UUID,
		limit int,
		offset int,
	) ([]models.MCPTokenSummary, error)
	RevokeTokenForOwner func(
		ctx context.Context,
		tokenID uuid.UUID,
		ownerUserID uuid.UUID,
	) (*models.MCPTokenSummary, error)
	MCPTokenPepper string
}

type sessionAuthDependencies struct {
	manager                  *auth.SessionManager
	oidcProvider             auth.OIDCLoginProvider
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
	shareEngram func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		engramID uuid.UUID,
	) (*models.EngramVisibilityRecord, error)
	unshareEngram func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		engramID uuid.UUID,
	) (*models.EngramVisibilityRecord, error)
	createEngramLink func(
		ctx context.Context,
		input SessionEngramLinkCreateInput,
	) (*models.EngramLinkRecord, error)
	listEngramLinks func(
		ctx context.Context,
		input SessionEngramLinkListInput,
	) ([]models.EngramLinkRecord, error)
	updateEngramLink func(
		ctx context.Context,
		input SessionEngramLinkUpdateInput,
	) (*models.EngramLinkRecord, error)
	archiveEngramLink func(
		ctx context.Context,
		input SessionEngramLinkArchiveInput,
	) (*models.EngramLinkRecord, error)
	suggestEngramLinks func(
		ctx context.Context,
		input SessionEngramLinkSuggestInput,
	) ([]models.EngramLinkSuggestion, error)
	traceEngramLinks func(
		ctx context.Context,
		input SessionEngramTraceInput,
	) ([]models.EngramLinkTraversalStep, error)
	hygieneEngramLinks func(
		ctx context.Context,
		input SessionEngramLinkHygieneInput,
	) ([]models.EngramLinkHygieneRecommendation, error)
	createTokenForOwner func(
		ctx context.Context,
		ownerUserID uuid.UUID,
		payload models.MCPTokenCreateRequest,
		pepper string,
	) (*models.MCPTokenCreateResponse, error)
	listTokenSummaries func(
		ctx context.Context,
		ownerUserID uuid.UUID,
		limit int,
		offset int,
	) ([]models.MCPTokenSummary, error)
	revokeTokenForOwner func(
		ctx context.Context,
		tokenID uuid.UUID,
		ownerUserID uuid.UUID,
	) (*models.MCPTokenSummary, error)
	mcpTokenPepper string
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
		oidcProvider:             dependencies.OIDCProvider,
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
		shareEngram:              dependencies.ShareEngram,
		unshareEngram:            dependencies.UnshareEngram,
		createEngramLink:         dependencies.CreateEngramLink,
		listEngramLinks:          dependencies.ListEngramLinks,
		updateEngramLink:         dependencies.UpdateEngramLink,
		archiveEngramLink:        dependencies.ArchiveEngramLink,
		suggestEngramLinks:       dependencies.SuggestEngramLinks,
		traceEngramLinks:         dependencies.TraceEngramLinks,
		hygieneEngramLinks:       dependencies.HygieneEngramLinks,
		createTokenForOwner:      dependencies.CreateTokenForOwner,
		listTokenSummaries:       dependencies.ListTokenSummaries,
		revokeTokenForOwner:      dependencies.RevokeTokenForOwner,
		mcpTokenPepper:           dependencies.MCPTokenPepper,
	}
}

// MountSessionAuthRoutes registers session login/logout/csrf/me endpoints.
func MountSessionAuthRoutes(router chi.Router, dependencies SessionAuthDependencies) {
	deps := newSessionAuthDependencies(dependencies)
	router.Route("/api/v1/session", func(session chi.Router) {
		session.Get("/csrf", deps.handleCSRF)
		session.Post("/login", deps.handleLogin)
		session.Post("/logout", deps.handleLogout)
		session.Get("/oidc/start", deps.handleOIDCStart)
		session.Get("/oidc/callback", deps.handleOIDCCallback)
	})

	router.Get("/api/v1/me", deps.handleCurrentUser)
	router.Get("/api/v1/users", deps.handleListUsers)
	router.Post("/api/v1/users", deps.handleCreateUser)
	router.Patch("/api/v1/users/{user_id}", deps.handleUpdateUser)
	router.Post("/api/v1/mcp/tokens", deps.handleCreateMCPToken)
	router.Get("/api/v1/mcp/tokens", deps.handleListMCPTokens)
	router.Post("/api/v1/mcp/tokens/{token_id}/revoke", deps.handleRevokeMCPToken)
	router.Post("/api/v1/engrams", deps.handleCreateEngram)
	router.Get("/api/v1/engrams", deps.handleListEngrams)
	router.Post("/api/v1/engrams/query", deps.handleQueryEngrams)
	router.Get("/api/v1/engrams/{engram_id}/sources", deps.handleListEngramSources)
	router.Get("/api/v1/engrams/{engram_id}/rehydrate", deps.handleRehydrateEngram)
	router.Post("/api/v1/engrams/{engram_id}/share", deps.handleShareEngram)
	router.Post("/api/v1/engrams/{engram_id}/unshare", deps.handleUnshareEngram)
	router.Post("/api/v1/engrams/{engram_id}/links", deps.handleCreateEngramLink)
	router.Get("/api/v1/engrams/{engram_id}/links", deps.handleListEngramLinks)
	router.Patch("/api/v1/engrams/links/{link_id}", deps.handleUpdateEngramLink)
	router.Delete("/api/v1/engrams/links/{link_id}", deps.handleArchiveEngramLink)
	router.Post("/api/v1/engrams/{engram_id}/links/suggest", deps.handleSuggestEngramLinks)
	router.Post("/api/v1/engrams/{engram_id}/links/hygiene", deps.handleHygieneEngramLinks)
	router.Post("/api/v1/engrams/{engram_id}/trace", deps.handleTraceEngramLinks)
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

func (dependencies sessionAuthDependencies) handleOIDCStart(writer http.ResponseWriter, request *http.Request) {
	if !dependencies.validateCoreDependencies(writer) {
		return
	}
	if !dependencies.requireOIDCProvider(writer) {
		return
	}
	state, ok := dependencies.ensureCSRFSessionState(writer, request)
	if !ok {
		return
	}
	oidcState, oidcNonce, ok := dependencies.issueOIDCPendingState(writer)
	if !ok {
		return
	}
	nextPath := safeNextPath(request.URL.Query().Get("next"))
	if nextPath == "" {
		nextPath = "/ui"
	}
	state.OIDCState = oidcState
	state.OIDCNonce = oidcNonce
	state.OIDCNext = nextPath
	if !dependencies.writeSessionCookie(writer, state) {
		return
	}
	redirectURL, err := dependencies.oidcProvider.AuthCodeURL(oidcState, oidcNonce)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "failed to start oidc login"})
		return
	}
	dependencies.logAuditEventRequest(
		request,
		sessionAuditEvent{
			eventType: "oidc_login_start",
			success:   true,
			detail:    nextPath,
		},
	)
	http.Redirect(writer, request, redirectURL, http.StatusSeeOther)
}

func (dependencies sessionAuthDependencies) handleOIDCCallback(writer http.ResponseWriter, request *http.Request) {
	if !dependencies.validateCoreDependencies(writer) {
		return
	}
	if !dependencies.requireOIDCProvider(writer) {
		return
	}
	decodedState, code, ok := dependencies.resolveOIDCCallbackRequest(writer, request)
	if !ok {
		return
	}
	if !dependencies.consumeOIDCPendingState(writer, decodedState) {
		return
	}
	identity, err := dependencies.oidcProvider.AuthenticateCode(request.Context(), code, decodedState.OIDCNonce)
	if err != nil {
		dependencies.logOIDCFailure(
			request,
			"token_exchange_or_verification_failed",
			func() {
				http.Redirect(writer, request, "/login", http.StatusSeeOther)
			},
		)
		return
	}
	record, failureDetail, ok := dependencies.lookupAuthenticatedOIDCUser(writer, request, identity.Username)
	if !ok {
		dependencies.logAuditEventRequest(
			request,
			sessionAuditEvent{
				eventType: "oidc_login_failed",
				success:   false,
				username:  strings.TrimSpace(identity.Username),
				detail:    failureDetail,
				metadata: map[string]any{
					"subject": identity.Subject,
					"email":   identity.Email,
				},
			},
		)
		return
	}
	dependencies.logAuditEventRequest(
		request,
		sessionAuditEvent{
			eventType: "oidc_login_success",
			success:   true,
			username:  record.Username,
			metadata: map[string]any{
				"subject": identity.Subject,
				"email":   identity.Email,
			},
		},
	)
	dependencies.writeUILoginSuccess(writer, request, record, decodedState.OIDCNext)
}

func (dependencies sessionAuthDependencies) requireOIDCProvider(writer http.ResponseWriter) bool {
	if dependencies.hasOIDCProvider() {
		return true
	}
	writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "oidc login is not enabled"})
	return false
}

func (dependencies sessionAuthDependencies) resolveOIDCCallbackRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (auth.SessionState, string, bool) {
	decodedState, err := dependencies.manager.DecodeRequest(request)
	if err != nil {
		http.Redirect(writer, request, "/login", http.StatusSeeOther)
		return auth.SessionState{}, "", false
	}
	providerState := strings.TrimSpace(request.URL.Query().Get("state"))
	if providerState == "" || providerState != strings.TrimSpace(decodedState.OIDCState) {
		dependencies.logOIDCFailure(
			request,
			"state_mismatch",
			func() {
				writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "invalid oidc state"})
			},
		)
		return auth.SessionState{}, "", false
	}
	code := strings.TrimSpace(request.URL.Query().Get("code"))
	if code == "" {
		dependencies.logOIDCFailure(
			request,
			"missing_code",
			func() {
				writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "missing oidc authorization code"})
			},
		)
		return auth.SessionState{}, "", false
	}
	return decodedState, code, true
}

func (dependencies sessionAuthDependencies) logOIDCFailure(
	request *http.Request,
	detail string,
	onFailure func(),
) {
	dependencies.logAuditEventRequest(
		request,
		sessionAuditEvent{
			eventType: "oidc_login_failed",
			success:   false,
			detail:    detail,
		},
	)
	onFailure()
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

func (dependencies sessionAuthDependencies) hasOIDCProvider() bool {
	return dependencies.oidcProvider != nil && dependencies.oidcProvider.Enabled()
}

func (dependencies sessionAuthDependencies) issueOIDCPendingState(
	writer http.ResponseWriter,
) (string, string, bool) {
	oidcState, err := dependencies.generateCSRFToken()
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "failed to generate oidc state"})
		return "", "", false
	}
	oidcNonce, err := dependencies.generateCSRFToken()
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "failed to generate oidc nonce"})
		return "", "", false
	}
	return oidcState, oidcNonce, true
}

func (dependencies sessionAuthDependencies) consumeOIDCPendingState(
	writer http.ResponseWriter,
	state auth.SessionState,
) bool {
	state.OIDCState = ""
	state.OIDCNonce = ""
	state.OIDCNext = ""
	return dependencies.writeSessionCookie(writer, state)
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

func (dependencies sessionAuthDependencies) lookupAuthenticatedOIDCUser(
	writer http.ResponseWriter,
	request *http.Request,
	username string,
) (*models.UserAuthRecord, string, bool) {
	if dependencies.lookupUserByUsername == nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "oidc user lookup is not configured"})
		return nil, "user_lookup_not_configured", false
	}
	normalizedUsername := strings.TrimSpace(username)
	if normalizedUsername == "" {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"detail": "oidc identity missing username"})
		return nil, "identity_missing_username", false
	}
	record, err := dependencies.lookupUserByUsername(request.Context(), normalizedUsername)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return nil, "user_lookup_error", false
	}
	if record == nil || !record.IsActive {
		writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "oidc user is not authorized"})
		return nil, "user_not_authorized", false
	}
	return record, "", true
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

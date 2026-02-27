package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"engram/internal/audit"
	"engram/internal/auth"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const sessionUITestPassword = "StrongPassword-12345"

var csrfTokenInputPattern = regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)

type sessionUIFormRequestSpec struct {
	Method string
	Path   string
	Values url.Values
	Cookie *http.Cookie
}

type sessionUILoginCredentials struct {
	Username string
	Password string
}

func assertRedirect(t *testing.T, response *httptest.ResponseRecorder, expectedLocation string) {
	t.Helper()
	if response.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", response.Code)
	}
	if response.Header().Get("Location") != expectedLocation {
		t.Fatalf("expected redirect to %q, got %q", expectedLocation, response.Header().Get("Location"))
	}
}

func assertRouteRedirectsToLogin(t *testing.T, handler http.Handler, path string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	assertRedirect(t, response, "/login")
}

func loginSessionUIUser(
	t *testing.T,
	handler http.Handler,
	manager *auth.SessionManager,
	credentials sessionUILoginCredentials,
) *http.Cookie {
	t.Helper()
	csrfToken, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)
	loginValues := url.Values{}
	loginValues.Set("username", credentials.Username)
	loginValues.Set("password", credentials.Password)
	loginValues.Set("csrf_token", csrfToken)

	loginRequest := newSessionUIFormRequest(t, sessionUIFormRequestSpec{
		Method: http.MethodPost,
		Path:   "/login",
		Values: loginValues,
		Cookie: loginCookie,
	})
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, loginRequest)
	assertRedirect(t, loginResponse, "/ui")
	authenticatedCookie := findResponseCookie(loginResponse, manager.CookieName())
	if authenticatedCookie == nil {
		t.Fatalf("expected session cookie after login")
	}
	return authenticatedCookie
}

func TestMountSessionUIRoutesHomeRedirectsToLoginWhenUnauthenticated(t *testing.T) {
	handler, _, _ := buildSessionUITestHandler(t)
	assertRouteRedirectsToLogin(t, handler, "/")
}

func TestMountSessionUIRoutesDashboardRedirectsToLoginWhenUnauthenticated(t *testing.T) {
	handler, _, _ := buildSessionUITestHandler(t)
	assertRouteRedirectsToLogin(t, handler, "/ui")
}

func TestMountSessionUIRoutesAdminRedirectsToLoginWhenUnauthenticated(t *testing.T) {
	handler, _, _ := buildSessionUITestHandler(t)
	assertRouteRedirectsToLogin(t, handler, "/ui/admin")
}

func TestMountSessionUIRoutesLoginRejectsInvalidCredentials(t *testing.T) {
	handler, manager, _ := buildSessionUITestHandler(t)
	csrfToken, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)

	values := url.Values{}
	values.Set("username", "wrong")
	values.Set("password", "wrong")
	values.Set("csrf_token", csrfToken)

	request := newSessionUIFormRequest(t, sessionUIFormRequestSpec{
		Method: http.MethodPost,
		Path:   "/login",
		Values: values,
		Cookie: loginCookie,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "Invalid username or password.") {
		t.Fatalf("expected invalid credentials message in response body")
	}
}

func TestMountSessionUIRoutesLoginRejectsInvalidCSRF(t *testing.T) {
	handler, manager, record := buildSessionUITestHandler(t)
	_, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)

	values := url.Values{}
	values.Set("username", record.Username)
	values.Set("password", sessionUITestPassword)
	values.Set("csrf_token", "bad-token")

	request := newSessionUIFormRequest(t, sessionUIFormRequestSpec{
		Method: http.MethodPost,
		Path:   "/login",
		Values: values,
		Cookie: loginCookie,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", response.Code)
	}
}

func TestMountSessionUIRoutesLoginRateLimitAfterRepeatedFailures(t *testing.T) {
	handler, manager, _ := buildSessionUITestHandler(t)

	for attempt := 0; attempt < 5; attempt++ {
		csrfToken, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)
		values := url.Values{}
		values.Set("username", "admin")
		values.Set("password", "wrong")
		values.Set("csrf_token", csrfToken)
		request := newSessionUIFormRequest(t, sessionUIFormRequestSpec{
			Method: http.MethodPost,
			Path:   "/login",
			Values: values,
			Cookie: loginCookie,
		})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", response.Code)
		}
	}

	csrfToken, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)
	blockedValues := url.Values{}
	blockedValues.Set("username", "admin")
	blockedValues.Set("password", "wrong")
	blockedValues.Set("csrf_token", csrfToken)
	blockedRequest := newSessionUIFormRequest(t, sessionUIFormRequestSpec{
		Method: http.MethodPost,
		Path:   "/login",
		Values: blockedValues,
		Cookie: loginCookie,
	})
	blockedResponse := httptest.NewRecorder()
	handler.ServeHTTP(blockedResponse, blockedRequest)
	if blockedResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", blockedResponse.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(blockedResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json payload, got %v", err)
	}
	if !strings.Contains(payload["detail"], "Too many login attempts") {
		t.Fatalf("expected rate-limit detail message, got %q", payload["detail"])
	}
}

func TestMountSessionUIRoutesLoginAndLogoutWorkflow(t *testing.T) {
	handler, manager, record := buildSessionUITestHandler(t)
	authenticatedCookie := loginSessionUIUser(t, handler, manager, sessionUILoginCredentials{
		Username: record.Username,
		Password: sessionUITestPassword,
	})

	dashboardRequest := httptest.NewRequest(http.MethodGet, "/ui", nil)
	dashboardRequest.AddCookie(authenticatedCookie)
	dashboardResponse := httptest.NewRecorder()
	handler.ServeHTTP(dashboardResponse, dashboardRequest)
	if dashboardResponse.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", dashboardResponse.Code)
	}
	if !strings.Contains(dashboardResponse.Body.String(), "Engram Vault Test UI") {
		t.Fatalf("expected dashboard title in response body")
	}
	if !strings.Contains(dashboardResponse.Body.String(), record.Username) {
		t.Fatalf("expected username %q in dashboard", record.Username)
	}
	logoutCSRFToken := extractCSRFTokenFromHTML(t, dashboardResponse.Body.String())
	dashboardCookie := findResponseCookie(dashboardResponse, manager.CookieName())
	if dashboardCookie != nil {
		authenticatedCookie = dashboardCookie
	}

	logoutValues := url.Values{}
	logoutValues.Set("csrf_token", logoutCSRFToken)
	logoutRequest := newSessionUIFormRequest(t, sessionUIFormRequestSpec{
		Method: http.MethodPost,
		Path:   "/logout",
		Values: logoutValues,
		Cookie: authenticatedCookie,
	})
	logoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(logoutResponse, logoutRequest)
	assertRedirect(t, logoutResponse, "/login")
	clearedCookie := findResponseCookie(logoutResponse, manager.CookieName())
	if clearedCookie == nil {
		t.Fatalf("expected clear-session cookie after logout")
	}

	blockedDashboardRequest := httptest.NewRequest(http.MethodGet, "/ui", nil)
	blockedDashboardRequest.AddCookie(clearedCookie)
	blockedDashboardResponse := httptest.NewRecorder()
	handler.ServeHTTP(blockedDashboardResponse, blockedDashboardRequest)
	assertRedirect(t, blockedDashboardResponse, "/login")
}

func TestMountSessionUIRoutesAdminRejectsNonAdminRole(t *testing.T) {
	handler, manager, record := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{
			userRole: models.UserRoleViewer,
		},
	)
	authenticatedCookie := loginSessionUIUser(t, handler, manager, sessionUILoginCredentials{
		Username: record.Username,
		Password: sessionUITestPassword,
	})

	adminRequest := httptest.NewRequest(http.MethodGet, "/ui/admin", nil)
	adminRequest.AddCookie(authenticatedCookie)
	adminResponse := httptest.NewRecorder()
	handler.ServeHTTP(adminResponse, adminRequest)
	if adminResponse.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", adminResponse.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(adminResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json payload, got %v", err)
	}
	if payload["detail"] != "Admin role required" {
		t.Fatalf("expected admin role detail, got %q", payload["detail"])
	}
}

func TestMountSessionUIRoutesAdminAllowsAdminRole(t *testing.T) {
	handler, manager, record := buildSessionUITestHandler(t)
	authenticatedCookie := loginSessionUIUser(t, handler, manager, sessionUILoginCredentials{
		Username: record.Username,
		Password: sessionUITestPassword,
	})

	adminRequest := httptest.NewRequest(http.MethodGet, "/ui/admin", nil)
	adminRequest.AddCookie(authenticatedCookie)
	adminResponse := httptest.NewRecorder()
	handler.ServeHTTP(adminResponse, adminRequest)
	if adminResponse.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", adminResponse.Code)
	}
	if !strings.Contains(adminResponse.Body.String(), "Engram Vault Admin") {
		t.Fatalf("expected admin page title")
	}
}

func TestMountSessionUIRoutesAdminCreateMCPTokenRedirectsOnSuccess(t *testing.T) {
	var (
		capturedOwnerUserID uuid.UUID
		capturedPayload     models.MCPTokenCreateRequest
	)
	handler, manager, record := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{
			mcpTokenPepper: "ui-admin-test-pepper",
			createTokenForOwner: func(
				_ context.Context,
				ownerUserID uuid.UUID,
				payload models.MCPTokenCreateRequest,
				pepper string,
			) (*models.MCPTokenCreateResponse, error) {
				capturedOwnerUserID = ownerUserID
				capturedPayload = payload
				if pepper != "ui-admin-test-pepper" {
					t.Fatalf("expected configured pepper, got %q", pepper)
				}
				return &models.MCPTokenCreateResponse{
					TokenID:           uuid.MustParse("00000000-0000-0000-0000-000000000990"),
					Name:              payload.Name,
					Scope:             models.MCPTokenScopeWrite,
					AllowedTools:      payload.AllowedTools,
					AllowedProjectIDs: payload.AllowedProjectIDs,
					TokenSecretHint:   "abc123...xyz9",
					Token:             "engram_mcp_token",
					ExpiresAt:         time.Unix(1_800_000_000, 0).UTC(),
					CreatedAt:         time.Unix(1_700_000_000, 0).UTC(),
				}, nil
			},
		},
	)
	authenticatedCookie := loginSessionUIUser(t, handler, manager, sessionUILoginCredentials{
		Username: record.Username,
		Password: sessionUITestPassword,
	})
	adminCSRFToken, authenticatedCookie := fetchAdminCSRFTokenAndCookie(t, handler, manager, authenticatedCookie)

	createValues := url.Values{}
	createValues.Set("csrf_token", adminCSRFToken)
	createValues.Set("name", "UI Token")
	createValues.Set("scope", "write")
	createValues.Set("allowed_tools", "chat_send_message, chat_send_message, project_list")
	createValues.Set("allowed_project_ids", "project-1, project-2")
	createValues.Set("expires_in_days", "30")

	createRequest := newSessionUIFormRequest(t, sessionUIFormRequestSpec{
		Method: http.MethodPost,
		Path:   "/ui/admin/mcp-tokens/create",
		Values: createValues,
		Cookie: authenticatedCookie,
	})
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)
	assertRedirect(t, createResponse, "/ui/admin")

	if capturedOwnerUserID != record.UserID {
		t.Fatalf("expected owner_user_id %s, got %s", record.UserID, capturedOwnerUserID)
	}
	if capturedPayload.Name != "UI Token" {
		t.Fatalf("expected token name to be forwarded, got %q", capturedPayload.Name)
	}
	if capturedPayload.Scope != "write" {
		t.Fatalf("expected write scope, got %q", capturedPayload.Scope)
	}
	if capturedPayload.ExpiresInDays != 30 {
		t.Fatalf("expected expires_in_days 30, got %d", capturedPayload.ExpiresInDays)
	}
	if len(capturedPayload.AllowedTools) != 2 {
		t.Fatalf("expected deduplicated allowed_tools, got %#v", capturedPayload.AllowedTools)
	}
}

func TestMountSessionUIRoutesAdminCreateMCPTokenRejectsInvalidCSRF(t *testing.T) {
	handler, manager, record := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{
			mcpTokenPepper: "ui-admin-test-pepper",
			createTokenForOwner: func(
				_ context.Context,
				_ uuid.UUID,
				_ models.MCPTokenCreateRequest,
				_ string,
			) (*models.MCPTokenCreateResponse, error) {
				t.Fatalf("createTokenForOwner should not be called when csrf is invalid")
				return nil, nil
			},
		},
	)
	authenticatedCookie := loginSessionUIUser(t, handler, manager, sessionUILoginCredentials{
		Username: record.Username,
		Password: sessionUITestPassword,
	})

	createValues := url.Values{}
	createValues.Set("csrf_token", "bad-token")
	createValues.Set("name", "UI Token")
	createValues.Set("scope", "read")

	createRequest := newSessionUIFormRequest(t, sessionUIFormRequestSpec{
		Method: http.MethodPost,
		Path:   "/ui/admin/mcp-tokens/create",
		Values: createValues,
		Cookie: authenticatedCookie,
	})
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", createResponse.Code)
	}
}

func TestMountSessionUIRoutesAdminRevokeMCPTokenRedirectsOnSuccess(t *testing.T) {
	var (
		capturedTokenID     uuid.UUID
		capturedOwnerUserID uuid.UUID
	)
	handler, manager, record := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{
			revokeTokenForOwner: func(
				_ context.Context,
				tokenID uuid.UUID,
				ownerUserID uuid.UUID,
			) (*models.MCPTokenSummary, error) {
				capturedTokenID = tokenID
				capturedOwnerUserID = ownerUserID
				return &models.MCPTokenSummary{
					TokenID: tokenID,
					Name:    "UI Token",
					Scope:   models.MCPTokenScopeRead,
				}, nil
			},
		},
	)
	authenticatedCookie := loginSessionUIUser(t, handler, manager, sessionUILoginCredentials{
		Username: record.Username,
		Password: sessionUITestPassword,
	})
	adminCSRFToken, authenticatedCookie := fetchAdminCSRFTokenAndCookie(t, handler, manager, authenticatedCookie)

	tokenID := uuid.MustParse("00000000-0000-0000-0000-000000000991")
	revokeValues := url.Values{}
	revokeValues.Set("csrf_token", adminCSRFToken)
	revokeValues.Set("reason", "cleanup")
	revokeRequest := newSessionUIFormRequest(t, sessionUIFormRequestSpec{
		Method: http.MethodPost,
		Path:   "/ui/admin/mcp-tokens/" + tokenID.String() + "/revoke",
		Values: revokeValues,
		Cookie: authenticatedCookie,
	})
	revokeResponse := httptest.NewRecorder()
	handler.ServeHTTP(revokeResponse, revokeRequest)
	assertRedirect(t, revokeResponse, "/ui/admin")

	if capturedTokenID != tokenID {
		t.Fatalf("expected token id %s, got %s", tokenID, capturedTokenID)
	}
	if capturedOwnerUserID != record.UserID {
		t.Fatalf("expected owner_user_id %s, got %s", record.UserID, capturedOwnerUserID)
	}
}

func TestMountSessionUIRoutesAdminRevokeMCPTokenReturnsNotFoundWhenMissing(t *testing.T) {
	handler, manager, record := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{
			revokeTokenForOwner: func(
				_ context.Context,
				_ uuid.UUID,
				_ uuid.UUID,
			) (*models.MCPTokenSummary, error) {
				return nil, nil
			},
		},
	)
	authenticatedCookie := loginSessionUIUser(t, handler, manager, sessionUILoginCredentials{
		Username: record.Username,
		Password: sessionUITestPassword,
	})
	adminCSRFToken, authenticatedCookie := fetchAdminCSRFTokenAndCookie(t, handler, manager, authenticatedCookie)

	revokeValues := url.Values{}
	revokeValues.Set("csrf_token", adminCSRFToken)
	revokeRequest := newSessionUIFormRequest(t, sessionUIFormRequestSpec{
		Method: http.MethodPost,
		Path:   "/ui/admin/mcp-tokens/00000000-0000-0000-0000-000000000992/revoke",
		Values: revokeValues,
		Cookie: authenticatedCookie,
	})
	revokeResponse := httptest.NewRecorder()
	handler.ServeHTTP(revokeResponse, revokeRequest)

	if revokeResponse.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", revokeResponse.Code)
	}
}

func TestMountSessionUIRoutesLoginRedirectPathSanitization(t *testing.T) {
	testCases := []struct {
		name             string
		nextPath         string
		expectedRedirect string
	}{
		{name: "trusted relative path", nextPath: "/ui/admin", expectedRedirect: "/ui/admin"},
		{name: "absolute url rejected", nextPath: "https://malicious.example/phish", expectedRedirect: "/ui"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler, manager, record := buildSessionUITestHandler(t)
			csrfToken, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)

			values := url.Values{}
			values.Set("username", record.Username)
			values.Set("password", sessionUITestPassword)
			values.Set("csrf_token", csrfToken)
			values.Set("next_path", testCase.nextPath)

			request := newSessionUIFormRequest(t, sessionUIFormRequestSpec{
				Method: http.MethodPost,
				Path:   "/login",
				Values: values,
				Cookie: loginCookie,
			})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			assertRedirect(t, response, testCase.expectedRedirect)
		})
	}
}

func TestMountSessionUIRoutesLogoutRejectsInvalidCSRF(t *testing.T) {
	handler, manager, record := buildSessionUITestHandler(t)
	authenticatedCookie := loginSessionUIUser(t, handler, manager, sessionUILoginCredentials{
		Username: record.Username,
		Password: sessionUITestPassword,
	})

	badLogoutValues := url.Values{}
	badLogoutValues.Set("csrf_token", "bad-token")
	badLogoutRequest := newSessionUIFormRequest(t, sessionUIFormRequestSpec{
		Method: http.MethodPost,
		Path:   "/logout",
		Values: badLogoutValues,
		Cookie: authenticatedCookie,
	})
	badLogoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(badLogoutResponse, badLogoutRequest)
	if badLogoutResponse.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", badLogoutResponse.Code)
	}

	stillAuthenticatedRequest := httptest.NewRequest(http.MethodGet, "/ui", nil)
	stillAuthenticatedRequest.AddCookie(authenticatedCookie)
	stillAuthenticatedResponse := httptest.NewRecorder()
	handler.ServeHTTP(stillAuthenticatedResponse, stillAuthenticatedRequest)
	if stillAuthenticatedResponse.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", stillAuthenticatedResponse.Code)
	}
}

func TestMountSessionUIRoutesLoginFailureWritesAuditLog(t *testing.T) {
	auditLogPath := filepath.Join(t.TempDir(), "audit.log")
	handler, manager, _ := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{auditLogPath: auditLogPath},
	)
	csrfToken, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)

	values := url.Values{}
	values.Set("username", "admin")
	values.Set("password", "wrong")
	values.Set("csrf_token", csrfToken)
	request := newSessionUIFormRequest(t, sessionUIFormRequestSpec{
		Method: http.MethodPost,
		Path:   "/login",
		Values: values,
		Cookie: loginCookie,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}

	content, err := os.ReadFile(auditLogPath)
	if err != nil {
		t.Fatalf("expected audit log to be written: %v", err)
	}
	if !strings.Contains(string(content), "\"event_type\":\"login_failed\"") {
		t.Fatalf("expected login_failed audit event in log")
	}
}

type sessionUITestHandlerOptions struct {
	auditLogPath        string
	loginAttemptGuard   SessionLoginAttemptGuard
	userRole            models.UserRole
	createTokenForOwner func(
		ctx context.Context,
		ownerUserID uuid.UUID,
		payload models.MCPTokenCreateRequest,
		pepper string,
	) (*models.MCPTokenCreateResponse, error)
	revokeTokenForOwner func(
		ctx context.Context,
		tokenID uuid.UUID,
		ownerUserID uuid.UUID,
	) (*models.MCPTokenSummary, error)
	mcpTokenPepper string
}

func buildSessionUITestHandler(
	t *testing.T,
	options ...sessionUITestHandlerOptions,
) (http.Handler, *auth.SessionManager, *models.UserAuthRecord) {
	t.Helper()
	handlerOptions := resolveSessionUITestHandlerOptions(options)
	manager := newSessionUITestSessionManager(t)
	record := newSessionUITestUserRecord(t, handlerOptions.userRole)
	lookupByUsername, lookupByID := newSessionUITestLookups(record)
	router := chi.NewRouter()
	dependencies := SessionAuthDependencies{
		SessionManager:       manager,
		LookupUserByUsername: lookupByUsername,
		LookupUserByID:       lookupByID,
		VerifyPassword:       auth.VerifyPassword,
		GenerateCSRFToken:    auth.GenerateCSRFToken,
		LoginAttemptGuard:    resolveSessionUITestLoginAttemptGuard(handlerOptions.loginAttemptGuard),
		LogAuditEvent:        newSessionUITestAuditLogger(handlerOptions.auditLogPath),
		CreateTokenForOwner:  handlerOptions.createTokenForOwner,
		RevokeTokenForOwner:  handlerOptions.revokeTokenForOwner,
		MCPTokenPepper:       handlerOptions.mcpTokenPepper,
	}
	MountSessionAuthRoutes(router, dependencies)
	MountSessionUIRoutes(router, dependencies)
	return SessionActorMiddleware(manager, lookupByID)(router), manager, record
}

func resolveSessionUITestHandlerOptions(options []sessionUITestHandlerOptions) sessionUITestHandlerOptions {
	if len(options) == 0 {
		return sessionUITestHandlerOptions{}
	}
	return options[0]
}

func newSessionUITestSessionManager(t *testing.T) *auth.SessionManager {
	t.Helper()
	manager, err := auth.NewSessionManager("dev-session-secret-for-tests", auth.DefaultSessionCookieName)
	if err != nil {
		t.Fatalf("expected manager creation to succeed: %v", err)
	}
	return manager
}

func newSessionUITestUserRecord(t *testing.T, role models.UserRole) *models.UserAuthRecord {
	t.Helper()
	passwordHash, err := auth.HashPassword(sessionUITestPassword, []byte{0x01, 0x02, 0x03, 0x04})
	if err != nil {
		t.Fatalf("expected password hashing to succeed: %v", err)
	}
	if role == "" {
		role = models.UserRoleAdmin
	}
	return &models.UserAuthRecord{
		UserID:       uuid.MustParse("00000000-0000-0000-0000-000000000712"),
		Username:     "admin",
		PasswordHash: passwordHash,
		Role:         role,
		IsActive:     true,
	}
}

func newSessionUITestLookups(
	record *models.UserAuthRecord,
) (SessionUserByUsernameLookup, SessionUserLookup) {
	lookupByUsername := func(_ context.Context, username string) (*models.UserAuthRecord, error) {
		if username != record.Username {
			return nil, nil
		}
		return record, nil
	}
	lookupByID := func(_ context.Context, userID uuid.UUID) (*models.UserAuthRecord, error) {
		if userID != record.UserID {
			return nil, nil
		}
		return record, nil
	}
	return lookupByUsername, lookupByID
}

func resolveSessionUITestLoginAttemptGuard(guard SessionLoginAttemptGuard) SessionLoginAttemptGuard {
	if guard != nil {
		return guard
	}
	return auth.NewLoginAttemptGuard(5, 300, 900)
}

func newSessionUITestAuditLogger(path string) SessionAuditLogger {
	if path == "" {
		return nil
	}
	logger := audit.NewLogger(audit.LoggerOptions{
		Path:          path,
		MaxEventBytes: 32_768,
	})
	return func(
		request *http.Request,
		eventType string,
		success bool,
		username string,
		detail string,
		metadata map[string]any,
	) {
		_ = logger.LogRequestEvent(request, eventType, success, username, detail, metadata)
	}
}

func fetchLoginCSRFTokenAndCookie(
	t *testing.T,
	handler http.Handler,
	manager *auth.SessionManager,
	cookie *http.Cookie,
) (string, *http.Cookie) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/login", nil)
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	csrfToken := extractCSRFTokenFromHTML(t, response.Body.String())
	nextCookie := findResponseCookie(response, manager.CookieName())
	if nextCookie == nil {
		nextCookie = cookie
	}
	if nextCookie == nil {
		t.Fatalf("expected session cookie from login page")
	}
	return csrfToken, nextCookie
}

func extractCSRFTokenFromHTML(t *testing.T, html string) string {
	t.Helper()
	matches := csrfTokenInputPattern.FindStringSubmatch(html)
	if len(matches) != 2 {
		t.Fatalf("expected csrf token input in html response")
	}
	return matches[1]
}

func fetchAdminCSRFTokenAndCookie(
	t *testing.T,
	handler http.Handler,
	manager *auth.SessionManager,
	cookie *http.Cookie,
) (string, *http.Cookie) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/ui/admin", nil)
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	csrfToken := extractCSRFTokenFromHTML(t, response.Body.String())
	nextCookie := findResponseCookie(response, manager.CookieName())
	if nextCookie == nil {
		nextCookie = cookie
	}
	return csrfToken, nextCookie
}

func newSessionUIFormRequest(
	t *testing.T,
	spec sessionUIFormRequestSpec,
) *http.Request {
	t.Helper()
	request := httptest.NewRequest(spec.Method, spec.Path, strings.NewReader(spec.Values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if spec.Cookie != nil {
		request.AddCookie(spec.Cookie)
	}
	return request
}

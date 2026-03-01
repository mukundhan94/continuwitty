package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"engram/internal/auth"
	"engram/internal/models"
)

const sessionUITestOIDCAuthorizeURL = "https://accounts.example.com/oauth/authorize?client_id=engram-web"

type sessionUITestOIDCProvider struct {
	enabled          bool
	authURL          string
	authURLError     error
	identity         *auth.OIDCIdentity
	authenticateErr  error
	authURLCalls     []sessionUITestOIDCAuthURLCall
	authenticateCall []sessionUITestOIDCAuthenticateCall
}

type sessionUITestOIDCAuthURLCall struct {
	State string
	Nonce string
}

type sessionUITestOIDCAuthenticateCall struct {
	Code  string
	Nonce string
}

type oidcCallbackFailureCase struct {
	name                string
	provider            *sessionUITestOIDCProvider
	lookupByUsername    SessionUserByUsernameLookup
	expectedStatus      int
	expectedDetail      string
	expectedRedirect    string
	expectedAuditDetail string
}

type oidcIdentityFailureSpec struct {
	name                string
	subject             string
	username            string
	email               string
	lookupByUsername    SessionUserByUsernameLookup
	expectedStatus      int
	expectedDetail      string
	expectedAuditDetail string
}

func (provider *sessionUITestOIDCProvider) Enabled() bool {
	return provider != nil && provider.enabled
}

func (provider *sessionUITestOIDCProvider) AuthCodeURL(state string, nonce string) (string, error) {
	provider.authURLCalls = append(provider.authURLCalls, sessionUITestOIDCAuthURLCall{
		State: state,
		Nonce: nonce,
	})
	if provider.authURLError != nil {
		return "", provider.authURLError
	}
	if provider.authURL == "" {
		return "https://accounts.example.com/oauth/authorize", nil
	}
	return provider.authURL, nil
}

func (provider *sessionUITestOIDCProvider) AuthenticateCode(
	_ context.Context,
	code string,
	nonce string,
) (*auth.OIDCIdentity, error) {
	provider.authenticateCall = append(provider.authenticateCall, sessionUITestOIDCAuthenticateCall{
		Code:  code,
		Nonce: nonce,
	})
	if provider.authenticateErr != nil {
		return nil, provider.authenticateErr
	}
	if provider.identity != nil {
		return provider.identity, nil
	}
	return &auth.OIDCIdentity{
		Subject:  "subject-1",
		Username: "admin",
		Email:    "admin@example.com",
	}, nil
}

func TestMountSessionUIRoutesLoginPageShowsOIDCLinkWhenEnabled(t *testing.T) {
	handler, manager, _ := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{
			oidcProvider: &sessionUITestOIDCProvider{enabled: true},
		},
	)
	_, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)

	request := httptest.NewRequest(http.MethodGet, "/login?next=%2Fui%2Fadmin", nil)
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "/login/oidc?next=%2Fui%2Fadmin") {
		t.Fatalf("expected encoded oidc login link in login page")
	}
}

func TestMountSessionUIRoutesOIDCStartStoresPendingState(t *testing.T) {
	testCases := []struct {
		name             string
		path             string
		expectedNextPath string
	}{
		{
			name:             "safe next path",
			path:             "/login/oidc?next=%2Fui%2Fadmin",
			expectedNextPath: "/ui/admin",
		},
		{
			name:             "unsafe external next path",
			path:             "/login/oidc?next=https%3A%2F%2Fevil.example%2Fsteal",
			expectedNextPath: "/ui",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			oidcProvider := &sessionUITestOIDCProvider{
				enabled: true,
				authURL: sessionUITestOIDCAuthorizeURL,
			}
			handler, manager, _ := buildSessionUITestHandler(
				t,
				sessionUITestHandlerOptions{oidcProvider: oidcProvider},
			)
			_, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)

			request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
			request.AddCookie(loginCookie)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			assertRedirect(t, response, oidcProvider.authURL)
			assertOIDCStartCall(t, oidcProvider)
			assertPendingOIDCSessionState(t, manager, response, testCase.expectedNextPath)
		})
	}
}

func TestMountSessionUIRoutesOIDCStartReturnsInternalErrorWhenProviderFails(t *testing.T) {
	oidcProvider := &sessionUITestOIDCProvider{
		enabled:      true,
		authURLError: errors.New("provider unavailable"),
	}
	handler, manager, _ := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{oidcProvider: oidcProvider},
	)
	_, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)

	request := httptest.NewRequest(http.MethodGet, "/login/oidc", nil)
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	assertJSONDetail(t, response, http.StatusInternalServerError, "failed to start oidc login")
}

func TestMountSessionUIRoutesOIDCCallbackAuthenticatesAndRedirects(t *testing.T) {
	oidcProvider := &sessionUITestOIDCProvider{
		enabled: true,
		authURL: sessionUITestOIDCAuthorizeURL,
		identity: &auth.OIDCIdentity{
			Subject:  "subject-2",
			Username: "admin",
			Email:    "admin@example.com",
		},
	}
	handler, manager, _ := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{oidcProvider: oidcProvider},
	)
	pendingCookie, pendingState := startOIDCLoginFlow(t, handler, manager)

	callbackResponse := performOIDCCallbackRequest(handler, pendingCookie, pendingState.OIDCState, "auth-code-1")

	assertRedirect(t, callbackResponse, "/ui/admin")
	assertOIDCCallbackCall(t, oidcProvider, pendingState.OIDCNonce)
	assertAuthenticatedOIDCSessionState(t, manager, callbackResponse)
}

func TestMountSessionUIRoutesOIDCCallbackRejectsReplayAfterPendingStateConsumed(t *testing.T) {
	auditLogPath := filepath.Join(t.TempDir(), "oidc-callback-replay-audit.log")
	oidcProvider := &sessionUITestOIDCProvider{
		enabled: true,
		authURL: sessionUITestOIDCAuthorizeURL,
		identity: &auth.OIDCIdentity{
			Subject:  "subject-2",
			Username: "admin",
			Email:    "admin@example.com",
		},
	}
	handler, manager, _ := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{
			oidcProvider: oidcProvider,
			auditLogPath: auditLogPath,
		},
	)
	pendingCookie, pendingState := startOIDCLoginFlow(t, handler, manager)

	firstCallback := performOIDCCallbackRequest(handler, pendingCookie, pendingState.OIDCState, "auth-code-1")
	assertRedirect(t, firstCallback, "/ui/admin")
	assertOIDCCallbackCall(t, oidcProvider, pendingState.OIDCNonce)

	consumedStateCookie := findResponseCookie(firstCallback, manager.CookieName())
	if consumedStateCookie == nil {
		t.Fatalf("expected callback response to include authenticated session cookie")
	}
	replayCallback := performOIDCCallbackRequest(handler, consumedStateCookie, pendingState.OIDCState, "auth-code-2")

	assertJSONDetail(t, replayCallback, http.StatusForbidden, "invalid oidc state")
	if len(oidcProvider.authenticateCall) != 1 {
		t.Fatalf("expected oidc callback exchange to remain single-use for consumed pending state")
	}
	assertAuditLogContains(t, auditLogPath, "\"event_type\":\"oidc_login_failed\"")
	assertAuditLogContains(t, auditLogPath, "\"detail\":\"state_mismatch\"")
}

func TestMountSessionUIRoutesOIDCCallbackRejectsInvalidRequest(t *testing.T) {
	testCases := []struct {
		name                string
		providerState       string
		usePendingState     bool
		authorizationCode   string
		expectedStatus      int
		expectedDetail      string
		expectedAuditDetail string
	}{
		{
			name:                "state mismatch",
			providerState:       "invalid-state",
			authorizationCode:   "auth-code-1",
			expectedStatus:      http.StatusForbidden,
			expectedDetail:      "invalid oidc state",
			expectedAuditDetail: "state_mismatch",
		},
		{
			name:                "missing authorization code",
			usePendingState:     true,
			authorizationCode:   "",
			expectedStatus:      http.StatusBadRequest,
			expectedDetail:      "missing oidc authorization code",
			expectedAuditDetail: "missing_code",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			auditLogPath := filepath.Join(t.TempDir(), "oidc-invalid-request-audit.log")
			oidcProvider := &sessionUITestOIDCProvider{
				enabled: true,
				authURL: sessionUITestOIDCAuthorizeURL,
			}
			handler, manager, _ := buildSessionUITestHandler(
				t,
				sessionUITestHandlerOptions{
					oidcProvider: oidcProvider,
					auditLogPath: auditLogPath,
				},
			)
			pendingCookie, pendingState := startOIDCLoginFlow(t, handler, manager)
			providerState := testCase.providerState
			if testCase.usePendingState {
				providerState = pendingState.OIDCState
			}

			callbackResponse := performOIDCCallbackRequest(
				handler,
				pendingCookie,
				providerState,
				testCase.authorizationCode,
			)

			assertJSONDetail(t, callbackResponse, testCase.expectedStatus, testCase.expectedDetail)
			assertOIDCCallbackNotCalled(t, oidcProvider)
			assertAuditLogContains(t, auditLogPath, "\"event_type\":\"oidc_login_failed\"")
			assertAuditLogContains(t, auditLogPath, "\"detail\":\""+testCase.expectedAuditDetail+"\"")
		})
	}
}

func TestMountSessionUIRoutesOIDCCallbackFailurePaths(t *testing.T) {
	for _, testCase := range buildOIDCCallbackFailureCases() {
		t.Run(testCase.name, func(t *testing.T) {
			runOIDCCallbackFailureCase(t, testCase)
		})
	}
}

func buildOIDCCallbackFailureCases() []oidcCallbackFailureCase {
	return []oidcCallbackFailureCase{
		oidcProviderVerificationFailureCase(),
		oidcIdentityFailureCase(oidcIdentityFailureSpec{
			name:                "identity without username",
			subject:             "subject-identity-missing-username",
			username:            "",
			email:               "missing@example.com",
			expectedStatus:      http.StatusUnauthorized,
			expectedDetail:      "oidc identity missing username",
			expectedAuditDetail: "identity_missing_username",
		}),
		oidcIdentityFailureCase(oidcIdentityFailureSpec{
			name:                "unmapped user",
			subject:             "subject-unknown-user",
			username:            "ghost-user",
			email:               "ghost@example.com",
			expectedStatus:      http.StatusForbidden,
			expectedDetail:      "oidc user is not authorized",
			expectedAuditDetail: "user_not_authorized",
		}),
		oidcIdentityFailureCase(oidcIdentityFailureSpec{
			name:                "inactive mapped user",
			subject:             "subject-inactive-user",
			username:            "admin",
			email:               "admin@example.com",
			lookupByUsername:    oidcInactiveUserLookup,
			expectedStatus:      http.StatusForbidden,
			expectedDetail:      "oidc user is not authorized",
			expectedAuditDetail: "user_not_authorized",
		}),
	}
}

func oidcProviderVerificationFailureCase() oidcCallbackFailureCase {
	return oidcCallbackFailureCase{
		name: "provider verification failure",
		provider: &sessionUITestOIDCProvider{
			enabled:         true,
			authURL:         sessionUITestOIDCAuthorizeURL,
			authenticateErr: errors.New("id token validation failed"),
		},
		expectedRedirect:    "/login",
		expectedAuditDetail: "token_exchange_or_verification_failed",
	}
}

func oidcIdentityFailureCase(spec oidcIdentityFailureSpec) oidcCallbackFailureCase {
	return oidcCallbackFailureCase{
		name: spec.name,
		provider: &sessionUITestOIDCProvider{
			enabled: true,
			authURL: sessionUITestOIDCAuthorizeURL,
			identity: &auth.OIDCIdentity{
				Subject:  spec.subject,
				Username: spec.username,
				Email:    spec.email,
			},
		},
		lookupByUsername:    spec.lookupByUsername,
		expectedStatus:      spec.expectedStatus,
		expectedDetail:      spec.expectedDetail,
		expectedAuditDetail: spec.expectedAuditDetail,
	}
}

func oidcInactiveUserLookup(_ context.Context, username string) (*models.UserAuthRecord, error) {
	if username != "admin" {
		return nil, nil
	}
	return &models.UserAuthRecord{
		Username: "admin",
		Role:     models.UserRoleAdmin,
		IsActive: false,
	}, nil
}

func runOIDCCallbackFailureCase(t *testing.T, testCase oidcCallbackFailureCase) {
	t.Helper()
	auditLogPath := filepath.Join(t.TempDir(), "oidc-callback-failure-audit.log")
	handler, manager, _ := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{
			oidcProvider:     testCase.provider,
			lookupByUsername: testCase.lookupByUsername,
			auditLogPath:     auditLogPath,
		},
	)
	pendingCookie, pendingState := startOIDCLoginFlow(t, handler, manager)
	callbackResponse := performOIDCCallbackRequest(handler, pendingCookie, pendingState.OIDCState, "auth-code-1")

	if testCase.expectedRedirect != "" {
		assertRedirect(t, callbackResponse, testCase.expectedRedirect)
	} else {
		assertJSONDetail(t, callbackResponse, testCase.expectedStatus, testCase.expectedDetail)
	}
	assertOIDCCallbackCall(t, testCase.provider, pendingState.OIDCNonce)
	assertOIDCPendingStateCleared(t, manager, callbackResponse)
	assertAuditLogContains(t, auditLogPath, "\"event_type\":\"oidc_login_failed\"")
	assertAuditLogContains(t, auditLogPath, "\"detail\":\""+testCase.expectedAuditDetail+"\"")
}

func TestMountSessionUIRoutesOIDCStartReturnsNotFoundWhenDisabled(t *testing.T) {
	handler, manager, _ := buildSessionUITestHandler(t)
	_, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)

	request := httptest.NewRequest(http.MethodGet, "/login/oidc", nil)
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
}

func startOIDCLoginFlow(
	t *testing.T,
	handler http.Handler,
	manager *auth.SessionManager,
) (*http.Cookie, auth.SessionState) {
	t.Helper()
	_, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)
	startRequest := httptest.NewRequest(http.MethodGet, "/login/oidc?next=%2Fui%2Fadmin", nil)
	startRequest.AddCookie(loginCookie)
	startResponse := httptest.NewRecorder()
	handler.ServeHTTP(startResponse, startRequest)
	assertRedirect(t, startResponse, sessionUITestOIDCAuthorizeURL)
	pendingCookie := findResponseCookie(startResponse, manager.CookieName())
	if pendingCookie == nil {
		t.Fatalf("expected pending session cookie after oidc start")
	}
	pendingState, err := manager.Decode(pendingCookie.Value)
	if err != nil {
		t.Fatalf("expected pending state to decode: %v", err)
	}
	return pendingCookie, pendingState
}

func performOIDCCallbackRequest(
	handler http.Handler,
	pendingCookie *http.Cookie,
	state string,
	code string,
) *httptest.ResponseRecorder {
	values := url.Values{}
	values.Set("state", state)
	if strings.TrimSpace(code) != "" {
		values.Set("code", code)
	}
	callbackRequest := httptest.NewRequest(http.MethodGet, "/login/oidc/callback?"+values.Encode(), nil)
	callbackRequest.AddCookie(pendingCookie)
	callbackResponse := httptest.NewRecorder()
	handler.ServeHTTP(callbackResponse, callbackRequest)
	return callbackResponse
}

func assertOIDCStartCall(t *testing.T, provider *sessionUITestOIDCProvider) {
	t.Helper()
	if len(provider.authURLCalls) != 1 {
		t.Fatalf("expected one oidc auth URL call, got %d", len(provider.authURLCalls))
	}
}

func assertPendingOIDCSessionState(
	t *testing.T,
	manager *auth.SessionManager,
	response *httptest.ResponseRecorder,
	expectedNextPath string,
) {
	t.Helper()
	pendingCookie := findResponseCookie(response, manager.CookieName())
	if pendingCookie == nil {
		t.Fatalf("expected session cookie to include pending oidc state")
	}
	state, err := manager.Decode(pendingCookie.Value)
	if err != nil {
		t.Fatalf("expected pending session cookie to decode: %v", err)
	}
	if state.OIDCState == "" || state.OIDCNonce == "" {
		t.Fatalf("expected oidc state and nonce to be set in session")
	}
	if state.OIDCNext != expectedNextPath {
		t.Fatalf("expected oidc next path %q, got %q", expectedNextPath, state.OIDCNext)
	}
}

func assertOIDCCallbackCall(
	t *testing.T,
	provider *sessionUITestOIDCProvider,
	expectedNonce string,
) {
	t.Helper()
	if len(provider.authenticateCall) != 1 {
		t.Fatalf("expected one oidc authenticate call, got %d", len(provider.authenticateCall))
	}
	if provider.authenticateCall[0].Code != "auth-code-1" {
		t.Fatalf("expected auth code to be forwarded, got %q", provider.authenticateCall[0].Code)
	}
	if provider.authenticateCall[0].Nonce != expectedNonce {
		t.Fatalf("expected oidc nonce to match pending state")
	}
}

func assertOIDCCallbackNotCalled(t *testing.T, provider *sessionUITestOIDCProvider) {
	t.Helper()
	if len(provider.authenticateCall) != 0 {
		t.Fatalf("expected oidc authenticate not to be called")
	}
}

func assertJSONDetail(
	t *testing.T,
	response *httptest.ResponseRecorder,
	expectedStatus int,
	expectedDetail string,
) {
	t.Helper()
	if response.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, response.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON response payload: %v", err)
	}
	if payload["detail"] != expectedDetail {
		t.Fatalf("expected detail %q, got %q", expectedDetail, payload["detail"])
	}
}

func assertOIDCPendingStateCleared(
	t *testing.T,
	manager *auth.SessionManager,
	response *httptest.ResponseRecorder,
) {
	t.Helper()
	sessionCookie := findResponseCookie(response, manager.CookieName())
	if sessionCookie == nil {
		t.Fatalf("expected callback response to include a session cookie")
	}
	state, err := manager.Decode(sessionCookie.Value)
	if err != nil {
		t.Fatalf("expected callback cookie to decode: %v", err)
	}
	if state.OIDCState != "" {
		t.Fatalf("expected oidc state to be cleared")
	}
	if state.OIDCNonce != "" {
		t.Fatalf("expected oidc nonce to be cleared")
	}
	if state.OIDCNext != "" {
		t.Fatalf("expected oidc next path to be cleared")
	}
}

func assertAuditLogContains(t *testing.T, auditLogPath string, expected string) {
	t.Helper()
	content, err := os.ReadFile(auditLogPath)
	if err != nil {
		t.Fatalf("expected audit log to be readable: %v", err)
	}
	if !strings.Contains(string(content), expected) {
		t.Fatalf("expected audit log to contain %q", expected)
	}
}

func assertAuthenticatedOIDCSessionState(
	t *testing.T,
	manager *auth.SessionManager,
	response *httptest.ResponseRecorder,
) {
	t.Helper()
	authenticatedCookie := findResponseCookie(response, manager.CookieName())
	if authenticatedCookie == nil {
		t.Fatalf("expected authenticated session cookie after oidc callback")
	}
	authenticatedState, err := manager.Decode(authenticatedCookie.Value)
	if err != nil {
		t.Fatalf("expected authenticated cookie to decode: %v", err)
	}
	if authenticatedState.User == nil || authenticatedState.User.Username != "admin" {
		t.Fatalf("expected authenticated user in session, got %#v", authenticatedState.User)
	}
	if authenticatedState.CSRFToken == "" {
		t.Fatalf("expected rotated csrf token in authenticated session")
	}
}

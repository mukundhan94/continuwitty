package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"engram/internal/auth"
)

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

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func anyStringContains(values []string, expectedSubstring string) bool {
	for _, value := range values {
		if strings.Contains(value, expectedSubstring) {
			return true
		}
	}
	return false
}

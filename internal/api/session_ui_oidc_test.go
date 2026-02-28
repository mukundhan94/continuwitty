package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"engram/internal/auth"
)

type sessionUITestOIDCProvider struct {
	enabled          bool
	authURL          string
	identity         *auth.OIDCIdentity
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

func (provider *sessionUITestOIDCProvider) Enabled() bool {
	return provider != nil && provider.enabled
}

func (provider *sessionUITestOIDCProvider) AuthCodeURL(state string, nonce string) (string, error) {
	provider.authURLCalls = append(provider.authURLCalls, sessionUITestOIDCAuthURLCall{
		State: state,
		Nonce: nonce,
	})
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

func TestMountSessionUIRoutesOIDCStartRedirectsAndStoresPendingState(t *testing.T) {
	oidcProvider := &sessionUITestOIDCProvider{
		enabled: true,
		authURL: "https://accounts.example.com/oauth/authorize?client_id=engram-web",
	}
	handler, manager, _ := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{oidcProvider: oidcProvider},
	)
	_, loginCookie := fetchLoginCSRFTokenAndCookie(t, handler, manager, nil)

	request := httptest.NewRequest(http.MethodGet, "/login/oidc?next=%2Fui%2Fadmin", nil)
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	assertRedirect(t, response, oidcProvider.authURL)

	assertOIDCStartCall(t, oidcProvider)
	assertPendingOIDCSessionState(t, manager, response, "/ui/admin")
}

func TestMountSessionUIRoutesOIDCCallbackAuthenticatesAndRedirects(t *testing.T) {
	oidcProvider := &sessionUITestOIDCProvider{
		enabled: true,
		authURL: "https://accounts.example.com/oauth/authorize?client_id=engram-web",
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

	callbackPath := "/login/oidc/callback?state=" + url.QueryEscape(pendingState.OIDCState) + "&code=auth-code-1"
	callbackRequest := httptest.NewRequest(http.MethodGet, callbackPath, nil)
	callbackRequest.AddCookie(pendingCookie)
	callbackResponse := httptest.NewRecorder()
	handler.ServeHTTP(callbackResponse, callbackRequest)

	assertRedirect(t, callbackResponse, "/ui/admin")
	assertOIDCCallbackCall(t, oidcProvider, pendingState.OIDCNonce)
	assertAuthenticatedOIDCSessionState(t, manager, callbackResponse)
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
	assertRedirect(t, startResponse, "https://accounts.example.com/oauth/authorize?client_id=engram-web")
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

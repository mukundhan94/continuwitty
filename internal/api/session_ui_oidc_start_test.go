package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"engram/internal/config"
	internaloauth "engram/internal/oauth"

	"github.com/google/uuid"
)

func TestOAuthRoutesMountedWithDependencies(t *testing.T) {
	settings := config.Settings{
		AppSemanticVersion: "1.2.3",
		AppCommitSHA:       "abc1234",
		OAuthEnabled:       true,
	}
	serviceCalled := false
	router := NewRouterWithDependencies(
		settings,
		RouterDependencies{
			OAuthRegistration: fakeOAuthRegistrationRouteService{
				handleRegisterFn: func(
					_ context.Context,
					_ config.Settings,
					_ internaloauth.RegistrationRequest,
					_ *internaloauth.SessionUser,
				) (internaloauth.RegistrationResponse, error) {
					serviceCalled = true
					return internaloauth.RegistrationResponse{
						ClientID:                "engram_client_123",
						ClientName:              "Engram MCP Client",
						RedirectURIs:            []string{"https://client.example/callback"},
						GrantTypes:              []string{"authorization_code"},
						ResponseTypes:           []string{"code"},
						TokenEndpointAuthMethod: "none",
						ClientIDIssuedAt:        1767225600,
					}, nil
				},
			},
		},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/oauth/register",
		strings.NewReader(`{"redirect_uris":["https://client.example/callback"]}`),
	)
	request = WithAdminActor(
		request,
		AdminActor{
			UserID: uuid.MustParse("00000000-0000-0000-0000-000000000963"),
			Role:   "admin",
		},
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	if !serviceCalled {
		t.Fatalf("expected oauth registration service to be called")
	}
}

func TestOAuthAuthorizationRoutesMountedWithDependencies(t *testing.T) {
	settings := config.Settings{
		AppSemanticVersion: "1.2.3",
		AppCommitSHA:       "abc1234",
		OAuthEnabled:       true,
	}
	serviceCalled := false
	router := NewRouterWithDependencies(
		settings,
		RouterDependencies{
			OAuthAuthorization: fakeOAuthAuthorizationRouteService{
				handleAuthorizeFn: func(
					_ context.Context,
					_ config.Settings,
					_ internaloauth.AuthorizationRequest,
				) (internaloauth.AuthorizationResult, error) {
					serviceCalled = true
					return internaloauth.AuthorizationResult{
						RedirectURL: "https://client.example/callback?code=abc123",
					}, nil
				},
			},
		},
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/oauth/authorize?response_type=code&client_id=client-1&redirect_uri=https://client.example/callback",
		nil,
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", response.Code)
	}
	if !serviceCalled {
		t.Fatalf("expected oauth authorization service to be called")
	}
}

func TestOAuthTokenRoutesMountedWithDependencies(t *testing.T) {
	settings := config.Settings{
		AppSemanticVersion: "1.2.3",
		AppCommitSHA:       "abc1234",
		OAuthEnabled:       true,
	}
	serviceCalled := false
	router := NewRouterWithDependencies(
		settings,
		RouterDependencies{
			OAuthToken: fakeOAuthTokenRouteService{
				handleTokenFn: func(
					_ context.Context,
					_ config.Settings,
					_ internaloauth.TokenRequest,
				) (internaloauth.TokenResult, error) {
					serviceCalled = true
					return internaloauth.TokenResult{
						ErrorCode:        "invalid_grant",
						ErrorDescription: "Authorization code is invalid.",
						StatusCode:       http.StatusBadRequest,
					}, nil
				},
			},
		},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/oauth/token",
		strings.NewReader("grant_type=authorization_code&code=abc&redirect_uri=https%3A%2F%2Fclient.example%2Fcallback&client_id=client-1&code_verifier=verifier"),
	)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
	if !serviceCalled {
		t.Fatalf("expected oauth token service to be called")
	}
}

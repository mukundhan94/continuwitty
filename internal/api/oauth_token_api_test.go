package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"engram/internal/config"
	internaloauth "engram/internal/oauth"

	"github.com/go-chi/chi/v5"
)

type fakeOAuthTokenRouteService struct {
	handleTokenFn func(
		ctx context.Context,
		settings config.Settings,
		request internaloauth.TokenRequest,
	) (internaloauth.TokenResult, error)
}

func (service fakeOAuthTokenRouteService) HandleToken(
	ctx context.Context,
	settings config.Settings,
	request internaloauth.TokenRequest,
) (internaloauth.TokenResult, error) {
	if service.handleTokenFn == nil {
		return internaloauth.TokenResult{}, nil
	}
	return service.handleTokenFn(ctx, settings, request)
}

func TestMountOAuthTokenRoutesRegistersEndpoint(t *testing.T) {
	router := chi.NewRouter()
	MountOAuthTokenRoutes(router, config.Settings{OAuthEnabled: true}, fakeOAuthTokenRouteService{})

	routes := collectChatRoutes(t, router)
	requireChatRoute(t, routes, "/oauth/token")
}

func TestOAuthTokenRouteWritesSuccessResponse(t *testing.T) {
	capturedRequest := internaloauth.TokenRequest{}
	router := chi.NewRouter()
	MountOAuthTokenRoutes(
		router,
		config.Settings{OAuthEnabled: true},
		fakeOAuthTokenRouteService{
			handleTokenFn: func(
				_ context.Context,
				_ config.Settings,
				request internaloauth.TokenRequest,
			) (internaloauth.TokenResult, error) {
				capturedRequest = request
				return internaloauth.TokenResult{
					AccessToken:      "engram_mcp_token",
					TokenType:        "Bearer",
					ExpiresInSeconds: 3600,
					Scope:            "mcp:read",
				}, nil
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

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if capturedRequest.GrantType != "authorization_code" {
		t.Fatalf("expected grant_type authorization_code, got %q", capturedRequest.GrantType)
	}
	payload := decodeOAuthTokenResponseMap(t, response)
	if payload["access_token"] != "engram_mcp_token" {
		t.Fatalf("expected access token in response, got %#v", payload["access_token"])
	}
}

func TestOAuthTokenRouteMapsOAuthErrors(t *testing.T) {
	router := chi.NewRouter()
	MountOAuthTokenRoutes(
		router,
		config.Settings{OAuthEnabled: true},
		fakeOAuthTokenRouteService{
			handleTokenFn: func(
				_ context.Context,
				_ config.Settings,
				_ internaloauth.TokenRequest,
			) (internaloauth.TokenResult, error) {
				return internaloauth.TokenResult{
					ErrorCode:        "invalid_grant",
					ErrorDescription: "Authorization code is invalid.",
					StatusCode:       http.StatusBadRequest,
				}, nil
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
	payload := decodeOAuthTokenResponseMap(t, response)
	if payload["error"] != "invalid_grant" {
		t.Fatalf("expected invalid_grant error code, got %#v", payload["error"])
	}
}

func TestOAuthTokenRouteValidationErrors(t *testing.T) {
	testCases := []struct {
		name           string
		settings       config.Settings
		expectedStatus int
		expectedKey    string
		expectedValue  string
	}{
		{
			name:           "missing required fields",
			settings:       config.Settings{OAuthEnabled: true},
			expectedStatus: http.StatusBadRequest,
			expectedKey:    "error",
			expectedValue:  "invalid_request",
		},
		{
			name:           "oauth disabled",
			settings:       config.Settings{OAuthEnabled: false},
			expectedStatus: http.StatusNotFound,
			expectedKey:    "detail",
			expectedValue:  "OAuth endpoints disabled",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			router := chi.NewRouter()
			MountOAuthTokenRoutes(router, testCase.settings, fakeOAuthTokenRouteService{})

			request := httptest.NewRequest(
				http.MethodPost,
				"/oauth/token",
				strings.NewReader("grant_type=authorization_code&client_id=client-1"),
			)
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != testCase.expectedStatus {
				t.Fatalf("expected status %d, got %d", testCase.expectedStatus, response.Code)
			}
			payload := decodeOAuthTokenResponseMap(t, response)
			if payload[testCase.expectedKey] != testCase.expectedValue {
				t.Fatalf(
					"expected %q=%q, got %#v",
					testCase.expectedKey,
					testCase.expectedValue,
					payload[testCase.expectedKey],
				)
			}
		})
	}
}

func decodeOAuthTokenResponseMap(
	t *testing.T,
	response *httptest.ResponseRecorder,
) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return payload
}

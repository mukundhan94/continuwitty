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
	"github.com/google/uuid"
)

type fakeOAuthAuthorizationRouteService struct {
	handleAuthorizeFn func(
		ctx context.Context,
		settings config.Settings,
		request internaloauth.AuthorizationRequest,
	) (internaloauth.AuthorizationResult, error)
}

func (service fakeOAuthAuthorizationRouteService) HandleAuthorize(
	ctx context.Context,
	settings config.Settings,
	request internaloauth.AuthorizationRequest,
) (internaloauth.AuthorizationResult, error) {
	if service.handleAuthorizeFn == nil {
		return internaloauth.AuthorizationResult{}, nil
	}
	return service.handleAuthorizeFn(ctx, settings, request)
}

func TestMountOAuthAuthorizationRoutesRegistersEndpoint(t *testing.T) {
	router := chi.NewRouter()
	MountOAuthAuthorizationRoutes(
		router,
		config.Settings{OAuthEnabled: true},
		fakeOAuthAuthorizationRouteService{},
	)

	routes := collectChatRoutes(t, router)
	requireChatRoute(t, routes, "/oauth/authorize")
}

func TestOAuthAuthorizeRouteHandlesServiceResult(t *testing.T) {
	testCases := []struct {
		name              string
		serviceResult     internaloauth.AuthorizationResult
		expectedStatus    int
		expectedLocation  string
		expectedErrorCode string
	}{
		{
			name: "redirect response",
			serviceResult: internaloauth.AuthorizationResult{
				RedirectURL: "https://client.example/callback?code=abc123",
			},
			expectedStatus:   http.StatusSeeOther,
			expectedLocation: "https://client.example/callback?code=abc123",
		},
		{
			name: "oauth error response",
			serviceResult: internaloauth.AuthorizationResult{
				ErrorCode:        "invalid_client",
				ErrorDescription: "Unknown client_id.",
				StatusCode:       http.StatusUnauthorized,
			},
			expectedStatus:    http.StatusUnauthorized,
			expectedErrorCode: "invalid_client",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			router := chi.NewRouter()
			MountOAuthAuthorizationRoutes(
				router,
				config.Settings{OAuthEnabled: true},
				fakeOAuthAuthorizationRouteService{
					handleAuthorizeFn: func(
						_ context.Context,
						_ config.Settings,
						_ internaloauth.AuthorizationRequest,
					) (internaloauth.AuthorizationResult, error) {
						return testCase.serviceResult, nil
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

			if response.Code != testCase.expectedStatus {
				t.Fatalf("expected status %d, got %d", testCase.expectedStatus, response.Code)
			}
			if testCase.expectedLocation != "" {
				if response.Header().Get("Location") != testCase.expectedLocation {
					t.Fatalf("unexpected redirect location %q", response.Header().Get("Location"))
				}
				return
			}
			payload := decodeOAuthAuthorizeResponseMap(t, response)
			if payload["error"] != testCase.expectedErrorCode {
				t.Fatalf("expected oauth error code %q, got %#v", testCase.expectedErrorCode, payload["error"])
			}
		})
	}
}

func TestOAuthAuthorizeRouteRejectsMissingRequiredFields(t *testing.T) {
	serviceCalled := false
	router := chi.NewRouter()
	MountOAuthAuthorizationRoutes(
		router,
		config.Settings{OAuthEnabled: true},
		fakeOAuthAuthorizationRouteService{
			handleAuthorizeFn: func(
				_ context.Context,
				_ config.Settings,
				_ internaloauth.AuthorizationRequest,
			) (internaloauth.AuthorizationResult, error) {
				serviceCalled = true
				return internaloauth.AuthorizationResult{}, nil
			},
		},
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/oauth/authorize?client_id=client-1&redirect_uri=https://client.example/callback",
		nil,
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
	if serviceCalled {
		t.Fatalf("expected authorization service not to be called")
	}
	payload := decodeOAuthAuthorizeResponseMap(t, response)
	if payload["error"] != "invalid_request" {
		t.Fatalf("expected invalid_request error code, got %#v", payload["error"])
	}
}

func TestOAuthAuthorizeRouteReturnsNotFoundWhenDisabled(t *testing.T) {
	router := chi.NewRouter()
	MountOAuthAuthorizationRoutes(
		router,
		config.Settings{OAuthEnabled: false},
		fakeOAuthAuthorizationRouteService{},
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/oauth/authorize?response_type=code&client_id=client-1&redirect_uri=https://client.example/callback",
		nil,
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
	payload := decodeOAuthAuthorizeResponseMap(t, response)
	if payload["detail"] != "OAuth endpoints disabled" {
		t.Fatalf("expected disabled detail, got %#v", payload["detail"])
	}
}

func TestOAuthAuthorizeRouteBuildsRequestContext(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000964")
	captured := internaloauth.AuthorizationRequest{}
	router := chi.NewRouter()
	MountOAuthAuthorizationRoutes(
		router,
		config.Settings{OAuthEnabled: true},
		fakeOAuthAuthorizationRouteService{
			handleAuthorizeFn: func(
				_ context.Context,
				_ config.Settings,
				request internaloauth.AuthorizationRequest,
			) (internaloauth.AuthorizationResult, error) {
				captured = request
				return internaloauth.AuthorizationResult{
					RedirectURL: "https://client.example/callback?code=abc123",
				}, nil
			},
		},
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"https://api.local/oauth/authorize?response_type=code&client_id=client-1&redirect_uri=https://client.example/callback&scope=mcp:read",
		nil,
	)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "admin"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", response.Code)
	}
	if !strings.Contains(captured.RequestPathWithQuery, "response_type=code") {
		t.Fatalf("expected request path to preserve query, got %q", captured.RequestPathWithQuery)
	}
	if captured.Issuer != "https://api.local" {
		t.Fatalf("expected issuer https://api.local, got %q", captured.Issuer)
	}
	if captured.SessionUser == nil || captured.SessionUser.UserID != actorID.String() {
		t.Fatalf("expected actor-backed session user, got %#v", captured.SessionUser)
	}
}

func decodeOAuthAuthorizeResponseMap(
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

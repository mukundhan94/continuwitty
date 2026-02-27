package api

import (
	"context"
	"crypto/tls"
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

type fakeOAuthRegistrationRouteService struct {
	handleRegisterFn func(
		ctx context.Context,
		settings config.Settings,
		payload internaloauth.RegistrationRequest,
		sessionUser *internaloauth.SessionUser,
	) (internaloauth.RegistrationResponse, error)
}

func (service fakeOAuthRegistrationRouteService) HandleRegister(
	ctx context.Context,
	settings config.Settings,
	payload internaloauth.RegistrationRequest,
	sessionUser *internaloauth.SessionUser,
) (internaloauth.RegistrationResponse, error) {
	if service.handleRegisterFn == nil {
		return internaloauth.RegistrationResponse{}, nil
	}
	return service.handleRegisterFn(ctx, settings, payload, sessionUser)
}

func TestMountOAuthRoutesRegistersEndpoints(t *testing.T) {
	router := chi.NewRouter()
	MountOAuthRoutes(router, config.Settings{OAuthEnabled: true}, fakeOAuthRegistrationRouteService{})

	routes := collectChatRoutes(t, router)
	requiredRoutes := []string{
		"/.well-known/oauth-authorization-server",
		"/.well-known/openid-configuration",
		"/.well-known/oauth-protected-resource",
		"/.well-known/oauth-protected-resource/*",
		"/oauth/register",
	}
	for _, route := range requiredRoutes {
		requireChatRoute(t, routes, route)
	}
}

func TestOAuthAuthorizationServerMetadataUsesConfiguredIssuer(t *testing.T) {
	router := chi.NewRouter()
	MountOAuthRoutes(
		router,
		config.Settings{
			OAuthEnabled:   true,
			OAuthIssuerURL: "https://auth.example.com/",
		},
		fakeOAuthRegistrationRouteService{},
	)

	request := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	metadata := decodeOAuthJSONMap(t, response)
	if metadata["issuer"] != "https://auth.example.com" {
		t.Fatalf("expected configured issuer, got %#v", metadata["issuer"])
	}
	if metadata["registration_endpoint"] != "https://auth.example.com/oauth/register" {
		t.Fatalf("expected registration endpoint, got %#v", metadata["registration_endpoint"])
	}
}

func TestOpenIDConfigurationAddsOIDCFields(t *testing.T) {
	router := chi.NewRouter()
	MountOAuthRoutes(router, config.Settings{OAuthEnabled: true}, fakeOAuthRegistrationRouteService{})

	request := httptest.NewRequest(
		http.MethodGet,
		"https://api.local/.well-known/openid-configuration",
		nil,
	)
	request.TLS = &tls.ConnectionState{}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	metadata := decodeOAuthJSONMap(t, response)
	requireOAuthStringSliceFieldContains(t, metadata, "subject_types_supported", "public")
	requireOAuthFieldExists(t, metadata, "claims_supported")
	if metadata["issuer"] != "https://api.local" {
		t.Fatalf("expected request-derived issuer, got %#v", metadata["issuer"])
	}
}

func TestOAuthProtectedResourceMetadataScopedPath(t *testing.T) {
	router := chi.NewRouter()
	MountOAuthRoutes(router, config.Settings{OAuthEnabled: true}, fakeOAuthRegistrationRouteService{})

	request := httptest.NewRequest(
		http.MethodGet,
		"http://api.local/.well-known/oauth-protected-resource/scoped/resource",
		nil,
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	metadata := decodeOAuthJSONMap(t, response)
	if metadata["resource"] != "http://api.local/scoped/resource" {
		t.Fatalf("expected scoped resource path, got %#v", metadata["resource"])
	}
}

func TestOAuthRegisterClientWritesCreatedResponse(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000961")
	capturedPayload := internaloauth.RegistrationRequest{}
	var capturedSessionUser *internaloauth.SessionUser
	service := fakeOAuthRegistrationRouteService{
		handleRegisterFn: func(
			_ context.Context,
			_ config.Settings,
			payload internaloauth.RegistrationRequest,
			sessionUser *internaloauth.SessionUser,
		) (internaloauth.RegistrationResponse, error) {
			capturedPayload = payload
			capturedSessionUser = sessionUser
			return internaloauth.RegistrationResponse{
				ClientID:                "engram_client_123",
				ClientName:              payload.ClientName,
				RedirectURIs:            payload.RedirectURIs,
				GrantTypes:              []string{"authorization_code"},
				ResponseTypes:           []string{"code"},
				TokenEndpointAuthMethod: "none",
				ClientIDIssuedAt:        1767225600,
			}, nil
		},
	}
	router := chi.NewRouter()
	MountOAuthRoutes(router, config.Settings{OAuthEnabled: true}, service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/oauth/register",
		strings.NewReader(`{"redirect_uris":["https://client.example/callback"]}`),
	)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "admin"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	responsePayload := decodeOAuthJSONMap(t, response)
	if responsePayload["client_id"] != "engram_client_123" {
		t.Fatalf("expected RFC client_id field, got %#v", responsePayload["client_id"])
	}
	if _, present := responsePayload["ClientID"]; present {
		t.Fatalf("expected no legacy ClientID field in oauth registration response")
	}
	if capturedPayload.ClientName != defaultOAuthClientName {
		t.Fatalf("expected default client name %q, got %q", defaultOAuthClientName, capturedPayload.ClientName)
	}
	if capturedSessionUser == nil || capturedSessionUser.UserID != actorID.String() {
		t.Fatalf("expected actor-backed session user, got %#v", capturedSessionUser)
	}
}

func TestOAuthRegisterClientMapsRegistrationErrors(t *testing.T) {
	router := chi.NewRouter()
	MountOAuthRoutes(
		router,
		config.Settings{OAuthEnabled: true},
		fakeOAuthRegistrationRouteService{
			handleRegisterFn: func(
				_ context.Context,
				_ config.Settings,
				_ internaloauth.RegistrationRequest,
				_ *internaloauth.SessionUser,
			) (internaloauth.RegistrationResponse, error) {
				return internaloauth.RegistrationResponse{}, internaloauth.RegistrationError{
					ErrorCode:   "invalid_client_metadata",
					Description: "Unsupported token_endpoint_auth_method.",
					StatusCode:  http.StatusBadRequest,
				}
			},
		},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/oauth/register",
		strings.NewReader(`{"redirect_uris":["https://client.example/callback"]}`),
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
	payload := decodeOAuthJSONMap(t, response)
	if payload["error"] != "invalid_client_metadata" {
		t.Fatalf("expected oauth error code, got %#v", payload["error"])
	}
}

func TestOAuthRouteValidationErrors(t *testing.T) {
	testCases := []struct {
		name        string
		settings    config.Settings
		expectation oauthRouteDetailExpectation
	}{
		{
			name:     "well known route disabled",
			settings: config.Settings{OAuthEnabled: false},
			expectation: oauthRouteDetailExpectation{
				method:         http.MethodGet,
				path:           "/.well-known/oauth-authorization-server",
				body:           "",
				expectedStatus: http.StatusNotFound,
				expectedDetail: "OAuth endpoints disabled",
			},
		},
		{
			name:     "registration payload invalid json",
			settings: config.Settings{OAuthEnabled: true},
			expectation: oauthRouteDetailExpectation{
				method:         http.MethodPost,
				path:           "/oauth/register",
				body:           "{invalid",
				expectedStatus: http.StatusBadRequest,
				expectedDetail: "invalid json body",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			router := chi.NewRouter()
			MountOAuthRoutes(router, testCase.settings, fakeOAuthRegistrationRouteService{})
			assertOAuthRouteDetail(t, router, testCase.expectation)
		})
	}
}

func decodeOAuthJSONMap(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return payload
}

func requireOAuthFieldExists(t *testing.T, payload map[string]any, key string) {
	t.Helper()
	if _, ok := payload[key]; !ok {
		t.Fatalf("expected %q field to exist", key)
	}
}

func requireOAuthStringSliceFieldContains(
	t *testing.T,
	payload map[string]any,
	key string,
	expected string,
) {
	t.Helper()
	rawValue, ok := payload[key]
	if !ok {
		t.Fatalf("expected %q field to exist", key)
	}
	values, ok := rawValue.([]any)
	if !ok {
		t.Fatalf("expected %q to be []any, got %T", key, rawValue)
	}
	for _, value := range values {
		if parsedValue, parseOK := value.(string); parseOK && parsedValue == expected {
			return
		}
	}
	t.Fatalf("expected %q to include %q, got %#v", key, expected, values)
}

type oauthRouteDetailExpectation struct {
	method         string
	path           string
	body           string
	expectedStatus int
	expectedDetail string
}

func assertOAuthRouteDetail(
	t *testing.T,
	router chi.Router,
	expectation oauthRouteDetailExpectation,
) {
	t.Helper()
	request := httptest.NewRequest(
		expectation.method,
		expectation.path,
		strings.NewReader(expectation.body),
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != expectation.expectedStatus {
		t.Fatalf("expected status %d, got %d", expectation.expectedStatus, response.Code)
	}
	payload := decodeOAuthJSONMap(t, response)
	if payload["detail"] != expectation.expectedDetail {
		t.Fatalf("expected detail %q, got %#v", expectation.expectedDetail, payload["detail"])
	}
}

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"engram/internal/admin"
	"engram/internal/auth"
	"engram/internal/config"
	"engram/internal/models"
	internaloauth "engram/internal/oauth"
	"engram/internal/workflow"

	"github.com/google/uuid"
)

func TestHealthz(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	router := NewRouter(settings)

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("expected status payload to be 'ok', got %q", payload["status"])
	}
}

func TestVersionEndpoint(t *testing.T) {
	settings := config.Settings{
		AppSemanticVersion:      "1.2.3",
		AppCommitSHA:            "abc1234",
		ChatPromptPolicyVersion: "chat-policy-v9",
		MCPToolPolicyVersion:    "mcp-policy-v4",
		EvalSuiteVersion:        "eval-suite-v2",
	}
	router := NewRouter(settings)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	expectedFields := map[string]string{
		"semantic_version":           settings.AppSemanticVersion,
		"release":                    "v" + settings.AppSemanticVersion,
		"commit_id":                  settings.AppCommitSHA,
		"chat_prompt_policy_version": settings.ChatPromptPolicyVersion,
		"mcp_tool_policy_version":    settings.MCPToolPolicyVersion,
		"eval_suite_version":         settings.EvalSuiteVersion,
	}
	for key, expectedValue := range expectedFields {
		assertVersionFieldValue(t, payload, key, expectedValue)
	}
}

func TestObservabilityMetricsRouteMountedWithRecorder(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	metrics := NewInMemoryRequestMetrics()
	router := NewRouterWithDependencies(
		settings,
		RouterDependencies{
			RequestMetrics: metrics,
		},
	)

	healthRequest := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthResponse := httptest.NewRecorder()
	router.ServeHTTP(healthResponse, healthRequest)
	if healthResponse.Code != http.StatusOK {
		t.Fatalf("expected health status 200, got %d", healthResponse.Code)
	}

	metricsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	metricsResponse := httptest.NewRecorder()
	router.ServeHTTP(metricsResponse, metricsRequest)
	if metricsResponse.Code != http.StatusOK {
		t.Fatalf("expected metrics status 200, got %d", metricsResponse.Code)
	}
	var payload RequestMetricsSnapshot
	if err := json.Unmarshal(metricsResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode metrics response: %v", err)
	}
	if payload.Totals.Requests < 1 {
		t.Fatalf("expected at least one recorded request")
	}
	if payload.ByRoute["GET /healthz"].Count < 1 {
		t.Fatalf("expected recorded /healthz route stats")
	}
}

func TestMemoryAdminRoutesNotMountedWithoutDependencies(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	router := NewRouter(settings)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/memory/sessions", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}

	chatRequest := httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions", nil)
	chatResponse := httptest.NewRecorder()
	router.ServeHTTP(chatResponse, chatRequest)
	if chatResponse.Code != http.StatusNotFound {
		t.Fatalf("expected chat status 404, got %d", chatResponse.Code)
	}

	projectsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	projectsResponse := httptest.NewRecorder()
	router.ServeHTTP(projectsResponse, projectsRequest)
	if projectsResponse.Code != http.StatusNotFound {
		t.Fatalf("expected projects status 404, got %d", projectsResponse.Code)
	}

	ingestionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/ingestion/documents", nil)
	ingestionResponse := httptest.NewRecorder()
	router.ServeHTTP(ingestionResponse, ingestionRequest)
	if ingestionResponse.Code != http.StatusNotFound {
		t.Fatalf("expected ingestion status 404, got %d", ingestionResponse.Code)
	}
}

func TestMemoryAdminRoutesMountedWithDependencies(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	serviceCalled := false
	actorCalled := false
	service := &fakeMemoryAdminService{
		listSessionsFn: func(_ context.Context, request admin.MemoryAdminListRequest) ([]models.AdminChatSessionRecord, error) {
			serviceCalled = true
			if request.Limit != 3 {
				t.Fatalf("expected limit 3, got %d", request.Limit)
			}
			if request.Offset != 1 {
				t.Fatalf("expected offset 1, got %d", request.Offset)
			}
			return []models.AdminChatSessionRecord{}, nil
		},
	}
	router := NewRouterWithDependencies(
		settings,
		RouterDependencies{
			MemoryAdminService: service,
			RequireAdminActor: func(_ *http.Request) (AdminActor, error) {
				actorCalled = true
				return AdminActor{
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000211"),
					Role:   "admin",
				}, nil
			},
		},
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/memory/sessions?limit=3&offset=1", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !actorCalled {
		t.Fatalf("expected actor resolver to be called")
	}
	if !serviceCalled {
		t.Fatalf("expected list sessions service to be called")
	}
}

func TestSessionAuthRoutesNotMountedWithoutDependencies(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	router := NewRouter(settings)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/session/csrf", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}

	usersRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	usersResponse := httptest.NewRecorder()
	router.ServeHTTP(usersResponse, usersRequest)
	if usersResponse.Code != http.StatusNotFound {
		t.Fatalf("expected users status 404, got %d", usersResponse.Code)
	}

	mcpTokensRequest := httptest.NewRequest(http.MethodGet, "/api/v1/mcp/tokens", nil)
	mcpTokensResponse := httptest.NewRecorder()
	router.ServeHTTP(mcpTokensResponse, mcpTokensRequest)
	if mcpTokensResponse.Code != http.StatusNotFound {
		t.Fatalf("expected mcp tokens status 404, got %d", mcpTokensResponse.Code)
	}

	engramsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/engrams", nil)
	engramsResponse := httptest.NewRecorder()
	router.ServeHTTP(engramsResponse, engramsRequest)
	if engramsResponse.Code != http.StatusNotFound {
		t.Fatalf("expected engrams status 404, got %d", engramsResponse.Code)
	}

	uiRequest := httptest.NewRequest(http.MethodGet, "/login", nil)
	uiResponse := httptest.NewRecorder()
	router.ServeHTTP(uiResponse, uiRequest)

	if uiResponse.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", uiResponse.Code)
	}

	adminUIRequest := httptest.NewRequest(http.MethodGet, "/ui/admin", nil)
	adminUIResponse := httptest.NewRecorder()
	router.ServeHTTP(adminUIResponse, adminUIRequest)
	if adminUIResponse.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", adminUIResponse.Code)
	}
}

func TestSessionAuthRoutesMountedWithDependencies(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	manager, err := auth.NewSessionManager("dev-session-secret-for-tests", auth.DefaultSessionCookieName)
	if err != nil {
		t.Fatalf("expected session manager creation to succeed: %v", err)
	}

	router := NewRouterWithDependencies(
		settings,
		RouterDependencies{
			SessionAuth: SessionAuthDependencies{
				SessionManager:    manager,
				GenerateCSRFToken: auth.GenerateCSRFToken,
			},
		},
	)

	assertRouteStatus(
		t,
		router,
		"/api/v1/session/csrf",
		http.StatusOK,
	)
	assertRouteStatus(
		t,
		router,
		"/login",
		http.StatusOK,
	)
	assertRouteRedirect(
		t,
		router,
		"/ui/admin",
		"/login",
	)
	assertRouteStatus(
		t,
		router,
		"/api/v1/users",
		http.StatusInternalServerError,
	)
	assertRouteStatus(
		t,
		router,
		"/api/v1/mcp/tokens",
		http.StatusInternalServerError,
	)
	assertRouteStatus(
		t,
		router,
		"/api/v1/engrams",
		http.StatusInternalServerError,
	)
}

func TestChatRoutesMountedWithDependencies(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	chatRouter := CreateChatRouter(
		newFakeChatSessionService(),
		fakeChatStreamService{},
		fakeChatSessionDerivativeService{},
		staticChatActorResolver(uuid.MustParse("00000000-0000-0000-0000-000000000090")),
	)
	router := NewRouterWithDependencies(
		settings,
		RouterDependencies{ChatRouter: chatRouter},
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
}

func TestDataRoutesMountedWithDependencies(t *testing.T) {
	testCases := []struct {
		name              string
		path              string
		buildDependencies func(serviceCalled *bool) RouterDependencies
	}{
		{
			name: "projects",
			path: "/api/v1/projects",
			buildDependencies: func(serviceCalled *bool) RouterDependencies {
				return RouterDependencies{
					ProjectsService: fakeProjectService{
						listProjectsFn: func(
							_ context.Context,
							_ ProjectListRouteRequest,
						) ([]models.ProjectRecord, error) {
							*serviceCalled = true
							return []models.ProjectRecord{}, nil
						},
					},
				}
			},
		},
		{
			name: "ingestion",
			path: "/api/v1/ingestion/documents",
			buildDependencies: func(serviceCalled *bool) RouterDependencies {
				return RouterDependencies{
					IngestionService: fakeIngestionService{
						listDocumentsFn: func(
							_ context.Context,
							_ IngestionListDocumentsRouteRequest,
						) ([]models.DocumentRecord, error) {
							*serviceCalled = true
							return []models.DocumentRecord{}, nil
						},
					},
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			serviceCalled := false
			statusCode := exerciseActorScopedDependencyRoute(
				t,
				testCase.buildDependencies(&serviceCalled),
				testCase.path,
			)
			if statusCode != http.StatusOK {
				t.Fatalf("expected status 200, got %d", statusCode)
			}
			if !serviceCalled {
				t.Fatalf("expected route service to be called")
			}
		})
	}
}

func TestAgentWorkflowRoutesMountedWithDependencies(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	serviceCalled := false
	router := NewRouterWithDependencies(
		settings,
		RouterDependencies{
			AgentWorkflow: fakeAgentWorkflowRouteService{
				runFn: func(_ context.Context, request workflow.AgentRunRequest) (workflow.AgentState, error) {
					serviceCalled = true
					return workflow.AgentState{
						ThreadID: request.ThreadID,
						Status:   "completed",
					}, nil
				},
			},
		},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/agent-runs",
		strings.NewReader(`{"project_id":"project-agent","thread_id":"thread-1","objective":"objective"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !serviceCalled {
		t.Fatalf("expected workflow service to be called")
	}
}

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

func assertRouteStatus(
	t *testing.T,
	router http.Handler,
	path string,
	expectedStatus int,
) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != expectedStatus {
		t.Fatalf("expected status %d for %s, got %d", expectedStatus, path, response.Code)
	}
}

func assertVersionFieldValue(
	t *testing.T,
	payload map[string]string,
	key string,
	expectedValue string,
) {
	t.Helper()
	if payload[key] != expectedValue {
		t.Fatalf("expected %s %q, got %q", key, expectedValue, payload[key])
	}
}

func assertRouteRedirect(
	t *testing.T,
	router http.Handler,
	path string,
	expectedLocation string,
) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect status 303 for %s, got %d", path, response.Code)
	}
	if response.Header().Get("Location") != expectedLocation {
		t.Fatalf("expected redirect location %q for %s, got %q", expectedLocation, path, response.Header().Get("Location"))
	}
}

func exerciseActorScopedDependencyRoute(
	t *testing.T,
	dependencies RouterDependencies,
	path string,
) int {
	t.Helper()
	router := NewRouterWithDependencies(
		config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"},
		dependencies,
	)
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request = WithAdminActor(
		request,
		AdminActor{
			UserID: uuid.MustParse("00000000-0000-0000-0000-000000000242"),
			Role:   "analyst",
		},
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response.Code
}

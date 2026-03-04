package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"engram/internal/admin"
	"engram/internal/auth"
	"engram/internal/config"
	"engram/internal/models"
	"engram/internal/workflow"

	"github.com/google/uuid"
)

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
		"/app/admin/sessions",
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

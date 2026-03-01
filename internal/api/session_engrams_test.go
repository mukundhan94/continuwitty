package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"engram/internal/auth"
	"engram/internal/models"
	"engram/internal/projects"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type sessionEngramRoutesHandlerOptions struct {
	actor                    *models.UserAuthRecord
	resolveProjectIDForWrite func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		projectID string,
	) (SessionProjectResolution, error)
	createEngram func(
		ctx context.Context,
		payload models.MemoryEngramCreate,
		ownerUserID uuid.UUID,
	) (*models.EngramCreateResponse, error)
	listEngrams func(
		ctx context.Context,
		projectID *string,
		limit int,
		offset int,
		actorUserID uuid.UUID,
	) ([]models.EngramSummary, error)
	queryEngrams func(
		ctx context.Context,
		request models.EngramQueryRequest,
		actorUserID uuid.UUID,
	) ([]models.EngramQueryResult, error)
	getRehydrationBundle func(
		ctx context.Context,
		engramID uuid.UUID,
		actorUserID uuid.UUID,
	) (*models.RehydrationBundle, error)
	getEngramSources func(
		ctx context.Context,
		engramID uuid.UUID,
		limit int,
		actorUserID uuid.UUID,
	) ([]models.EngramSourceRecord, error)
	shareEngram func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		engramID uuid.UUID,
	) (*models.EngramVisibilityRecord, error)
	unshareEngram func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		engramID uuid.UUID,
	) (*models.EngramVisibilityRecord, error)
	createEngramLink func(
		ctx context.Context,
		input SessionEngramLinkCreateInput,
	) (*models.EngramLinkRecord, error)
	listEngramLinks func(
		ctx context.Context,
		input SessionEngramLinkListInput,
	) ([]models.EngramLinkRecord, error)
	updateEngramLink func(
		ctx context.Context,
		input SessionEngramLinkUpdateInput,
	) (*models.EngramLinkRecord, error)
	archiveEngramLink func(
		ctx context.Context,
		input SessionEngramLinkArchiveInput,
	) (*models.EngramLinkRecord, error)
	suggestEngramLinks func(
		ctx context.Context,
		input SessionEngramLinkSuggestInput,
	) ([]models.EngramLinkSuggestion, error)
	traceEngramLinks func(
		ctx context.Context,
		input SessionEngramTraceInput,
	) ([]models.EngramLinkTraversalStep, error)
	hygieneEngramLinks func(
		ctx context.Context,
		input SessionEngramLinkHygieneInput,
	) ([]models.EngramLinkHygieneRecommendation, error)
}

type sessionEngramLoginCredentials struct {
	username string
	password string
}

type engramRequestSpec struct {
	method string
	path   string
	body   map[string]any
}

type engramCollectionAssertion struct {
	requestSpec engramRequestSpec
	expectedID  uuid.UUID
	contextName string
}

type engramCollectionRouteType int

const (
	engramCollectionRouteList engramCollectionRouteType = iota + 1
	engramCollectionRouteQuery
)

type engramCollectionRouteCase struct {
	name        string
	routeType   engramCollectionRouteType
	engramID    uuid.UUID
	requestSpec engramRequestSpec
}

func TestMountSessionAuthRoutesEngramCollectionRoutesUseRepository(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAdmin)
	for _, testCase := range engramCollectionRouteCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			assertCollectionRouteUsesRepository(
				t,
				buildCollectionRouteOptions(t, actor, testCase),
				engramCollectionAssertion{
					requestSpec: testCase.requestSpec,
					expectedID:  testCase.engramID,
					contextName: testCase.name,
				},
			)
		})
	}
}

func engramCollectionRouteCases() []engramCollectionRouteCase {
	return []engramCollectionRouteCase{
		{
			name:      "list engrams",
			routeType: engramCollectionRouteList,
			engramID:  uuid.MustParse("00000000-0000-0000-0000-000000000901"),
			requestSpec: engramRequestSpec{
				method: http.MethodGet,
				path:   "/api/v1/engrams?project_id=proj-1&limit=10&offset=2",
			},
		},
		{
			name:      "query engrams",
			routeType: engramCollectionRouteQuery,
			engramID:  uuid.MustParse("00000000-0000-0000-0000-000000000903"),
			requestSpec: engramRequestSpec{
				method: http.MethodPost,
				path:   "/api/v1/engrams/query",
				body:   map[string]any{"query": "durable memory", "top_k": 5},
			},
		},
	}
}

func buildCollectionRouteOptions(
	t *testing.T,
	actor *models.UserAuthRecord,
	testCase engramCollectionRouteCase,
) sessionEngramRoutesHandlerOptions {
	switch testCase.routeType {
	case engramCollectionRouteList:
		return sessionEngramRoutesHandlerOptions{
			actor: actor,
			listEngrams: func(
				_ context.Context,
				projectID *string,
				limit int,
				offset int,
				actorUserID uuid.UUID,
			) ([]models.EngramSummary, error) {
				requireEqual(t, "proj-1", *projectID)
				requireEqual(t, 10, limit)
				requireEqual(t, 2, offset)
				requireEqual(t, actor.UserID, actorUserID)
				return []models.EngramSummary{
					{
						EngramID:        testCase.engramID,
						ProjectID:       "proj-1",
						Title:           "Title",
						Abstract:        "Abstract",
						CreatedAt:       time.Date(2026, 2, 22, 6, 0, 0, 0, time.UTC),
						Tags:            []string{"tag"},
						Keywords:        []string{"kw"},
						VisibilityScope: "private",
					},
				}, nil
			},
		}
	case engramCollectionRouteQuery:
		return sessionEngramRoutesHandlerOptions{
			actor: actor,
			queryEngrams: func(
				_ context.Context,
				request models.EngramQueryRequest,
				actorUserID uuid.UUID,
			) ([]models.EngramQueryResult, error) {
				requireEqual(t, "durable memory", request.Query)
				requireEqual(t, 5, request.TopK)
				requireEqual(t, actor.UserID, actorUserID)
				return []models.EngramQueryResult{
					{
						EngramID:        testCase.engramID,
						ProjectID:       "proj-1",
						Title:           "LangGraph decision",
						Abstract:        "Used for checkpoints",
						CreatedAt:       time.Date(2026, 2, 22, 7, 0, 0, 0, time.UTC),
						Tags:            []string{"memory"},
						Keywords:        []string{"langgraph"},
						VisibilityScope: "private",
						Distance:        0.1,
					},
				}, nil
			},
		}
	default:
		t.Fatalf("unsupported engram collection route type: %d", testCase.routeType)
		return sessionEngramRoutesHandlerOptions{}
	}
}

func TestMountSessionAuthRoutesCreateEngramUsesProjectResolution(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAdmin)
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000902")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			resolveProjectIDForWrite: func(
				_ context.Context,
				actorUserID uuid.UUID,
				actorRole models.UserRole,
				projectID string,
			) (SessionProjectResolution, error) {
				requireEqual(t, actor.UserID, actorUserID)
				requireEqual(t, models.UserRoleAdmin, actorRole)
				requireEqual(t, "proj-1", projectID)
				return SessionProjectResolution{
					ProjectID:          "proj-1",
					UsedDefaultProject: false,
				}, nil
			},
			createEngram: func(
				_ context.Context,
				payload models.MemoryEngramCreate,
				ownerUserID uuid.UUID,
			) (*models.EngramCreateResponse, error) {
				requireEqual(t, "proj-1", payload.ProjectID)
				requireEqual(t, actor.UserID, ownerUserID)
				return &models.EngramCreateResponse{
					EngramID:  engramID,
					CreatedAt: time.Date(2026, 2, 22, 6, 30, 0, 0, time.UTC),
				}, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{
			username: actor.Username,
			password: "StrongPassword-12345",
		},
	)

	body := map[string]any{
		"project_id":                "proj-1",
		"title":                     "t",
		"abstract":                  "a",
		"detailed_summary_markdown": "d",
	}
	encodedBody, _ := json.Marshal(body)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/engrams", bytes.NewReader(encodedBody))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	requireEqual(t, http.StatusOK, response.Code)
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode create engram response: %v", err)
	}
	requireEqual(t, engramID.String(), payload["engram_id"].(string))
	requireEqual(t, "proj-1", payload["resolved_project_id"].(string))
	requireEqual(t, false, payload["used_default_project"].(bool))
}

func TestMountSessionAuthRoutesRehydrateReturns404WhenMissing(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAdmin)
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			getRehydrationBundle: func(
				_ context.Context,
				_ uuid.UUID,
				_ uuid.UUID,
			) (*models.RehydrationBundle, error) {
				return nil, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{
			username: actor.Username,
			password: "StrongPassword-12345",
		},
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/engrams/00000000-0000-0000-0000-000000000904/rehydrate",
		nil,
	)
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	requireEqual(t, http.StatusNotFound, response.Code)
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode rehydrate response: %v", err)
	}
	requireEqual(t, "Engram not found", payload["detail"].(string))
}

func TestMountSessionAuthRoutesListEngramSourcesUsesRepository(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAdmin)
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000905")
	sourceID := uuid.MustParse("00000000-0000-0000-0000-000000000906")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			getRehydrationBundle: func(
				_ context.Context,
				receivedEngramID uuid.UUID,
				receivedActorUserID uuid.UUID,
			) (*models.RehydrationBundle, error) {
				requireEqual(t, engramID, receivedEngramID)
				requireEqual(t, actor.UserID, receivedActorUserID)
				return &models.RehydrationBundle{EngramID: engramID, ProjectID: "proj-1"}, nil
			},
			getEngramSources: func(
				_ context.Context,
				receivedEngramID uuid.UUID,
				limit int,
				receivedActorUserID uuid.UUID,
			) ([]models.EngramSourceRecord, error) {
				requireEqual(t, engramID, receivedEngramID)
				requireEqual(t, 11, limit)
				requireEqual(t, actor.UserID, receivedActorUserID)
				return []models.EngramSourceRecord{
					{
						SourceID:   sourceID,
						EngramID:   engramID,
						CapturedAt: time.Date(2026, 2, 22, 7, 30, 0, 0, time.UTC),
						URL:        "https://example.com/source",
						Snippet:    stringPointer("snippet"),
					},
				}, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{
			username: actor.Username,
			password: "StrongPassword-12345",
		},
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/engrams/"+engramID.String()+"/sources?limit=11",
		nil,
	)
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	requireEqual(t, http.StatusOK, response.Code)
	var payload []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode sources response: %v", err)
	}
	requireEqual(t, 1, len(payload))
	requireEqual(t, sourceID.String(), payload[0]["source_id"].(string))
}

func TestMountSessionAuthRoutesCreateEngramRequiresProjectOrDefault(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAdmin)
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			resolveProjectIDForWrite: func(
				_ context.Context,
				_ uuid.UUID,
				_ models.UserRole,
				_ string,
			) (SessionProjectResolution, error) {
				return SessionProjectResolution{}, ErrProjectIDRequiredWhenNoDefaultProject
			},
			createEngram: func(
				_ context.Context,
				_ models.MemoryEngramCreate,
				_ uuid.UUID,
			) (*models.EngramCreateResponse, error) {
				t.Fatalf("create engram should not be called when project resolution fails")
				return nil, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{
			username: actor.Username,
			password: "StrongPassword-12345",
		},
	)

	body := map[string]any{
		"title":                     "t",
		"abstract":                  "a",
		"detailed_summary_markdown": "d",
	}
	encodedBody, _ := json.Marshal(body)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/engrams", bytes.NewReader(encodedBody))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	requireEqual(t, http.StatusUnprocessableEntity, response.Code)
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode create validation response: %v", err)
	}
	requireEqual(t, ErrProjectIDRequiredWhenNoDefaultProject.Error(), payload["detail"].(string))
}

func TestMountSessionAuthRoutesShareEngramReturnsUpdatedVisibility(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAnalyst)
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000909")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			shareEngram: func(
				_ context.Context,
				actorUserID uuid.UUID,
				actorRole models.UserRole,
				receivedEngramID uuid.UUID,
			) (*models.EngramVisibilityRecord, error) {
				requireEqual(t, actor.UserID, actorUserID)
				requireEqual(t, models.UserRoleAnalyst, actorRole)
				requireEqual(t, engramID, receivedEngramID)
				return &models.EngramVisibilityRecord{
					EngramID:        engramID,
					ProjectID:       "proj-1",
					VisibilityScope: models.VisibilityScopeProject,
				}, nil
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{
			username: actor.Username,
			password: "StrongPassword-12345",
		},
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/engrams/"+engramID.String()+"/share", nil)
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	requireEqual(t, http.StatusOK, response.Code)
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode share response: %v", err)
	}
	requireEqual(t, engramID.String(), payload["engram_id"].(string))
	requireEqual(t, "project", payload["visibility_scope"].(string))
}

func TestMountSessionAuthRoutesShareEngramMapsForbidden(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleViewer)
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000910")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			shareEngram: func(
				_ context.Context,
				_ uuid.UUID,
				_ models.UserRole,
				_ uuid.UUID,
			) (*models.EngramVisibilityRecord, error) {
				return nil, projects.ErrEngramShareForbidden
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{
			username: actor.Username,
			password: "StrongPassword-12345",
		},
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/engrams/"+engramID.String()+"/share", nil)
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	requireEqual(t, http.StatusForbidden, response.Code)
}

func TestMountSessionAuthRoutesUnshareEngramMapsNotFound(t *testing.T) {
	actor := newSessionRoutesTestActor(t, models.UserRoleAdmin)
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000911")
	handler, manager := buildSessionEngramRoutesTestHandler(
		t,
		sessionEngramRoutesHandlerOptions{
			actor: actor,
			unshareEngram: func(
				_ context.Context,
				_ uuid.UUID,
				_ models.UserRole,
				_ uuid.UUID,
			) (*models.EngramVisibilityRecord, error) {
				return nil, errors.Join(projects.ErrEngramNotFound)
			},
		},
	)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{
			username: actor.Username,
			password: "StrongPassword-12345",
		},
	)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/engrams/"+engramID.String()+"/unshare", nil)
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	requireEqual(t, http.StatusNotFound, response.Code)
}

func buildSessionEngramRoutesTestHandler(
	t *testing.T,
	options sessionEngramRoutesHandlerOptions,
) (http.Handler, *auth.SessionManager) {
	t.Helper()
	actor := options.actor
	if actor == nil {
		actor = newSessionRoutesTestActor(t, models.UserRoleAdmin)
	}

	manager, err := auth.NewSessionManager("dev-session-secret-for-tests", auth.DefaultSessionCookieName)
	if err != nil {
		t.Fatalf("expected manager creation to succeed: %v", err)
	}

	lookupByUsername := func(_ context.Context, username string) (*models.UserAuthRecord, error) {
		if username == actor.Username {
			return actor, nil
		}
		return nil, nil
	}
	lookupByID := func(_ context.Context, userID uuid.UUID) (*models.UserAuthRecord, error) {
		if userID == actor.UserID {
			return actor, nil
		}
		return nil, nil
	}

	router := chi.NewRouter()
	MountSessionAuthRoutes(
		router,
		SessionAuthDependencies{
			SessionManager:           manager,
			LookupUserByUsername:     lookupByUsername,
			LookupUserByID:           lookupByID,
			VerifyPassword:           auth.VerifyPassword,
			GenerateCSRFToken:        auth.GenerateCSRFToken,
			ResolveProjectIDForWrite: options.resolveProjectIDForWrite,
			CreateEngram:             options.createEngram,
			ListEngrams:              options.listEngrams,
			QueryEngrams:             options.queryEngrams,
			GetRehydrationBundle:     options.getRehydrationBundle,
			GetEngramSources:         options.getEngramSources,
			ShareEngram:              options.shareEngram,
			UnshareEngram:            options.unshareEngram,
			CreateEngramLink:         options.createEngramLink,
			ListEngramLinks:          options.listEngramLinks,
			UpdateEngramLink:         options.updateEngramLink,
			ArchiveEngramLink:        options.archiveEngramLink,
			SuggestEngramLinks:       options.suggestEngramLinks,
			HygieneEngramLinks:       options.hygieneEngramLinks,
			TraceEngramLinks:         options.traceEngramLinks,
		},
	)
	return SessionActorMiddleware(manager, lookupByID)(router), manager
}

func loginSessionEngramActor(
	t *testing.T,
	handler http.Handler,
	manager *auth.SessionManager,
	credentials sessionEngramLoginCredentials,
) *http.Cookie {
	t.Helper()

	csrfToken, sessionCookie := fetchCSRFTokenAndCookie(t, handler, manager)
	loginBody := map[string]string{
		"username":   credentials.username,
		"password":   credentials.password,
		"csrf_token": csrfToken,
	}
	encodedBody, _ := json.Marshal(loginBody)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/session/login", bytes.NewReader(encodedBody))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(sessionCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	requireEqual(t, http.StatusOK, response.Code)
	loginCookie := findResponseCookie(response, manager.CookieName())
	if loginCookie == nil {
		t.Fatalf("expected login cookie to be present")
	}
	return loginCookie
}

func stringPointer(value string) *string {
	return &value
}

func assertCollectionRouteUsesRepository(
	t *testing.T,
	options sessionEngramRoutesHandlerOptions,
	assertion engramCollectionAssertion,
) {
	t.Helper()
	actor := options.actor
	if actor == nil {
		actor = newSessionRoutesTestActor(t, models.UserRoleAdmin)
		options.actor = actor
	}
	handler, manager := buildSessionEngramRoutesTestHandler(t, options)
	loginCookie := loginSessionEngramActor(
		t,
		handler,
		manager,
		sessionEngramLoginCredentials{
			username: actor.Username,
			password: "StrongPassword-12345",
		},
	)
	assertEngramCollectionRouteReturnsSingleID(t, handler, loginCookie, assertion)
}

func assertEngramCollectionRouteReturnsSingleID(
	t *testing.T,
	handler http.Handler,
	loginCookie *http.Cookie,
	assertion engramCollectionAssertion,
) {
	t.Helper()
	response := executeEngramRequest(t, handler, loginCookie, assertion.requestSpec)
	requireEqual(t, http.StatusOK, response.Code)
	var payload []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode %s response: %v", assertion.contextName, err)
	}
	requireEqual(t, 1, len(payload))
	requireEqual(t, assertion.expectedID.String(), payload[0]["engram_id"].(string))
}

func executeEngramRequest(
	t *testing.T,
	handler http.Handler,
	loginCookie *http.Cookie,
	requestSpec engramRequestSpec,
) *httptest.ResponseRecorder {
	t.Helper()
	var requestBody []byte
	if requestSpec.body != nil {
		requestBody, _ = json.Marshal(requestSpec.body)
	}
	request := httptest.NewRequest(requestSpec.method, requestSpec.path, bytes.NewReader(requestBody))
	if requestSpec.body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.AddCookie(loginCookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

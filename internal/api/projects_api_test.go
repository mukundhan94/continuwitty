package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/projects"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type fakeProjectService struct {
	listProjectsFn        func(ctx context.Context, request ProjectListRouteRequest) ([]models.ProjectRecord, error)
	createProjectFn       func(ctx context.Context, request ProjectCreateRouteRequest) (*models.ProjectRecord, error)
	getDefaultProjectIDFn func(ctx context.Context, actorUserID uuid.UUID) (*string, error)
	setDefaultProjectIDFn func(ctx context.Context, request ProjectDefaultUpdateRouteRequest) (string, error)
}

func (service fakeProjectService) ListProjects(
	ctx context.Context,
	request ProjectListRouteRequest,
) ([]models.ProjectRecord, error) {
	if service.listProjectsFn == nil {
		return []models.ProjectRecord{}, nil
	}
	return service.listProjectsFn(ctx, request)
}

func (service fakeProjectService) CreateProject(
	ctx context.Context,
	request ProjectCreateRouteRequest,
) (*models.ProjectRecord, error) {
	if service.createProjectFn == nil {
		return nil, nil
	}
	return service.createProjectFn(ctx, request)
}

func (service fakeProjectService) GetDefaultProjectID(
	ctx context.Context,
	actorUserID uuid.UUID,
) (*string, error) {
	if service.getDefaultProjectIDFn == nil {
		return nil, nil
	}
	return service.getDefaultProjectIDFn(ctx, actorUserID)
}

func (service fakeProjectService) SetDefaultProjectID(
	ctx context.Context,
	request ProjectDefaultUpdateRouteRequest,
) (string, error) {
	if service.setDefaultProjectIDFn == nil {
		return "", nil
	}
	return service.setDefaultProjectIDFn(ctx, request)
}

func TestMountProjectRoutesRegistersEndpoints(t *testing.T) {
	router := chi.NewRouter()
	MountProjectRoutes(router, fakeProjectService{})
	routes := collectChatRoutes(t, router)

	requiredRoutes := []string{
		"/api/v1/projects",
		"/api/v1/projects/default",
	}
	for _, route := range requiredRoutes {
		requireChatRoute(t, routes, route)
	}
}

func TestListProjectsHandlerUsesDefaultPaging(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000931")
	capturedIncludeArchived := false
	capturedLimit := 0
	capturedOffset := 0
	capturedRole := models.UserRole("")
	service := fakeProjectService{
		listProjectsFn: func(_ context.Context, request ProjectListRouteRequest) ([]models.ProjectRecord, error) {
			if request.ActorUserID != actorID {
				t.Fatalf("expected actor id %s, got %s", actorID, request.ActorUserID)
			}
			capturedRole = request.ActorRole
			capturedIncludeArchived = request.IncludeArchived
			capturedLimit = request.Limit
			capturedOffset = request.Offset
			return []models.ProjectRecord{}, nil
		},
	}
	router := chi.NewRouter()
	MountProjectRoutes(router, service)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "analyst"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if capturedRole != models.UserRoleAnalyst {
		t.Fatalf("expected role analyst, got %s", capturedRole)
	}
	if capturedIncludeArchived {
		t.Fatalf("expected include_archived false by default")
	}
	if capturedLimit != defaultProjectListLimit {
		t.Fatalf("expected default limit %d, got %d", defaultProjectListLimit, capturedLimit)
	}
	if capturedOffset != defaultProjectListOffset {
		t.Fatalf("expected default offset %d, got %d", defaultProjectListOffset, capturedOffset)
	}
}

func TestCreateProjectHandlerWritesCreatedResponse(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000932")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000933")
	capturedRequest := projects.CreateProjectRequest{}
	service := fakeProjectService{
		createProjectFn: func(_ context.Context, request ProjectCreateRouteRequest) (*models.ProjectRecord, error) {
			if request.ActorUserID != actorID {
				t.Fatalf("expected actor id %s, got %s", actorID, request.ActorUserID)
			}
			if request.ActorRole != models.UserRoleAdmin {
				t.Fatalf("expected actor role admin, got %s", request.ActorRole)
			}
			capturedRequest = projects.CreateProjectRequest{
				ProjectID:   request.Payload.ProjectID,
				Name:        request.Payload.Name,
				Description: request.Payload.Description,
				OwnerUserID: request.Payload.OwnerUserID,
			}
			now := time.Date(2026, 2, 26, 16, 0, 0, 0, time.UTC)
			return &models.ProjectRecord{
				ProjectID:   request.Payload.ProjectID,
				Name:        request.Payload.Name,
				Description: request.Payload.Description,
				OwnerUserID: ownerUserID,
				IsArchived:  false,
				CreatedAt:   now,
				UpdatedAt:   now,
			}, nil
		},
	}
	router := chi.NewRouter()
	MountProjectRoutes(router, service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects",
		strings.NewReader(`{"project_id":"project-alpha","name":"Project Alpha","description":"Alpha project","owner_user_id":"00000000-0000-0000-0000-000000000933"}`),
	)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "admin"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	if capturedRequest.ProjectID != "project-alpha" || capturedRequest.Name != "Project Alpha" {
		t.Fatalf("expected captured request to include payload fields, got %#v", capturedRequest)
	}
	var payload models.ProjectRecord
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ProjectID != "project-alpha" {
		t.Fatalf("expected project id project-alpha, got %q", payload.ProjectID)
	}
}

func TestGetDefaultProjectHandlerWritesResponse(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000934")
	defaultProjectID := "project-default"
	service := fakeProjectService{
		getDefaultProjectIDFn: func(_ context.Context, actorUserID uuid.UUID) (*string, error) {
			if actorUserID != actorID {
				t.Fatalf("expected actor id %s, got %s", actorID, actorUserID)
			}
			return &defaultProjectID, nil
		},
	}
	router := chi.NewRouter()
	MountProjectRoutes(router, service)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/projects/default", nil)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "viewer"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var payload models.ProjectDefaultResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.DefaultProjectID == nil || *payload.DefaultProjectID != defaultProjectID {
		t.Fatalf("expected default_project_id %q, got %#v", defaultProjectID, payload.DefaultProjectID)
	}
}

func TestSetDefaultProjectHandlerMapsProjectNotFound(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000935")
	service := fakeProjectService{
		setDefaultProjectIDFn: func(_ context.Context, request ProjectDefaultUpdateRouteRequest) (string, error) {
			if request.ActorUserID != actorID {
				t.Fatalf("expected actor id %s, got %s", actorID, request.ActorUserID)
			}
			if request.ActorRole != models.UserRoleAnalyst {
				t.Fatalf("expected actor role analyst, got %s", request.ActorRole)
			}
			if request.ProjectID != "missing-project" {
				t.Fatalf("expected project id missing-project, got %s", request.ProjectID)
			}
			return "", projects.ErrProjectNotFound
		},
	}
	router := chi.NewRouter()
	MountProjectRoutes(router, service)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/projects/default",
		strings.NewReader(`{"project_id":"missing-project"}`),
	)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "analyst"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["detail"] != projects.ErrProjectNotFound.Error() {
		t.Fatalf("expected detail %q, got %q", projects.ErrProjectNotFound.Error(), payload["detail"])
	}
}

func TestProjectRoutesRequireAuthenticatedActor(t *testing.T) {
	router := chi.NewRouter()
	MountProjectRoutes(router, fakeProjectService{})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
}

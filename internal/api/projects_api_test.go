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
	listProjectMembersFn  func(ctx context.Context, request ProjectMemberListRouteRequest) ([]models.ProjectMemberRecord, error)
	addProjectMemberFn    func(ctx context.Context, request ProjectMemberCreateRouteRequest) (*models.ProjectMemberRecord, error)
	updateProjectMemberFn func(ctx context.Context, request ProjectMemberUpdateRouteRequest) (*models.ProjectMemberRecord, error)
	removeProjectMemberFn func(ctx context.Context, request ProjectMemberDeleteRouteRequest) error
	listProjectAuditFn    func(ctx context.Context, request ProjectAuditListRouteRequest) ([]models.ProjectAuditEventRecord, error)
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

func (service fakeProjectService) ListProjectMembers(
	ctx context.Context,
	request ProjectMemberListRouteRequest,
) ([]models.ProjectMemberRecord, error) {
	if service.listProjectMembersFn == nil {
		return []models.ProjectMemberRecord{}, nil
	}
	return service.listProjectMembersFn(ctx, request)
}

func (service fakeProjectService) AddProjectMember(
	ctx context.Context,
	request ProjectMemberCreateRouteRequest,
) (*models.ProjectMemberRecord, error) {
	if service.addProjectMemberFn == nil {
		return nil, nil
	}
	return service.addProjectMemberFn(ctx, request)
}

func (service fakeProjectService) UpdateProjectMember(
	ctx context.Context,
	request ProjectMemberUpdateRouteRequest,
) (*models.ProjectMemberRecord, error) {
	if service.updateProjectMemberFn == nil {
		return nil, nil
	}
	return service.updateProjectMemberFn(ctx, request)
}

func (service fakeProjectService) RemoveProjectMember(
	ctx context.Context,
	request ProjectMemberDeleteRouteRequest,
) error {
	if service.removeProjectMemberFn == nil {
		return nil
	}
	return service.removeProjectMemberFn(ctx, request)
}

func (service fakeProjectService) ListProjectAuditEvents(
	ctx context.Context,
	request ProjectAuditListRouteRequest,
) ([]models.ProjectAuditEventRecord, error) {
	if service.listProjectAuditFn == nil {
		return []models.ProjectAuditEventRecord{}, nil
	}
	return service.listProjectAuditFn(ctx, request)
}

func TestMountProjectRoutesRegistersEndpoints(t *testing.T) {
	router := chi.NewRouter()
	MountProjectRoutes(router, fakeProjectService{})
	routes := collectChatRoutes(t, router)

	requiredRoutes := []string{
		"/api/v1/projects",
		"/api/v1/projects/default",
		"/api/v1/projects/{project_id}/members",
		"/api/v1/projects/{project_id}/members/{user_id}",
		"/api/v1/projects/{project_id}/audit-events",
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

func TestListProjectMembersHandlerUsesQueryDefaults(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000936")
	captured := ProjectMemberListRouteRequest{}
	service := fakeProjectService{
		listProjectMembersFn: func(_ context.Context, request ProjectMemberListRouteRequest) ([]models.ProjectMemberRecord, error) {
			captured = request
			return []models.ProjectMemberRecord{}, nil
		},
	}
	router := chi.NewRouter()
	MountProjectRoutes(router, service)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/projects/engram-vault/members", nil)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "admin"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if captured.ProjectID != "engram-vault" {
		t.Fatalf("expected project id engram-vault, got %q", captured.ProjectID)
	}
	if captured.ActorUserID != actorID {
		t.Fatalf("expected actor %s, got %s", actorID, captured.ActorUserID)
	}
	if captured.ActorRole != models.UserRoleAdmin {
		t.Fatalf("expected admin actor role, got %s", captured.ActorRole)
	}
	if captured.Limit != defaultProjectMemberLimit || captured.Offset != defaultProjectMemberOffset {
		t.Fatalf("unexpected paging defaults: %#v", captured)
	}
}

func TestCreateProjectMemberHandlerRejectsInvalidRole(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000937")
	router := chi.NewRouter()
	MountProjectRoutes(router, fakeProjectService{})

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/projects/engram-vault/members",
		strings.NewReader(`{"user_id":"00000000-0000-0000-0000-000000000099","role":"invalid"}`),
	)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "admin"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", response.Code)
	}
}

func TestListProjectAuditEventsHandlerMapsForbidden(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000938")
	service := fakeProjectService{
		listProjectAuditFn: func(_ context.Context, request ProjectAuditListRouteRequest) ([]models.ProjectAuditEventRecord, error) {
			if request.ProjectID != "engram-vault" {
				t.Fatalf("expected project id engram-vault, got %q", request.ProjectID)
			}
			return nil, projects.ErrProjectAuditForbidden
		},
	}
	router := chi.NewRouter()
	MountProjectRoutes(router, service)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/projects/engram-vault/audit-events", nil)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "viewer"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", response.Code)
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

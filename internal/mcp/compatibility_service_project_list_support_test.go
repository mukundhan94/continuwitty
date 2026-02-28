package mcp

import (
	"context"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

type projectListCallCapture struct {
	actorUserID     uuid.UUID
	actorRole       models.UserRole
	includeArchived bool
	limit           int
	offset          int
}

type fakeProjectListService struct {
	capture     *projectListCallCapture
	ownerUserID uuid.UUID
}

func (service fakeProjectListService) ListProjects(
	_ context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	includeArchived bool,
	limit int,
	offset int,
) ([]models.ProjectRecord, error) {
	if service.capture != nil {
		service.capture.actorUserID = actorUserID
		service.capture.actorRole = actorRole
		service.capture.includeArchived = includeArchived
		service.capture.limit = limit
		service.capture.offset = offset
	}
	return []models.ProjectRecord{buildProjectRecord(service.ownerUserID)}, nil
}

func (fakeProjectListService) GetDefaultProjectID(
	_ context.Context,
	_ uuid.UUID,
) (*string, error) {
	return nil, nil
}

func (fakeProjectListService) CreateProject(
	_ context.Context,
	_ uuid.UUID,
	_ models.UserRole,
	_ projects.CreateProjectRequest,
) (*models.ProjectRecord, error) {
	return nil, nil
}

func (fakeProjectListService) SetDefaultProjectID(
	_ context.Context,
	_ uuid.UUID,
	_ models.UserRole,
	_ string,
) (string, error) {
	return "", nil
}

func (fakeProjectListService) ListProjectMembers(
	_ context.Context,
	_ uuid.UUID,
	_ models.UserRole,
	_ string,
	_ bool,
	_ int,
	_ int,
) ([]models.ProjectMemberRecord, error) {
	return []models.ProjectMemberRecord{}, nil
}

func (fakeProjectListService) AddProjectMember(
	_ context.Context,
	_ uuid.UUID,
	_ models.UserRole,
	_ projects.ProjectMemberCreateRequest,
) (*models.ProjectMemberRecord, error) {
	return nil, nil
}

func (fakeProjectListService) UpdateProjectMember(
	_ context.Context,
	_ uuid.UUID,
	_ models.UserRole,
	_ projects.ProjectMemberUpdateRequest,
) (*models.ProjectMemberRecord, error) {
	return nil, nil
}

func (fakeProjectListService) RemoveProjectMember(
	_ context.Context,
	_ uuid.UUID,
	_ models.UserRole,
	_ projects.ProjectMemberRemoveRequest,
) error {
	return nil
}

func (fakeProjectListService) ShareEngram(
	_ context.Context,
	_ uuid.UUID,
	_ models.UserRole,
	_ uuid.UUID,
) (*models.EngramVisibilityRecord, error) {
	return nil, nil
}

func (fakeProjectListService) UnshareEngram(
	_ context.Context,
	_ uuid.UUID,
	_ models.UserRole,
	_ uuid.UUID,
) (*models.EngramVisibilityRecord, error) {
	return nil, nil
}

func dispatchProjectListForTest(
	t *testing.T,
	actorUserID uuid.UUID,
	capture *projectListCallCapture,
) Frame {
	t.Helper()
	ownerUserID := uuid.MustParse("22000000-0000-0000-0000-000000000099")
	service := NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{
			ProjectService: fakeProjectListService{capture: capture, ownerUserID: ownerUserID},
		},
	)
	return runCompatibilityRequestWithService(
		t,
		service,
		StreamCallRequest{
			Request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "tools-call",
				Method:  "tools/call",
				Params: map[string]any{
					"name": "project_list",
					"arguments": map[string]any{
						"include_archived": true,
						"limit":            12.0,
						"offset":           3.0,
					},
				},
			},
			Actor: Actor{UserID: actorUserID, Role: "analyst"},
		},
	)
}

func assertProjectListCall(
	t *testing.T,
	capture projectListCallCapture,
	actorUserID uuid.UUID,
) {
	t.Helper()
	if capture.actorUserID != actorUserID {
		t.Fatalf("expected actor_user_id to be forwarded")
	}
	if capture.actorRole != models.UserRoleAnalyst {
		t.Fatalf("expected actor role to be forwarded")
	}
	if !capture.includeArchived {
		t.Fatalf("expected include_archived to be forwarded")
	}
	if capture.limit != 12 {
		t.Fatalf("expected limit to be forwarded")
	}
	if capture.offset != 3 {
		t.Fatalf("expected offset to be forwarded")
	}
}

func assertProjectListPayload(t *testing.T, frame Frame) {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	structuredContent := mapFromMap(t, result, "structuredContent")
	projects, ok := structuredContent["projects"].([]models.ProjectRecord)
	if !ok {
		t.Fatalf("expected project list payload")
	}
	if len(projects) != 1 || projects[0].ProjectID != "proj-alpha" {
		t.Fatalf("expected one listed project in payload")
	}
}

func buildProjectRecord(ownerUserID uuid.UUID) models.ProjectRecord {
	return models.ProjectRecord{
		ProjectID:   "proj-alpha",
		Name:        "Alpha",
		Description: "primary",
		OwnerUserID: ownerUserID,
		CreatedAt:   time.Unix(0, 0).UTC(),
		UpdatedAt:   time.Unix(0, 0).UTC(),
	}
}

package mcp

import (
	"context"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

func TestCompatibilityServiceProjectCreateToolsCall(t *testing.T) {
	actorUserID := uuid.MustParse("24000000-0000-0000-0000-000000000240")
	ownerUserID := uuid.MustParse("24000000-0000-0000-0000-000000000241")
	service := &fakeProjectCreateService{result: projectRecordForCreate(ownerUserID)}

	frame := runCompatibilityRequestWithService(
		t,
		newProjectDefaultCompatibilityService(service),
		toolsCallRequest(
			actorUserID.String(),
			"project_create",
			map[string]any{
				"project_id":    "proj-created",
				"name":          "Created",
				"description":   "From MCP",
				"owner_user_id": ownerUserID.String(),
			},
		),
	)

	result := resultPayloadFromFrame(t, frame)
	structuredContent := mapFromMap(t, result, "structuredContent")
	project := asProjectRecord(t, structuredContent["project"])
	if project.ProjectID != "proj-created" {
		t.Fatalf("expected created project payload")
	}
	assertProjectCreateCall(
		t,
		service.call,
		projectCreateExpectation{
			actorUserID: actorUserID,
			actorRole:   models.UserRoleAnalyst,
			ownerUserID: ownerUserID,
			projectID:   "proj-created",
		},
	)
}

func TestCompatibilityServiceProjectCreateDirectMethod(t *testing.T) {
	actorUserID := uuid.MustParse("24100000-0000-0000-0000-000000000241")
	service := &fakeProjectCreateService{result: projectRecordForCreate(actorUserID)}

	frame := runCompatibilityRequestWithService(
		t,
		newProjectDefaultCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"project.create",
			map[string]any{
				"project_id":  "proj-direct",
				"name":        "Direct",
				"description": "Path",
			},
		),
	)
	result := resultPayloadFromFrame(t, frame)
	project := asProjectRecord(t, result["project"])
	if project.ProjectID != "proj-created" {
		t.Fatalf("expected direct project payload")
	}
}

func TestCompatibilityServiceProjectCreateErrors(t *testing.T) {
	testCases := []struct {
		name         string
		service      *fakeProjectCreateService
		request      StreamCallRequest
		expectedCode int
	}{
		{
			name:    "invalid owner user id",
			service: &fakeProjectCreateService{},
			request: toolsCallRequest(
				"24200000-0000-0000-0000-000000000242",
				"project_create",
				map[string]any{
					"project_id":    "proj-invalid-owner",
					"owner_user_id": "invalid-uuid",
				},
			),
			expectedCode: -32602,
		},
		{
			name:    "blank project id",
			service: &fakeProjectCreateService{err: projects.ErrProjectIDMustNotBeBlank},
			request: toolsCallRequest(
				"24300000-0000-0000-0000-000000000243",
				"project_create",
				map[string]any{"project_id": ""},
			),
			expectedCode: -32602,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newProjectDefaultCompatibilityService(testCase.service),
				testCase.request,
			)
			errorPayload := errorPayloadFromFrame(t, frame)
			requireErrorCode(t, errorPayload, testCase.expectedCode)
		})
	}
}

func asProjectRecord(t *testing.T, value any) models.ProjectRecord {
	t.Helper()
	switch typed := value.(type) {
	case models.ProjectRecord:
		return typed
	case *models.ProjectRecord:
		if typed == nil {
			t.Fatalf("expected non-nil project pointer")
		}
		return *typed
	default:
		t.Fatalf("expected project payload, got %T", value)
		return models.ProjectRecord{}
	}
}

func assertProjectCreateCall(
	t *testing.T,
	call projectCreateCall,
	expected projectCreateExpectation,
) {
	t.Helper()
	if call.actorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.actorRole != expected.actorRole {
		t.Fatalf("expected actor role forwarded")
	}
	if call.request.ProjectID != expected.projectID {
		t.Fatalf("expected project id forwarded")
	}
	if call.request.OwnerUserID == nil || *call.request.OwnerUserID != expected.ownerUserID {
		t.Fatalf("expected owner_user_id forwarded")
	}
}

func projectRecordForCreate(ownerUserID uuid.UUID) *models.ProjectRecord {
	createdAt := time.Unix(0, 0).UTC()
	return &models.ProjectRecord{
		ProjectID:   "proj-created",
		Name:        "Created",
		Description: "From MCP",
		OwnerUserID: ownerUserID,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
}

type projectCreateCall struct {
	actorUserID uuid.UUID
	actorRole   models.UserRole
	request     projects.CreateProjectRequest
}

type projectCreateExpectation struct {
	actorUserID uuid.UUID
	actorRole   models.UserRole
	ownerUserID uuid.UUID
	projectID   string
}

type fakeProjectCreateService struct {
	fakeProjectListService
	result *models.ProjectRecord
	err    error
	call   projectCreateCall
}

func (service *fakeProjectCreateService) CreateProject(
	_ context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request projects.CreateProjectRequest,
) (*models.ProjectRecord, error) {
	service.call = projectCreateCall{actorUserID: actorUserID, actorRole: actorRole, request: request}
	if service.err != nil {
		return nil, service.err
	}
	if service.result != nil {
		return service.result, nil
	}
	return projectRecordForCreate(actorUserID), nil
}

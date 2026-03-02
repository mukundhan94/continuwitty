package mcp

import (
	"context"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

func TestCompatibilityServiceProjectMemberAddParity(t *testing.T) {
	actorUserID := uuid.MustParse("50010000-0000-0000-0000-000000000501")
	memberUserID := uuid.MustParse("50010000-0000-0000-0000-000000000502")
	service := &fakeProjectMembershipService{
		addResult: projectMemberRecordForMCP("engram-vault", memberUserID, models.ProjectMemberRoleViewer),
	}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name: "direct",
			request: directToolRequest(
				actorUserID.String(),
				"project.member_add",
				map[string]any{
					"project_id": "engram-vault",
					"user_id":    memberUserID.String(),
					"role":       "viewer",
				},
			),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name: "tools call",
			request: toolsCallRequest(
				actorUserID.String(),
				"project_member_add",
				map[string]any{
					"project_id": "engram-vault",
					"user_id":    memberUserID.String(),
					"role":       "viewer",
				},
			),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newProjectDefaultCompatibilityService(service),
				testCase.request,
			)
			member := projectMemberPayloadFromFrame(t, frame, testCase.asToolsCallPath)
			if member.UserID != memberUserID || member.ProjectID != "engram-vault" {
				t.Fatalf("expected member payload from service")
			}
			call := service.addCall
			if call.actorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			if call.actorRole != testCase.expectedRole {
				t.Fatalf("expected actor role forwarded")
			}
			if call.request.UserID != memberUserID || call.request.Role != models.ProjectMemberRoleViewer {
				t.Fatalf("expected member add request payload forwarded")
			}
		})
	}
}

func TestCompatibilityServiceProjectMemberListUsesRepository(t *testing.T) {
	actorUserID := uuid.MustParse("50020000-0000-0000-0000-000000000501")
	memberUserID := uuid.MustParse("50020000-0000-0000-0000-000000000502")
	service := &fakeProjectMembershipService{
		listResult: []models.ProjectMemberRecord{
			*projectMemberRecordForMCP("engram-vault", memberUserID, models.ProjectMemberRoleEditor),
		},
	}

	listFrame := runCompatibilityRequestWithService(
		t,
		newProjectDefaultCompatibilityService(service),
		toolsCallRequest(
			actorUserID.String(),
			"project_member_list",
			map[string]any{
				"project_id":      "engram-vault",
				"include_revoked": false,
				"limit":           12.0,
				"offset":          3.0,
			},
		),
	)
	listResult := mapFromMap(t, resultPayloadFromFrame(t, listFrame), "structuredContent")
	members, ok := listResult["members"].([]models.ProjectMemberRecord)
	if !ok {
		t.Fatalf("expected members payload")
	}
	if len(members) != 1 || members[0].Role != models.ProjectMemberRoleEditor {
		t.Fatalf("expected listed members")
	}
	assertProjectMemberListCall(
		t,
		service.listCall,
		projectMemberListCallExpectation{
			projectID: "engram-vault",
			limit:     12,
			offset:    3,
		},
	)
}

func TestCompatibilityServiceProjectMemberRemoveUsesRepository(t *testing.T) {
	actorUserID := uuid.MustParse("50020000-0000-0000-0000-000000000501")
	memberUserID := uuid.MustParse("50020000-0000-0000-0000-000000000502")
	service := &fakeProjectMembershipService{}
	removeFrame := runCompatibilityRequestWithService(
		t,
		newProjectDefaultCompatibilityService(service),
		toolsCallRequest(
			actorUserID.String(),
			"project_member_remove",
			map[string]any{
				"project_id": "engram-vault",
				"user_id":    memberUserID.String(),
			},
		),
	)
	removePayload := mapFromMap(t, resultPayloadFromFrame(t, removeFrame), "structuredContent")
	if removePayload["removed"] != true {
		t.Fatalf("expected removed=true payload")
	}
	if service.removeCall.request.UserID != memberUserID || service.removeCall.request.ProjectID != "engram-vault" {
		t.Fatalf("expected member remove request forwarded")
	}
}

func TestCompatibilityServiceProjectMemberValidationAndErrorMapping(t *testing.T) {
	service := newProjectDefaultCompatibilityService(&fakeProjectMembershipService{})
	testCases := []struct {
		name   string
		method string
		params map[string]any
	}{
		{name: "list missing project id", method: "project_member_list", params: map[string]any{}},
		{name: "add invalid user id", method: "project_member_add", params: map[string]any{"project_id": "engram-vault", "user_id": "bad", "role": "viewer"}},
		{name: "update invalid role", method: "project_member_update", params: map[string]any{"project_id": "engram-vault", "user_id": uuid.NewString(), "role": "bad"}},
		{name: "remove missing user id", method: "project_member_remove", params: map[string]any{"project_id": "engram-vault"}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest("50030000-0000-0000-0000-000000000501", testCase.method, testCase.params),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	forbiddenFrame := runCompatibilityRequestWithService(
		t,
		newProjectDefaultCompatibilityService(
			&fakeProjectMembershipService{addErr: projects.ErrProjectMemberManagementForbidden},
		),
		toolsCallRequest(
			"50030000-0000-0000-0000-000000000502",
			"project_member_add",
			map[string]any{
				"project_id": "engram-vault",
				"user_id":    uuid.NewString(),
				"role":       "viewer",
			},
		),
	)
	forbiddenPayload := errorPayloadFromFrame(t, forbiddenFrame)
	requireErrorCode(t, forbiddenPayload, -32602)
	assertErrorStatusCode(t, forbiddenPayload, 403)
	assertErrorDetail(t, forbiddenPayload, projects.ErrProjectMemberManagementForbidden.Error())

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newProjectDefaultCompatibilityService(
			&fakeProjectMembershipService{removeErr: projects.ErrProjectMemberNotFound},
		),
		toolsCallRequest(
			"50030000-0000-0000-0000-000000000503",
			"project_member_remove",
			map[string]any{
				"project_id": "engram-vault",
				"user_id":    uuid.NewString(),
			},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertErrorStatusCode(t, notFoundPayload, 404)
	assertErrorDetail(t, notFoundPayload, projects.ErrProjectMemberNotFound.Error())
}

func projectMemberPayloadFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.ProjectMemberRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	memberValue, exists := payload["member"]
	if !exists {
		t.Fatalf("expected member payload")
	}
	switch typed := memberValue.(type) {
	case models.ProjectMemberRecord:
		return typed
	case *models.ProjectMemberRecord:
		if typed == nil {
			t.Fatalf("expected non-nil member pointer")
		}
		return *typed
	default:
		t.Fatalf("unexpected member payload type %T", memberValue)
		return models.ProjectMemberRecord{}
	}
}

func projectMemberRecordForMCP(
	projectID string,
	userID uuid.UUID,
	role models.ProjectMemberRole,
) *models.ProjectMemberRecord {
	now := time.Unix(0, 0).UTC()
	return &models.ProjectMemberRecord{
		ProjectID: projectID,
		UserID:    userID,
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type projectMemberListCall struct {
	actorUserID    uuid.UUID
	actorRole      models.UserRole
	projectID      string
	includeRevoked bool
	limit          int
	offset         int
}

type projectMemberMutateCall[T any] struct {
	actorUserID uuid.UUID
	actorRole   models.UserRole
	request     T
}

type projectMemberListCallExpectation struct {
	projectID string
	limit     int
	offset    int
}

func assertProjectMemberListCall(
	t *testing.T,
	call projectMemberListCall,
	expected projectMemberListCallExpectation,
) {
	t.Helper()
	if call.projectID != expected.projectID {
		t.Fatalf("expected project id %q, got %q", expected.projectID, call.projectID)
	}
	if call.limit != expected.limit {
		t.Fatalf("expected limit %d, got %d", expected.limit, call.limit)
	}
	if call.offset != expected.offset {
		t.Fatalf("expected offset %d, got %d", expected.offset, call.offset)
	}
}

func assignProjectMemberMutateCall[T any](
	call *projectMemberMutateCall[T],
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request T,
) {
	*call = projectMemberMutateCall[T]{
		actorUserID: actorUserID,
		actorRole:   actorRole,
		request:     request,
	}
}

func projectMemberMutateResult(
	result *models.ProjectMemberRecord,
	err error,
) (*models.ProjectMemberRecord, error) {
	if err != nil {
		return nil, err
	}
	return result, nil
}

type fakeProjectMembershipService struct {
	fakeProjectListService
	listResult   []models.ProjectMemberRecord
	addResult    *models.ProjectMemberRecord
	updateResult *models.ProjectMemberRecord
	listErr      error
	addErr       error
	updateErr    error
	removeErr    error
	listCall     projectMemberListCall
	addCall      projectMemberMutateCall[projects.ProjectMemberCreateRequest]
	updateCall   projectMemberMutateCall[projects.ProjectMemberUpdateRequest]
	removeCall   projectMemberMutateCall[projects.ProjectMemberRemoveRequest]
}

func (service *fakeProjectMembershipService) ListProjectMembers(
	_ context.Context,
	actor projects.ActorContext,
	request projects.ProjectMemberListRequest,
) ([]models.ProjectMemberRecord, error) {
	service.listCall = projectMemberListCall{
		actorUserID:    actor.UserID,
		actorRole:      actor.Role,
		projectID:      request.ProjectID,
		includeRevoked: request.IncludeRevoked,
		limit:          request.Limit,
		offset:         request.Offset,
	}
	if service.listErr != nil {
		return nil, service.listErr
	}
	return service.listResult, nil
}

func (service *fakeProjectMembershipService) AddProjectMember(
	_ context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request projects.ProjectMemberCreateRequest,
) (*models.ProjectMemberRecord, error) {
	assignProjectMemberMutateCall(&service.addCall, actorUserID, actorRole, request)
	return projectMemberMutateResult(service.addResult, service.addErr)
}

func (service *fakeProjectMembershipService) UpdateProjectMember(
	_ context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request projects.ProjectMemberUpdateRequest,
) (*models.ProjectMemberRecord, error) {
	assignProjectMemberMutateCall(&service.updateCall, actorUserID, actorRole, request)
	return projectMemberMutateResult(service.updateResult, service.updateErr)
}

func (service *fakeProjectMembershipService) RemoveProjectMember(
	_ context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request projects.ProjectMemberRemoveRequest,
) error {
	assignProjectMemberMutateCall(&service.removeCall, actorUserID, actorRole, request)
	return service.removeErr
}

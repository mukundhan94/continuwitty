package mcp

import (
	"context"
	"testing"

	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

func TestCompatibilityServiceProjectGetDefaultResponses(t *testing.T) {
	defaultProjectID := "proj-default"
	service := &fakeProjectDefaultService{defaultProjectID: &defaultProjectID}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name:            "direct",
			request:         directToolRequest("23000000-0000-0000-0000-000000000023", "project.get_default", nil),
			asToolsCallPath: false,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest("23100000-0000-0000-0000-000000000231", "project_get_default", nil),
			asToolsCallPath: true,
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
			if testCase.asToolsCallPath {
				result := resultPayloadFromFrame(t, frame)
				structuredContent := mapFromMap(t, result, "structuredContent")
				assertDefaultProjectIDPayload(t, structuredContent, defaultProjectID)
				return
			}
			result := resultPayloadFromFrame(t, frame)
			assertDefaultProjectIDPayload(t, result, defaultProjectID)
		})
	}
}

func TestCompatibilityServiceProjectSetDefaultToolsCall(t *testing.T) {
	serviceDependency := &fakeProjectDefaultService{setResult: "proj-next"}
	actorUserID := "23200000-0000-0000-0000-000000000232"

	frame := runCompatibilityRequestWithService(
		t,
		newProjectDefaultCompatibilityService(serviceDependency),
		toolsCallRequest(
			actorUserID,
			"project_set_default",
			map[string]any{"project_id": "proj-next"},
		),
	)

	result := resultPayloadFromFrame(t, frame)
	structuredContent := mapFromMap(t, result, "structuredContent")
	assertDefaultProjectIDPayload(t, structuredContent, "proj-next")
	assertProjectSetCall(t, serviceDependency.setCall, actorUserID, "proj-next")
}

func TestCompatibilityServiceProjectSetDefaultErrors(t *testing.T) {
	testCases := []struct {
		name         string
		service      *fakeProjectDefaultService
		request      StreamCallRequest
		expectedCode int
	}{
		{
			name:         "missing project id",
			service:      &fakeProjectDefaultService{},
			request:      toolsCallRequest("23300000-0000-0000-0000-000000000233", "project_set_default", map[string]any{}),
			expectedCode: -32602,
		},
		{
			name:         "project not found",
			service:      &fakeProjectDefaultService{setErr: projects.ErrProjectNotFound},
			request:      toolsCallRequest("23400000-0000-0000-0000-000000000234", "project_set_default", map[string]any{"project_id": "missing"}),
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

func newProjectDefaultCompatibilityService(projectService ProjectListService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{ProjectService: projectService},
	)
}

func directToolRequest(actorUserID string, method string, params map[string]any) StreamCallRequest {
	return StreamCallRequest{
		Request: JSONRPCRequest{JSONRPC: "2.0", ID: "direct", Method: method, Params: params},
		Actor: Actor{
			UserID: uuid.MustParse(actorUserID),
			Role:   "viewer",
		},
	}
}

func toolsCallRequest(actorUserID string, toolName string, arguments map[string]any) StreamCallRequest {
	return StreamCallRequest{
		Request: JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      "tools-call",
			Method:  "tools/call",
			Params: map[string]any{
				"name":      toolName,
				"arguments": arguments,
			},
		},
		Actor: Actor{
			UserID: uuid.MustParse(actorUserID),
			Role:   "Analyst",
		},
	}
}

func assertDefaultProjectIDPayload(t *testing.T, payload map[string]any, expected string) {
	t.Helper()
	if payload["default_project_id"] != expected {
		t.Fatalf("expected default_project_id %q", expected)
	}
}

func assertProjectSetCall(
	t *testing.T,
	call projectSetCall,
	actorUserID string,
	expectedProjectID string,
) {
	t.Helper()
	if call.actorUserID != uuid.MustParse(actorUserID) {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.actorRole != models.UserRoleAnalyst {
		t.Fatalf("expected normalized actor role forwarded")
	}
	if call.projectID != expectedProjectID {
		t.Fatalf("expected project_id forwarded")
	}
}

type projectSetCall struct {
	actorUserID uuid.UUID
	actorRole   models.UserRole
	projectID   string
}

type fakeProjectDefaultService struct {
	fakeProjectListService
	defaultProjectID *string
	getErr           error
	setResult        string
	setErr           error
	setCall          projectSetCall
}

func (service *fakeProjectDefaultService) GetDefaultProjectID(
	_ context.Context,
	_ uuid.UUID,
) (*string, error) {
	return service.defaultProjectID, service.getErr
}

func (service *fakeProjectDefaultService) SetDefaultProjectID(
	_ context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
) (string, error) {
	service.setCall = projectSetCall{actorUserID: actorUserID, actorRole: actorRole, projectID: projectID}
	if service.setErr != nil {
		return "", service.setErr
	}
	if service.setResult != "" {
		return service.setResult, nil
	}
	return projectID, nil
}

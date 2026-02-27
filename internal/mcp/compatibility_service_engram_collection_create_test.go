package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/admin"
	"engram/internal/models"
	"engram/internal/projects"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramCollectionCreateParity(t *testing.T) {
	actorUserID := uuid.MustParse("39920000-0000-0000-0000-000000000398")
	collection := models.EngramCollectionRecord{
		CollectionID: uuid.MustParse("39920000-0000-0000-0000-000000000399"),
		ProjectID:    "project-target",
		OwnerUserID:  actorUserID,
		Name:         "Ops",
		Description:  "Runbooks",
	}
	response := &EngramCollectionCreateResponse{
		Collection:         collection,
		ResolvedProjectID:  "project-target",
		UsedDefaultProject: true,
	}
	createService := &fakeEngramCollectionCreateService{response: response}
	params := map[string]any{
		"project_id":  "project-requested",
		"name":        "Ops",
		"description": "Runbooks",
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.collection_create", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_collection_create", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramCollectionCreateCompatibilityService(createService),
				testCase.request,
			)
			actual := collectionCreateResponseFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*response, actual) {
				t.Fatalf("expected collection-create payload to match service output")
			}
			assertCollectionCreateCall(
				t,
				createService.call,
				collectionCreateCallExpectation{
					actorUserID: actorUserID,
					actorRole:   testCase.expectedRole,
					projectID:   "project-requested",
					name:        "Ops",
					description: "Runbooks",
				},
			)
		})
	}
}

func TestCompatibilityServiceEngramCollectionCreateUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39920000-0000-0000-0000-000000000400")
	createService := &fakeEngramCollectionCreateService{
		response: &EngramCollectionCreateResponse{
			Collection: models.EngramCollectionRecord{CollectionID: uuid.MustParse("39920000-0000-0000-0000-000000000401")},
		},
	}
	frame := runCompatibilityRequestWithService(
		t,
		newEngramCollectionCreateCompatibilityService(createService),
		directToolRequest(
			actorUserID.String(),
			"engram.collection_create",
			map[string]any{"name": "Ops"},
		),
	)
	_ = collectionCreateResponseFromFrame(t, frame, false)
	assertCollectionCreateCall(
		t,
		createService.call,
		collectionCreateCallExpectation{
			actorUserID: actorUserID,
			actorRole:   models.UserRoleViewer,
			projectID:   "",
			name:        "Ops",
			description: "",
		},
	)
}

func TestCompatibilityServiceEngramCollectionCreateValidationAndErrors(t *testing.T) {
	service := newEngramCollectionCreateCompatibilityService(&fakeEngramCollectionCreateService{})
	validationCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing name", params: map[string]any{}},
		{name: "blank name", params: map[string]any{"name": ""}},
	}
	for _, testCase := range validationCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39920000-0000-0000-0000-000000000402",
					"engram_collection_create",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	errorCases := []struct {
		name           string
		err            error
		expectedCode   int
		expectedStatus *int
		expectedDetail *string
	}{
		{name: "project id required", err: projects.ErrProjectIDMustNotBeBlank, expectedCode: -32602},
		{name: "project not found", err: projects.ErrProjectNotFound, expectedCode: -32602, expectedStatus: createStatusPtr(404), expectedDetail: stringPtr("Project not found")},
		{name: "missing default", err: projects.ErrProjectIDRequiredWhenNoDefaultProject, expectedCode: -32602, expectedStatus: createStatusPtr(422)},
		{name: "hidden default", err: projects.ErrDefaultProjectNotAccessible, expectedCode: -32602, expectedStatus: createStatusPtr(422)},
		{name: "admin project required", err: admin.ErrProjectIDRequired, expectedCode: -32602},
		{name: "duplicate", err: repository.ErrCollectionNameExists, expectedCode: -32602, expectedStatus: createStatusPtr(409), expectedDetail: stringPtr(repository.ErrCollectionNameExists.Error())},
		{name: "internal", err: errors.New("boom"), expectedCode: -32603},
	}
	for _, testCase := range errorCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramCollectionCreateCompatibilityService(
					&fakeEngramCollectionCreateService{err: testCase.err},
				),
				toolsCallRequest(
					"39920000-0000-0000-0000-000000000403",
					"engram_collection_create",
					map[string]any{"name": "Ops"},
				),
			)
			errorPayload := errorPayloadFromFrame(t, frame)
			requireErrorCode(t, errorPayload, testCase.expectedCode)
			if testCase.expectedStatus != nil {
				assertErrorStatusCode(t, errorPayload, *testCase.expectedStatus)
			}
			if testCase.expectedDetail != nil {
				assertErrorDetail(t, errorPayload, *testCase.expectedDetail)
			}
		})
	}
}

func newEngramCollectionCreateCompatibilityService(service EngramCollectionCreateService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramCollectionCreate: service},
	)
}

func collectionCreateResponseFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) EngramCollectionCreateResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	collection, ok := payload["collection"].(models.EngramCollectionRecord)
	if !ok {
		t.Fatalf("expected collection payload")
	}
	resolvedProjectID, ok := payload["resolved_project_id"].(string)
	if !ok {
		t.Fatalf("expected resolved_project_id payload")
	}
	usedDefaultProject, ok := payload["used_default_project"].(bool)
	if !ok {
		t.Fatalf("expected used_default_project payload")
	}
	return EngramCollectionCreateResponse{
		Collection:         collection,
		ResolvedProjectID:  resolvedProjectID,
		UsedDefaultProject: usedDefaultProject,
	}
}

type collectionCreateCallExpectation struct {
	actorUserID uuid.UUID
	actorRole   models.UserRole
	projectID   string
	name        string
	description string
}

func assertCollectionCreateCall(
	t *testing.T,
	call EngramCollectionCreateRequest,
	expected collectionCreateCallExpectation,
) {
	t.Helper()
	if call.ActorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.ActorRole != expected.actorRole {
		t.Fatalf("expected actor role forwarded")
	}
	if call.ProjectID != expected.projectID {
		t.Fatalf("expected project_id forwarded")
	}
	if call.Name != expected.name {
		t.Fatalf("expected name forwarded")
	}
	if call.Description != expected.description {
		t.Fatalf("expected description forwarded")
	}
}

type fakeEngramCollectionCreateService struct {
	response *EngramCollectionCreateResponse
	err      error
	call     EngramCollectionCreateRequest
}

func (service *fakeEngramCollectionCreateService) CreateCollection(
	_ context.Context,
	request EngramCollectionCreateRequest,
) (*EngramCollectionCreateResponse, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.response == nil {
		return nil, nil
	}
	response := *service.response
	return &response, nil
}

func createStatusPtr(value int) *int {
	return &value
}

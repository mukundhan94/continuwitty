package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	internalexport "engram/internal/export"
	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceProjectExportBundleParity(t *testing.T) {
	actorUserID := uuid.MustParse("39970000-0000-0000-0000-000000000398")
	collectionID := uuid.MustParse("39970000-0000-0000-0000-000000000399")
	service := &fakeProjectExportService{
		response: &ProjectExportResponse{
			Bundle: map[string]any{
				"schema_version": "1.0",
				"project": map[string]any{
					"project_id": "project-target",
				},
			},
		},
	}
	params := map[string]any{
		"project_id":         "project-target",
		"collection_ids":     []any{collectionID.String()},
		"include_embeddings": true,
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "project.export_bundle", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "project_export_bundle", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newProjectExportCompatibilityService(service),
				testCase.request,
			)
			bundle := exportBundleFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(service.response.Bundle, bundle) {
				t.Fatalf("expected export bundle payload to match service output")
			}
			assertProjectExportCall(
				t,
				service.call,
				projectExportCallExpectation{
					actorUserID:       actorUserID,
					actorRole:         testCase.expectedRole,
					projectID:         "project-target",
					collectionIDs:     []uuid.UUID{collectionID},
					includeEmbeddings: true,
				},
			)
		})
	}
}

func TestCompatibilityServiceProjectExportBundleUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39970000-0000-0000-0000-000000000400")
	service := &fakeProjectExportService{response: &ProjectExportResponse{Bundle: map[string]any{}}}
	frame := runCompatibilityRequestWithService(
		t,
		newProjectExportCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"project.export_bundle",
			map[string]any{"project_id": "project-target"},
		),
	)
	_ = exportBundleFromFrame(t, frame, false)
	assertProjectExportCall(
		t,
		service.call,
		projectExportCallExpectation{
			actorUserID:       actorUserID,
			actorRole:         models.UserRoleViewer,
			projectID:         "project-target",
			collectionIDs:     []uuid.UUID{},
			includeEmbeddings: false,
		},
	)
}

func TestCompatibilityServiceProjectExportBundleValidationAndErrors(t *testing.T) {
	service := newProjectExportCompatibilityService(&fakeProjectExportService{})
	validationCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing project id", params: map[string]any{}},
		{name: "invalid collection ids type", params: map[string]any{"project_id": "project", "collection_ids": "bad"}},
		{name: "invalid collection ids item", params: map[string]any{"project_id": "project", "collection_ids": []any{"bad"}}},
		{name: "invalid include embeddings", params: map[string]any{"project_id": "project", "include_embeddings": "bad"}},
	}
	for _, testCase := range validationCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39970000-0000-0000-0000-000000000401",
					"project_export_bundle",
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
		{name: "project not found", err: internalexport.ErrProjectNotFound, expectedCode: -32602, expectedStatus: projectTransferStatusPtr(404), expectedDetail: stringPtr("Project not found")},
		{name: "collection not found", err: internalexport.ErrCollectionNotFoundForProject, expectedCode: -32602, expectedStatus: projectTransferStatusPtr(404), expectedDetail: stringPtr("Collection not found for project")},
		{name: "internal", err: errors.New("boom"), expectedCode: -32603},
	}
	for _, testCase := range errorCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newProjectExportCompatibilityService(
					&fakeProjectExportService{err: testCase.err},
				),
				toolsCallRequest(
					"39970000-0000-0000-0000-000000000402",
					"project_export_bundle",
					map[string]any{"project_id": "project-target"},
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

func newProjectExportCompatibilityService(service ProjectExportService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{ProjectExport: service},
	)
}

func exportBundleFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) map[string]any {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	bundle, ok := payload["bundle"].(map[string]any)
	if !ok {
		t.Fatalf("expected bundle payload")
	}
	return bundle
}

type projectExportCallExpectation struct {
	actorUserID       uuid.UUID
	actorRole         models.UserRole
	projectID         string
	collectionIDs     []uuid.UUID
	includeEmbeddings bool
}

func assertProjectExportCall(
	t *testing.T,
	call ProjectExportRequest,
	expected projectExportCallExpectation,
) {
	t.Helper()
	if call.ActorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.ActorRole != expected.actorRole {
		t.Fatalf("expected actor role forwarded")
	}
	if call.ProjectID != expected.projectID {
		t.Fatalf("expected project id forwarded")
	}
	if !reflect.DeepEqual(call.CollectionIDs, expected.collectionIDs) {
		t.Fatalf("expected collection ids forwarded")
	}
	if call.IncludeEmbeddings != expected.includeEmbeddings {
		t.Fatalf("expected include_embeddings forwarded")
	}
}

type fakeProjectExportService struct {
	response *ProjectExportResponse
	err      error
	call     ProjectExportRequest
}

func (service *fakeProjectExportService) ExportProjectBundle(
	_ context.Context,
	request ProjectExportRequest,
) (*ProjectExportResponse, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.response == nil {
		return nil, nil
	}
	bundle := map[string]any{}
	for key, value := range service.response.Bundle {
		bundle[key] = value
	}
	return &ProjectExportResponse{Bundle: bundle}, nil
}

func projectTransferStatusPtr(value int) *int {
	return &value
}

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	internalexport "engram/internal/export"
	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceProjectImportBundleParity(t *testing.T) {
	actorUserID := uuid.MustParse("39980000-0000-0000-0000-000000000398")
	summary := map[string]any{
		"target_project_id": "project-target",
		"imported_engrams":  1,
	}
	service := &fakeProjectImportService{
		response: &ProjectImportResponse{Summary: summary},
	}
	params := map[string]any{
		"project_id":      "project-target",
		"bundle":          map[string]any{"schema_version": "1.0"},
		"conflict_policy": "rename",
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "project.import_bundle", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "project_import_bundle", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newProjectImportCompatibilityService(service),
				testCase.request,
			)
			actualSummary := importSummaryFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(summary, actualSummary) {
				t.Fatalf("expected import summary payload to match service output")
			}
			callBundle := decodeBundleBytesMap(t, service.call.BundleBytes)
			expectedBundle := map[string]any{"schema_version": "1.0"}
			if !reflect.DeepEqual(expectedBundle, callBundle) {
				t.Fatalf("expected bundle payload forwarded")
			}
			assertProjectImportCall(
				t,
				service.call,
				projectImportCallExpectation{
					actorUserID:     actorUserID,
					actorRole:       testCase.expectedRole,
					targetProjectID: "project-target",
					conflictPolicy:  "rename",
				},
			)
		})
	}
}

func TestCompatibilityServiceProjectImportBundleUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39980000-0000-0000-0000-000000000400")
	service := &fakeProjectImportService{response: &ProjectImportResponse{Summary: map[string]any{}}}
	bundleJSON := `{"schema_version":"1.0"}`
	frame := runCompatibilityRequestWithService(
		t,
		newProjectImportCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"project.import_bundle",
			map[string]any{
				"project_id":  "project-target",
				"bundle_json": bundleJSON,
			},
		),
	)
	_ = importSummaryFromFrame(t, frame, false)
	assertProjectImportCall(
		t,
		service.call,
		projectImportCallExpectation{
			actorUserID:     actorUserID,
			actorRole:       models.UserRoleViewer,
			targetProjectID: "project-target",
			conflictPolicy:  "skip",
		},
	)
	if string(service.call.BundleBytes) != bundleJSON {
		t.Fatalf("expected bundle_json bytes forwarded")
	}
}

func TestCompatibilityServiceProjectImportBundleValidationAndErrors(t *testing.T) {
	service := newProjectImportCompatibilityService(&fakeProjectImportService{})
	validationCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing project id", params: map[string]any{}},
		{name: "invalid conflict policy", params: map[string]any{"project_id": "project", "bundle": map[string]any{}, "conflict_policy": "bad"}},
		{name: "missing bundle", params: map[string]any{"project_id": "project"}},
		{name: "invalid bundle", params: map[string]any{"project_id": "project", "bundle": "bad"}},
		{name: "invalid bundle json", params: map[string]any{"project_id": "project", "bundle_json": 123}},
	}
	for _, testCase := range validationCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39980000-0000-0000-0000-000000000401",
					"project_import_bundle",
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
		{name: "import file empty", err: internalexport.ErrImportFileEmpty, expectedCode: -32602, expectedStatus: projectTransferStatusPtr(422), expectedDetail: stringPtr("Import file is empty")},
		{name: "zip missing export", err: internalexport.ErrImportZipMissingExportJSON, expectedCode: -32602, expectedStatus: projectTransferStatusPtr(422), expectedDetail: stringPtr("ZIP missing export.json")},
		{name: "unsupported format", err: internalexport.ErrUnsupportedImportFileFormat, expectedCode: -32602, expectedStatus: projectTransferStatusPtr(422), expectedDetail: stringPtr("Unsupported import file format")},
		{name: "invalid bundle", err: internalexport.ErrInvalidExportBundle, expectedCode: -32602, expectedStatus: projectTransferStatusPtr(422), expectedDetail: stringPtr("Invalid export bundle")},
		{name: "internal", err: errors.New("boom"), expectedCode: -32603},
	}
	for _, testCase := range errorCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newProjectImportCompatibilityService(
					&fakeProjectImportService{err: testCase.err},
				),
				toolsCallRequest(
					"39980000-0000-0000-0000-000000000402",
					"project_import_bundle",
					map[string]any{
						"project_id": "project-target",
						"bundle":     map[string]any{"schema_version": "1.0"},
					},
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

func newProjectImportCompatibilityService(service ProjectImportService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{ProjectImport: service},
	)
}

func importSummaryFromFrame(
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
	summary, ok := payload["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary payload")
	}
	return summary
}

func decodeBundleBytesMap(t *testing.T, bundleBytes []byte) map[string]any {
	t.Helper()
	bundle := map[string]any{}
	if err := json.Unmarshal(bundleBytes, &bundle); err != nil {
		t.Fatalf("expected bundle bytes to be JSON: %v", err)
	}
	return bundle
}

type projectImportCallExpectation struct {
	actorUserID     uuid.UUID
	actorRole       models.UserRole
	targetProjectID string
	conflictPolicy  string
}

func assertProjectImportCall(
	t *testing.T,
	call ProjectImportRequest,
	expected projectImportCallExpectation,
) {
	t.Helper()
	if call.ActorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.ActorRole != expected.actorRole {
		t.Fatalf("expected actor role forwarded")
	}
	if call.TargetProjectID != expected.targetProjectID {
		t.Fatalf("expected target project id forwarded")
	}
	if call.ConflictPolicy != expected.conflictPolicy {
		t.Fatalf("expected conflict policy forwarded")
	}
}

type fakeProjectImportService struct {
	response *ProjectImportResponse
	err      error
	call     ProjectImportRequest
}

func (service *fakeProjectImportService) ImportProjectBundle(
	_ context.Context,
	request ProjectImportRequest,
) (*ProjectImportResponse, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.response == nil {
		return nil, nil
	}
	summary := map[string]any{}
	for key, value := range service.response.Summary {
		summary[key] = value
	}
	return &ProjectImportResponse{Summary: summary}, nil
}

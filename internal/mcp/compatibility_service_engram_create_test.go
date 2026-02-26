package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramCreateParity(t *testing.T) {
	actorUserID := uuid.MustParse("39700000-0000-0000-0000-000000000397")
	params := map[string]any{
		"project_id":                " proj-alpha ",
		"title":                     "  Ops snapshot  ",
		"abstract":                  "  Summary  ",
		"detailed_summary_markdown": "  Details  ",
		"thread_id":                 "thread-1",
		"visibility_scope":          "project",
		"tags":                      []any{"alpha"},
		"keywords":                  []any{"beta"},
	}
	created := &models.EngramCreateResponse{
		EngramID:           uuid.MustParse("39700000-0000-0000-0000-000000000398"),
		CreatedAt:          time.Unix(1700003970, 0).UTC(),
		ResolvedProjectID:  stringPtr("proj-alpha"),
		UsedDefaultProject: false,
	}
	service := &fakeEngramCreateService{created: created}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.create", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_create", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramCreateCompatibilityService(service),
				testCase.request,
			)
			engram := engramCreateResponseFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*created, engram) {
				t.Fatalf("expected create payload to match service output")
			}
			assertEngramCreateCall(
				t,
				service.call,
				engramCreateExpectation{
					actorUserID: actorUserID,
					actorRole:   testCase.expectedRole,
					payload: models.MemoryEngramCreate{
						ProjectID:               "proj-alpha",
						ThreadID:                stringPtr("thread-1"),
						Title:                   "Ops snapshot",
						Abstract:                "Summary",
						DetailedSummaryMarkdown: "Details",
						Decisions:               []models.Decision{},
						Assumptions:             []string{},
						OpenQuestions:           []string{},
						Claims:                  []models.Claim{},
						Tags:                    []string{"alpha"},
						Keywords:                []string{"beta"},
						Artifacts:               []models.ArtifactIn{},
						VisibilityScope:         string(models.VisibilityScopeProject),
					},
				},
			)
		})
	}
}

func TestCompatibilityServiceEngramCreateUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39710000-0000-0000-0000-000000000397")
	service := &fakeEngramCreateService{
		created: &models.EngramCreateResponse{
			EngramID:           uuid.MustParse("39710000-0000-0000-0000-000000000398"),
			CreatedAt:          time.Unix(1700003971, 0).UTC(),
			ResolvedProjectID:  stringPtr("proj-default"),
			UsedDefaultProject: true,
		},
	}
	frame := runCompatibilityRequestWithService(
		t,
		newEngramCreateCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"engram.create",
			map[string]any{
				"title":                     "Snapshot",
				"detailed_summary_markdown": "Details",
			},
		),
	)
	_ = engramCreateResponseFromFrame(t, frame, false)
	if service.call.Payload.VisibilityScope != string(models.VisibilityScopePrivate) {
		t.Fatalf("expected default visibility scope to be private")
	}
}

func TestCompatibilityServiceEngramCreateValidationAndErrorMappings(t *testing.T) {
	service := newEngramCreateCompatibilityService(&fakeEngramCreateService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "invalid payload type", params: map[string]any{"title": "x", "detailed_summary_markdown": "y", "tags": "bad"}},
		{name: "missing title", params: map[string]any{"detailed_summary_markdown": "y"}},
		{name: "missing summary", params: map[string]any{"title": "x"}},
		{name: "invalid visibility", params: map[string]any{"title": "x", "detailed_summary_markdown": "y", "visibility_scope": "bad"}},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39720000-0000-0000-0000-000000000397",
					"engram_create",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	notFoundFrame := runCompatibilityRequestWithService(
		t,
		newEngramCreateCompatibilityService(
			&fakeEngramCreateService{err: projects.ErrProjectNotFound},
		),
		toolsCallRequest(
			"39720000-0000-0000-0000-000000000398",
			"engram_create",
			map[string]any{"title": "x", "detailed_summary_markdown": "y", "project_id": "missing"},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32602)
	assertErrorStatusCode(t, notFoundPayload, 404)
	assertErrorDetail(t, notFoundPayload, "Project not found")

	defaultMissingFrame := runCompatibilityRequestWithService(
		t,
		newEngramCreateCompatibilityService(
			&fakeEngramCreateService{err: projects.ErrProjectIDRequiredWhenNoDefaultProject},
		),
		toolsCallRequest(
			"39720000-0000-0000-0000-000000000399",
			"engram_create",
			map[string]any{"title": "x", "detailed_summary_markdown": "y"},
		),
	)
	defaultMissingPayload := errorPayloadFromFrame(t, defaultMissingFrame)
	requireErrorCode(t, defaultMissingPayload, -32602)
	assertErrorStatusCode(t, defaultMissingPayload, 422)

	defaultHiddenFrame := runCompatibilityRequestWithService(
		t,
		newEngramCreateCompatibilityService(
			&fakeEngramCreateService{err: projects.ErrDefaultProjectNotAccessible},
		),
		toolsCallRequest(
			"39720000-0000-0000-0000-000000000400",
			"engram_create",
			map[string]any{"title": "x", "detailed_summary_markdown": "y"},
		),
	)
	defaultHiddenPayload := errorPayloadFromFrame(t, defaultHiddenFrame)
	requireErrorCode(t, defaultHiddenPayload, -32602)
	assertErrorStatusCode(t, defaultHiddenPayload, 422)

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramCreateCompatibilityService(
			&fakeEngramCreateService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39720000-0000-0000-0000-000000000401",
			"engram_create",
			map[string]any{"title": "x", "detailed_summary_markdown": "y"},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramCreateCompatibilityService(service EngramCreateService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramCreate: service},
	)
}

func engramCreateResponseFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.EngramCreateResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	engram, ok := payload["engram"].(models.EngramCreateResponse)
	if !ok {
		t.Fatalf("expected engram create payload")
	}
	return engram
}

type engramCreateExpectation struct {
	actorUserID uuid.UUID
	actorRole   models.UserRole
	payload     models.MemoryEngramCreate
}

func assertEngramCreateCall(
	t *testing.T,
	call EngramCreateRequest,
	expected engramCreateExpectation,
) {
	t.Helper()
	if call.ActorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.ActorRole != expected.actorRole {
		t.Fatalf("expected actor role forwarded")
	}
	if !reflect.DeepEqual(call.Payload, expected.payload) {
		t.Fatalf("expected payload forwarded")
	}
}

func assertErrorDetail(t *testing.T, errorPayload map[string]any, expected string) {
	t.Helper()
	data := mapFromMap(t, errorPayload, "data")
	if data["detail"] != expected {
		t.Fatalf("expected detail %q in error data", expected)
	}
}

type fakeEngramCreateService struct {
	created *models.EngramCreateResponse
	err     error
	call    EngramCreateRequest
}

func (service *fakeEngramCreateService) CreateEngram(
	_ context.Context,
	request EngramCreateRequest,
) (*models.EngramCreateResponse, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.created == nil {
		return nil, nil
	}
	response := *service.created
	return &response, nil
}

package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramRehydrateParity(t *testing.T) {
	actorUserID := uuid.MustParse("39600000-0000-0000-0000-000000000396")
	engramID := uuid.MustParse("39600000-0000-0000-0000-000000000397")
	bundle := &models.RehydrationBundle{
		EngramID:                engramID,
		ProjectID:               "proj-alpha",
		Title:                   "Roadmap",
		CompactSummary:          "Summary",
		DetailedSummaryMarkdown: "Details",
		TopCitations: []models.RehydrationCitation{
			{
				URL:        "https://example.com/source",
				Snippet:    "Evidence",
				CapturedAt: time.Unix(1700003960, 0).UTC(),
			},
		},
		ContextMarkdown: "Context",
		VisibilityScope: "private",
	}
	service := &fakeEngramRehydrateService{bundle: bundle}
	params := map[string]any{"engram_id": engramID.String()}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.rehydrate", params),
			asToolsCallPath: false,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_rehydrate", params),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramRehydrateCompatibilityService(service),
				testCase.request,
			)
			gotBundle := rehydrationBundleFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*bundle, gotBundle) {
				t.Fatalf("expected bundle payload to match service output")
			}
			if service.call.ActorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			if service.call.EngramID != engramID {
				t.Fatalf("expected engram id forwarded")
			}
		})
	}
}

func TestCompatibilityServiceEngramRehydrateValidationAndErrors(t *testing.T) {
	service := newEngramRehydrateCompatibilityService(&fakeEngramRehydrateService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing engram id", params: map[string]any{}},
		{name: "invalid engram id", params: map[string]any{"engram_id": "bad"}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39610000-0000-0000-0000-000000000396",
					"engram_rehydrate",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	notFoundID := uuid.MustParse("39610000-0000-0000-0000-000000000397")
	notFoundFrame := runCompatibilityRequestWithService(
		t,
		service,
		toolsCallRequest(
			"39610000-0000-0000-0000-000000000398",
			"engram_rehydrate",
			map[string]any{"engram_id": notFoundID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32004)
	assertRehydrateNotFoundData(t, notFoundPayload, notFoundID)

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramRehydrateCompatibilityService(
			&fakeEngramRehydrateService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39610000-0000-0000-0000-000000000399",
			"engram_rehydrate",
			map[string]any{"engram_id": notFoundID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramRehydrateCompatibilityService(service EngramRehydrateService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramRehydrate: service},
	)
}

func rehydrationBundleFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.RehydrationBundle {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	bundle, ok := payload["bundle"].(models.RehydrationBundle)
	if !ok {
		t.Fatalf("expected bundle payload")
	}
	return bundle
}

func assertRehydrateNotFoundData(t *testing.T, errorPayload map[string]any, engramID uuid.UUID) {
	t.Helper()
	data := mapFromMap(t, errorPayload, "data")
	if data["engram_id"] != engramID.String() {
		t.Fatalf("expected engram_id in not-found error data")
	}
}

type fakeEngramRehydrateService struct {
	bundle *models.RehydrationBundle
	err    error
	call   EngramRehydrateRequest
}

func (service *fakeEngramRehydrateService) RehydrateEngram(
	_ context.Context,
	request EngramRehydrateRequest,
) (*models.RehydrationBundle, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.bundle == nil {
		return nil, nil
	}
	bundle := *service.bundle
	return &bundle, nil
}

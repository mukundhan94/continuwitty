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

func TestCompatibilityServiceEngramQueryParity(t *testing.T) {
	actorUserID := uuid.MustParse("39500000-0000-0000-0000-000000000395")
	createdAfter := mustParseRFC3339(t, "2026-01-01T00:00:00Z")
	createdBefore := mustParseRFC3339(t, "2026-02-01T00:00:00Z")
	projectID := "proj-alpha"
	params := map[string]any{
		"query":          "roadmap",
		"top_k":          7.0,
		"project_id":     projectID,
		"tags":           []any{"ops", "planning"},
		"keywords":       []any{"risk"},
		"created_after":  createdAfter.Format(time.RFC3339),
		"created_before": createdBefore.Format(time.RFC3339),
	}
	service := &fakeEngramQueryService{
		results: []models.EngramQueryResult{
			{
				EngramID:    uuid.MustParse("39500000-0000-0000-0000-000000000396"),
				ProjectID:   projectID,
				Title:       "Roadmap",
				Abstract:    "Quarterly plan",
				CreatedAt:   time.Unix(1700003950, 0).UTC(),
				Distance:    0.1,
				Keywords:    []string{"risk"},
				Tags:        []string{"ops"},
				OwnerUserID: uuidPtr(uuid.MustParse("39500000-0000-0000-0000-000000000397")),
			},
		},
	}
	expected := EngramQueryDispatchRequest{
		ActorUserID: actorUserID,
		Payload: models.EngramQueryRequest{
			Query:         "roadmap",
			TopK:          7,
			ProjectID:     &projectID,
			Tags:          []string{"ops", "planning"},
			Keywords:      []string{"risk"},
			CreatedAfter:  &createdAfter,
			CreatedBefore: &createdBefore,
		},
	}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.query", params),
			asToolsCallPath: false,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_query", params),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramQueryCompatibilityService(service),
				testCase.request,
			)
			results := engramQueryResultsFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(service.results, results) {
				t.Fatalf("expected query results payload to match service output")
			}
			assertEngramQueryCall(t, service.call, expected)
		})
	}
}

func TestCompatibilityServiceEngramQueryUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39510000-0000-0000-0000-000000000395")
	service := &fakeEngramQueryService{results: []models.EngramQueryResult{}}
	frame := runCompatibilityRequestWithService(
		t,
		newEngramQueryCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"engram.query",
			map[string]any{"query": "roadmap"},
		),
	)
	_ = engramQueryResultsFromFrame(t, frame, false)
	assertEngramQueryCall(
		t,
		service.call,
		EngramQueryDispatchRequest{
			ActorUserID: actorUserID,
			Payload: models.EngramQueryRequest{
				Query:    "roadmap",
				TopK:     5,
				Tags:     []string{},
				Keywords: []string{},
			},
		},
	)
}

func TestCompatibilityServiceEngramQueryValidationAndErrors(t *testing.T) {
	service := newEngramQueryCompatibilityService(&fakeEngramQueryService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing query", params: map[string]any{}},
		{name: "blank query", params: map[string]any{"query": "   "}},
		{name: "invalid top_k type", params: map[string]any{"query": "x", "top_k": "bad"}},
		{name: "invalid top_k low", params: map[string]any{"query": "x", "top_k": 0.0}},
		{name: "invalid top_k high", params: map[string]any{"query": "x", "top_k": 51.0}},
		{name: "invalid tags type", params: map[string]any{"query": "x", "tags": "bad"}},
		{name: "invalid tag item", params: map[string]any{"query": "x", "tags": []any{"ok", 1}}},
		{name: "invalid keywords type", params: map[string]any{"query": "x", "keywords": "bad"}},
		{name: "invalid created_after", params: map[string]any{"query": "x", "created_after": "bad"}},
		{name: "invalid created_before", params: map[string]any{"query": "x", "created_before": "bad"}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39520000-0000-0000-0000-000000000395",
					"engram_query",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramQueryCompatibilityService(
			&fakeEngramQueryService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39520000-0000-0000-0000-000000000396",
			"engram_query",
			map[string]any{"query": "x"},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramQueryCompatibilityService(service EngramQueryService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramQuery: service},
	)
}

func engramQueryResultsFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) []models.EngramQueryResult {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	results, ok := payload["results"].([]models.EngramQueryResult)
	if !ok {
		t.Fatalf("expected results payload")
	}
	return results
}

func assertEngramQueryCall(
	t *testing.T,
	call EngramQueryDispatchRequest,
	expected EngramQueryDispatchRequest,
) {
	t.Helper()
	if !reflect.DeepEqual(expected, call) {
		t.Fatalf("expected query call %+v, got %+v", expected, call)
	}
}

func mustParseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("expected valid time, got %v", err)
	}
	return parsed
}

func uuidPtr(value uuid.UUID) *uuid.UUID {
	return &value
}

type fakeEngramQueryService struct {
	results []models.EngramQueryResult
	err     error
	call    EngramQueryDispatchRequest
}

func (service *fakeEngramQueryService) QueryEngrams(
	_ context.Context,
	request EngramQueryDispatchRequest,
) ([]models.EngramQueryResult, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	return append([]models.EngramQueryResult(nil), service.results...), nil
}

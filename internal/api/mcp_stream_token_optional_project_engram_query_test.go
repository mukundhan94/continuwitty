package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/mcp"
	"engram/internal/models"
)

type engramQueryRouteScenario struct {
	name      string
	requestID string
	toolsCall bool
}

func TestMountMCPRoutesTokenProjectAutofillForEngramQuery(t *testing.T) {
	tests := []engramQueryRouteScenario{
		{name: "tools/call", requestID: "query-tools", toolsCall: true},
		{name: "direct", requestID: "query-direct", toolsCall: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			queryService := &capturingMCPStreamEngramQueryService{}
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{EngramQuery: queryService},
				[]string{"project-token"},
			)
			response := postEngramQueryRouteRequest(t, router, tc)
			assertMCPToolsCallStatusOK(t, response)
			if queryService.call.Payload.ProjectID == nil || *queryService.call.Payload.ProjectID != "project-token" {
				t.Fatalf("expected token project autofill for engram.query")
			}
		})
	}
}

func TestMountMCPRoutesTokenProjectAutofillRequiresExplicitProjectForEngramQuery(t *testing.T) {
	tests := []engramQueryRouteScenario{
		{name: "tools/call", requestID: "query-tools-denied", toolsCall: true},
		{name: "direct", requestID: "query-direct-denied", toolsCall: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{EngramQuery: &capturingMCPStreamEngramQueryService{}},
				[]string{"project-a", "project-b"},
			)
			response := postEngramQueryRouteRequest(t, router, tc)
			assertMCPToolsCallStatusOK(t, response)
			assertMCPMissingProjectIDReasonError(
				t,
				response,
				"token_has_multiple_allowed_projects",
			)
		})
	}
}

func postEngramQueryRouteRequest(
	t *testing.T,
	router http.Handler,
	scenario engramQueryRouteScenario,
) *httptest.ResponseRecorder {
	t.Helper()
	if scenario.toolsCall {
		return postMCPToolsCallRequest(t, router, mcpToolsCallRequest{
			RequestID: scenario.requestID,
			ToolName:  "engram_query",
			Arguments: map[string]any{"query": "roadmap"},
		})
	}
	return postMCPJSONRPCRequest(t, router, mcpJSONRPCPostRequest{
		RequestID: scenario.requestID,
		Method:    "engram.query",
		Params:    map[string]any{"query": "roadmap"},
	})
}

type capturingMCPStreamEngramQueryService struct {
	results []models.EngramQueryResult
	call    mcp.EngramQueryDispatchRequest
}

func (service *capturingMCPStreamEngramQueryService) QueryEngrams(
	_ context.Context,
	request mcp.EngramQueryDispatchRequest,
) ([]models.EngramQueryResult, error) {
	service.call = request
	return append([]models.EngramQueryResult(nil), service.results...), nil
}

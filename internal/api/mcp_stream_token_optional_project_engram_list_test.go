package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/mcp"
	"engram/internal/models"
)

type engramListRouteScenario struct {
	name      string
	requestID string
	toolsCall bool
}

func TestMountMCPRoutesTokenProjectAutofillForEngramList(t *testing.T) {
	tests := []engramListRouteScenario{
		{name: "tools/call", requestID: "list-tools", toolsCall: true},
		{name: "direct", requestID: "list-direct", toolsCall: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			listService := &capturingMCPStreamEngramListService{}
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{EngramList: listService},
				[]string{"project-token"},
			)
			response := postEngramListRouteRequest(t, router, tc)
			assertMCPToolsCallStatusOK(t, response)
			if listService.call.ProjectID == nil || *listService.call.ProjectID != "project-token" {
				t.Fatalf("expected token project autofill for engram.list")
			}
		})
	}
}

func TestMountMCPRoutesTokenProjectAutofillRequiresExplicitProjectForEngramList(t *testing.T) {
	tests := []engramListRouteScenario{
		{name: "tools/call", requestID: "list-tools-denied", toolsCall: true},
		{name: "direct", requestID: "list-direct-denied", toolsCall: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{EngramList: &capturingMCPStreamEngramListService{}},
				[]string{"project-a", "project-b"},
			)
			response := postEngramListRouteRequest(t, router, tc)
			assertMCPToolsCallStatusOK(t, response)
			assertMCPMissingProjectIDReasonError(
				t,
				response,
				"token_has_multiple_allowed_projects",
			)
		})
	}
}

func postEngramListRouteRequest(
	t *testing.T,
	router http.Handler,
	scenario engramListRouteScenario,
) *httptest.ResponseRecorder {
	t.Helper()
	if scenario.toolsCall {
		return postMCPToolsCallRequest(t, router, mcpToolsCallRequest{
			RequestID: scenario.requestID,
			ToolName:  "engram_list",
			Arguments: map[string]any{},
		})
	}
	return postMCPJSONRPCRequest(t, router, mcpJSONRPCPostRequest{
		RequestID: scenario.requestID,
		Method:    "engram.list",
		Params:    map[string]any{},
	})
}

type capturingMCPStreamEngramListService struct {
	engrams []models.AdminEngramRecord
	call    mcp.EngramListRequest
}

func (service *capturingMCPStreamEngramListService) ListEngrams(
	_ context.Context,
	request mcp.EngramListRequest,
) ([]models.AdminEngramRecord, error) {
	service.call = request
	return append([]models.AdminEngramRecord(nil), service.engrams...), nil
}

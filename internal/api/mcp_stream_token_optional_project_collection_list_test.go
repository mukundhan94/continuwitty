package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/mcp"
	"engram/internal/models"
)

type collectionListRouteScenario struct {
	name      string
	requestID string
	toolsCall bool
}

func TestMountMCPRoutesTokenProjectAutofillForCollectionList(t *testing.T) {
	tests := []collectionListRouteScenario{
		{name: "tools/call", requestID: "collection-tools", toolsCall: true},
		{name: "direct", requestID: "collection-direct", toolsCall: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			listService := &capturingMCPStreamCollectionListService{}
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{EngramCollectionList: listService},
				[]string{"project-token"},
			)
			response := postCollectionListRouteRequest(t, router, tc)
			assertMCPToolsCallStatusOK(t, response)
			if listService.call.ProjectID == nil || *listService.call.ProjectID != "project-token" {
				t.Fatalf("expected token project autofill for engram.collection_list")
			}
		})
	}
}

func TestMountMCPRoutesTokenProjectAutofillRequiresExplicitProjectForCollectionList(t *testing.T) {
	tests := []collectionListRouteScenario{
		{name: "tools/call", requestID: "collection-tools-denied", toolsCall: true},
		{name: "direct", requestID: "collection-direct-denied", toolsCall: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{EngramCollectionList: &capturingMCPStreamCollectionListService{}},
				[]string{"project-a", "project-b"},
			)
			response := postCollectionListRouteRequest(t, router, tc)
			assertMCPToolsCallStatusOK(t, response)
			assertMCPMissingProjectIDReasonError(
				t,
				response,
				"token_has_multiple_allowed_projects",
			)
		})
	}
}

func postCollectionListRouteRequest(
	t *testing.T,
	router http.Handler,
	scenario collectionListRouteScenario,
) *httptest.ResponseRecorder {
	t.Helper()
	if scenario.toolsCall {
		return postMCPToolsCallRequest(t, router, mcpToolsCallRequest{
			RequestID: scenario.requestID,
			ToolName:  "engram_collection_list",
			Arguments: map[string]any{},
		})
	}
	return postMCPJSONRPCRequest(t, router, mcpJSONRPCPostRequest{
		RequestID: scenario.requestID,
		Method:    "engram.collection_list",
		Params:    map[string]any{},
	})
}

type capturingMCPStreamCollectionListService struct {
	collections []models.EngramCollectionRecord
	call        mcp.EngramCollectionListRequest
}

func (service *capturingMCPStreamCollectionListService) ListCollections(
	_ context.Context,
	request mcp.EngramCollectionListRequest,
) ([]models.EngramCollectionRecord, error) {
	service.call = request
	return append([]models.EngramCollectionRecord(nil), service.collections...), nil
}

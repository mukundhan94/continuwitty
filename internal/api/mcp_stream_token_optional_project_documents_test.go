package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/mcp"
	"engram/internal/models"
)

type projectDocumentsRouteScenario struct {
	name      string
	requestID string
	toolsCall bool
}

func TestMountMCPRoutesTokenProjectAutofillForProjectDocuments(t *testing.T) {
	tests := []projectDocumentsRouteScenario{
		{name: "tools/call", requestID: "docs-tools", toolsCall: true},
		{name: "direct", requestID: "docs-direct", toolsCall: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			documentService := &capturingMCPStreamProjectDocumentListService{}
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{ProjectDocumentService: documentService},
				[]string{"project-token"},
			)
			response := postProjectDocumentsRouteRequest(t, router, tc)
			assertMCPToolsCallStatusOK(t, response)
			if documentService.call.ProjectID == nil || *documentService.call.ProjectID != "project-token" {
				t.Fatalf("expected token project autofill for chat.list_project_documents")
			}
		})
	}
}

func TestMountMCPRoutesTokenProjectAutofillRequiresExplicitProjectForProjectDocuments(t *testing.T) {
	tests := []projectDocumentsRouteScenario{
		{name: "tools/call", requestID: "docs-tools-denied", toolsCall: true},
		{name: "direct", requestID: "docs-direct-denied", toolsCall: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{ProjectDocumentService: &capturingMCPStreamProjectDocumentListService{}},
				[]string{"project-a", "project-b"},
			)
			response := postProjectDocumentsRouteRequest(t, router, tc)
			assertMCPToolsCallStatusOK(t, response)
			assertMCPMissingProjectIDReasonError(
				t,
				response,
				"token_has_multiple_allowed_projects",
			)
		})
	}
}

func postProjectDocumentsRouteRequest(
	t *testing.T,
	router http.Handler,
	scenario projectDocumentsRouteScenario,
) *httptest.ResponseRecorder {
	t.Helper()
	if scenario.toolsCall {
		return postMCPToolsCallRequest(t, router, mcpToolsCallRequest{
			RequestID: scenario.requestID,
			ToolName:  "chat_list_project_documents",
			Arguments: map[string]any{},
		})
	}
	return postMCPJSONRPCRequest(t, router, mcpJSONRPCPostRequest{
		RequestID: scenario.requestID,
		Method:    "chat.list_project_documents",
		Params:    map[string]any{},
	})
}

type capturingMCPStreamProjectDocumentListService struct {
	documents []models.DocumentRecord
	call      mcp.ProjectDocumentListRequest
}

func (service *capturingMCPStreamProjectDocumentListService) ListProjectDocuments(
	_ context.Context,
	request mcp.ProjectDocumentListRequest,
) ([]models.DocumentRecord, error) {
	service.call = request
	return append([]models.DocumentRecord(nil), service.documents...), nil
}

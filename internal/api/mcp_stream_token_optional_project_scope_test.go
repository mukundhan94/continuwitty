package api

import (
	"testing"

	"engram/internal/mcp"
)

type tokenOptionalProjectScopeScenario struct {
	name      string
	requestID string
	toolName  string
	method    string
	params    map[string]any
}

func TestMountMCPRoutesTokenProjectScopeRejectsOutOfScopeOptionalProjectToolsCall(t *testing.T) {
	for _, tc := range tokenOptionalProjectScopeScenarios() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{},
				[]string{"project-token"},
			)
			response := postMCPToolsCallRequest(t, router, mcpToolsCallRequest{
				RequestID: tc.requestID + "-tools",
				ToolName:  tc.toolName,
				Arguments: tc.params,
			})
			assertMCPToolsCallStatusOK(t, response)
			assertMCPProjectScopeDeniedError(t, response, "project-other")
		})
	}
}

func TestMountMCPRoutesTokenProjectScopeRejectsOutOfScopeOptionalProjectDirect(t *testing.T) {
	for _, tc := range tokenOptionalProjectScopeScenarios() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{},
				[]string{"project-token"},
			)
			response := postMCPJSONRPCRequest(t, router, mcpJSONRPCPostRequest{
				RequestID: tc.requestID + "-direct",
				Method:    tc.method,
				Params:    tc.params,
			})
			assertMCPToolsCallStatusOK(t, response)
			assertMCPProjectScopeDeniedError(t, response, "project-other")
		})
	}
}

func tokenOptionalProjectScopeScenarios() []tokenOptionalProjectScopeScenario {
	return []tokenOptionalProjectScopeScenario{
		{
			name:      "chat.list_sessions",
			requestID: "chat-list-sessions-denied",
			toolName:  "chat_list_sessions",
			method:    "chat.list_sessions",
			params: map[string]any{
				"project_id": "project-other",
			},
		},
		{
			name:      "engram.query",
			requestID: "engram-query-denied",
			toolName:  "engram_query",
			method:    "engram.query",
			params: map[string]any{
				"project_id": "project-other",
				"query":      "Denied",
			},
		},
		{
			name:      "engram.list",
			requestID: "engram-list-denied",
			toolName:  "engram_list",
			method:    "engram.list",
			params: map[string]any{
				"project_id": "project-other",
			},
		},
		{
			name:      "chat.list_project_documents",
			requestID: "chat-list-project-documents-denied",
			toolName:  "chat_list_project_documents",
			method:    "chat.list_project_documents",
			params: map[string]any{
				"project_id": "project-other",
			},
		},
		{
			name:      "engram.collection_list",
			requestID: "engram-collection-list-denied",
			toolName:  "engram_collection_list",
			method:    "engram.collection_list",
			params: map[string]any{
				"project_id": "project-other",
			},
		},
	}
}

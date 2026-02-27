package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"engram/internal/mcp"
)

type tokenProjectDeniedToolsCallScenario struct {
	name      string
	requestID string
	toolName  string
	arguments map[string]any
}

type tokenProjectDeniedDirectScenario struct {
	name      string
	requestID string
	method    string
	params    map[string]any
}

func TestMountMCPRoutesTokenProjectScopeRejectsOutOfScopeToolsCallCreateTools(t *testing.T) {
	for _, tc := range tokenProjectDeniedToolsCallScenarios() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{},
				[]string{"project-token"},
			)
			response := postMCPToolsCallRequest(t, router, mcpToolsCallRequest{
				RequestID: tc.requestID,
				ToolName:  tc.toolName,
				Arguments: tc.arguments,
			})
			assertMCPToolsCallStatusOK(t, response)
			assertMCPProjectScopeDeniedError(t, response, "project-other")
		})
	}
}

func TestMountMCPRoutesTokenProjectScopeRejectsOutOfScopeDirectCreateTools(t *testing.T) {
	for _, tc := range tokenProjectDeniedDirectScenarios() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{},
				[]string{"project-token"},
			)
			response := postMCPJSONRPCRequest(t, router, mcpJSONRPCPostRequest{
				RequestID: tc.requestID,
				Method:    tc.method,
				Params:    tc.params,
			})
			assertMCPToolsCallStatusOK(t, response)
			assertMCPProjectScopeDeniedError(t, response, "project-other")
		})
	}
}

func tokenProjectDeniedToolsCallScenarios() []tokenProjectDeniedToolsCallScenario {
	return []tokenProjectDeniedToolsCallScenario{
		{
			name:      "engram.create",
			requestID: "create-denied",
			toolName:  "engram_create",
			arguments: map[string]any{
				"project_id":                "project-other",
				"title":                     "Denied",
				"detailed_summary_markdown": "Details",
			},
		},
		{
			name:      "engram.create_from_conversation",
			requestID: "create-conv-denied",
			toolName:  "engram_create_from_conversation",
			arguments: map[string]any{
				"project_id":            "project-other",
				"title":                 "Denied",
				"conversation_markdown": "Details",
			},
		},
		{
			name:      "chat.save_as_engram without session",
			requestID: "chat-save-denied",
			toolName:  "chat_save_as_engram",
			arguments: map[string]any{
				"project_id":            "project-other",
				"title":                 "Denied",
				"conversation_markdown": "Details",
			},
		},
	}
}

func tokenProjectDeniedDirectScenarios() []tokenProjectDeniedDirectScenario {
	return []tokenProjectDeniedDirectScenario{
		{
			name:      "engram.create",
			requestID: "create-direct-denied",
			method:    "engram.create",
			params: map[string]any{
				"project_id":                "project-other",
				"title":                     "Denied",
				"detailed_summary_markdown": "Details",
			},
		},
		{
			name:      "engram.create_from_conversation",
			requestID: "create-conv-direct-denied",
			method:    "engram.create_from_conversation",
			params: map[string]any{
				"project_id":            "project-other",
				"title":                 "Denied",
				"conversation_markdown": "Details",
			},
		},
		{
			name:      "chat.save_as_engram without session",
			requestID: "chat-save-direct-denied",
			method:    "chat.save_as_engram",
			params: map[string]any{
				"project_id":            "project-other",
				"title":                 "Denied",
				"conversation_markdown": "Details",
			},
		},
	}
}

func assertMCPProjectScopeDeniedError(
	t *testing.T,
	response *httptest.ResponseRecorder,
	expectedProjectID string,
) {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json response: %v", err)
	}
	errorPayload, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected json-rpc error payload")
	}
	if errorPayload["code"] != float64(-32003) {
		t.Fatalf("expected project-scope denied code -32003")
	}
	data, ok := errorPayload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected error data payload")
	}
	if data["project_id"] != expectedProjectID {
		t.Fatalf("expected denied project_id %q payload", expectedProjectID)
	}
}

package mcp

import (
	"context"
	"testing"

	"engram/internal/governance"
	"engram/internal/models"
)

func TestCompatibilityServiceInitialize(t *testing.T) {
	frame := runCompatibilityRequest(
		t,
		StreamCallRequest{Request: JSONRPCRequest{JSONRPC: "2.0", ID: "init", Method: "initialize"}},
	)

	result, ok := frame["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected initialize result payload")
	}
	protocolVersion, _ := result["protocolVersion"].(string)
	if protocolVersion != mcpProtocolVersion {
		t.Fatalf("expected protocol version %q, got %q", mcpProtocolVersion, protocolVersion)
	}
	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatalf("expected serverInfo payload")
	}
	serverVersion, _ := serverInfo["version"].(string)
	if serverVersion != "1.2.3" {
		t.Fatalf("expected server version 1.2.3, got %q", serverVersion)
	}
	policy, ok := result["policy"].(map[string]any)
	if !ok {
		t.Fatalf("expected policy payload")
	}
	policyVersion, _ := policy["tool_policy_version"].(string)
	if policyVersion != governance.DefaultMCPToolPolicyVersion {
		t.Fatalf("expected tool policy version %q, got %q", governance.DefaultMCPToolPolicyVersion, policyVersion)
	}
	evalSuiteVersion, _ := policy["eval_suite_version"].(string)
	if evalSuiteVersion != governance.DefaultEvalSuiteVersion {
		t.Fatalf("expected eval suite version %q, got %q", governance.DefaultEvalSuiteVersion, evalSuiteVersion)
	}
}

func TestCompatibilityServiceToolsList(t *testing.T) {
	frame := runCompatibilityRequest(
		t,
		StreamCallRequest{Request: JSONRPCRequest{JSONRPC: "2.0", ID: "tools-list", Method: "tools/list"}},
	)

	toolNames := toolNamesFromFrame(t, frame)
	if len(toolNames) != len(toolCatalogOrder) {
		t.Fatalf("expected %d tools, got %d", len(toolCatalogOrder), len(toolNames))
	}
	if !containsString(toolNames, "chat_create_session") {
		t.Fatalf("expected chat_create_session in tool list")
	}
}

func TestCompatibilityServiceToolsListRespectsReadScope(t *testing.T) {
	frame := runCompatibilityRequest(
		t,
		StreamCallRequest{
			Request: JSONRPCRequest{JSONRPC: "2.0", ID: "tools-list", Method: "tools/list"},
			TokenAuth: &models.MCPTokenAuthContext{
				Scope: models.MCPTokenScopeRead,
			},
		},
	)

	toolNames := toolNamesFromFrame(t, frame)
	if containsString(toolNames, "chat_create_session") {
		t.Fatalf("expected write tool to be hidden for read scope")
	}
	if !containsString(toolNames, "engram_query") {
		t.Fatalf("expected read tool engram_query to remain visible")
	}
}

func TestCompatibilityServiceToolsListIncludesCatalogMetadataForConversationSave(t *testing.T) {
	frame := runCompatibilityRequest(
		t,
		StreamCallRequest{Request: JSONRPCRequest{JSONRPC: "2.0", ID: "tools-list", Method: "tools/list"}},
	)
	tool := findPublicToolByName(t, frame, "chat_save_as_engram")
	description, _ := tool["description"].(string)
	expectedDescription := "Save as engram. Supports either a chat session snapshot or direct conversation markdown when no session_id exists."
	if description != expectedDescription {
		t.Fatalf("expected chat_save_as_engram description parity")
	}
	inputSchema := mapFromMap(t, tool, "inputSchema")
	properties := mapFromMap(t, inputSchema, "properties")
	if _, exists := properties["session_id"]; !exists {
		t.Fatalf("expected session_id property in chat_save_as_engram schema")
	}
	if _, exists := properties["conversation_markdown"]; !exists {
		t.Fatalf("expected conversation_markdown property in chat_save_as_engram schema")
	}
}

func TestCompatibilityServiceToolsListIncludesProjectScopeForCurationRefresh(t *testing.T) {
	frame := runCompatibilityRequest(
		t,
		StreamCallRequest{Request: JSONRPCRequest{JSONRPC: "2.0", ID: "tools-list", Method: "tools/list"}},
	)
	tool := findPublicToolByName(t, frame, "engram_curation_refresh_links")
	inputSchema := mapFromMap(t, tool, "inputSchema")
	properties := mapFromMap(t, inputSchema, "properties")
	if _, exists := properties["project_id"]; !exists {
		t.Fatalf("expected project_id property in engram_curation_refresh_links schema")
	}
}

func TestCompatibilityServiceToolsListIncludesFeedbackSignalFieldsForEngramFeedback(t *testing.T) {
	frame := runCompatibilityRequest(
		t,
		StreamCallRequest{Request: JSONRPCRequest{JSONRPC: "2.0", ID: "tools-list", Method: "tools/list"}},
	)
	tool := findPublicToolByName(t, frame, "engram_feedback")
	inputSchema := mapFromMap(t, tool, "inputSchema")
	properties := mapFromMap(t, inputSchema, "properties")
	if _, exists := properties["relevance_score"]; !exists {
		t.Fatalf("expected relevance_score property in engram_feedback schema")
	}
	if _, exists := properties["session_id"]; !exists {
		t.Fatalf("expected session_id property in engram_feedback schema")
	}
	if _, exists := properties["integration_depth"]; !exists {
		t.Fatalf("expected integration_depth property in engram_feedback schema")
	}
}

func TestCompatibilityServiceToolsListIncludesAuthorityFilterForEngramQuery(t *testing.T) {
	frame := runCompatibilityRequest(
		t,
		StreamCallRequest{Request: JSONRPCRequest{JSONRPC: "2.0", ID: "tools-list", Method: "tools/list"}},
	)
	tool := findPublicToolByName(t, frame, "engram_query")
	inputSchema := mapFromMap(t, tool, "inputSchema")
	properties := mapFromMap(t, inputSchema, "properties")
	if _, exists := properties["source_session_quality_min"]; !exists {
		t.Fatalf("expected source_session_quality_min property in engram_query schema")
	}
}

func TestBuildVisiblePublicToolCatalogClonesInputSchemas(t *testing.T) {
	tools := buildVisiblePublicToolCatalog(nil)
	first := findToolByPublicName(tools, "chat_save_as_engram")
	firstSchema := mapFromMap(t, first, "inputSchema")
	firstProperties := mapFromMap(t, firstSchema, "properties")
	sessionField := mapFromMap(t, firstProperties, "session_id")
	sessionField["format"] = "mutated"

	tools = buildVisiblePublicToolCatalog(nil)
	second := findToolByPublicName(tools, "chat_save_as_engram")
	secondSchema := mapFromMap(t, second, "inputSchema")
	secondProperties := mapFromMap(t, secondSchema, "properties")
	secondSessionField := mapFromMap(t, secondProperties, "session_id")
	if secondSessionField["format"] != "uuid" {
		t.Fatalf("expected catalog input schemas to be cloned per request")
	}
}

func TestCompatibilityServiceErrorMappings(t *testing.T) {
	testCases := []struct {
		name         string
		request      StreamCallRequest
		expectedCode int
	}{
		{
			name: "tools/call missing name",
			request: StreamCallRequest{
				Request: JSONRPCRequest{
					JSONRPC: "2.0",
					ID:      "tools-call",
					Method:  "tools/call",
					Params:  map[string]any{},
				},
			},
			expectedCode: -32602,
		},
		{
			name: "tools/call invalid arguments",
			request: StreamCallRequest{
				Request: JSONRPCRequest{
					JSONRPC: "2.0",
					ID:      "tools-call",
					Method:  "tools/call",
					Params: map[string]any{
						"name":      "project_list",
						"arguments": "invalid",
					},
				},
			},
			expectedCode: -32602,
		},
		{
			name: "unknown method",
			request: StreamCallRequest{
				Request: JSONRPCRequest{
					JSONRPC: "2.0",
					ID:      "custom",
					Method:  "unknown.method",
				},
			},
			expectedCode: -32601,
		},
		{
			name: "known tool not implemented",
			request: StreamCallRequest{
				Request: JSONRPCRequest{
					JSONRPC: "2.0",
					ID:      "tools-call",
					Method:  "tools/call",
					Params:  map[string]any{"name": "engram_query"},
				},
			},
			expectedCode: -32000,
		},
		{
			name: "known direct tool method not implemented",
			request: StreamCallRequest{
				Request: JSONRPCRequest{
					JSONRPC: "2.0",
					ID:      "direct-call",
					Method:  "engram_query",
				},
			},
			expectedCode: -32000,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			assertRequestErrorCode(t, testCase.request, testCase.expectedCode)
		})
	}
}

func TestCompatibilityServiceErrorMappingsChatSaveMissingConversationMarkdown(t *testing.T) {
	frame := runCompatibilityRequestWithService(
		t,
		NewCompatibilityServiceWithDependencies(
			"1.2.3",
			CompatibilityServiceDependencies{
				EngramCreateConversation: &fakeEngramCreateConversationService{},
			},
		),
		directToolRequest(
			"39135000-0000-0000-0000-000000000390",
			"chat.save_as_engram",
			map[string]any{},
		),
	)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32602)
	data := mapFromMap(t, errorPayload, "data")
	if data["missing"] != "conversation_markdown" {
		t.Fatalf("expected missing conversation_markdown error data")
	}
}

func TestCompatibilityServiceRejectsWriteToolsForReadToken(t *testing.T) {
	testCases := []StreamCallRequest{
		{
			Request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "direct-call",
				Method:  "chat.send_message",
			},
			TokenAuth: &models.MCPTokenAuthContext{
				Scope: models.MCPTokenScopeRead,
			},
		},
		{
			Request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "tools-call",
				Method:  "tools/call",
				Params:  map[string]any{"name": "chat_send_message"},
			},
			TokenAuth: &models.MCPTokenAuthContext{
				Scope: models.MCPTokenScopeRead,
			},
		},
	}

	for _, request := range testCases {
		assertRequestErrorCode(t, request, -32003)
	}
}

func singleFrame(t *testing.T, frames <-chan Frame) Frame {
	t.Helper()
	frame, ok := <-frames
	if !ok {
		t.Fatalf("expected frame, channel closed")
	}
	if _, stillOpen := <-frames; stillOpen {
		t.Fatalf("expected a single frame")
	}
	return frame
}

func runCompatibilityRequest(t *testing.T, request StreamCallRequest) Frame {
	t.Helper()
	service := NewCompatibilityService("1.2.3")
	return singleFrame(t, service.StreamCall(context.Background(), request))
}

func assertRequestErrorCode(t *testing.T, request StreamCallRequest, expectedCode int) {
	t.Helper()
	frame := runCompatibilityRequest(t, request)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, expectedCode)
}

func toolNamesFromFrame(t *testing.T, frame Frame) []string {
	t.Helper()
	rawTools := rawToolsFromFrame(t, frame)
	toolNames := make([]string, 0, len(rawTools))
	for _, tool := range rawTools {
		name, _ := tool["name"].(string)
		toolNames = append(toolNames, name)
	}
	return toolNames
}

func rawToolsFromFrame(t *testing.T, frame Frame) []map[string]any {
	t.Helper()
	result, ok := frame["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result payload in frame")
	}
	rawTools, ok := result["tools"].([]map[string]any)
	if !ok {
		t.Fatalf("expected tools payload")
	}
	return rawTools
}

func findPublicToolByName(t *testing.T, frame Frame, publicName string) map[string]any {
	t.Helper()
	return findToolByPublicName(rawToolsFromFrame(t, frame), publicName)
}

func findToolByPublicName(tools []map[string]any, publicName string) map[string]any {
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		if name == publicName {
			return tool
		}
	}
	return map[string]any{}
}

func errorPayloadFromFrame(t *testing.T, frame Frame) map[string]any {
	t.Helper()
	errorPayload, ok := frame["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error payload")
	}
	return errorPayload
}

func requireErrorCode(t *testing.T, errorPayload map[string]any, expected int) {
	t.Helper()
	code, ok := errorPayload["code"].(int)
	if !ok {
		t.Fatalf("expected integer error code")
	}
	if code != expected {
		t.Fatalf("expected error code %d, got %d", expected, code)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

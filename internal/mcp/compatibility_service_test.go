package mcp

import (
	"context"
	"testing"

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
	result, ok := frame["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result payload in frame")
	}
	rawTools, ok := result["tools"].([]map[string]any)
	if !ok {
		t.Fatalf("expected tools payload")
	}
	toolNames := make([]string, 0, len(rawTools))
	for _, tool := range rawTools {
		name, _ := tool["name"].(string)
		toolNames = append(toolNames, name)
	}
	return toolNames
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

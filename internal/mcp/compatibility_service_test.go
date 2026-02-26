package mcp

import (
	"context"
	"testing"
)

func TestCompatibilityServiceInitialize(t *testing.T) {
	service := NewCompatibilityService("1.2.3")
	frame := singleFrame(t, service.StreamCall(
		context.Background(),
		StreamCallRequest{Request: JSONRPCRequest{JSONRPC: "2.0", ID: "init", Method: "initialize"}},
	))

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
	service := NewCompatibilityService("1.2.3")
	frame := singleFrame(t, service.StreamCall(
		context.Background(),
		StreamCallRequest{Request: JSONRPCRequest{JSONRPC: "2.0", ID: "tools-list", Method: "tools/list"}},
	))

	result, ok := frame["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected tools/list result payload")
	}
	tools, ok := result["tools"].([]map[string]any)
	if !ok {
		t.Fatalf("expected tools list payload")
	}
	if len(tools) != 0 {
		t.Fatalf("expected no tools in compatibility service, got %d", len(tools))
	}
}

func TestCompatibilityServiceToolsCallMissingName(t *testing.T) {
	service := NewCompatibilityService("1.2.3")
	frame := singleFrame(t, service.StreamCall(
		context.Background(),
		StreamCallRequest{
			Request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "tools-call",
				Method:  "tools/call",
				Params:  map[string]any{},
			},
		},
	))

	errorPayload, ok := frame["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error payload")
	}
	code, ok := errorPayload["code"].(int)
	if !ok {
		t.Fatalf("expected integer error code")
	}
	if code != -32602 {
		t.Fatalf("expected invalid params code -32602, got %d", code)
	}
}

func TestCompatibilityServiceUnknownMethod(t *testing.T) {
	service := NewCompatibilityService("1.2.3")
	frame := singleFrame(t, service.StreamCall(
		context.Background(),
		StreamCallRequest{Request: JSONRPCRequest{JSONRPC: "2.0", ID: "custom", Method: "engram.query"}},
	))

	errorPayload, ok := frame["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error payload")
	}
	code, ok := errorPayload["code"].(int)
	if !ok {
		t.Fatalf("expected integer error code")
	}
	if code != -32601 {
		t.Fatalf("expected method not found code -32601, got %d", code)
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

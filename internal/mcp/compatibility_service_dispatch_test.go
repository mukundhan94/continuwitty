package mcp

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestCompatibilityServiceToolsCallUserGetProfile(t *testing.T) {
	actorUserID := uuid.MustParse("20000000-0000-0000-0000-000000000002")
	frame := runCompatibilityRequestWithService(
		t,
		NewCompatibilityService("1.2.3"),
		StreamCallRequest{
			Request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "tools-call",
				Method:  "tools/call",
				Params:  map[string]any{"name": "user_get_profile"},
			},
			Actor: Actor{UserID: actorUserID, Username: "alice", Role: "Admin"},
		},
	)

	result := resultPayloadFromFrame(t, frame)
	if result["tool_name"] != "user_get_profile" {
		t.Fatalf("expected tools/call tool_name to echo request name")
	}
	structuredContent := mapFromMap(t, result, "structuredContent")
	profile := mapFromMap(t, structuredContent, "profile")

	if profile["user_id"] != actorUserID {
		t.Fatalf("expected profile user_id to echo actor id")
	}
	if profile["role"] != "admin" {
		t.Fatalf("expected profile role to be normalized")
	}
}

func TestCompatibilityServiceDirectUserGetProfile(t *testing.T) {
	actorUserID := uuid.MustParse("21000000-0000-0000-0000-000000000021")
	frame := runCompatibilityRequestWithService(
		t,
		NewCompatibilityService("1.2.3"),
		StreamCallRequest{
			Request: JSONRPCRequest{JSONRPC: "2.0", ID: "direct-call", Method: "user.get_profile"},
			Actor:   Actor{UserID: actorUserID, Role: "analyst"},
		},
	)

	result := resultPayloadFromFrame(t, frame)
	profile := mapFromMap(t, result, "profile")
	if profile["user_id"] != actorUserID {
		t.Fatalf("expected profile user_id to echo actor id")
	}
}

func runCompatibilityRequestWithService(t *testing.T, service Service, request StreamCallRequest) Frame {
	t.Helper()
	return singleFrame(t, service.StreamCall(context.Background(), request))
}

func resultPayloadFromFrame(t *testing.T, frame Frame) map[string]any {
	t.Helper()
	result, ok := frame["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result payload")
	}
	return result
}

func mapFromMap(t *testing.T, source map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := source[key].(map[string]any)
	if !ok {
		t.Fatalf("expected map payload at key %q", key)
	}
	return value
}

package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"engram/internal/mcp"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
)

type tokenAllowedToolsDirectScenario struct {
	name      string
	requestID string
	method    string
}

func TestMountMCPRoutesTokenAllowedToolsRejectsDisallowedToolsCall(t *testing.T) {
	router := newMCPTokenAllowedToolsRouter([]string{"engram.query"})
	response := postMCPToolsCallRequest(t, router, mcpToolsCallRequest{
		RequestID: "tools-call-disallowed",
		ToolName:  "chat_send_message",
		Arguments: map[string]any{
			"session_id":   "40330000-0000-0000-0000-000000000001",
			"content_text": "Denied",
		},
	})
	assertMCPToolsCallStatusOK(t, response)
	assertMCPToolDeniedByAllowlistError(t, response)
}

func TestMountMCPRoutesTokenAllowedToolsRejectsDisallowedDirectWriteTool(t *testing.T) {
	router := newMCPTokenAllowedToolsRouter([]string{"engram.query"})
	tests := []tokenAllowedToolsDirectScenario{
		{name: "dotted method", requestID: "direct-dot-disallowed", method: "chat.send_message"},
		{name: "underscore alias", requestID: "direct-underscore-disallowed", method: "chat_send_message"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			response := postMCPJSONRPCRequest(t, router, mcpJSONRPCPostRequest{
				RequestID: tc.requestID,
				Method:    tc.method,
				Params: map[string]any{
					"session_id":   "40330000-0000-0000-0000-000000000001",
					"content_text": "Denied",
				},
			})
			assertMCPToolsCallStatusOK(t, response)
			assertMCPToolDeniedByAllowlistError(t, response)
		})
	}
}

func newMCPTokenAllowedToolsRouter(allowedTools []string) *chi.Mux {
	router := chi.NewRouter()
	MountMCPRoutes(
		router,
		mcp.NewCompatibilityService("1.2.3"),
		newTokenScopedMCPActorResolver(&models.MCPTokenAuthContext{
			Scope:        models.MCPTokenScopeWrite,
			AllowedTools: allowedTools,
		}),
		nil,
	)
	return router
}

func assertMCPToolDeniedByAllowlistError(t *testing.T, response *httptest.ResponseRecorder) {
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
		t.Fatalf("expected token-policy denied code -32003")
	}
	data, ok := errorPayload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected error data payload")
	}
	tool, ok := data["tool"].(string)
	if !ok || tool == "" {
		t.Fatalf("expected denied tool in error data")
	}
	if data["token_scope"] != "write" {
		t.Fatalf("expected token_scope write")
	}
}

package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"engram/internal/mcp"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
)

type tokenScopeDirectWriteScenario struct {
	name      string
	requestID string
	method    string
}

func TestMountMCPRoutesTokenReadScopeRejectsDirectWriteTools(t *testing.T) {
	router := newMCPTokenReadScopeRouter()
	tests := []tokenScopeDirectWriteScenario{
		{name: "dotted method", requestID: "direct-dot", method: "chat.send_message"},
		{name: "underscore alias", requestID: "direct-underscore", method: "chat_send_message"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			response := postMCPJSONRPCRequest(t, router, mcpJSONRPCPostRequest{
				RequestID: tc.requestID,
				Method:    tc.method,
				Params: map[string]any{
					"session_id":   "40320000-0000-0000-0000-000000000001",
					"content_text": "Denied",
				},
			})
			assertMCPToolsCallStatusOK(t, response)
			assertMCPReadScopeDeniedError(t, response)
		})
	}
}

func newMCPTokenReadScopeRouter() *chi.Mux {
	router := chi.NewRouter()
	MountMCPRoutes(
		router,
		mcp.NewCompatibilityService("1.2.3"),
		newTokenScopedMCPActorResolver(&models.MCPTokenAuthContext{Scope: models.MCPTokenScopeRead}),
		nil,
	)
	return router
}

func assertMCPReadScopeDeniedError(t *testing.T, response *httptest.ResponseRecorder) {
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
		t.Fatalf("expected insufficient-scope code -32003")
	}
	data, ok := errorPayload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected scope detail payload")
	}
	if data["required_scope"] != "write" {
		t.Fatalf("expected required_scope write")
	}
	if data["token_scope"] != "read" {
		t.Fatalf("expected token_scope read")
	}
}

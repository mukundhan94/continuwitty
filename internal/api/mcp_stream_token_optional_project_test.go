package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/mcp"
	"engram/internal/models"
)

type sessionListRouteScenario struct {
	name      string
	requestID string
	toolsCall bool
}

func TestMountMCPRoutesTokenProjectAutofillForSessionList(t *testing.T) {
	tests := []sessionListRouteScenario{
		{name: "tools/call", requestID: "sessions-tools", toolsCall: true},
		{name: "direct", requestID: "sessions-direct", toolsCall: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			sessionList := &capturingMCPStreamSessionListService{}
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{SessionService: sessionList},
				[]string{"project-token"},
			)
			response := postSessionListRouteRequest(t, router, tc)
			assertMCPToolsCallStatusOK(t, response)
			if sessionList.call.ProjectID == nil || *sessionList.call.ProjectID != "project-token" {
				t.Fatalf("expected token project autofill for chat.list_sessions")
			}
		})
	}
}

func TestMountMCPRoutesTokenProjectAutofillRequiresExplicitProjectForSessionList(t *testing.T) {
	tests := []sessionListRouteScenario{
		{name: "tools/call", requestID: "sessions-tools-denied", toolsCall: true},
		{name: "direct", requestID: "sessions-direct-denied", toolsCall: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{SessionService: &capturingMCPStreamSessionListService{}},
				[]string{"project-a", "project-b"},
			)
			response := postSessionListRouteRequest(t, router, tc)
			assertMCPToolsCallStatusOK(t, response)
			assertMCPMissingProjectIDReasonError(
				t,
				response,
				"token_has_multiple_allowed_projects",
			)
		})
	}
}

func postSessionListRouteRequest(
	t *testing.T,
	router http.Handler,
	scenario sessionListRouteScenario,
) *httptest.ResponseRecorder {
	t.Helper()
	if scenario.toolsCall {
		return postMCPToolsCallRequest(t, router, mcpToolsCallRequest{
			RequestID: scenario.requestID,
			ToolName:  "chat_list_sessions",
			Arguments: map[string]any{},
		})
	}
	return postMCPJSONRPCRequest(t, router, mcpJSONRPCPostRequest{
		RequestID: scenario.requestID,
		Method:    "chat.list_sessions",
		Params:    map[string]any{},
	})
}

func assertMCPMissingProjectIDReasonError(
	t *testing.T,
	response *httptest.ResponseRecorder,
	expectedReason string,
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
	if errorPayload["code"] != float64(-32602) {
		t.Fatalf("expected invalid-params code -32602")
	}
	data, ok := errorPayload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected error data payload")
	}
	if data["missing"] != "project_id" {
		t.Fatalf("expected missing project_id payload")
	}
	if data["reason"] != expectedReason {
		t.Fatalf("expected reason %q in payload", expectedReason)
	}
}

type capturingMCPStreamSessionListService struct {
	sessions []models.ChatSessionRecord
	call     mcp.SessionListRequest
}

func (service *capturingMCPStreamSessionListService) ListSessions(
	_ context.Context,
	request mcp.SessionListRequest,
) ([]models.ChatSessionRecord, error) {
	service.call = request
	return append([]models.ChatSessionRecord(nil), service.sessions...), nil
}

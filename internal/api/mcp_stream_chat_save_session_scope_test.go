package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"engram/internal/mcp"
	"engram/internal/models"

	"github.com/google/uuid"
)

type chatSaveSessionStreamRequest struct {
	name      string
	requestID string
	method    string
}

func TestMountMCPRoutesTokenProjectScopeChatSaveAsEngramPrefersSessionProject(t *testing.T) {
	sessionID := uuid.MustParse("40300000-0000-0000-0000-000000000001")
	tests := []chatSaveSessionStreamRequest{
		{name: "tools/call", requestID: "chat-save-tools", method: "tools/call"},
		{name: "direct", requestID: "chat-save-direct", method: "chat.save_as_engram"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			sessionGet := &capturingMCPStreamSessionGetService{
				session: &models.ChatSessionRecord{
					SessionID: sessionID,
					ProjectID: "project-session",
				},
			}
			saveSession := &capturingMCPStreamSessionSaveAsEngramService{
				response: &models.SaveSessionAsEngramResponse{
					EngramID:  uuid.MustParse("40310000-0000-0000-0000-000000000001"),
					SessionID: sessionID,
					CreatedAt: time.Now().UTC(),
				},
			}
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{
					SessionGet:          sessionGet,
					SessionSaveAsEngram: saveSession,
				},
				[]string{"project-session"},
			)
			response := postChatSaveSessionStreamRequest(t, router, chatSaveSessionStreamPostRequest{
				RequestID: tc.requestID,
				Method:    tc.method,
				SessionID: sessionID,
			})
			assertMCPToolsCallStatusOK(t, response)
			assertMCPResponseHasNoError(t, response)
			if sessionGet.call.SessionID != sessionID {
				t.Fatalf("expected session lookup for chat.save_as_engram session scope")
			}
			if saveSession.call.SessionID != sessionID {
				t.Fatalf("expected save-session call to use requested session id")
			}
		})
	}
}

type chatSaveSessionStreamPostRequest struct {
	RequestID string
	Method    string
	SessionID uuid.UUID
}

func postChatSaveSessionStreamRequest(
	t *testing.T,
	router http.Handler,
	request chatSaveSessionStreamPostRequest,
) *httptest.ResponseRecorder {
	t.Helper()
	params := map[string]any{
		"session_id": request.SessionID.String(),
		"project_id": "project-other",
		"title":      "Session snapshot",
	}
	if request.Method == "tools/call" {
		return postMCPToolsCallRequest(t, router, mcpToolsCallRequest{
			RequestID: request.RequestID,
			ToolName:  "chat_save_as_engram",
			Arguments: params,
		})
	}
	return postMCPJSONRPCRequest(t, router, mcpJSONRPCPostRequest{
		RequestID: request.RequestID,
		Method:    request.Method,
		Params:    params,
	})
}

func assertMCPResponseHasNoError(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json response: %v", err)
	}
	if rawError, exists := payload["error"]; exists && rawError != nil {
		t.Fatalf("expected success response, received error payload: %#v", rawError)
	}
}

type capturingMCPStreamSessionGetService struct {
	session *models.ChatSessionRecord
	call    mcp.SessionGetRequest
}

func (service *capturingMCPStreamSessionGetService) GetSession(
	_ context.Context,
	request mcp.SessionGetRequest,
) (*models.ChatSessionRecord, error) {
	service.call = request
	return clonePointer(service.session), nil
}

type capturingMCPStreamSessionSaveAsEngramService struct {
	response *models.SaveSessionAsEngramResponse
	call     mcp.SessionSaveAsEngramRequest
}

func (service *capturingMCPStreamSessionSaveAsEngramService) SaveSessionAsEngram(
	_ context.Context,
	request mcp.SessionSaveAsEngramRequest,
) (*models.SaveSessionAsEngramResponse, error) {
	service.call = request
	return clonePointer(service.response), nil
}

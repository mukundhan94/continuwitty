package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"engram/internal/mcp"
	"engram/internal/models"

	"github.com/google/uuid"
)

type chatCreateSessionRouteScenario struct {
	name      string
	requestID string
	toolsCall bool
}

func TestMountMCPRoutesTokenProjectAutofillForChatCreateSession(t *testing.T) {
	tests := []chatCreateSessionRouteScenario{
		{name: "tools/call", requestID: "chat-create-tools", toolsCall: true},
		{name: "direct", requestID: "chat-create-direct", toolsCall: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			createService := &capturingMCPStreamSessionCreateService{
				response: &models.ChatSessionRecord{
					SessionID:               uuid.MustParse("40350000-0000-0000-0000-000000000001"),
					OwnerUserID:             uuid.MustParse("40000000-0000-0000-0000-000000000004"),
					ProjectID:               "project-token",
					Title:                   "Token fallback",
					Provider:                models.ChatProviderOpenAI,
					ModelID:                 "gpt-4o-mini",
					SystemPrompt:            "",
					VisibilityScope:         models.VisibilityScopePrivate,
					AutosaveEnabled:         false,
					AutosaveStrategy:        models.ChatAutosaveStrategyOff,
					AutosaveIntervalMinutes: 30,
					AutosaveMinMessages:     6,
					RetentionDays:           30,
					RetentionMaxSnapshots:   60,
					CreatedAt:               time.Now().UTC(),
					UpdatedAt:               time.Now().UTC(),
				},
			}
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{SessionCreate: createService},
				[]string{"project-token"},
			)
			response := postChatCreateSessionRouteRequest(t, router, tc)
			assertMCPToolsCallStatusOK(t, response)
			if createService.call.Payload.ProjectID != "project-token" {
				t.Fatalf("expected token project autofill for chat.create_session")
			}
		})
	}
}

func TestMountMCPRoutesTokenProjectAutofillRequiresExplicitProjectForChatCreateSession(t *testing.T) {
	tests := []chatCreateSessionRouteScenario{
		{name: "tools/call", requestID: "chat-create-tools-denied", toolsCall: true},
		{name: "direct", requestID: "chat-create-direct-denied", toolsCall: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{SessionCreate: &capturingMCPStreamSessionCreateService{}},
				[]string{"project-a", "project-b"},
			)
			response := postChatCreateSessionRouteRequest(t, router, tc)
			assertMCPToolsCallStatusOK(t, response)
			assertMCPMissingProjectIDReasonError(
				t,
				response,
				"token_has_multiple_allowed_projects",
			)
		})
	}
}

func postChatCreateSessionRouteRequest(
	t *testing.T,
	router http.Handler,
	scenario chatCreateSessionRouteScenario,
) *httptest.ResponseRecorder {
	t.Helper()
	params := map[string]any{"title": "Token fallback"}
	if scenario.toolsCall {
		return postMCPToolsCallRequest(t, router, mcpToolsCallRequest{
			RequestID: scenario.requestID,
			ToolName:  "chat_create_session",
			Arguments: params,
		})
	}
	return postMCPJSONRPCRequest(t, router, mcpJSONRPCPostRequest{
		RequestID: scenario.requestID,
		Method:    "chat.create_session",
		Params:    params,
	})
}

type capturingMCPStreamSessionCreateService struct {
	response *models.ChatSessionRecord
	call     mcp.SessionCreateRequest
}

func (service *capturingMCPStreamSessionCreateService) CreateSession(
	_ context.Context,
	request mcp.SessionCreateRequest,
) (*models.ChatSessionRecord, error) {
	service.call = request
	return clonePointer(service.response), nil
}

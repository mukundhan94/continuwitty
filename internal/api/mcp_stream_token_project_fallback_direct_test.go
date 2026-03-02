package api

import (
	"testing"

	"engram/internal/mcp"
	"engram/internal/models"

	"github.com/google/uuid"
)

type directConversationTokenProjectScenario struct {
	name             string
	requestID        string
	method           string
	params           map[string]any
	responseEngramID string
}

func TestMountMCPRoutesTokenProjectAutofillForDirectConversationTools(t *testing.T) {
	for _, tc := range directConversationTokenProjectScenarios() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			service := &capturingMCPCreateFromConversationService{
				response: &mcp.EngramCreateFromConversationResponse{
					Engram: models.EngramCreateResponse{
						EngramID: uuid.MustParse(tc.responseEngramID),
					},
					EnrichmentReport: map[string]any{},
				},
			}
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{EngramCreateConversation: service},
				[]string{"project-token"},
			)
			response := postMCPJSONRPCRequest(
				t,
				router,
				mcpJSONRPCPostRequest{
					RequestID: tc.requestID,
					Method:    tc.method,
					Params:    tc.params,
				},
			)
			assertMCPToolsCallStatusOK(t, response)
			if service.call.ProjectID != "project-token" {
				t.Fatalf("expected token project autofill for %s", tc.method)
			}
		})
	}
}

func TestMountMCPRoutesTokenProjectAutofillRequiresExplicitProjectForDirectConversationTools(t *testing.T) {
	for _, tc := range directConversationTokenProjectScenarios() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			service := &capturingMCPCreateFromConversationService{
				response: &mcp.EngramCreateFromConversationResponse{
					Engram: models.EngramCreateResponse{
						EngramID: uuid.MustParse(tc.responseEngramID),
					},
					EnrichmentReport: map[string]any{},
				},
			}
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{EngramCreateConversation: service},
				[]string{"project-a", "project-b"},
			)
			response := postMCPJSONRPCRequest(
				t,
				router,
				mcpJSONRPCPostRequest{
					RequestID: tc.requestID,
					Method:    tc.method,
					Params:    tc.params,
				},
			)
			assertMCPToolsCallStatusOK(t, response)
			assertMCPMissingProjectIDError(t, response)
		})
	}
}

func directConversationTokenProjectScenarios() []directConversationTokenProjectScenario {
	return []directConversationTokenProjectScenario{
		{
			name:      "engram.create_from_conversation",
			requestID: "create-conv-direct",
			method:    "engram.create_from_conversation",
			params: map[string]any{
				"title":                 "Token fallback conversation",
				"conversation_markdown": "Details",
			},
			responseEngramID: "40200000-0000-0000-0000-000000000001",
		},
		{
			name:      "chat.save_as_engram without session",
			requestID: "chat-save-direct",
			method:    "chat.save_as_engram",
			params: map[string]any{
				"title":                 "Token fallback save",
				"conversation_markdown": "Details",
			},
			responseEngramID: "40210000-0000-0000-0000-000000000001",
		},
	}
}

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/mcp"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type tokenProjectAutofillScenario struct {
	name      string
	requestID string
	toolName  string
	arguments map[string]any
	setup     func() (mcp.CompatibilityServiceDependencies, func() string)
}

type tokenProjectRequiredScenario struct {
	name      string
	requestID string
	toolName  string
	arguments map[string]any
	deps      mcp.CompatibilityServiceDependencies
}

func TestMountMCPRoutesTokenProjectAutofillForCreateTools(t *testing.T) {
	for _, tc := range tokenProjectAutofillScenarios() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps, projectFor := tc.setup()
			router := newMCPTokenProjectFallbackRouter(deps, []string{"project-token"})
			response := postMCPToolsCallRequest(t, router, mcpToolsCallRequest{
				RequestID: tc.requestID,
				ToolName:  tc.toolName,
				Arguments: tc.arguments,
			})
			assertMCPToolsCallStatusOK(t, response)
			if projectFor() != "project-token" {
				t.Fatalf("expected token project autofill for %s", tc.toolName)
			}
		})
	}
}

func TestMountMCPRoutesTokenProjectAutofillRequiresExplicitProjectForMultiProjectToken(t *testing.T) {
	for _, tc := range tokenProjectRequiredScenarios() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			router := newMCPTokenProjectFallbackRouter(tc.deps, []string{"project-a", "project-b"})
			response := postMCPToolsCallRequest(t, router, mcpToolsCallRequest{
				RequestID: tc.requestID,
				ToolName:  tc.toolName,
				Arguments: tc.arguments,
			})
			assertMCPToolsCallStatusOK(t, response)
			assertMCPMissingProjectIDError(t, response)
		})
	}
}

func tokenProjectAutofillScenarios() []tokenProjectAutofillScenario {
	return []tokenProjectAutofillScenario{
		{
			name:      "engram.create",
			requestID: "create",
			toolName:  "engram_create",
			arguments: map[string]any{
				"title":                     "Token fallback",
				"detailed_summary_markdown": "Details",
			},
			setup: func() (mcp.CompatibilityServiceDependencies, func() string) {
				service := &capturingMCPCreateEngramService{
					response: &models.EngramCreateResponse{
						EngramID: uuid.MustParse("40100000-0000-0000-0000-000000000001"),
					},
				}
				return mcp.CompatibilityServiceDependencies{EngramCreate: service}, func() string {
					return service.call.Payload.ProjectID
				}
			},
		},
		{
			name:      "engram.create_from_conversation",
			requestID: "create-conv",
			toolName:  "engram_create_from_conversation",
			arguments: map[string]any{
				"title":                 "Token fallback conversation",
				"conversation_markdown": "Details",
			},
			setup: func() (mcp.CompatibilityServiceDependencies, func() string) {
				service := &capturingMCPCreateFromConversationService{
					response: &mcp.EngramCreateFromConversationResponse{
						Engram: models.EngramCreateResponse{
							EngramID: uuid.MustParse("40120000-0000-0000-0000-000000000001"),
						},
						EnrichmentReport: map[string]any{},
					},
				}
				return mcp.CompatibilityServiceDependencies{EngramCreateConversation: service}, func() string {
					return service.call.ProjectID
				}
			},
		},
		{
			name:      "chat.save_as_engram without session",
			requestID: "chat-save",
			toolName:  "chat_save_as_engram",
			arguments: map[string]any{
				"title":                 "Token fallback save",
				"conversation_markdown": "Details",
			},
			setup: func() (mcp.CompatibilityServiceDependencies, func() string) {
				service := &capturingMCPCreateFromConversationService{
					response: &mcp.EngramCreateFromConversationResponse{
						Engram: models.EngramCreateResponse{
							EngramID: uuid.MustParse("40140000-0000-0000-0000-000000000001"),
						},
						EnrichmentReport: map[string]any{},
					},
				}
				return mcp.CompatibilityServiceDependencies{EngramCreateConversation: service}, func() string {
					return service.call.ProjectID
				}
			},
		},
	}
}

func tokenProjectRequiredScenarios() []tokenProjectRequiredScenario {
	return []tokenProjectRequiredScenario{
		{
			name:      "engram.create",
			requestID: "create",
			toolName:  "engram_create",
			arguments: map[string]any{
				"title":                     "Denied",
				"detailed_summary_markdown": "Details",
			},
			deps: mcp.CompatibilityServiceDependencies{
				EngramCreate: &capturingMCPCreateEngramService{
					response: &models.EngramCreateResponse{EngramID: uuid.MustParse("40110000-0000-0000-0000-000000000001")},
				},
			},
		},
		{
			name:      "engram.create_from_conversation",
			requestID: "create-conv",
			toolName:  "engram_create_from_conversation",
			arguments: map[string]any{
				"title":                 "Denied",
				"conversation_markdown": "Details",
			},
			deps: mcp.CompatibilityServiceDependencies{
				EngramCreateConversation: &capturingMCPCreateFromConversationService{
					response: &mcp.EngramCreateFromConversationResponse{
						Engram:           models.EngramCreateResponse{EngramID: uuid.MustParse("40130000-0000-0000-0000-000000000001")},
						EnrichmentReport: map[string]any{},
					},
				},
			},
		},
		{
			name:      "chat.save_as_engram without session",
			requestID: "chat-save",
			toolName:  "chat_save_as_engram",
			arguments: map[string]any{
				"title":                 "Denied",
				"conversation_markdown": "Details",
			},
			deps: mcp.CompatibilityServiceDependencies{
				EngramCreateConversation: &capturingMCPCreateFromConversationService{
					response: &mcp.EngramCreateFromConversationResponse{
						Engram:           models.EngramCreateResponse{EngramID: uuid.MustParse("40150000-0000-0000-0000-000000000001")},
						EnrichmentReport: map[string]any{},
					},
				},
			},
		},
	}
}

func newMCPTokenProjectFallbackRouter(
	deps mcp.CompatibilityServiceDependencies,
	allowedProjectIDs []string,
) *chi.Mux {
	router := chi.NewRouter()
	MountMCPRoutes(
		router,
		mcp.NewCompatibilityServiceWithDependencies("1.2.3", deps),
		newTokenScopedMCPActorResolver(&models.MCPTokenAuthContext{
			Scope:             models.MCPTokenScopeWrite,
			AllowedProjectIDs: append([]string(nil), allowedProjectIDs...),
		}),
		nil,
	)
	return router
}

func postMCPToolsCallRequest(
	t *testing.T,
	router http.Handler,
	callRequest mcpToolsCallRequest,
) *httptest.ResponseRecorder {
	return postMCPJSONRPCRequest(
		t,
		router,
		mcpJSONRPCPostRequest{
			RequestID: callRequest.RequestID,
			Method:    "tools/call",
			Params: map[string]any{
				"name":      callRequest.ToolName,
				"arguments": callRequest.Arguments,
			},
		},
	)
}

func postMCPJSONRPCRequest(
	t *testing.T,
	router http.Handler,
	postRequest mcpJSONRPCPostRequest,
) *httptest.ResponseRecorder {
	t.Helper()
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      postRequest.RequestID,
		"method":  postRequest.Method,
		"params":  postRequest.Params,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal json-rpc request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/stream", bytes.NewReader(body))
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

type mcpToolsCallRequest struct {
	RequestID string
	ToolName  string
	Arguments map[string]any
}

type mcpJSONRPCPostRequest struct {
	RequestID string
	Method    string
	Params    map[string]any
}

func assertMCPToolsCallStatusOK(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected tools/call status 200, got %d", response.Code)
	}
}

func assertMCPMissingProjectIDError(t *testing.T, response *httptest.ResponseRecorder) {
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
}

type capturingMCPCreateEngramService struct {
	response *models.EngramCreateResponse
	call     mcp.EngramCreateRequest
}

func (service *capturingMCPCreateEngramService) CreateEngram(
	_ context.Context,
	request mcp.EngramCreateRequest,
) (*models.EngramCreateResponse, error) {
	service.call = request
	return clonePointer(service.response), nil
}

type capturingMCPCreateFromConversationService struct {
	response *mcp.EngramCreateFromConversationResponse
	call     mcp.EngramCreateFromConversationRequest
}

func (service *capturingMCPCreateFromConversationService) CreateEngramFromConversation(
	_ context.Context,
	request mcp.EngramCreateFromConversationRequest,
) (*mcp.EngramCreateFromConversationResponse, error) {
	service.call = request
	return clonePointer(service.response), nil
}

func clonePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

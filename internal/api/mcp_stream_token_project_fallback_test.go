package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"engram/internal/mcp"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestMountMCPRoutesTokenProjectAutofillForEngramCreate(t *testing.T) {
	createService := &capturingMCPCreateEngramService{
		response: &models.EngramCreateResponse{
			EngramID: uuid.MustParse("40100000-0000-0000-0000-000000000001"),
		},
	}
	router := chi.NewRouter()
	MountMCPRoutes(
		router,
		mcp.NewCompatibilityServiceWithDependencies(
			"1.2.3",
			mcp.CompatibilityServiceDependencies{EngramCreate: createService},
		),
		newTokenScopedMCPActorResolver(&models.MCPTokenAuthContext{
			Scope:             models.MCPTokenScopeWrite,
			AllowedProjectIDs: []string{"project-token"},
		}),
		nil,
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/mcp/stream",
		strings.NewReader(`{"jsonrpc":"2.0","id":"create","method":"tools/call","params":{"name":"engram_create","arguments":{"title":"Token fallback","detailed_summary_markdown":"Details"}}}`),
	)
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected tools/call status 200, got %d", response.Code)
	}
	if createService.call.Payload.ProjectID != "project-token" {
		t.Fatalf("expected token project autofill for engram.create")
	}
}

func TestMountMCPRoutesTokenProjectAutofillRequiresExplicitProjectForMultiProjectToken(t *testing.T) {
	router := chi.NewRouter()
	MountMCPRoutes(
		router,
		mcp.NewCompatibilityServiceWithDependencies(
			"1.2.3",
			mcp.CompatibilityServiceDependencies{
				EngramCreate: &capturingMCPCreateEngramService{
					response: &models.EngramCreateResponse{
						EngramID: uuid.MustParse("40110000-0000-0000-0000-000000000001"),
					},
				},
			},
		),
		newTokenScopedMCPActorResolver(&models.MCPTokenAuthContext{
			Scope:             models.MCPTokenScopeWrite,
			AllowedProjectIDs: []string{"project-a", "project-b"},
		}),
		nil,
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/mcp/stream",
		strings.NewReader(`{"jsonrpc":"2.0","id":"create","method":"tools/call","params":{"name":"engram_create","arguments":{"title":"Denied","detailed_summary_markdown":"Details"}}}`),
	)
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected tools/call status 200, got %d", response.Code)
	}
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
	if service.response == nil {
		return nil, nil
	}
	response := *service.response
	return &response, nil
}

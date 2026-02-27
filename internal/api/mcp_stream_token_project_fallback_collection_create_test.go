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

type collectionCreateRouteScenario struct {
	name      string
	requestID string
	toolsCall bool
}

func TestMountMCPRoutesTokenProjectAutofillForCollectionCreate(t *testing.T) {
	tests := []collectionCreateRouteScenario{
		{name: "tools/call", requestID: "collection-create-tools", toolsCall: true},
		{name: "direct", requestID: "collection-create-direct", toolsCall: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			createService := &capturingMCPStreamCollectionCreateService{
				response: &mcp.EngramCollectionCreateResponse{
					Collection: models.EngramCollectionRecord{
						CollectionID: uuid.MustParse("40340000-0000-0000-0000-000000000001"),
						ProjectID:    "project-token",
						OwnerUserID:  uuid.MustParse("40000000-0000-0000-0000-000000000004"),
						Name:         "Collection",
						Description:  "Details",
						CreatedAt:    time.Now().UTC(),
						UpdatedAt:    time.Now().UTC(),
					},
					ResolvedProjectID:  "project-token",
					UsedDefaultProject: true,
				},
			}
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{EngramCollectionCreate: createService},
				[]string{"project-token"},
			)
			response := postCollectionCreateRouteRequest(t, router, tc)
			assertMCPToolsCallStatusOK(t, response)
			if createService.call.ProjectID != "project-token" {
				t.Fatalf("expected token project autofill for engram.collection_create")
			}
		})
	}
}

func TestMountMCPRoutesTokenProjectAutofillRequiresExplicitProjectForCollectionCreate(t *testing.T) {
	tests := []collectionCreateRouteScenario{
		{name: "tools/call", requestID: "collection-create-tools-denied", toolsCall: true},
		{name: "direct", requestID: "collection-create-direct-denied", toolsCall: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			router := newMCPTokenProjectFallbackRouter(
				mcp.CompatibilityServiceDependencies{EngramCollectionCreate: &capturingMCPStreamCollectionCreateService{}},
				[]string{"project-a", "project-b"},
			)
			response := postCollectionCreateRouteRequest(t, router, tc)
			assertMCPToolsCallStatusOK(t, response)
			assertMCPMissingProjectIDReasonError(
				t,
				response,
				"token_has_multiple_allowed_projects",
			)
		})
	}
}

func postCollectionCreateRouteRequest(
	t *testing.T,
	router http.Handler,
	scenario collectionCreateRouteScenario,
) *httptest.ResponseRecorder {
	t.Helper()
	params := map[string]any{
		"name":        "Collection",
		"description": "Details",
	}
	if scenario.toolsCall {
		return postMCPToolsCallRequest(t, router, mcpToolsCallRequest{
			RequestID: scenario.requestID,
			ToolName:  "engram_collection_create",
			Arguments: params,
		})
	}
	return postMCPJSONRPCRequest(t, router, mcpJSONRPCPostRequest{
		RequestID: scenario.requestID,
		Method:    "engram.collection_create",
		Params:    params,
	})
}

type capturingMCPStreamCollectionCreateService struct {
	response *mcp.EngramCollectionCreateResponse
	call     mcp.EngramCollectionCreateRequest
}

func (service *capturingMCPStreamCollectionCreateService) CreateCollection(
	_ context.Context,
	request mcp.EngramCollectionCreateRequest,
) (*mcp.EngramCollectionCreateResponse, error) {
	service.call = request
	return clonePointer(service.response), nil
}

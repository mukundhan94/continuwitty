package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/admin"
	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramCollectionAddItemsParity(t *testing.T) {
	actorUserID := uuid.MustParse("39950000-0000-0000-0000-000000000398")
	collectionID := uuid.MustParse("39950000-0000-0000-0000-000000000399")
	engramA := uuid.MustParse("39950000-0000-0000-0000-0000000003a0")
	engramB := uuid.MustParse("39950000-0000-0000-0000-0000000003a1")
	service := &fakeEngramCollectionAddItemsService{
		response: &EngramCollectionAddItemsResponse{Added: 2},
	}
	params := map[string]any{
		"collection_id": collectionID.String(),
		"engram_ids":    []any{engramA.String(), engramB.String()},
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.collection_add_items", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_collection_add_items", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramCollectionAddItemsCompatibilityService(service),
				testCase.request,
			)
			result := collectionAddItemsResultFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*service.response, result) {
				t.Fatalf("expected collection-add-items payload to match service output")
			}
			assertCollectionAddItemsCall(
				t,
				service.call,
				collectionAddItemsCallExpectation{
					actorUserID:  actorUserID,
					actorRole:    testCase.expectedRole,
					collectionID: collectionID,
					engramIDs:    []uuid.UUID{engramA, engramB},
				},
			)
		})
	}
}

func TestCompatibilityServiceEngramCollectionAddItemsUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39950000-0000-0000-0000-000000000400")
	collectionID := uuid.MustParse("39950000-0000-0000-0000-000000000401")
	service := &fakeEngramCollectionAddItemsService{response: &EngramCollectionAddItemsResponse{Added: 0}}
	frame := runCompatibilityRequestWithService(
		t,
		newEngramCollectionAddItemsCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"engram.collection_add_items",
			map[string]any{"collection_id": collectionID.String()},
		),
	)
	_ = collectionAddItemsResultFromFrame(t, frame, false)
	assertCollectionAddItemsCall(
		t,
		service.call,
		collectionAddItemsCallExpectation{
			actorUserID:  actorUserID,
			actorRole:    models.UserRoleViewer,
			collectionID: collectionID,
			engramIDs:    []uuid.UUID{},
		},
	)
}

func TestCompatibilityServiceEngramCollectionAddItemsValidationAndErrors(t *testing.T) {
	service := newEngramCollectionAddItemsCompatibilityService(&fakeEngramCollectionAddItemsService{})
	validationCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing collection id", params: map[string]any{}},
		{name: "invalid collection id", params: map[string]any{"collection_id": "bad"}},
		{name: "invalid engram ids type", params: map[string]any{"collection_id": uuid.NewString(), "engram_ids": "bad"}},
		{name: "invalid engram ids item", params: map[string]any{"collection_id": uuid.NewString(), "engram_ids": []any{"bad"}}},
	}
	for _, testCase := range validationCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39950000-0000-0000-0000-000000000402",
					"engram_collection_add_items",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	notFoundID := uuid.MustParse("39950000-0000-0000-0000-000000000403")
	notFoundCases := []struct {
		name    string
		service Service
	}{
		{name: "not found", service: service},
		{name: "not found error", service: newEngramCollectionAddItemsCompatibilityService(&fakeEngramCollectionAddItemsService{err: admin.ErrCollectionNotFound})},
	}
	for _, testCase := range notFoundCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				testCase.service,
				toolsCallRequest(
					"39950000-0000-0000-0000-000000000404",
					"engram_collection_add_items",
					map[string]any{"collection_id": notFoundID.String()},
				),
			)
			errorPayload := errorPayloadFromFrame(t, frame)
			requireErrorCode(t, errorPayload, -32004)
			assertCollectionMutationNotFoundData(t, errorPayload, notFoundID)
		})
	}

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramCollectionAddItemsCompatibilityService(
			&fakeEngramCollectionAddItemsService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39950000-0000-0000-0000-000000000405",
			"engram_collection_add_items",
			map[string]any{"collection_id": notFoundID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramCollectionAddItemsCompatibilityService(service EngramCollectionAddItemsService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramCollectionAddItems: service},
	)
}

func collectionAddItemsResultFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) EngramCollectionAddItemsResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	addResult, ok := payload["result"].(EngramCollectionAddItemsResponse)
	if !ok {
		t.Fatalf("expected collection-add-items result payload")
	}
	return addResult
}

type collectionAddItemsCallExpectation struct {
	actorUserID  uuid.UUID
	actorRole    models.UserRole
	collectionID uuid.UUID
	engramIDs    []uuid.UUID
}

func assertCollectionAddItemsCall(
	t *testing.T,
	call EngramCollectionAddItemsRequest,
	expected collectionAddItemsCallExpectation,
) {
	t.Helper()
	if call.ActorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.ActorRole != expected.actorRole {
		t.Fatalf("expected actor role forwarded")
	}
	if call.CollectionID != expected.collectionID {
		t.Fatalf("expected collection id forwarded")
	}
	if !reflect.DeepEqual(call.EngramIDs, expected.engramIDs) {
		t.Fatalf("expected engram ids forwarded")
	}
}

type fakeEngramCollectionAddItemsService struct {
	response *EngramCollectionAddItemsResponse
	err      error
	call     EngramCollectionAddItemsRequest
}

func (service *fakeEngramCollectionAddItemsService) AddCollectionItems(
	_ context.Context,
	request EngramCollectionAddItemsRequest,
) (*EngramCollectionAddItemsResponse, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.response == nil {
		return nil, nil
	}
	response := *service.response
	return &response, nil
}

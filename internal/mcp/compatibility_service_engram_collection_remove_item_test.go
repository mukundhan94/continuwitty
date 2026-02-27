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

func TestCompatibilityServiceEngramCollectionRemoveItemParity(t *testing.T) {
	actorUserID := uuid.MustParse("39960000-0000-0000-0000-000000000398")
	collectionID := uuid.MustParse("39960000-0000-0000-0000-000000000399")
	engramID := uuid.MustParse("39960000-0000-0000-0000-0000000003a0")
	service := &fakeEngramCollectionRemoveItemService{
		response: &EngramCollectionRemoveItemResponse{Removed: true},
	}
	params := map[string]any{
		"collection_id": collectionID.String(),
		"engram_id":     engramID.String(),
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.collection_remove_items", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_collection_remove_items", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramCollectionRemoveItemCompatibilityService(service),
				testCase.request,
			)
			result := collectionRemoveItemResultFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*service.response, result) {
				t.Fatalf("expected collection-remove-item payload to match service output")
			}
			assertCollectionRemoveItemCall(
				t,
				service.call,
				collectionRemoveItemCallExpectation{
					actorUserID:  actorUserID,
					actorRole:    testCase.expectedRole,
					collectionID: collectionID,
					engramID:     engramID,
				},
			)
		})
	}
}

func TestCompatibilityServiceEngramCollectionRemoveItemValidationAndErrors(t *testing.T) {
	service := newEngramCollectionRemoveItemCompatibilityService(&fakeEngramCollectionRemoveItemService{})
	validationCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing collection id", params: map[string]any{"engram_id": uuid.NewString()}},
		{name: "invalid collection id", params: map[string]any{"collection_id": "bad", "engram_id": uuid.NewString()}},
		{name: "missing engram id", params: map[string]any{"collection_id": uuid.NewString()}},
		{name: "invalid engram id", params: map[string]any{"collection_id": uuid.NewString(), "engram_id": "bad"}},
	}
	for _, testCase := range validationCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39960000-0000-0000-0000-000000000400",
					"engram_collection_remove_items",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	notFoundID := uuid.MustParse("39960000-0000-0000-0000-000000000401")
	engramID := uuid.MustParse("39960000-0000-0000-0000-000000000402")
	notFoundCases := []struct {
		name    string
		service Service
	}{
		{name: "not found", service: service},
		{name: "not found error", service: newEngramCollectionRemoveItemCompatibilityService(&fakeEngramCollectionRemoveItemService{err: admin.ErrCollectionNotFound})},
	}
	for _, testCase := range notFoundCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				testCase.service,
				toolsCallRequest(
					"39960000-0000-0000-0000-000000000403",
					"engram_collection_remove_items",
					map[string]any{
						"collection_id": notFoundID.String(),
						"engram_id":     engramID.String(),
					},
				),
			)
			errorPayload := errorPayloadFromFrame(t, frame)
			requireErrorCode(t, errorPayload, -32004)
			assertCollectionMutationNotFoundData(t, errorPayload, notFoundID)
		})
	}

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramCollectionRemoveItemCompatibilityService(
			&fakeEngramCollectionRemoveItemService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39960000-0000-0000-0000-000000000404",
			"engram_collection_remove_items",
			map[string]any{
				"collection_id": notFoundID.String(),
				"engram_id":     engramID.String(),
			},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramCollectionRemoveItemCompatibilityService(service EngramCollectionRemoveItemService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramCollectionRemove: service},
	)
}

func collectionRemoveItemResultFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) EngramCollectionRemoveItemResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	removeResult, ok := payload["result"].(EngramCollectionRemoveItemResponse)
	if !ok {
		t.Fatalf("expected collection-remove-item result payload")
	}
	return removeResult
}

type collectionRemoveItemCallExpectation struct {
	actorUserID  uuid.UUID
	actorRole    models.UserRole
	collectionID uuid.UUID
	engramID     uuid.UUID
}

func assertCollectionRemoveItemCall(
	t *testing.T,
	call EngramCollectionRemoveItemRequest,
	expected collectionRemoveItemCallExpectation,
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
	if call.EngramID != expected.engramID {
		t.Fatalf("expected engram id forwarded")
	}
}

type fakeEngramCollectionRemoveItemService struct {
	response *EngramCollectionRemoveItemResponse
	err      error
	call     EngramCollectionRemoveItemRequest
}

func (service *fakeEngramCollectionRemoveItemService) RemoveCollectionItem(
	_ context.Context,
	request EngramCollectionRemoveItemRequest,
) (*EngramCollectionRemoveItemResponse, error) {
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

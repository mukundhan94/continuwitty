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

func TestCompatibilityServiceEngramCollectionDeleteParity(t *testing.T) {
	actorUserID := uuid.MustParse("39940000-0000-0000-0000-000000000398")
	collectionID := uuid.MustParse("39940000-0000-0000-0000-000000000399")
	service := &fakeEngramCollectionDeleteService{
		response: &EngramCollectionDeleteResponse{Deleted: true},
	}
	params := map[string]any{
		"collection_id": collectionID.String(),
		"reason":        "cleanup",
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.collection_delete", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_collection_delete", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramCollectionDeleteCompatibilityService(service),
				testCase.request,
			)
			result := collectionDeleteResultFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*service.response, result) {
				t.Fatalf("expected collection delete payload to match service output")
			}
			assertCollectionDeleteCall(
				t,
				service.call,
				collectionDeleteCallExpectation{
					actorUserID:  actorUserID,
					actorRole:    testCase.expectedRole,
					collectionID: collectionID,
					reason:       stringPtr("cleanup"),
				},
			)
		})
	}
}

func TestCompatibilityServiceEngramCollectionDeleteUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39940000-0000-0000-0000-000000000400")
	collectionID := uuid.MustParse("39940000-0000-0000-0000-000000000401")
	service := &fakeEngramCollectionDeleteService{response: &EngramCollectionDeleteResponse{Deleted: true}}
	frame := runCompatibilityRequestWithService(
		t,
		newEngramCollectionDeleteCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"engram.collection_delete",
			map[string]any{"collection_id": collectionID.String()},
		),
	)
	_ = collectionDeleteResultFromFrame(t, frame, false)
	if service.call.Reason != nil {
		t.Fatalf("expected reason omitted by default")
	}
}

func TestCompatibilityServiceEngramCollectionDeleteValidationAndErrors(t *testing.T) {
	service := newEngramCollectionDeleteCompatibilityService(&fakeEngramCollectionDeleteService{})
	validationCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing collection id", params: map[string]any{}},
		{name: "invalid collection id", params: map[string]any{"collection_id": "bad"}},
		{name: "invalid reason", params: map[string]any{"collection_id": uuid.NewString(), "reason": 123}},
	}
	for _, testCase := range validationCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39940000-0000-0000-0000-000000000402",
					"engram_collection_delete",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	notFoundID := uuid.MustParse("39940000-0000-0000-0000-000000000403")
	notFoundCases := []struct {
		name    string
		service Service
	}{
		{name: "not found", service: service},
		{name: "not found error", service: newEngramCollectionDeleteCompatibilityService(&fakeEngramCollectionDeleteService{err: admin.ErrCollectionNotFound})},
	}
	for _, testCase := range notFoundCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				testCase.service,
				toolsCallRequest(
					"39940000-0000-0000-0000-000000000404",
					"engram_collection_delete",
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
		newEngramCollectionDeleteCompatibilityService(
			&fakeEngramCollectionDeleteService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39940000-0000-0000-0000-000000000405",
			"engram_collection_delete",
			map[string]any{"collection_id": notFoundID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramCollectionDeleteCompatibilityService(service EngramCollectionDeleteService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramCollectionDelete: service},
	)
}

func collectionDeleteResultFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) EngramCollectionDeleteResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	deleteResult, ok := payload["result"].(EngramCollectionDeleteResponse)
	if !ok {
		t.Fatalf("expected collection-delete result payload")
	}
	return deleteResult
}

type collectionDeleteCallExpectation struct {
	actorUserID  uuid.UUID
	actorRole    models.UserRole
	collectionID uuid.UUID
	reason       *string
}

func assertCollectionDeleteCall(
	t *testing.T,
	call EngramCollectionDeleteRequest,
	expected collectionDeleteCallExpectation,
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
	assertOptionalDeleteReason(t, call.Reason, expected.reason)
}

type fakeEngramCollectionDeleteService struct {
	response *EngramCollectionDeleteResponse
	err      error
	call     EngramCollectionDeleteRequest
}

func (service *fakeEngramCollectionDeleteService) DeleteCollection(
	_ context.Context,
	request EngramCollectionDeleteRequest,
) (*EngramCollectionDeleteResponse, error) {
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

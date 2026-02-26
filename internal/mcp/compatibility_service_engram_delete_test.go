package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramDeleteParity(t *testing.T) {
	actorUserID := uuid.MustParse("39810000-0000-0000-0000-000000000398")
	engramID := uuid.MustParse("39810000-0000-0000-0000-000000000399")
	deleteService := &fakeEngramDeleteService{
		response: &EngramDeleteResponse{
			EngramID: engramID,
			Deleted:  true,
		},
	}
	params := map[string]any{
		"engram_id": engramID.String(),
		"reason":    "cleanup",
	}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.delete", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_delete", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramDeleteCompatibilityService(deleteService),
				testCase.request,
			)
			result := engramDeleteResultFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*deleteService.response, result) {
				t.Fatalf("expected delete payload to match service output")
			}
			assertEngramDeleteCall(
				t,
				deleteService.call,
				engramDeleteCallExpectation{
					actorUserID: actorUserID,
					actorRole:   testCase.expectedRole,
					engramID:    engramID,
					reason:      stringPtr("cleanup"),
				},
			)
		})
	}
}

func TestCompatibilityServiceEngramDeleteUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39820000-0000-0000-0000-000000000398")
	engramID := uuid.MustParse("39820000-0000-0000-0000-000000000399")
	deleteService := &fakeEngramDeleteService{
		response: &EngramDeleteResponse{
			EngramID: engramID,
			Deleted:  true,
		},
	}
	frame := runCompatibilityRequestWithService(
		t,
		newEngramDeleteCompatibilityService(deleteService),
		directToolRequest(
			actorUserID.String(),
			"engram.delete",
			map[string]any{"engram_id": engramID.String()},
		),
	)
	_ = engramDeleteResultFromFrame(t, frame, false)
	if deleteService.call.Reason != nil {
		t.Fatalf("expected reason to remain nil when omitted")
	}
}

func TestCompatibilityServiceEngramDeleteValidationAndErrors(t *testing.T) {
	service := newEngramDeleteCompatibilityService(&fakeEngramDeleteService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing engram id", params: map[string]any{}},
		{name: "invalid engram id", params: map[string]any{"engram_id": "bad"}},
		{name: "invalid reason", params: map[string]any{"engram_id": uuid.NewString(), "reason": 123}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39830000-0000-0000-0000-000000000398",
					"engram_delete",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	notFoundID := uuid.MustParse("39830000-0000-0000-0000-000000000399")
	notFoundFrame := runCompatibilityRequestWithService(
		t,
		service,
		toolsCallRequest(
			"39830000-0000-0000-0000-000000000400",
			"engram_delete",
			map[string]any{"engram_id": notFoundID.String()},
		),
	)
	notFoundPayload := errorPayloadFromFrame(t, notFoundFrame)
	requireErrorCode(t, notFoundPayload, -32004)
	assertEngramMutationNotFoundData(t, notFoundPayload, notFoundID)

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramDeleteCompatibilityService(
			&fakeEngramDeleteService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39830000-0000-0000-0000-000000000401",
			"engram_delete",
			map[string]any{"engram_id": notFoundID.String()},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramDeleteCompatibilityService(service EngramDeleteService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramDelete: service},
	)
}

func engramDeleteResultFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) EngramDeleteResponse {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	deleteResult, ok := payload["result"].(EngramDeleteResponse)
	if !ok {
		t.Fatalf("expected engram-delete result payload")
	}
	return deleteResult
}

func assertEngramMutationNotFoundData(
	t *testing.T,
	errorPayload map[string]any,
	engramID uuid.UUID,
) {
	t.Helper()
	data := mapFromMap(t, errorPayload, "data")
	if data["engram_id"] != engramID.String() {
		t.Fatalf("expected engram_id in not-found error data")
	}
}

type engramDeleteCallExpectation struct {
	actorUserID uuid.UUID
	actorRole   models.UserRole
	engramID    uuid.UUID
	reason      *string
}

func assertEngramDeleteCall(
	t *testing.T,
	call EngramDeleteRequest,
	expected engramDeleteCallExpectation,
) {
	t.Helper()
	if call.ActorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.ActorRole != expected.actorRole {
		t.Fatalf("expected normalized actor role forwarded")
	}
	if call.EngramID != expected.engramID {
		t.Fatalf("expected engram id forwarded")
	}
	assertOptionalDeleteReason(t, call.Reason, expected.reason)
}

type fakeEngramDeleteService struct {
	response *EngramDeleteResponse
	err      error
	call     EngramDeleteRequest
}

func (service *fakeEngramDeleteService) DeleteEngram(
	_ context.Context,
	request EngramDeleteRequest,
) (*EngramDeleteResponse, error) {
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

package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/admin"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramCollectionUpdateParity(t *testing.T) {
	actorUserID := uuid.MustParse("39930000-0000-0000-0000-000000000398")
	collectionID := uuid.MustParse("39930000-0000-0000-0000-000000000399")
	expectedUpdatedAt := mustParseCollectionRFC3339(t, "2026-02-21T00:00:00Z")
	updated := &models.EngramCollectionRecord{
		CollectionID: collectionID,
		ProjectID:    "project-target",
		OwnerUserID:  actorUserID,
		Name:         "Ops Updated",
		Description:  "Updated",
	}
	service := &fakeEngramCollectionUpdateService{updated: updated}
	params := map[string]any{
		"collection_id":       collectionID.String(),
		"name":                "Ops Updated",
		"description":         "Updated",
		"expected_updated_at": expectedUpdatedAt.Format(time.RFC3339),
	}
	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
		expectedRole    models.UserRole
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "engram.collection_update", params),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "engram_collection_update", params),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newEngramCollectionUpdateCompatibilityService(service),
				testCase.request,
			)
			collection := collectionUpdateResponseFromFrame(t, frame, testCase.asToolsCallPath)
			if !reflect.DeepEqual(*updated, collection) {
				t.Fatalf("expected collection update payload to match service output")
			}
			assertCollectionUpdateCall(
				t,
				service.call,
				collectionUpdateCallExpectation{
					actorUserID:       actorUserID,
					actorRole:         testCase.expectedRole,
					collectionID:      collectionID,
					name:              stringPtr("Ops Updated"),
					description:       stringPtr("Updated"),
					expectedUpdatedAt: &expectedUpdatedAt,
				},
			)
		})
	}
}

func TestCompatibilityServiceEngramCollectionUpdateUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39930000-0000-0000-0000-000000000400")
	collectionID := uuid.MustParse("39930000-0000-0000-0000-000000000401")
	service := &fakeEngramCollectionUpdateService{updated: &models.EngramCollectionRecord{CollectionID: collectionID}}
	frame := runCompatibilityRequestWithService(
		t,
		newEngramCollectionUpdateCompatibilityService(service),
		directToolRequest(
			actorUserID.String(),
			"engram.collection_update",
			map[string]any{"collection_id": collectionID.String()},
		),
	)
	_ = collectionUpdateResponseFromFrame(t, frame, false)
	assertCollectionUpdateCall(
		t,
		service.call,
		collectionUpdateCallExpectation{
			actorUserID:       actorUserID,
			actorRole:         models.UserRoleViewer,
			collectionID:      collectionID,
			name:              nil,
			description:       nil,
			expectedUpdatedAt: nil,
		},
	)
}

func TestCompatibilityServiceEngramCollectionUpdateValidationAndErrors(t *testing.T) {
	service := newEngramCollectionUpdateCompatibilityService(&fakeEngramCollectionUpdateService{})
	validationCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "missing collection id", params: map[string]any{}},
		{name: "invalid collection id", params: map[string]any{"collection_id": "bad"}},
		{name: "invalid name", params: map[string]any{"collection_id": uuid.NewString(), "name": 123}},
		{name: "invalid description", params: map[string]any{"collection_id": uuid.NewString(), "description": 123}},
		{name: "invalid expected updated at", params: map[string]any{"collection_id": uuid.NewString(), "expected_updated_at": "bad"}},
	}
	for _, testCase := range validationCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39930000-0000-0000-0000-000000000402",
					"engram_collection_update",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	notFoundID := uuid.MustParse("39930000-0000-0000-0000-000000000403")
	errorCases := []struct {
		name           string
		service        Service
		expectedCode   int
		expectedStatus *int
		expectedDetail *string
		expectNotFound bool
	}{
		{name: "not found", service: service, expectedCode: -32004, expectNotFound: true},
		{name: "not found error", service: newEngramCollectionUpdateCompatibilityService(&fakeEngramCollectionUpdateService{err: admin.ErrCollectionNotFound}), expectedCode: -32004, expectNotFound: true},
		{name: "stale", service: newEngramCollectionUpdateCompatibilityService(&fakeEngramCollectionUpdateService{err: admin.ErrCollectionStale}), expectedCode: -32602, expectedStatus: collectionStatusPtr(409), expectedDetail: stringPtr(admin.ErrCollectionStale.Error())},
		{name: "duplicate", service: newEngramCollectionUpdateCompatibilityService(&fakeEngramCollectionUpdateService{err: repository.ErrCollectionNameExists}), expectedCode: -32602, expectedStatus: collectionStatusPtr(409), expectedDetail: stringPtr(repository.ErrCollectionNameExists.Error())},
		{name: "internal", service: newEngramCollectionUpdateCompatibilityService(&fakeEngramCollectionUpdateService{err: errors.New("boom")}), expectedCode: -32603},
	}
	for _, testCase := range errorCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				testCase.service,
				toolsCallRequest(
					"39930000-0000-0000-0000-000000000404",
					"engram_collection_update",
					map[string]any{"collection_id": notFoundID.String()},
				),
			)
			errorPayload := errorPayloadFromFrame(t, frame)
			requireErrorCode(t, errorPayload, testCase.expectedCode)
			if testCase.expectNotFound {
				assertCollectionMutationNotFoundData(t, errorPayload, notFoundID)
			}
			if testCase.expectedStatus != nil {
				assertErrorStatusCode(t, errorPayload, *testCase.expectedStatus)
			}
			if testCase.expectedDetail != nil {
				assertErrorDetail(t, errorPayload, *testCase.expectedDetail)
			}
		})
	}
}

func newEngramCollectionUpdateCompatibilityService(service EngramCollectionUpdateService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramCollectionUpdate: service},
	)
}

func collectionUpdateResponseFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) models.EngramCollectionRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	collection, ok := payload["collection"].(models.EngramCollectionRecord)
	if !ok {
		t.Fatalf("expected collection update payload")
	}
	return collection
}

type collectionUpdateCallExpectation struct {
	actorUserID       uuid.UUID
	actorRole         models.UserRole
	collectionID      uuid.UUID
	expectedUpdatedAt *time.Time
	name              *string
	description       *string
}

func assertCollectionUpdateCall(
	t *testing.T,
	call EngramCollectionUpdateRequest,
	expected collectionUpdateCallExpectation,
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
	assertOptionalField(
		t,
		"expected_updated_at",
		call.ExpectedUpdatedAt,
		expected.expectedUpdatedAt,
		func(actual time.Time, expected time.Time) bool { return actual.Equal(expected) },
	)
	assertOptionalDeleteReason(t, call.Name, expected.name)
	assertOptionalDeleteReason(t, call.Description, expected.description)
}

type fakeEngramCollectionUpdateService struct {
	updated *models.EngramCollectionRecord
	err     error
	call    EngramCollectionUpdateRequest
}

func (service *fakeEngramCollectionUpdateService) UpdateCollection(
	_ context.Context,
	request EngramCollectionUpdateRequest,
) (*models.EngramCollectionRecord, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	if service.updated == nil {
		return nil, nil
	}
	updated := *service.updated
	return &updated, nil
}

func mustParseCollectionRFC3339(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("expected RFC3339 time in test: %v", err)
	}
	return parsed
}

func assertCollectionMutationNotFoundData(
	t *testing.T,
	errorPayload map[string]any,
	collectionID uuid.UUID,
) {
	t.Helper()
	data := mapFromMap(t, errorPayload, "data")
	if data["collection_id"] != collectionID.String() {
		t.Fatalf("expected collection_id in not-found error data")
	}
}

func collectionStatusPtr(value int) *int {
	return &value
}

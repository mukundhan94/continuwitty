package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceEngramCollectionListParity(t *testing.T) {
	actorUserID := uuid.MustParse("39400000-0000-0000-0000-000000000394")
	projectID := "proj-alpha"
	service := newCollectionListParityService(actorUserID, projectID)
	for _, testCase := range newCollectionListParityCases(actorUserID, projectID) {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			assertCollectionListParityCase(
				t,
				collectionListParityInput{
					service:     newEngramCollectionListCompatibilityService(service),
					listService: service,
					testCase:    testCase,
					expectedCall: collectionListExpectation{
						actorUserID:    actorUserID,
						actorRole:      testCase.expectedRole,
						projectID:      &projectID,
						includeDeleted: true,
						limit:          10,
						offset:         1,
					},
				},
			)
		})
	}
}

type collectionListParityCase struct {
	name            string
	request         StreamCallRequest
	asToolsCallPath bool
	expectedRole    models.UserRole
}

func newCollectionListParityService(
	actorUserID uuid.UUID,
	projectID string,
) *fakeEngramCollectionListService {
	return &fakeEngramCollectionListService{
		collections: []models.EngramCollectionRecord{
			{
				CollectionID: uuid.MustParse("39400000-0000-0000-0000-000000000395"),
				ProjectID:    projectID,
				OwnerUserID:  actorUserID,
				Name:         "Ops",
				Description:  "Ops notes",
				CreatedAt:    time.Unix(1700003940, 0).UTC(),
				UpdatedAt:    time.Unix(1700003940, 0).UTC(),
			},
		},
	}
}

func newCollectionListParityCases(
	actorUserID uuid.UUID,
	projectID string,
) []collectionListParityCase {
	return []collectionListParityCase{
		{
			name: "direct",
			request: directToolRequest(
				actorUserID.String(),
				"engram.collection_list",
				map[string]any{
					"project_id":      projectID,
					"include_deleted": true,
					"limit":           10.0,
					"offset":          1.0,
				},
			),
			asToolsCallPath: false,
			expectedRole:    models.UserRoleViewer,
		},
		{
			name: "tools call",
			request: toolsCallRequest(
				actorUserID.String(),
				"engram_collection_list",
				map[string]any{
					"project_id":      projectID,
					"include_deleted": true,
					"limit":           10.0,
					"offset":          1.0,
				},
			),
			asToolsCallPath: true,
			expectedRole:    models.UserRoleAnalyst,
		},
	}
}

type collectionListParityInput struct {
	service      Service
	listService  *fakeEngramCollectionListService
	testCase     collectionListParityCase
	expectedCall collectionListExpectation
}

func assertCollectionListParityCase(t *testing.T, input collectionListParityInput) {
	t.Helper()
	frame := runCompatibilityRequestWithService(t, input.service, input.testCase.request)
	collections := collectionsFromFrame(t, frame, input.testCase.asToolsCallPath)
	if !reflect.DeepEqual(input.listService.collections, collections) {
		t.Fatalf("expected collections payload to match service output")
	}
	assertCollectionListCall(t, input.listService.call, input.expectedCall)
}

func TestCompatibilityServiceEngramCollectionListUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39410000-0000-0000-0000-000000000394")
	service := &fakeEngramCollectionListService{collections: []models.EngramCollectionRecord{}}
	frame := runCompatibilityRequestWithService(
		t,
		newEngramCollectionListCompatibilityService(service),
		directToolRequest(actorUserID.String(), "engram.collection_list", map[string]any{}),
	)
	_ = collectionsFromFrame(t, frame, false)
	assertCollectionListCall(
		t,
		service.call,
		collectionListExpectation{
			actorUserID:    actorUserID,
			actorRole:      models.UserRoleViewer,
			projectID:      nil,
			includeDeleted: false,
			limit:          defaultCollectionListLimit,
			offset:         defaultCollectionListOffset,
		},
	)
}

func TestCompatibilityServiceEngramCollectionListValidationAndErrors(t *testing.T) {
	service := newEngramCollectionListCompatibilityService(&fakeEngramCollectionListService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "invalid include_deleted", params: map[string]any{"include_deleted": "bad"}},
		{name: "invalid limit", params: map[string]any{"limit": "bad"}},
		{name: "invalid offset", params: map[string]any{"offset": "bad"}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39420000-0000-0000-0000-000000000394",
					"engram_collection_list",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}

	internalFrame := runCompatibilityRequestWithService(
		t,
		newEngramCollectionListCompatibilityService(
			&fakeEngramCollectionListService{err: errors.New("boom")},
		),
		toolsCallRequest(
			"39420000-0000-0000-0000-000000000395",
			"engram_collection_list",
			map[string]any{"limit": 10.0},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, internalFrame), -32603)
}

func newEngramCollectionListCompatibilityService(service EngramCollectionListService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramCollectionList: service},
	)
}

func collectionsFromFrame(
	t *testing.T,
	frame Frame,
	asToolsCallPath bool,
) []models.EngramCollectionRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	collections, ok := payload["collections"].([]models.EngramCollectionRecord)
	if !ok {
		t.Fatalf("expected collections payload")
	}
	return collections
}

type collectionListExpectation struct {
	actorUserID    uuid.UUID
	actorRole      models.UserRole
	projectID      *string
	includeDeleted bool
	limit          int
	offset         int
}

func assertCollectionListCall(
	t *testing.T,
	call EngramCollectionListRequest,
	expected collectionListExpectation,
) {
	t.Helper()
	if call.ActorUserID != expected.actorUserID {
		t.Fatalf("expected actor user id forwarded")
	}
	if call.ActorRole != expected.actorRole {
		t.Fatalf("expected actor role forwarded")
	}
	if !reflect.DeepEqual(call.ProjectID, expected.projectID) {
		t.Fatalf("expected project id forwarded")
	}
	if call.IncludeDeleted != expected.includeDeleted {
		t.Fatalf("expected include_deleted forwarded")
	}
	if call.Limit != expected.limit {
		t.Fatalf("expected limit forwarded")
	}
	if call.Offset != expected.offset {
		t.Fatalf("expected offset forwarded")
	}
}

type fakeEngramCollectionListService struct {
	collections []models.EngramCollectionRecord
	err         error
	call        EngramCollectionListRequest
}

func (service *fakeEngramCollectionListService) ListCollections(
	_ context.Context,
	request EngramCollectionListRequest,
) ([]models.EngramCollectionRecord, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	return append([]models.EngramCollectionRecord(nil), service.collections...), nil
}

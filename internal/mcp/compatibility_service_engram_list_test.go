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

func TestCompatibilityServiceEngramListParity(t *testing.T) {
	actorUserID := uuid.MustParse("39200000-0000-0000-0000-000000000392")
	sessionID := uuid.MustParse("39200000-0000-0000-0000-000000000393")
	projectID := "proj-alpha"
	queryText := "risk"
	params := engramListParityParams(projectID, sessionID, queryText)
	service := newFakeEngramListService(projectID)

	runEngramListParityCase(
		t,
		service,
		"direct",
		directToolRequest(actorUserID.String(), "engram.list", params),
		false,
		EngramListRequest{
			ActorUserID:    actorUserID,
			ActorRole:      models.UserRoleViewer,
			ProjectID:      &projectID,
			SessionID:      &sessionID,
			QueryText:      &queryText,
			IncludeDeleted: true,
			Limit:          25,
			Offset:         2,
		},
	)
	runEngramListParityCase(
		t,
		service,
		"tools call",
		toolsCallRequest(actorUserID.String(), "engram_list", params),
		true,
		EngramListRequest{
			ActorUserID:    actorUserID,
			ActorRole:      models.UserRoleAnalyst,
			ProjectID:      &projectID,
			SessionID:      &sessionID,
			QueryText:      &queryText,
			IncludeDeleted: true,
			Limit:          25,
			Offset:         2,
		},
	)
}

func TestCompatibilityServiceEngramListUsesDefaults(t *testing.T) {
	actorUserID := uuid.MustParse("39210000-0000-0000-0000-000000000392")
	service := &fakeEngramListService{engrams: []models.AdminEngramRecord{}}
	frame := runCompatibilityRequestWithService(
		t,
		newEngramListCompatibilityService(service),
		directToolRequest(actorUserID.String(), "engram.list", map[string]any{}),
	)
	_ = engramsFromFrame(t, frame, false)
	assertEngramListCall(
		t,
		service.call,
		EngramListRequest{
			ActorUserID:    actorUserID,
			ActorRole:      models.UserRoleViewer,
			IncludeDeleted: false,
			Limit:          defaultEngramListLimit,
			Offset:         defaultEngramListOffset,
		},
	)
}

func TestCompatibilityServiceEngramListValidationErrors(t *testing.T) {
	service := newEngramListCompatibilityService(&fakeEngramListService{})
	testCases := []struct {
		name   string
		params map[string]any
	}{
		{name: "invalid include_deleted", params: map[string]any{"include_deleted": "nope"}},
		{name: "invalid limit", params: map[string]any{"limit": "bad"}},
		{name: "invalid offset", params: map[string]any{"offset": "bad"}},
		{name: "invalid session id", params: map[string]any{"session_id": "bad"}},
		{name: "invalid q", params: map[string]any{"q": 1}},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				service,
				toolsCallRequest(
					"39220000-0000-0000-0000-000000000392",
					"engram_list",
					testCase.params,
				),
			)
			requireErrorCode(t, errorPayloadFromFrame(t, frame), -32602)
		})
	}
}

func TestCompatibilityServiceEngramListServiceError(t *testing.T) {
	frame := runCompatibilityRequestWithService(
		t,
		newEngramListCompatibilityService(&fakeEngramListService{err: errors.New("boom")}),
		toolsCallRequest(
			"39230000-0000-0000-0000-000000000392",
			"engram_list",
			map[string]any{"limit": 10.0},
		),
	)
	requireErrorCode(t, errorPayloadFromFrame(t, frame), -32603)
}

func newEngramListCompatibilityService(service EngramListService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{EngramList: service},
	)
}

func engramsFromFrame(t *testing.T, frame Frame, asToolsCallPath bool) []models.AdminEngramRecord {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	engrams, ok := payload["engrams"].([]models.AdminEngramRecord)
	if !ok {
		t.Fatalf("expected engrams payload")
	}
	return engrams
}

func engramListParityParams(
	projectID string,
	sessionID uuid.UUID,
	queryText string,
) map[string]any {
	return map[string]any{
		"project_id":      projectID,
		"session_id":      sessionID.String(),
		"q":               queryText,
		"include_deleted": true,
		"limit":           25.0,
		"offset":          2.0,
	}
}

func newFakeEngramListService(projectID string) *fakeEngramListService {
	return &fakeEngramListService{
		engrams: []models.AdminEngramRecord{
			{
				EngramID:                uuid.MustParse("39200000-0000-0000-0000-000000000394"),
				ProjectID:               projectID,
				Title:                   "Ops Snapshot",
				Abstract:                "Summary",
				DetailedSummaryMarkdown: "Details",
				VisibilityScope:         models.VisibilityScopeProject,
				CreatedAt:               time.Unix(1700003920, 0).UTC(),
				UpdatedAt:               time.Unix(1700003920, 0).UTC(),
			},
		},
	}
}

func runEngramListParityCase(
	t *testing.T,
	service *fakeEngramListService,
	name string,
	request StreamCallRequest,
	asToolsCallPath bool,
	expectedCall EngramListRequest,
) {
	t.Run(name, func(t *testing.T) {
		frame := runCompatibilityRequestWithService(
			t,
			newEngramListCompatibilityService(service),
			request,
		)
		engrams := engramsFromFrame(t, frame, asToolsCallPath)
		if !reflect.DeepEqual(service.engrams, engrams) {
			t.Fatalf("expected engrams payload to match service output")
		}
		assertEngramListCall(t, service.call, expectedCall)
	})
}

func assertEngramListCall(t *testing.T, call EngramListRequest, expected EngramListRequest) {
	t.Helper()
	if !reflect.DeepEqual(expected, call) {
		t.Fatalf("expected call %+v, got %+v", expected, call)
	}
}

type fakeEngramListService struct {
	engrams []models.AdminEngramRecord
	err     error
	call    EngramListRequest
}

func (service *fakeEngramListService) ListEngrams(
	_ context.Context,
	request EngramListRequest,
) ([]models.AdminEngramRecord, error) {
	service.call = request
	if service.err != nil {
		return nil, service.err
	}
	return append([]models.AdminEngramRecord(nil), service.engrams...), nil
}

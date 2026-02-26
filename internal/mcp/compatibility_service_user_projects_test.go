package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestCompatibilityServiceUserListProjectsParity(t *testing.T) {
	actorUserID := uuid.MustParse("25000000-0000-0000-0000-000000000250")
	sessionService := &fakeSessionListService{
		sessions: []models.ChatSessionRecord{
			{ProjectID: "proj-b"},
			{ProjectID: "proj-a"},
			{ProjectID: "proj-a"},
			{ProjectID: "   "},
			{ProjectID: "proj-c"},
		},
	}

	testCases := []struct {
		name            string
		request         StreamCallRequest
		asToolsCallPath bool
	}{
		{
			name:            "direct",
			request:         directToolRequest(actorUserID.String(), "user.list_projects", nil),
			asToolsCallPath: false,
		},
		{
			name:            "tools call",
			request:         toolsCallRequest(actorUserID.String(), "user_list_projects", nil),
			asToolsCallPath: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			frame := runCompatibilityRequestWithService(
				t,
				newUserProjectsCompatibilityService(sessionService),
				testCase.request,
			)
			projectIDs := projectIDsFromFrame(t, frame, testCase.asToolsCallPath)
			expectedProjectIDs := []string{"proj-a", "proj-b", "proj-c"}
			if !reflect.DeepEqual(expectedProjectIDs, projectIDs) {
				t.Fatalf("expected sorted unique project_ids %v, got %v", expectedProjectIDs, projectIDs)
			}
			if sessionService.call.actorUserID != actorUserID {
				t.Fatalf("expected actor user id forwarded")
			}
			if sessionService.call.limit != defaultUserProjectsLimit || sessionService.call.offset != defaultUserProjectsOffset {
				t.Fatalf("expected default user project list window forwarded")
			}
		})
	}
}

func TestCompatibilityServiceUserListProjectsMapsSessionListError(t *testing.T) {
	frame := runCompatibilityRequestWithService(
		t,
		newUserProjectsCompatibilityService(&fakeSessionListService{err: errors.New("boom")}),
		toolsCallRequest("25100000-0000-0000-0000-000000000251", "user_list_projects", nil),
	)
	errorPayload := errorPayloadFromFrame(t, frame)
	requireErrorCode(t, errorPayload, -32603)
}

func newUserProjectsCompatibilityService(sessionService SessionListService) Service {
	return NewCompatibilityServiceWithDependencies(
		"1.2.3",
		CompatibilityServiceDependencies{SessionService: sessionService},
	)
}

func projectIDsFromFrame(t *testing.T, frame Frame, asToolsCallPath bool) []string {
	t.Helper()
	result := resultPayloadFromFrame(t, frame)
	payload := result
	if asToolsCallPath {
		payload = mapFromMap(t, result, "structuredContent")
	}
	rawProjectIDs, ok := payload["project_ids"]
	if !ok {
		t.Fatalf("expected project_ids payload")
	}
	return toStringSlice(t, rawProjectIDs)
}

func toStringSlice(t *testing.T, value any) []string {
	t.Helper()
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				t.Fatalf("expected string item in slice, got %T", item)
			}
			values = append(values, text)
		}
		return values
	default:
		t.Fatalf("expected []string payload, got %T", value)
		return nil
	}
}

type sessionListCall struct {
	actorUserID uuid.UUID
	limit       int
	offset      int
}

type fakeSessionListService struct {
	sessions []models.ChatSessionRecord
	err      error
	call     sessionListCall
}

func (service *fakeSessionListService) ListSessions(
	_ context.Context,
	actorUserID uuid.UUID,
	limit int,
	offset int,
) ([]models.ChatSessionRecord, error) {
	service.call = sessionListCall{actorUserID: actorUserID, limit: limit, offset: offset}
	if service.err != nil {
		return nil, service.err
	}
	return service.sessions, nil
}

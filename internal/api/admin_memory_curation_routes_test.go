package api

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"engram/internal/admin"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestMountMemoryAdminRoutesListMemoryCurationSuggestionsUsesRequestObject(t *testing.T) {
	captured := admin.MemoryCurationSuggestionListRequest{}
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000191")
	suggestionType := models.MemoryCurationSuggestionTypeLink
	status := models.MemoryCurationSuggestionStatusSuggested
	service := &fakeMemoryAdminService{
		listCurationFn: func(
			_ context.Context,
			request admin.MemoryCurationSuggestionListRequest,
		) ([]models.MemoryCurationSuggestion, error) {
			captured = request
			return []models.MemoryCurationSuggestion{}, nil
		},
	}
	router := curationAdminRouter(service, uuid.MustParse("00000000-0000-0000-0000-000000000192"))

	response := executeRequest(
		router,
		http.MethodGet,
		"/api/v1/admin/memory/engrams/curation/suggestions?project_id=engram-vault&session_id="+
			sessionID.String()+"&suggestion_type=link&status=suggested&limit=25&offset=7",
		nil,
	)
	requireEqual(t, http.StatusOK, response.Code)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	requireEqual(t, sessionID, derefUUID(captured.SessionID))
	requireEqual(t, suggestionType, *captured.SuggestionType)
	requireEqual(t, status, *captured.Status)
	requireEqual(t, 25, captured.Limit)
	requireEqual(t, 7, captured.Offset)
}

func TestMountMemoryAdminRoutesActionMemoryCurationSuggestionUsesActorAndPayload(t *testing.T) {
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-000000000193")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000194")
	capturedSuggestionID := uuid.Nil
	capturedActorUserID := uuid.Nil
	capturedRequest := admin.MemoryCurationSuggestionActionRequest{}
	service := &fakeMemoryAdminService{
		actionCurationFn: func(
			_ context.Context,
			receivedSuggestionID uuid.UUID,
			receivedActorUserID uuid.UUID,
			request admin.MemoryCurationSuggestionActionRequest,
		) (*models.MemoryCurationSuggestion, error) {
			capturedSuggestionID = receivedSuggestionID
			capturedActorUserID = receivedActorUserID
			capturedRequest = request
			return &models.MemoryCurationSuggestion{
				SuggestionID: suggestionID,
				Status:       models.MemoryCurationSuggestionStatusAccepted,
			}, nil
		},
	}
	router := curationAdminRouter(service, actorUserID)

	response := executeRequest(
		router,
		http.MethodPost,
		"/api/v1/admin/memory/engrams/curation/suggestions/"+suggestionID.String()+"/action",
		[]byte(`{"project_id":"engram-vault","status":"accepted"}`),
	)
	requireEqual(t, http.StatusOK, response.Code)
	requireEqual(t, suggestionID, capturedSuggestionID)
	requireEqual(t, actorUserID, capturedActorUserID)
	requireEqual(t, "engram-vault", derefString(capturedRequest.ProjectID))
	requireEqual(t, models.MemoryCurationSuggestionStatusAccepted, capturedRequest.Status)
}

func TestMountMemoryAdminRoutesCurationRoutesRejectInvalidValues(t *testing.T) {
	service := &fakeMemoryAdminService{
		listCurationFn: func(
			_ context.Context,
			_ admin.MemoryCurationSuggestionListRequest,
		) ([]models.MemoryCurationSuggestion, error) {
			return nil, errors.New("unexpected")
		},
		actionCurationFn: func(
			_ context.Context,
			_ uuid.UUID,
			_ uuid.UUID,
			_ admin.MemoryCurationSuggestionActionRequest,
		) (*models.MemoryCurationSuggestion, error) {
			return nil, errors.New("unexpected")
		},
	}
	router := curationAdminRouter(service, uuid.MustParse("00000000-0000-0000-0000-000000000195"))
	testCases := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{
			name:   "list invalid suggestion_type",
			method: http.MethodGet,
			path:   "/api/v1/admin/memory/engrams/curation/suggestions?suggestion_type=invalid",
		},
		{
			name:   "list invalid status",
			method: http.MethodGet,
			path:   "/api/v1/admin/memory/engrams/curation/suggestions?status=invalid",
		},
		{
			name:   "action invalid status",
			method: http.MethodPost,
			path:   "/api/v1/admin/memory/engrams/curation/suggestions/00000000-0000-0000-0000-000000000196/action",
			body:   []byte(`{"status":"suggested"}`),
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			response := executeRequest(router, testCase.method, testCase.path, testCase.body)
			requireEqual(t, http.StatusBadRequest, response.Code)
		})
	}
}

func curationAdminRouter(service MemoryAdminService, actorUserID uuid.UUID) http.Handler {
	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		return AdminActor{UserID: actorUserID, Role: "admin"}, nil
	})
	return router
}

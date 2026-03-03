package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"engram/internal/admin"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type fakeMemoryAdminService struct {
	listSessionsFn         func(ctx context.Context, request admin.MemoryAdminListRequest) ([]models.AdminChatSessionRecord, error)
	deleteSessionFn        func(ctx context.Context, sessionID, actorUserID uuid.UUID, payload admin.SessionDeleteRequest) (admin.SessionDeleteResponse, error)
	restoreSessionFn       func(ctx context.Context, sessionID uuid.UUID) (admin.SessionRestoreResponse, error)
	listEngramsFn          func(ctx context.Context, request admin.MemoryAdminEngramListRequest) ([]models.AdminEngramRecord, error)
	getEngramFn            func(ctx context.Context, engramID uuid.UUID, includeDeleted bool) (*models.AdminEngramRecord, error)
	updateEngramFn         func(ctx context.Context, engramID, actorUserID uuid.UUID, payload admin.EngramUpdateRequest) (*models.AdminEngramRecord, error)
	moveEngramFn           func(ctx context.Context, engramID uuid.UUID, actor admin.WriteActor, payload admin.EngramMoveRequest) (*models.AdminEngramRecord, error)
	deleteEngramFn         func(ctx context.Context, engramID, actorUserID uuid.UUID, payload admin.EngramDeleteRequest) (admin.EngramDeleteResponse, error)
	restoreEngramFn        func(ctx context.Context, engramID uuid.UUID) (admin.EngramRestoreResponse, error)
	refreshEngramFreshness func(ctx context.Context, request admin.EngramFreshnessRefreshRequest) (admin.EngramFreshnessRefreshResponse, error)
	refreshConsolidationFn func(ctx context.Context, request admin.EngramConsolidationSuggestionRefreshRequest) (admin.EngramConsolidationSuggestionRefreshResponse, error)
	listConsolidationFn    func(ctx context.Context, request admin.EngramConsolidationSuggestionListRequest) ([]models.EngramConsolidationSuggestion, error)
	actionConsolidationFn  func(ctx context.Context, suggestionID uuid.UUID, actorUserID uuid.UUID, request admin.EngramConsolidationSuggestionActionRequest) (*models.EngramConsolidationSuggestion, error)
	listCollectionsFn      func(ctx context.Context, request admin.MemoryAdminListRequest) ([]models.EngramCollectionRecord, error)
	createCollectionFn     func(ctx context.Context, actorUserID uuid.UUID, actorRole string, payload admin.CollectionCreateRequest) (*models.EngramCollectionRecord, error)
	updateCollectionFn     func(ctx context.Context, collectionID uuid.UUID, payload admin.CollectionUpdateRequest) (*models.EngramCollectionRecord, error)
	deleteCollectionFn     func(ctx context.Context, collectionID, actorUserID uuid.UUID, payload admin.CollectionDeleteRequest) (admin.CollectionDeleteResponse, error)
	addCollectionItemsFn   func(ctx context.Context, collectionID, actorUserID uuid.UUID, payload admin.CollectionItemsUpdateRequest) (admin.CollectionItemsAddResponse, error)
	removeCollectionItemFn func(ctx context.Context, collectionID, engramID uuid.UUID) (admin.CollectionItemRemoveResponse, error)
}

func requireFakeAdminHandler[T any](name string, handler T) T {
	value := reflect.ValueOf(handler)
	if !value.IsValid() || value.IsNil() {
		panic("unexpected " + name + " call")
	}
	return handler
}

func (f *fakeMemoryAdminService) ListSessions(ctx context.Context, request admin.MemoryAdminListRequest) ([]models.AdminChatSessionRecord, error) {
	return requireFakeAdminHandler("ListSessions", f.listSessionsFn)(ctx, request)
}

func (f *fakeMemoryAdminService) DeleteSession(ctx context.Context, sessionID, actorUserID uuid.UUID, payload admin.SessionDeleteRequest) (admin.SessionDeleteResponse, error) {
	return requireFakeAdminHandler("DeleteSession", f.deleteSessionFn)(ctx, sessionID, actorUserID, payload)
}

func (f *fakeMemoryAdminService) RestoreSession(ctx context.Context, sessionID uuid.UUID) (admin.SessionRestoreResponse, error) {
	return requireFakeAdminHandler("RestoreSession", f.restoreSessionFn)(ctx, sessionID)
}

func (f *fakeMemoryAdminService) ListEngrams(ctx context.Context, request admin.MemoryAdminEngramListRequest) ([]models.AdminEngramRecord, error) {
	return requireFakeAdminHandler("ListEngrams", f.listEngramsFn)(ctx, request)
}

func (f *fakeMemoryAdminService) GetEngram(ctx context.Context, engramID uuid.UUID, includeDeleted bool) (*models.AdminEngramRecord, error) {
	return requireFakeAdminHandler("GetEngram", f.getEngramFn)(ctx, engramID, includeDeleted)
}

func (f *fakeMemoryAdminService) UpdateEngram(ctx context.Context, engramID, actorUserID uuid.UUID, payload admin.EngramUpdateRequest) (*models.AdminEngramRecord, error) {
	return requireFakeAdminHandler("UpdateEngram", f.updateEngramFn)(ctx, engramID, actorUserID, payload)
}

func (f *fakeMemoryAdminService) MoveEngram(
	ctx context.Context,
	engramID uuid.UUID,
	actor admin.WriteActor,
	payload admin.EngramMoveRequest,
) (*models.AdminEngramRecord, error) {
	return requireFakeAdminHandler("MoveEngram", f.moveEngramFn)(ctx, engramID, actor, payload)
}

func (f *fakeMemoryAdminService) DeleteEngram(ctx context.Context, engramID, actorUserID uuid.UUID, payload admin.EngramDeleteRequest) (admin.EngramDeleteResponse, error) {
	return requireFakeAdminHandler("DeleteEngram", f.deleteEngramFn)(ctx, engramID, actorUserID, payload)
}

func (f *fakeMemoryAdminService) RestoreEngram(ctx context.Context, engramID uuid.UUID) (admin.EngramRestoreResponse, error) {
	return requireFakeAdminHandler("RestoreEngram", f.restoreEngramFn)(ctx, engramID)
}

func (f *fakeMemoryAdminService) RefreshEngramFreshness(
	ctx context.Context,
	request admin.EngramFreshnessRefreshRequest,
) (admin.EngramFreshnessRefreshResponse, error) {
	return requireFakeAdminHandler("RefreshEngramFreshness", f.refreshEngramFreshness)(ctx, request)
}

func (f *fakeMemoryAdminService) RefreshEngramConsolidationSuggestions(
	ctx context.Context,
	request admin.EngramConsolidationSuggestionRefreshRequest,
) (admin.EngramConsolidationSuggestionRefreshResponse, error) {
	return requireFakeAdminHandler("RefreshEngramConsolidationSuggestions", f.refreshConsolidationFn)(ctx, request)
}

func (f *fakeMemoryAdminService) ListEngramConsolidationSuggestions(
	ctx context.Context,
	request admin.EngramConsolidationSuggestionListRequest,
) ([]models.EngramConsolidationSuggestion, error) {
	return requireFakeAdminHandler("ListEngramConsolidationSuggestions", f.listConsolidationFn)(ctx, request)
}

func (f *fakeMemoryAdminService) ActionEngramConsolidationSuggestion(
	ctx context.Context,
	suggestionID uuid.UUID,
	actorUserID uuid.UUID,
	request admin.EngramConsolidationSuggestionActionRequest,
) (*models.EngramConsolidationSuggestion, error) {
	return requireFakeAdminHandler("ActionEngramConsolidationSuggestion", f.actionConsolidationFn)(
		ctx,
		suggestionID,
		actorUserID,
		request,
	)
}

func (f *fakeMemoryAdminService) ListCollections(ctx context.Context, request admin.MemoryAdminListRequest) ([]models.EngramCollectionRecord, error) {
	return requireFakeAdminHandler("ListCollections", f.listCollectionsFn)(ctx, request)
}

func (f *fakeMemoryAdminService) CreateCollection(ctx context.Context, actorUserID uuid.UUID, actorRole string, payload admin.CollectionCreateRequest) (*models.EngramCollectionRecord, error) {
	return requireFakeAdminHandler("CreateCollection", f.createCollectionFn)(ctx, actorUserID, actorRole, payload)
}

func (f *fakeMemoryAdminService) UpdateCollection(ctx context.Context, collectionID uuid.UUID, payload admin.CollectionUpdateRequest) (*models.EngramCollectionRecord, error) {
	return requireFakeAdminHandler("UpdateCollection", f.updateCollectionFn)(ctx, collectionID, payload)
}

func (f *fakeMemoryAdminService) DeleteCollection(ctx context.Context, collectionID, actorUserID uuid.UUID, payload admin.CollectionDeleteRequest) (admin.CollectionDeleteResponse, error) {
	return requireFakeAdminHandler("DeleteCollection", f.deleteCollectionFn)(ctx, collectionID, actorUserID, payload)
}

func (f *fakeMemoryAdminService) AddCollectionItems(ctx context.Context, collectionID, actorUserID uuid.UUID, payload admin.CollectionItemsUpdateRequest) (admin.CollectionItemsAddResponse, error) {
	return requireFakeAdminHandler("AddCollectionItems", f.addCollectionItemsFn)(ctx, collectionID, actorUserID, payload)
}

func (f *fakeMemoryAdminService) RemoveCollectionItem(ctx context.Context, collectionID, engramID uuid.UUID) (admin.CollectionItemRemoveResponse, error) {
	return requireFakeAdminHandler("RemoveCollectionItem", f.removeCollectionItemFn)(ctx, collectionID, engramID)
}

func TestMountMemoryAdminRoutesListSessionsForwardsQueryAndActorCheck(t *testing.T) {
	captured := admin.MemoryAdminListRequest{}
	service := &fakeMemoryAdminService{
		listSessionsFn: func(_ context.Context, request admin.MemoryAdminListRequest) ([]models.AdminChatSessionRecord, error) {
			captured = request
			return []models.AdminChatSessionRecord{}, nil
		},
	}
	actorCalled := false
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000111")

	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		actorCalled = true
		return AdminActor{UserID: ownerUserID, Role: "admin"}, nil
	})

	response := executeRequest(
		router,
		http.MethodGet,
		"/api/v1/admin/memory/sessions?project_id=engram-vault&owner_user_id="+ownerUserID.String()+"&include_deleted=true&limit=50&offset=10",
		nil,
	)
	requireEqual(t, http.StatusOK, response.Code)
	if !actorCalled {
		t.Fatalf("expected require admin actor callback")
	}
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	requireEqual(t, ownerUserID, derefUUID(captured.OwnerUserID))
	requireEqual(t, true, captured.IncludeDeleted)
	requireEqual(t, 50, captured.Limit)
	requireEqual(t, 10, captured.Offset)
}

func TestMountMemoryAdminRoutesDeleteSessionUsesActorAndPayload(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000121")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000122")
	capturedActorUserID := uuid.Nil
	capturedPayload := admin.SessionDeleteRequest{}
	service := &fakeMemoryAdminService{
		deleteSessionFn: func(_ context.Context, receivedSessionID, receivedActorUserID uuid.UUID, payload admin.SessionDeleteRequest) (admin.SessionDeleteResponse, error) {
			requireEqual(t, sessionID, receivedSessionID)
			capturedActorUserID = receivedActorUserID
			capturedPayload = payload
			return admin.SessionDeleteResponse{SessionID: receivedSessionID, Deleted: true, LinkedEngramsDeleted: 2}, nil
		},
	}
	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		return AdminActor{UserID: actorUserID, Role: "admin"}, nil
	})

	response := executeRequest(
		router,
		http.MethodDelete,
		"/api/v1/admin/memory/sessions/"+sessionID.String(),
		[]byte(`{"delete_linked_engrams":true,"reason":"cleanup"}`),
	)
	requireEqual(t, http.StatusOK, response.Code)
	requireEqual(t, actorUserID, capturedActorUserID)
	requireEqual(t, true, capturedPayload.DeleteLinkedEngrams)
	requireEqual(t, "cleanup", derefString(capturedPayload.Reason))
}

func TestMountMemoryAdminRoutesListEngramsUsesRequestObject(t *testing.T) {
	captured := admin.MemoryAdminEngramListRequest{}
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000131")
	service := &fakeMemoryAdminService{
		listEngramsFn: func(_ context.Context, request admin.MemoryAdminEngramListRequest) ([]models.AdminEngramRecord, error) {
			captured = request
			return []models.AdminEngramRecord{}, nil
		},
	}
	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		return AdminActor{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000132"), Role: "admin"}, nil
	})

	response := executeRequest(
		router,
		http.MethodGet,
		"/api/v1/admin/memory/engrams?project_id=engram-vault&session_id="+sessionID.String()+"&q=incident&include_deleted=true&limit=25&offset=5",
		nil,
	)
	requireEqual(t, http.StatusOK, response.Code)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	requireEqual(t, sessionID, derefUUID(captured.SessionID))
	requireEqual(t, "incident", derefString(captured.QueryText))
	requireEqual(t, true, captured.IncludeDeleted)
	requireEqual(t, 25, captured.Limit)
	requireEqual(t, 5, captured.Offset)
}

func TestMountMemoryAdminRoutesListEngramsIncludesDeletedAtNullForActiveRecords(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000133")
	now := time.Date(2026, 2, 27, 18, 30, 0, 0, time.UTC)
	service := &fakeMemoryAdminService{
		listEngramsFn: func(_ context.Context, _ admin.MemoryAdminEngramListRequest) ([]models.AdminEngramRecord, error) {
			return []models.AdminEngramRecord{
				{
					EngramID:                engramID,
					ProjectID:               "engram-vault",
					Title:                   "Linked engram",
					Abstract:                "Linked abstract",
					DetailedSummaryMarkdown: "Linked markdown",
					VisibilityScope:         models.VisibilityScopeProject,
					CreatedAt:               now,
					UpdatedAt:               now,
				},
			}, nil
		},
	}
	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		return AdminActor{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000134"), Role: "admin"}, nil
	})

	response := executeRequest(
		router,
		http.MethodGet,
		"/api/v1/admin/memory/engrams",
		nil,
	)
	requireEqual(t, http.StatusOK, response.Code)

	var payload []map[string]any
	requireNoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	requireEqual(t, 1, len(payload))
	deletedAt, exists := payload[0]["deleted_at"]
	if !exists {
		t.Fatalf("expected deleted_at field in admin engram payload")
	}
	if deletedAt != nil {
		t.Fatalf("expected deleted_at to be null, got %#v", deletedAt)
	}
}

func TestMountMemoryAdminRoutesRefreshEngramFreshnessUsesPayload(t *testing.T) {
	captured := admin.EngramFreshnessRefreshRequest{}
	referenceTime := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)
	service := &fakeMemoryAdminService{
		refreshEngramFreshness: func(
			_ context.Context,
			request admin.EngramFreshnessRefreshRequest,
		) (admin.EngramFreshnessRefreshResponse, error) {
			captured = request
			return admin.EngramFreshnessRefreshResponse{
				ProjectID:     request.ProjectID,
				HalfLifeDays:  40,
				ReferenceTime: referenceTime,
				UpdatedCount:  12,
			}, nil
		},
	}

	executeMemoryAdminRefreshRequest(
		t,
		service,
		"/api/v1/admin/memory/engrams/freshness/refresh",
		[]byte(`{"project_id":"engram-vault","half_life_days":40}`),
	)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	if captured.HalfLifeDays == nil {
		t.Fatalf("expected half_life_days to be forwarded")
	}
	requireEqual(t, 40.0, *captured.HalfLifeDays)
}

func TestMountMemoryAdminRoutesRefreshEngramConsolidationUsesPayload(t *testing.T) {
	captured := admin.EngramConsolidationSuggestionRefreshRequest{}
	suggestedAt := time.Date(2026, 3, 3, 13, 5, 0, 0, time.UTC)
	service := &fakeMemoryAdminService{
		refreshConsolidationFn: func(
			_ context.Context,
			request admin.EngramConsolidationSuggestionRefreshRequest,
		) (admin.EngramConsolidationSuggestionRefreshResponse, error) {
			captured = request
			return admin.EngramConsolidationSuggestionRefreshResponse{
				ProjectID:    request.ProjectID,
				MinGroupSize: 3,
				SuggestedAt:  suggestedAt,
				UpdatedCount: 5,
			}, nil
		},
	}

	executeMemoryAdminRefreshRequest(
		t,
		service,
		"/api/v1/admin/memory/engrams/consolidation/refresh",
		[]byte(`{"project_id":"engram-vault","min_group_size":3}`),
	)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	if captured.MinGroupSize == nil {
		t.Fatalf("expected min_group_size to be forwarded")
	}
	requireEqual(t, 3, *captured.MinGroupSize)
}

func TestMountMemoryAdminRoutesListEngramConsolidationSuggestionsUsesRequestObject(t *testing.T) {
	captured := admin.EngramConsolidationSuggestionListRequest{}
	status := models.ConsolidationSuggestionStatusSuggested
	service := &fakeMemoryAdminService{
		listConsolidationFn: func(
			_ context.Context,
			request admin.EngramConsolidationSuggestionListRequest,
		) ([]models.EngramConsolidationSuggestion, error) {
			captured = request
			return []models.EngramConsolidationSuggestion{}, nil
		},
	}
	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		return AdminActor{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000141"), Role: "admin"}, nil
	})

	response := executeRequest(
		router,
		http.MethodGet,
		"/api/v1/admin/memory/engrams/consolidation/suggestions?project_id=engram-vault&status=suggested&limit=15&offset=2",
		nil,
	)
	requireEqual(t, http.StatusOK, response.Code)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	requireEqual(t, status, *captured.Status)
	requireEqual(t, 15, captured.Limit)
	requireEqual(t, 2, captured.Offset)
}

func TestMountMemoryAdminRoutesListEngramConsolidationSuggestionsRejectsInvalidStatus(t *testing.T) {
	service := &fakeMemoryAdminService{
		listConsolidationFn: func(
			_ context.Context,
			_ admin.EngramConsolidationSuggestionListRequest,
		) ([]models.EngramConsolidationSuggestion, error) {
			return nil, errors.New("unexpected")
		},
	}
	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		return AdminActor{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000142"), Role: "admin"}, nil
	})

	response := executeRequest(
		router,
		http.MethodGet,
		"/api/v1/admin/memory/engrams/consolidation/suggestions?status=invalid",
		nil,
	)
	requireEqual(t, http.StatusBadRequest, response.Code)
}

func TestMountMemoryAdminRoutesActionConsolidationSuggestionUsesActorAndPayload(t *testing.T) {
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-000000000171")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000172")
	capturedSuggestionID := uuid.Nil
	capturedActorUserID := uuid.Nil
	capturedRequest := admin.EngramConsolidationSuggestionActionRequest{}
	service := &fakeMemoryAdminService{
		actionConsolidationFn: func(
			_ context.Context,
			receivedSuggestionID uuid.UUID,
			receivedActorUserID uuid.UUID,
			request admin.EngramConsolidationSuggestionActionRequest,
		) (*models.EngramConsolidationSuggestion, error) {
			capturedSuggestionID = receivedSuggestionID
			capturedActorUserID = receivedActorUserID
			capturedRequest = request
			return &models.EngramConsolidationSuggestion{
				SuggestionID: suggestionID,
				Status:       models.ConsolidationSuggestionStatusMerged,
			}, nil
		},
	}
	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		return AdminActor{UserID: actorUserID, Role: "admin"}, nil
	})

	response := executeRequest(
		router,
		http.MethodPost,
		"/api/v1/admin/memory/engrams/consolidation/suggestions/"+suggestionID.String()+"/action",
		[]byte(`{"project_id":"engram-vault","status":"merged"}`),
	)
	requireEqual(t, http.StatusOK, response.Code)
	requireEqual(t, suggestionID, capturedSuggestionID)
	requireEqual(t, actorUserID, capturedActorUserID)
	requireEqual(t, "engram-vault", derefString(capturedRequest.ProjectID))
	requireEqual(t, models.ConsolidationSuggestionStatusMerged, capturedRequest.Status)
}

func TestMountMemoryAdminRoutesActionConsolidationSuggestionRejectsInvalidStatus(t *testing.T) {
	service := &fakeMemoryAdminService{
		actionConsolidationFn: func(
			_ context.Context,
			_ uuid.UUID,
			_ uuid.UUID,
			_ admin.EngramConsolidationSuggestionActionRequest,
		) (*models.EngramConsolidationSuggestion, error) {
			return nil, errors.New("unexpected")
		},
	}
	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		return AdminActor{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000173"), Role: "admin"}, nil
	})

	response := executeRequest(
		router,
		http.MethodPost,
		"/api/v1/admin/memory/engrams/consolidation/suggestions/00000000-0000-0000-0000-000000000174/action",
		[]byte(`{"status":"invalid"}`),
	)
	requireEqual(t, http.StatusBadRequest, response.Code)
}

func TestMountMemoryAdminRoutesUpdateEngramMapsStaleTo409(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000141")
	service := &fakeMemoryAdminService{
		updateEngramFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ admin.EngramUpdateRequest) (*models.AdminEngramRecord, error) {
			return nil, admin.ErrEngramStale
		},
	}
	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		return AdminActor{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000142"), Role: "admin"}, nil
	})

	response := executeRequest(
		router,
		http.MethodPatch,
		"/api/v1/admin/memory/engrams/"+engramID.String(),
		[]byte(`{"title":"Updated"}`),
	)
	requireEqual(t, http.StatusConflict, response.Code)
	var payload map[string]string
	requireNoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	requireEqual(t, admin.ErrEngramStale.Error(), payload["detail"])
}

func TestMountMemoryAdminRoutesCreateCollectionReturns201(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000151")
	now := time.Date(2026, 2, 22, 21, 0, 0, 0, time.UTC)
	capturedActorRole := ""
	capturedPayload := admin.CollectionCreateRequest{}
	service := &fakeMemoryAdminService{
		createCollectionFn: func(_ context.Context, receivedActorUserID uuid.UUID, actorRole string, payload admin.CollectionCreateRequest) (*models.EngramCollectionRecord, error) {
			requireEqual(t, actorUserID, receivedActorUserID)
			capturedActorRole = actorRole
			capturedPayload = payload
			collectionID := uuid.MustParse("00000000-0000-0000-0000-000000000152")
			return &models.EngramCollectionRecord{
				CollectionID: collectionID,
				ProjectID:    payload.ProjectID,
				OwnerUserID:  receivedActorUserID,
				Name:         payload.Name,
				Description:  payload.Description,
				CreatedAt:    now,
				UpdatedAt:    now,
			}, nil
		},
	}
	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		return AdminActor{UserID: actorUserID, Role: "admin"}, nil
	})

	response := executeRequest(
		router,
		http.MethodPost,
		"/api/v1/admin/memory/collections",
		[]byte(`{"project_id":"engram-vault","name":"Collection","description":"desc"}`),
	)
	requireEqual(t, http.StatusCreated, response.Code)
	requireEqual(t, "admin", capturedActorRole)
	requireEqual(t, "engram-vault", capturedPayload.ProjectID)
	requireEqual(t, "Collection", capturedPayload.Name)
}

func TestMountMemoryAdminRoutesCreateCollectionMapsMissingProjectTo400(t *testing.T) {
	service := &fakeMemoryAdminService{
		createCollectionFn: func(_ context.Context, _ uuid.UUID, _ string, _ admin.CollectionCreateRequest) (*models.EngramCollectionRecord, error) {
			return nil, admin.ErrProjectIDRequired
		},
	}
	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		return AdminActor{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000161"), Role: "admin"}, nil
	})

	response := executeRequest(
		router,
		http.MethodPost,
		"/api/v1/admin/memory/collections",
		[]byte(`{"project_id":"","name":"Collection","description":"desc"}`),
	)
	requireEqual(t, http.StatusBadRequest, response.Code)
}

func TestMountMemoryAdminRoutesReturnsForbiddenWhenActorCheckFails(t *testing.T) {
	service := &fakeMemoryAdminService{
		listSessionsFn: func(_ context.Context, _ admin.MemoryAdminListRequest) ([]models.AdminChatSessionRecord, error) {
			return nil, errors.New("unexpected")
		},
	}
	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		return AdminActor{}, errors.New("forbidden")
	})

	response := executeRequest(router, http.MethodGet, "/api/v1/admin/memory/sessions", nil)
	requireEqual(t, http.StatusForbidden, response.Code)
}

func executeMemoryAdminRefreshRequest(
	t *testing.T,
	service MemoryAdminService,
	path string,
	body []byte,
) {
	t.Helper()
	router := chi.NewRouter()
	MountMemoryAdminRoutes(router, service, func(_ *http.Request) (AdminActor, error) {
		return AdminActor{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000139"), Role: "admin"}, nil
	})
	response := executeRequest(router, http.MethodPost, path, body)
	requireEqual(t, http.StatusOK, response.Code)
}

func executeRequest(router http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefUUID(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.Nil
	}
	return *value
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func requireEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

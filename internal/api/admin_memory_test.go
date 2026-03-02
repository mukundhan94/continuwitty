package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
	listCollectionsFn      func(ctx context.Context, request admin.MemoryAdminListRequest) ([]models.EngramCollectionRecord, error)
	createCollectionFn     func(ctx context.Context, actorUserID uuid.UUID, actorRole string, payload admin.CollectionCreateRequest) (*models.EngramCollectionRecord, error)
	updateCollectionFn     func(ctx context.Context, collectionID uuid.UUID, payload admin.CollectionUpdateRequest) (*models.EngramCollectionRecord, error)
	deleteCollectionFn     func(ctx context.Context, collectionID, actorUserID uuid.UUID, payload admin.CollectionDeleteRequest) (admin.CollectionDeleteResponse, error)
	addCollectionItemsFn   func(ctx context.Context, collectionID, actorUserID uuid.UUID, payload admin.CollectionItemsUpdateRequest) (admin.CollectionItemsAddResponse, error)
	removeCollectionItemFn func(ctx context.Context, collectionID, engramID uuid.UUID) (admin.CollectionItemRemoveResponse, error)
}

func (f *fakeMemoryAdminService) ListSessions(ctx context.Context, request admin.MemoryAdminListRequest) ([]models.AdminChatSessionRecord, error) {
	if f.listSessionsFn == nil {
		panic("unexpected ListSessions call")
	}
	return f.listSessionsFn(ctx, request)
}

func (f *fakeMemoryAdminService) DeleteSession(ctx context.Context, sessionID, actorUserID uuid.UUID, payload admin.SessionDeleteRequest) (admin.SessionDeleteResponse, error) {
	if f.deleteSessionFn == nil {
		panic("unexpected DeleteSession call")
	}
	return f.deleteSessionFn(ctx, sessionID, actorUserID, payload)
}

func (f *fakeMemoryAdminService) RestoreSession(ctx context.Context, sessionID uuid.UUID) (admin.SessionRestoreResponse, error) {
	if f.restoreSessionFn == nil {
		panic("unexpected RestoreSession call")
	}
	return f.restoreSessionFn(ctx, sessionID)
}

func (f *fakeMemoryAdminService) ListEngrams(ctx context.Context, request admin.MemoryAdminEngramListRequest) ([]models.AdminEngramRecord, error) {
	if f.listEngramsFn == nil {
		panic("unexpected ListEngrams call")
	}
	return f.listEngramsFn(ctx, request)
}

func (f *fakeMemoryAdminService) GetEngram(ctx context.Context, engramID uuid.UUID, includeDeleted bool) (*models.AdminEngramRecord, error) {
	if f.getEngramFn == nil {
		panic("unexpected GetEngram call")
	}
	return f.getEngramFn(ctx, engramID, includeDeleted)
}

func (f *fakeMemoryAdminService) UpdateEngram(ctx context.Context, engramID, actorUserID uuid.UUID, payload admin.EngramUpdateRequest) (*models.AdminEngramRecord, error) {
	if f.updateEngramFn == nil {
		panic("unexpected UpdateEngram call")
	}
	return f.updateEngramFn(ctx, engramID, actorUserID, payload)
}

func (f *fakeMemoryAdminService) MoveEngram(
	ctx context.Context,
	engramID uuid.UUID,
	actor admin.WriteActor,
	payload admin.EngramMoveRequest,
) (*models.AdminEngramRecord, error) {
	if f.moveEngramFn == nil {
		panic("unexpected MoveEngram call")
	}
	return f.moveEngramFn(ctx, engramID, actor, payload)
}

func (f *fakeMemoryAdminService) DeleteEngram(ctx context.Context, engramID, actorUserID uuid.UUID, payload admin.EngramDeleteRequest) (admin.EngramDeleteResponse, error) {
	if f.deleteEngramFn == nil {
		panic("unexpected DeleteEngram call")
	}
	return f.deleteEngramFn(ctx, engramID, actorUserID, payload)
}

func (f *fakeMemoryAdminService) RestoreEngram(ctx context.Context, engramID uuid.UUID) (admin.EngramRestoreResponse, error) {
	if f.restoreEngramFn == nil {
		panic("unexpected RestoreEngram call")
	}
	return f.restoreEngramFn(ctx, engramID)
}

func (f *fakeMemoryAdminService) ListCollections(ctx context.Context, request admin.MemoryAdminListRequest) ([]models.EngramCollectionRecord, error) {
	if f.listCollectionsFn == nil {
		panic("unexpected ListCollections call")
	}
	return f.listCollectionsFn(ctx, request)
}

func (f *fakeMemoryAdminService) CreateCollection(ctx context.Context, actorUserID uuid.UUID, actorRole string, payload admin.CollectionCreateRequest) (*models.EngramCollectionRecord, error) {
	if f.createCollectionFn == nil {
		panic("unexpected CreateCollection call")
	}
	return f.createCollectionFn(ctx, actorUserID, actorRole, payload)
}

func (f *fakeMemoryAdminService) UpdateCollection(ctx context.Context, collectionID uuid.UUID, payload admin.CollectionUpdateRequest) (*models.EngramCollectionRecord, error) {
	if f.updateCollectionFn == nil {
		panic("unexpected UpdateCollection call")
	}
	return f.updateCollectionFn(ctx, collectionID, payload)
}

func (f *fakeMemoryAdminService) DeleteCollection(ctx context.Context, collectionID, actorUserID uuid.UUID, payload admin.CollectionDeleteRequest) (admin.CollectionDeleteResponse, error) {
	if f.deleteCollectionFn == nil {
		panic("unexpected DeleteCollection call")
	}
	return f.deleteCollectionFn(ctx, collectionID, actorUserID, payload)
}

func (f *fakeMemoryAdminService) AddCollectionItems(ctx context.Context, collectionID, actorUserID uuid.UUID, payload admin.CollectionItemsUpdateRequest) (admin.CollectionItemsAddResponse, error) {
	if f.addCollectionItemsFn == nil {
		panic("unexpected AddCollectionItems call")
	}
	return f.addCollectionItemsFn(ctx, collectionID, actorUserID, payload)
}

func (f *fakeMemoryAdminService) RemoveCollectionItem(ctx context.Context, collectionID, engramID uuid.UUID) (admin.CollectionItemRemoveResponse, error) {
	if f.removeCollectionItemFn == nil {
		panic("unexpected RemoveCollectionItem call")
	}
	return f.removeCollectionItemFn(ctx, collectionID, engramID)
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

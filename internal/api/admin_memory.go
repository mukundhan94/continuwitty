package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"engram/internal/admin"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AdminActor identifies the authenticated admin actor for memory-admin operations.
type AdminActor struct {
	UserID uuid.UUID
	Role   string
}

// RequireAdminActor resolves and authorizes the current request actor.
type RequireAdminActor func(request *http.Request) (AdminActor, error)

// MemoryAdminService captures service methods required by memory-admin HTTP handlers.
type MemoryAdminService interface {
	ListSessions(ctx context.Context, request admin.MemoryAdminListRequest) ([]models.AdminChatSessionRecord, error)
	DeleteSession(ctx context.Context, sessionID, actorUserID uuid.UUID, payload admin.SessionDeleteRequest) (admin.SessionDeleteResponse, error)
	RestoreSession(ctx context.Context, sessionID uuid.UUID) (admin.SessionRestoreResponse, error)

	ListEngrams(ctx context.Context, request admin.MemoryAdminEngramListRequest) ([]models.AdminEngramRecord, error)
	GetEngram(ctx context.Context, engramID uuid.UUID, includeDeleted bool) (*models.AdminEngramRecord, error)
	UpdateEngram(ctx context.Context, engramID, actorUserID uuid.UUID, payload admin.EngramUpdateRequest) (*models.AdminEngramRecord, error)
	MoveEngram(ctx context.Context, engramID, actorUserID uuid.UUID, actorRole string, payload admin.EngramMoveRequest) (*models.AdminEngramRecord, error)
	DeleteEngram(ctx context.Context, engramID, actorUserID uuid.UUID, payload admin.EngramDeleteRequest) (admin.EngramDeleteResponse, error)
	RestoreEngram(ctx context.Context, engramID uuid.UUID) (admin.EngramRestoreResponse, error)

	ListCollections(ctx context.Context, request admin.MemoryAdminListRequest) ([]models.EngramCollectionRecord, error)
	CreateCollection(ctx context.Context, actorUserID uuid.UUID, actorRole string, payload admin.CollectionCreateRequest) (*models.EngramCollectionRecord, error)
	UpdateCollection(ctx context.Context, collectionID uuid.UUID, payload admin.CollectionUpdateRequest) (*models.EngramCollectionRecord, error)
	DeleteCollection(ctx context.Context, collectionID, actorUserID uuid.UUID, payload admin.CollectionDeleteRequest) (admin.CollectionDeleteResponse, error)
	AddCollectionItems(ctx context.Context, collectionID, actorUserID uuid.UUID, payload admin.CollectionItemsUpdateRequest) (admin.CollectionItemsAddResponse, error)
	RemoveCollectionItem(ctx context.Context, collectionID, engramID uuid.UUID) (admin.CollectionItemRemoveResponse, error)
}

// MountMemoryAdminRoutes registers /api/v1/admin/memory routes on the provided router.
func MountMemoryAdminRoutes(router chi.Router, service MemoryAdminService, requireAdminActor RequireAdminActor) {
	router.Route("/api/v1/admin/memory", func(memory chi.Router) {
		memory.Get("/sessions", func(writer http.ResponseWriter, request *http.Request) {
			if _, ok := requireActor(writer, request, requireAdminActor); !ok {
				return
			}

			projectID := optionalTrimmedString(request.URL.Query().Get("project_id"))
			ownerUserID, ok := parseOptionalUUIDQuery(writer, request, "owner_user_id")
			if !ok {
				return
			}
			includeDeleted, ok := parseOptionalBoolQuery(writer, request, "include_deleted", false)
			if !ok {
				return
			}
			limit, ok := parseOptionalIntQuery(writer, request, "limit", 200, 1, 1000)
			if !ok {
				return
			}
			offset, ok := parseOptionalIntQuery(writer, request, "offset", 0, 0, 1_000_000)
			if !ok {
				return
			}

			sessions, err := service.ListSessions(
				request.Context(),
				admin.MemoryAdminListRequest{
					ProjectID:      projectID,
					OwnerUserID:    ownerUserID,
					IncludeDeleted: includeDeleted,
					Limit:          limit,
					Offset:         offset,
				},
			)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, sessions)
		})

		memory.Delete("/sessions/{session_id}", func(writer http.ResponseWriter, request *http.Request) {
			actor, ok := requireActor(writer, request, requireAdminActor)
			if !ok {
				return
			}
			sessionID, ok := parsePathUUID(writer, request, "session_id")
			if !ok {
				return
			}

			payload := admin.SessionDeleteRequest{}
			if !decodeJSONAllowEmpty(writer, request, &payload) {
				return
			}
			response, err := service.DeleteSession(request.Context(), sessionID, actor.UserID, payload)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, response)
		})

		memory.Post("/sessions/{session_id}/restore", func(writer http.ResponseWriter, request *http.Request) {
			if _, ok := requireActor(writer, request, requireAdminActor); !ok {
				return
			}
			sessionID, ok := parsePathUUID(writer, request, "session_id")
			if !ok {
				return
			}
			response, err := service.RestoreSession(request.Context(), sessionID)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, response)
		})

		memory.Get("/engrams", func(writer http.ResponseWriter, request *http.Request) {
			if _, ok := requireActor(writer, request, requireAdminActor); !ok {
				return
			}

			projectID := optionalTrimmedString(request.URL.Query().Get("project_id"))
			sessionID, ok := parseOptionalUUIDQuery(writer, request, "session_id")
			if !ok {
				return
			}
			queryText := optionalTrimmedString(request.URL.Query().Get("q"))
			includeDeleted, ok := parseOptionalBoolQuery(writer, request, "include_deleted", false)
			if !ok {
				return
			}
			limit, ok := parseOptionalIntQuery(writer, request, "limit", 200, 1, 1000)
			if !ok {
				return
			}
			offset, ok := parseOptionalIntQuery(writer, request, "offset", 0, 0, 1_000_000)
			if !ok {
				return
			}

			records, err := service.ListEngrams(
				request.Context(),
				admin.MemoryAdminEngramListRequest{
					MemoryAdminListRequest: admin.MemoryAdminListRequest{
						ProjectID:      projectID,
						IncludeDeleted: includeDeleted,
						Limit:          limit,
						Offset:         offset,
					},
					SessionID: sessionID,
					QueryText: queryText,
				},
			)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, records)
		})

		memory.Get("/engrams/{engram_id}", func(writer http.ResponseWriter, request *http.Request) {
			if _, ok := requireActor(writer, request, requireAdminActor); !ok {
				return
			}
			engramID, ok := parsePathUUID(writer, request, "engram_id")
			if !ok {
				return
			}
			includeDeleted, ok := parseOptionalBoolQuery(writer, request, "include_deleted", true)
			if !ok {
				return
			}

			record, err := service.GetEngram(request.Context(), engramID, includeDeleted)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, record)
		})

		memory.Patch("/engrams/{engram_id}", func(writer http.ResponseWriter, request *http.Request) {
			actor, ok := requireActor(writer, request, requireAdminActor)
			if !ok {
				return
			}
			engramID, ok := parsePathUUID(writer, request, "engram_id")
			if !ok {
				return
			}
			payload := admin.EngramUpdateRequest{}
			if !decodeJSONAllowEmpty(writer, request, &payload) {
				return
			}
			updated, err := service.UpdateEngram(request.Context(), engramID, actor.UserID, payload)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, updated)
		})

		memory.Post("/engrams/{engram_id}/move", func(writer http.ResponseWriter, request *http.Request) {
			actor, ok := requireActor(writer, request, requireAdminActor)
			if !ok {
				return
			}
			engramID, ok := parsePathUUID(writer, request, "engram_id")
			if !ok {
				return
			}
			payload := admin.EngramMoveRequest{}
			if !decodeJSONAllowEmpty(writer, request, &payload) {
				return
			}
			moved, err := service.MoveEngram(request.Context(), engramID, actor.UserID, actor.Role, payload)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, moved)
		})

		memory.Delete("/engrams/{engram_id}", func(writer http.ResponseWriter, request *http.Request) {
			actor, ok := requireActor(writer, request, requireAdminActor)
			if !ok {
				return
			}
			engramID, ok := parsePathUUID(writer, request, "engram_id")
			if !ok {
				return
			}
			payload := admin.EngramDeleteRequest{}
			if !decodeJSONAllowEmpty(writer, request, &payload) {
				return
			}
			response, err := service.DeleteEngram(request.Context(), engramID, actor.UserID, payload)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, response)
		})

		memory.Post("/engrams/{engram_id}/restore", func(writer http.ResponseWriter, request *http.Request) {
			if _, ok := requireActor(writer, request, requireAdminActor); !ok {
				return
			}
			engramID, ok := parsePathUUID(writer, request, "engram_id")
			if !ok {
				return
			}
			response, err := service.RestoreEngram(request.Context(), engramID)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, response)
		})

		memory.Get("/collections", func(writer http.ResponseWriter, request *http.Request) {
			if _, ok := requireActor(writer, request, requireAdminActor); !ok {
				return
			}
			projectID := optionalTrimmedString(request.URL.Query().Get("project_id"))
			includeDeleted, ok := parseOptionalBoolQuery(writer, request, "include_deleted", false)
			if !ok {
				return
			}
			limit, ok := parseOptionalIntQuery(writer, request, "limit", 200, 1, 1000)
			if !ok {
				return
			}
			offset, ok := parseOptionalIntQuery(writer, request, "offset", 0, 0, 1_000_000)
			if !ok {
				return
			}
			records, err := service.ListCollections(
				request.Context(),
				admin.MemoryAdminListRequest{
					ProjectID:      projectID,
					IncludeDeleted: includeDeleted,
					Limit:          limit,
					Offset:         offset,
				},
			)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, records)
		})

		memory.Post("/collections", func(writer http.ResponseWriter, request *http.Request) {
			actor, ok := requireActor(writer, request, requireAdminActor)
			if !ok {
				return
			}
			payload := admin.CollectionCreateRequest{}
			if !decodeJSONAllowEmpty(writer, request, &payload) {
				return
			}
			record, err := service.CreateCollection(request.Context(), actor.UserID, actor.Role, payload)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusCreated, record)
		})

		memory.Patch("/collections/{collection_id}", func(writer http.ResponseWriter, request *http.Request) {
			if _, ok := requireActor(writer, request, requireAdminActor); !ok {
				return
			}
			collectionID, ok := parsePathUUID(writer, request, "collection_id")
			if !ok {
				return
			}
			payload := admin.CollectionUpdateRequest{}
			if !decodeJSONAllowEmpty(writer, request, &payload) {
				return
			}
			record, err := service.UpdateCollection(request.Context(), collectionID, payload)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, record)
		})

		memory.Delete("/collections/{collection_id}", func(writer http.ResponseWriter, request *http.Request) {
			actor, ok := requireActor(writer, request, requireAdminActor)
			if !ok {
				return
			}
			collectionID, ok := parsePathUUID(writer, request, "collection_id")
			if !ok {
				return
			}
			payload := admin.CollectionDeleteRequest{}
			if !decodeJSONAllowEmpty(writer, request, &payload) {
				return
			}
			response, err := service.DeleteCollection(request.Context(), collectionID, actor.UserID, payload)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, response)
		})

		memory.Post("/collections/{collection_id}/items", func(writer http.ResponseWriter, request *http.Request) {
			actor, ok := requireActor(writer, request, requireAdminActor)
			if !ok {
				return
			}
			collectionID, ok := parsePathUUID(writer, request, "collection_id")
			if !ok {
				return
			}
			payload := admin.CollectionItemsUpdateRequest{}
			if !decodeJSONAllowEmpty(writer, request, &payload) {
				return
			}
			response, err := service.AddCollectionItems(request.Context(), collectionID, actor.UserID, payload)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, response)
		})

		memory.Delete("/collections/{collection_id}/items/{engram_id}", func(writer http.ResponseWriter, request *http.Request) {
			if _, ok := requireActor(writer, request, requireAdminActor); !ok {
				return
			}
			collectionID, ok := parsePathUUID(writer, request, "collection_id")
			if !ok {
				return
			}
			engramID, ok := parsePathUUID(writer, request, "engram_id")
			if !ok {
				return
			}
			response, err := service.RemoveCollectionItem(request.Context(), collectionID, engramID)
			if err != nil {
				writeServiceError(writer, err)
				return
			}
			writeJSON(writer, http.StatusOK, response)
		})
	})
}

func requireActor(writer http.ResponseWriter, request *http.Request, requireAdminActor RequireAdminActor) (AdminActor, bool) {
	if requireAdminActor == nil {
		writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "forbidden"})
		return AdminActor{}, false
	}
	actor, err := requireAdminActor(request)
	if err != nil {
		writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "forbidden"})
		return AdminActor{}, false
	}
	return actor, true
}

func parsePathUUID(writer http.ResponseWriter, request *http.Request, param string) (uuid.UUID, bool) {
	value := strings.TrimSpace(chi.URLParam(request, param))
	parsed, err := uuid.Parse(value)
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid " + param})
		return uuid.Nil, false
	}
	return parsed, true
}

func parseOptionalUUIDQuery(writer http.ResponseWriter, request *http.Request, key string) (*uuid.UUID, bool) {
	value := strings.TrimSpace(request.URL.Query().Get(key))
	if value == "" {
		return nil, true
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid " + key})
		return nil, false
	}
	return &parsed, true
}

func parseOptionalBoolQuery(writer http.ResponseWriter, request *http.Request, key string, defaultValue bool) (bool, bool) {
	value := strings.TrimSpace(request.URL.Query().Get(key))
	if value == "" {
		return defaultValue, true
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid " + key})
		return false, false
	}
	return parsed, true
}

func parseOptionalIntQuery(writer http.ResponseWriter, request *http.Request, key string, defaultValue, minValue, maxValue int) (int, bool) {
	value := strings.TrimSpace(request.URL.Query().Get(key))
	if value == "" {
		return defaultValue, true
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minValue || parsed > maxValue {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid " + key})
		return 0, false
	}
	return parsed, true
}

func optionalTrimmedString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func decodeJSONAllowEmpty(writer http.ResponseWriter, request *http.Request, destination any) bool {
	if request.Body == nil {
		return true
	}
	defer request.Body.Close()

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid json body"})
		return false
	}
	return true
}

func writeServiceError(writer http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	detail := "internal error"
	switch {
	case errors.Is(err, admin.ErrSessionNotFound),
		errors.Is(err, admin.ErrEngramNotFound),
		errors.Is(err, admin.ErrCollectionNotFound):
		statusCode = http.StatusNotFound
		detail = err.Error()
	case errors.Is(err, admin.ErrProjectIDRequired):
		statusCode = http.StatusBadRequest
		detail = err.Error()
	case errors.Is(err, admin.ErrEngramStale),
		errors.Is(err, admin.ErrCollectionStale),
		errors.Is(err, repository.ErrCollectionNameExists):
		statusCode = http.StatusConflict
		detail = err.Error()
	}
	writeJSON(writer, statusCode, map[string]string{"detail": detail})
}

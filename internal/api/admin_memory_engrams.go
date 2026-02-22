package api

import (
	"context"
	"net/http"

	"engram/internal/admin"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func mountMemoryAdminEngramRoutes(memory chi.Router, service MemoryAdminService, requireAdminActor RequireAdminActor) {
	memory.Get("/engrams", listMemoryAdminEngramsRoute(service, requireAdminActor))
	memory.Get("/engrams/{engram_id}", getMemoryAdminEngramRoute(service, requireAdminActor))
	memory.Patch("/engrams/{engram_id}", updateMemoryAdminEngramRoute(service, requireAdminActor))
	memory.Post("/engrams/{engram_id}/move", moveMemoryAdminEngramRoute(service, requireAdminActor))
	memory.Delete("/engrams/{engram_id}", deleteMemoryAdminEngramRoute(service, requireAdminActor))
	memory.Post("/engrams/{engram_id}/restore", restoreMemoryAdminEngramRoute(service, requireAdminActor))
}

func listMemoryAdminEngramsRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if _, ok := requireActor(writer, request, requireAdminActor); !ok {
			return
		}
		listRequest, ok := parseMemoryAdminEngramListRequest(writer, request)
		if !ok {
			return
		}
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return service.ListEngrams(request.Context(), listRequest)
		})
	}
}

func getMemoryAdminEngramRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
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
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return service.GetEngram(request.Context(), engramID, includeDeleted)
		})
	}
}

func updateMemoryAdminEngramRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return engramActorPathPayloadRoute(
		requireAdminActor,
		func(ctx context.Context, actor AdminActor, engramID uuid.UUID, payload admin.EngramUpdateRequest) (any, error) {
			return service.UpdateEngram(ctx, engramID, actor.UserID, payload)
		},
	)
}

func moveMemoryAdminEngramRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return engramActorPathPayloadRoute(
		requireAdminActor,
		func(ctx context.Context, actor AdminActor, engramID uuid.UUID, payload admin.EngramMoveRequest) (any, error) {
			return service.MoveEngram(ctx, engramID, actor.UserID, actor.Role, payload)
		},
	)
}

func deleteMemoryAdminEngramRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return engramActorPathPayloadRoute(
		requireAdminActor,
		func(ctx context.Context, actor AdminActor, engramID uuid.UUID, payload admin.EngramDeleteRequest) (any, error) {
			return service.DeleteEngram(ctx, engramID, actor.UserID, payload)
		},
	)
}

func restoreMemoryAdminEngramRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return engramPathRoute(requireAdminActor, func(ctx context.Context, engramID uuid.UUID) (any, error) {
		return service.RestoreEngram(ctx, engramID)
	})
}

func engramPathRoute(
	requireAdminActor RequireAdminActor,
	execute func(ctx context.Context, engramID uuid.UUID) (any, error),
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if _, ok := requireActor(writer, request, requireAdminActor); !ok {
			return
		}
		engramID, ok := parsePathUUID(writer, request, "engram_id")
		if !ok {
			return
		}
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return execute(request.Context(), engramID)
		})
	}
}

func engramActorPathPayloadRoute[Payload any](
	requireAdminActor RequireAdminActor,
	execute func(ctx context.Context, actor AdminActor, engramID uuid.UUID, payload Payload) (any, error),
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actor, ok := requireActor(writer, request, requireAdminActor)
		if !ok {
			return
		}
		engramID, ok := parsePathUUID(writer, request, "engram_id")
		if !ok {
			return
		}
		var payload Payload
		if !decodeJSONAllowEmpty(writer, request, &payload) {
			return
		}
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return execute(request.Context(), actor, engramID, payload)
		})
	}
}

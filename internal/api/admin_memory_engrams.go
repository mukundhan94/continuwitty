package api

import (
	"context"
	"net/http"

	"engram/internal/admin"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func mountMemoryAdminEngramRoutes(memory chi.Router, service MemoryAdminService, requireAdminActor RequireAdminActor) {
	memory.Post("/engrams/freshness/refresh", refreshMemoryAdminEngramFreshnessRoute(service, requireAdminActor))
	memory.Get("/engrams", listMemoryAdminEngramsRoute(service, requireAdminActor))
	memory.Get("/engrams/{engram_id}", getMemoryAdminEngramRoute(service, requireAdminActor))
	memory.Patch("/engrams/{engram_id}", updateMemoryAdminEngramRoute(service, requireAdminActor))
	memory.Post("/engrams/{engram_id}/move", moveMemoryAdminEngramRoute(service, requireAdminActor))
	memory.Delete("/engrams/{engram_id}", deleteMemoryAdminEngramRoute(service, requireAdminActor))
	memory.Post("/engrams/{engram_id}/restore", restoreMemoryAdminEngramRoute(service, requireAdminActor))
}

func refreshMemoryAdminEngramFreshnessRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if _, ok := requireActor(writer, request, requireAdminActor); !ok {
			return
		}
		payload := struct {
			ProjectID    *string  `json:"project_id,omitempty"`
			HalfLifeDays *float64 `json:"half_life_days,omitempty"`
		}{}
		if !decodeJSONAllowEmpty(writer, request, &payload) {
			return
		}
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return service.RefreshEngramFreshness(
				request.Context(),
				admin.EngramFreshnessRefreshRequest{
					ProjectID:    payload.ProjectID,
					HalfLifeDays: payload.HalfLifeDays,
				},
			)
		})
	}
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

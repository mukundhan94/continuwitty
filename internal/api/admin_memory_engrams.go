package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func mountMemoryAdminEngramRoutes(memory chi.Router, service MemoryAdminService, requireAdminActor RequireAdminActor) {
	memory.Post("/engrams/freshness/refresh", refreshMemoryAdminEngramFreshnessRoute(service, requireAdminActor))
	memory.Post(
		"/engrams/consolidation/refresh",
		refreshMemoryAdminEngramConsolidationRoute(service, requireAdminActor),
	)
	memory.Get(
		"/engrams/consolidation/suggestions",
		listMemoryAdminEngramConsolidationRoute(service, requireAdminActor),
	)
	memory.Post(
		"/engrams/consolidation/suggestions/{suggestion_id}/action",
		actionMemoryAdminEngramConsolidationRoute(service, requireAdminActor),
	)
	memory.Post(
		"/engrams/contradictions/refresh",
		refreshMemoryAdminEngramContradictionRoute(service, requireAdminActor),
	)
	memory.Get(
		"/engrams/contradictions/alerts",
		listMemoryAdminEngramContradictionRoute(service, requireAdminActor),
	)
	memory.Post(
		"/engrams/contradictions/alerts/{alert_id}/resolve",
		resolveMemoryAdminEngramContradictionRoute(service, requireAdminActor),
	)
	memory.Get("/engrams", listMemoryAdminEngramsRoute(service, requireAdminActor))
	memory.Get("/engrams/{engram_id}", getMemoryAdminEngramRoute(service, requireAdminActor))
	memory.Patch("/engrams/{engram_id}", updateMemoryAdminEngramRoute(service, requireAdminActor))
	memory.Post("/engrams/{engram_id}/move", moveMemoryAdminEngramRoute(service, requireAdminActor))
	memory.Delete("/engrams/{engram_id}", deleteMemoryAdminEngramRoute(service, requireAdminActor))
	memory.Post("/engrams/{engram_id}/restore", restoreMemoryAdminEngramRoute(service, requireAdminActor))
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

package api

import (
	"net/http"

	"engram/internal/admin"

	"github.com/go-chi/chi/v5"
)

func mountMemoryAdminSessionRoutes(memory chi.Router, service MemoryAdminService, requireAdminActor RequireAdminActor) {
	memory.Get("/sessions", listMemoryAdminSessionsRoute(service, requireAdminActor))
	memory.Delete("/sessions/{session_id}", deleteMemoryAdminSessionRoute(service, requireAdminActor))
	memory.Post("/sessions/{session_id}/restore", restoreMemoryAdminSessionRoute(service, requireAdminActor))
}

func listMemoryAdminSessionsRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if _, ok := requireActor(writer, request, requireAdminActor); !ok {
			return
		}
		listRequest, ok := parseMemoryAdminListRequest(writer, request, true)
		if !ok {
			return
		}
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return service.ListSessions(request.Context(), listRequest)
		})
	}
}

func deleteMemoryAdminSessionRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
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
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return service.DeleteSession(request.Context(), sessionID, actor.UserID, payload)
		})
	}
}

func restoreMemoryAdminSessionRoute(service MemoryAdminService, requireAdminActor RequireAdminActor) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if _, ok := requireActor(writer, request, requireAdminActor); !ok {
			return
		}
		sessionID, ok := parsePathUUID(writer, request, "session_id")
		if !ok {
			return
		}
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return service.RestoreSession(request.Context(), sessionID)
		})
	}
}

package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

func adminActorRoute(
	requireAdminActor RequireAdminActor,
	execute func(writer http.ResponseWriter, request *http.Request, actor AdminActor),
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actor, ok := requireActor(writer, request, requireAdminActor)
		if !ok {
			return
		}
		execute(writer, request, actor)
	}
}

func adminActorPayloadRoute[Payload any](
	requireAdminActor RequireAdminActor,
	execute func(ctx context.Context, payload Payload) (any, error),
) http.HandlerFunc {
	return adminActorRoute(requireAdminActor, func(writer http.ResponseWriter, request *http.Request, _ AdminActor) {
		var payload Payload
		if !decodeJSONAllowEmpty(writer, request, &payload) {
			return
		}
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return execute(request.Context(), payload)
		})
	})
}

func adminActorListRoute[Request any](
	requireAdminActor RequireAdminActor,
	parseRequest func(writer http.ResponseWriter, request *http.Request) (Request, bool),
	execute func(ctx context.Context, request Request) (any, error),
) http.HandlerFunc {
	return adminActorRoute(requireAdminActor, func(writer http.ResponseWriter, request *http.Request, _ AdminActor) {
		listRequest, ok := parseRequest(writer, request)
		if !ok {
			return
		}
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return execute(request.Context(), listRequest)
		})
	})
}

func engramPathRoute(
	requireAdminActor RequireAdminActor,
	execute func(ctx context.Context, engramID uuid.UUID) (any, error),
) http.HandlerFunc {
	return adminActorRoute(requireAdminActor, func(writer http.ResponseWriter, request *http.Request, _ AdminActor) {
		engramID, ok := parsePathUUID(writer, request, "engram_id")
		if !ok {
			return
		}
		writeServiceCall(writer, http.StatusOK, func() (any, error) {
			return execute(request.Context(), engramID)
		})
	})
}

func engramActorPathPayloadRoute[Payload any](
	requireAdminActor RequireAdminActor,
	execute func(ctx context.Context, actor AdminActor, engramID uuid.UUID, payload Payload) (any, error),
) http.HandlerFunc {
	return adminActorRoute(requireAdminActor, func(writer http.ResponseWriter, request *http.Request, actor AdminActor) {
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
	})
}

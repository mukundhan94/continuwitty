package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

func handleEngramLinkQueryRoute[T any, R any](
	dependencies sessionAuthDependencies,
	writer http.ResponseWriter,
	request *http.Request,
	service func(context.Context, T) (R, error),
	decode func(http.ResponseWriter, *http.Request, uuid.UUID, uuid.UUID) (T, bool),
) {
	actor, ok := dependencies.requireAuthenticatedAPIActor(writer, request)
	if !ok {
		return
	}
	if service == nil {
		writeSessionUserDependenciesError(writer)
		return
	}
	sourceEngramID, ok := parsePathUUID(writer, request, "engram_id")
	if !ok {
		return
	}
	input, ok := decode(writer, request, sourceEngramID, actor.UserID)
	if !ok {
		return
	}
	result, err := service(request.Context(), input)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func (dependencies sessionAuthDependencies) handleSuggestEngramLinks(writer http.ResponseWriter, request *http.Request) {
	handleEngramLinkQueryRoute(
		dependencies,
		writer,
		request,
		dependencies.suggestEngramLinks,
		decodeSuggestEngramLinksInput,
	)
}

func (dependencies sessionAuthDependencies) handleTraceEngramLinks(writer http.ResponseWriter, request *http.Request) {
	handleEngramLinkQueryRoute(
		dependencies,
		writer,
		request,
		dependencies.traceEngramLinks,
		decodeTraceEngramLinksInput,
	)
}

func (dependencies sessionAuthDependencies) handleHygieneEngramLinks(writer http.ResponseWriter, request *http.Request) {
	handleEngramLinkQueryRoute(
		dependencies,
		writer,
		request,
		dependencies.hygieneEngramLinks,
		decodeHygieneEngramLinksInput,
	)
}

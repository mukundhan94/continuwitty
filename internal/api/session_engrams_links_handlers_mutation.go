package api

import "net/http"

func (dependencies sessionAuthDependencies) handleCreateEngramLink(writer http.ResponseWriter, request *http.Request) {
	actor, ok := dependencies.requireAuthenticatedAPIActor(writer, request)
	if !ok {
		return
	}
	if dependencies.createEngramLink == nil {
		writeSessionUserDependenciesError(writer)
		return
	}
	sourceEngramID, ok := parsePathUUID(writer, request, "engram_id")
	if !ok {
		return
	}
	input, ok := decodeCreateEngramLinkInput(writer, request, sourceEngramID, actor.UserID)
	if !ok {
		return
	}
	created, err := dependencies.createEngramLink(request.Context(), input)
	if err != nil {
		writeEngramLinkServiceError(writer, err)
		return
	}
	if created == nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "Engram source or target not found"})
		return
	}
	writeJSON(writer, http.StatusOK, created)
}

func (dependencies sessionAuthDependencies) handleListEngramLinks(writer http.ResponseWriter, request *http.Request) {
	actor, ok := dependencies.requireAuthenticatedAPIActor(writer, request)
	if !ok {
		return
	}
	if dependencies.listEngramLinks == nil {
		writeSessionUserDependenciesError(writer)
		return
	}
	sourceEngramID, ok := parsePathUUID(writer, request, "engram_id")
	if !ok {
		return
	}
	input, ok := decodeListEngramLinksInput(writer, request, sourceEngramID, actor.UserID)
	if !ok {
		return
	}
	links, err := dependencies.listEngramLinks(request.Context(), input)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	writeJSON(writer, http.StatusOK, links)
}

func (dependencies sessionAuthDependencies) handleUpdateEngramLink(writer http.ResponseWriter, request *http.Request) {
	actor, ok := dependencies.requireAuthenticatedAPIActor(writer, request)
	if !ok {
		return
	}
	if dependencies.updateEngramLink == nil {
		writeSessionUserDependenciesError(writer)
		return
	}
	linkID, ok := parsePathUUID(writer, request, "link_id")
	if !ok {
		return
	}
	input, ok := decodeUpdateEngramLinkInput(writer, request, linkID, actor.UserID)
	if !ok {
		return
	}
	updated, err := dependencies.updateEngramLink(request.Context(), input)
	if err != nil {
		writeEngramLinkServiceError(writer, err)
		return
	}
	if updated == nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "Engram link not found"})
		return
	}
	writeJSON(writer, http.StatusOK, updated)
}

func (dependencies sessionAuthDependencies) handleArchiveEngramLink(writer http.ResponseWriter, request *http.Request) {
	actor, ok := dependencies.requireAuthenticatedAPIActor(writer, request)
	if !ok {
		return
	}
	if dependencies.archiveEngramLink == nil {
		writeSessionUserDependenciesError(writer)
		return
	}
	linkID, ok := parsePathUUID(writer, request, "link_id")
	if !ok {
		return
	}
	archived, err := dependencies.archiveEngramLink(
		request.Context(),
		SessionEngramLinkArchiveInput{LinkID: linkID, ActorUserID: actor.UserID},
	)
	if err != nil {
		writeEngramLinkServiceError(writer, err)
		return
	}
	if archived == nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "Engram link not found"})
		return
	}
	writeJSON(writer, http.StatusOK, archived)
}

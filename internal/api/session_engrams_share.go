package api

import (
	"context"
	"errors"
	"net/http"

	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

func (dependencies sessionAuthDependencies) handleShareEngram(writer http.ResponseWriter, request *http.Request) {
	dependencies.handleEngramVisibilityMutation(writer, request, dependencies.shareEngram)
}

func (dependencies sessionAuthDependencies) handleUnshareEngram(writer http.ResponseWriter, request *http.Request) {
	dependencies.handleEngramVisibilityMutation(writer, request, dependencies.unshareEngram)
}

func (dependencies sessionAuthDependencies) handleEngramVisibilityMutation(
	writer http.ResponseWriter,
	request *http.Request,
	operation func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		engramID uuid.UUID,
	) (*models.EngramVisibilityRecord, error),
) {
	actor, ok := dependencies.requireAuthenticatedAPIActor(writer, request)
	if !ok {
		return
	}
	engramID, ok := parsePathUUID(writer, request, "engram_id")
	if !ok {
		return
	}
	if operation == nil {
		writeSessionUserDependenciesError(writer)
		return
	}
	updated, err := operation(request.Context(), actor.UserID, actor.Role, engramID)
	if err != nil {
		statusCode, detail := mapEngramVisibilityError(err)
		writeJSON(writer, statusCode, map[string]string{"detail": detail})
		return
	}
	if updated == nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": projects.ErrEngramNotFound.Error()})
		return
	}
	writeJSON(writer, http.StatusOK, updated)
}

func mapEngramVisibilityError(err error) (int, string) {
	switch {
	case errors.Is(err, projects.ErrEngramNotFound):
		return http.StatusNotFound, projects.ErrEngramNotFound.Error()
	case errors.Is(err, projects.ErrEngramShareForbidden),
		errors.Is(err, projects.ErrProjectWriteForbidden):
		return http.StatusForbidden, err.Error()
	default:
		return http.StatusInternalServerError, "internal error"
	}
}

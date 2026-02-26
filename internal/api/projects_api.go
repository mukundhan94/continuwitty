package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"engram/internal/models"
	"engram/internal/projects"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	defaultProjectListLimit  = 500
	minProjectListLimit      = 1
	maxProjectListLimit      = 1000
	defaultProjectListOffset = 0
)

var (
	errInvalidProjectQueryParam = errors.New("invalid project query parameter")
	errInvalidProjectActor      = errors.New("invalid project actor")
	errProjectNameRequired      = errors.New("name must not be blank")
)

// ProjectListRouteRequest captures list-project filters and actor scope.
type ProjectListRouteRequest struct {
	ActorUserID     uuid.UUID
	ActorRole       models.UserRole
	IncludeArchived bool
	Limit           int
	Offset          int
}

// ProjectCreateRouteRequest captures create-project payload and actor scope.
type ProjectCreateRouteRequest struct {
	ActorUserID uuid.UUID
	ActorRole   models.UserRole
	Payload     models.ProjectCreateRequest
}

// ProjectDefaultUpdateRouteRequest captures default-project update input.
type ProjectDefaultUpdateRouteRequest struct {
	ActorUserID uuid.UUID
	ActorRole   models.UserRole
	ProjectID   string
}

// ProjectService captures project route behavior used by REST handlers.
type ProjectService interface {
	ListProjects(ctx context.Context, request ProjectListRouteRequest) ([]models.ProjectRecord, error)
	CreateProject(ctx context.Context, request ProjectCreateRouteRequest) (*models.ProjectRecord, error)
	GetDefaultProjectID(ctx context.Context, actorUserID uuid.UUID) (*string, error)
	SetDefaultProjectID(ctx context.Context, request ProjectDefaultUpdateRouteRequest) (string, error)
}

func projectActorFromRequest(request *http.Request) (uuid.UUID, models.UserRole, error) {
	actor, ok := AdminActorFromContext(request.Context())
	if !ok {
		return uuid.Nil, "", errInvalidProjectActor
	}
	role, err := models.ParseUserRole(strings.TrimSpace(actor.Role))
	if err != nil {
		return uuid.Nil, "", errInvalidProjectActor
	}
	return actor.UserID, role, nil
}

func requireProjectActor(writer http.ResponseWriter, request *http.Request) (uuid.UUID, models.UserRole, bool) {
	actorUserID, actorRole, err := projectActorFromRequest(request)
	if err != nil {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"detail": "Unauthorized"})
		return uuid.Nil, "", false
	}
	return actorUserID, actorRole, true
}

func parseProjectListQuery(
	request *http.Request,
) (includeArchived bool, limit int, offset int, err error) {
	includeArchived, err = parseProjectIncludeArchivedQuery(request.URL.Query().Get("include_archived"))
	if err != nil {
		return false, 0, 0, err
	}
	limit, err = parseBoundedIntQueryParam(
		request.URL.Query().Get("limit"),
		defaultProjectListLimit,
		minProjectListLimit,
		maxProjectListLimit,
	)
	if err != nil {
		return false, 0, 0, errInvalidProjectQueryParam
	}
	offset, err = parseBoundedIntQueryParam(
		request.URL.Query().Get("offset"),
		defaultProjectListOffset,
		defaultProjectListOffset,
		int(^uint(0)>>1),
	)
	if err != nil {
		return false, 0, 0, errInvalidProjectQueryParam
	}
	return includeArchived, limit, offset, nil
}

func parseProjectIncludeArchivedQuery(rawValue string) (bool, error) {
	if strings.TrimSpace(rawValue) == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(rawValue)
	if err != nil {
		return false, errInvalidProjectQueryParam
	}
	return parsed, nil
}

func writeProjectRouteError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, projects.ErrProjectIDMustNotBeBlank),
		errors.Is(err, errProjectNameRequired):
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"detail": err.Error()})
	case errors.Is(err, projects.ErrProjectNotFound),
		errors.Is(err, projects.ErrUserNotFound):
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": err.Error()})
	default:
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "Internal server error"})
	}
}

func decodeProjectCreatePayload(request *http.Request) (models.ProjectCreateRequest, error) {
	payload := models.ProjectCreateRequest{}
	if err := decodeChatPayload(request, &payload); err != nil {
		return models.ProjectCreateRequest{}, err
	}
	if strings.TrimSpace(payload.ProjectID) == "" {
		return models.ProjectCreateRequest{}, projects.ErrProjectIDMustNotBeBlank
	}
	if strings.TrimSpace(payload.Name) == "" {
		return models.ProjectCreateRequest{}, errProjectNameRequired
	}
	return payload, nil
}

func listProjectsHandler(service ProjectService) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorUserID, actorRole, ok := requireProjectActor(writer, request)
		if !ok {
			return
		}
		includeArchived, limit, offset, err := parseProjectListQuery(request)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid query parameters"})
			return
		}
		records, err := service.ListProjects(
			request.Context(),
			ProjectListRouteRequest{
				ActorUserID:     actorUserID,
				ActorRole:       actorRole,
				IncludeArchived: includeArchived,
				Limit:           limit,
				Offset:          offset,
			},
		)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, records)
	}
}

func createProjectHandler(service ProjectService) http.HandlerFunc {
	return projectPayloadRouteHandler(
		decodeProjectCreatePayload,
		func(
			ctx context.Context,
			actorUserID uuid.UUID,
			actorRole models.UserRole,
			payload models.ProjectCreateRequest,
		) (*models.ProjectRecord, error) {
			return service.CreateProject(
				ctx,
				ProjectCreateRouteRequest{
					ActorUserID: actorUserID,
					ActorRole:   actorRole,
					Payload:     payload,
				},
			)
		},
		func(writer http.ResponseWriter, created *models.ProjectRecord) {
			writeJSON(writer, http.StatusCreated, created)
		},
	)
}

func getDefaultProjectHandler(service ProjectService) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorUserID, _, ok := requireProjectActor(writer, request)
		if !ok {
			return
		}
		defaultProjectID, err := service.GetDefaultProjectID(request.Context(), actorUserID)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, models.ProjectDefaultResponse{DefaultProjectID: defaultProjectID})
	}
}

func decodeDefaultProjectUpdatePayload(request *http.Request) (models.ProjectDefaultUpdateRequest, error) {
	payload := models.ProjectDefaultUpdateRequest{}
	if err := decodeChatPayload(request, &payload); err != nil {
		return models.ProjectDefaultUpdateRequest{}, err
	}
	if strings.TrimSpace(payload.ProjectID) == "" {
		return models.ProjectDefaultUpdateRequest{}, projects.ErrProjectIDMustNotBeBlank
	}
	return payload, nil
}

func setDefaultProjectHandler(service ProjectService) http.HandlerFunc {
	return projectPayloadRouteHandler(
		decodeDefaultProjectUpdatePayload,
		func(
			ctx context.Context,
			actorUserID uuid.UUID,
			actorRole models.UserRole,
			payload models.ProjectDefaultUpdateRequest,
		) (string, error) {
			return service.SetDefaultProjectID(
				ctx,
				ProjectDefaultUpdateRouteRequest{
					ActorUserID: actorUserID,
					ActorRole:   actorRole,
					ProjectID:   payload.ProjectID,
				},
			)
		},
		func(writer http.ResponseWriter, defaultProjectID string) {
			writeJSON(
				writer,
				http.StatusOK,
				models.ProjectDefaultResponse{DefaultProjectID: &defaultProjectID},
			)
		},
	)
}

func projectPayloadRouteHandler[Payload any, Result any](
	decodePayload func(request *http.Request) (Payload, error),
	operation func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		payload Payload,
	) (Result, error),
	writeSuccess func(writer http.ResponseWriter, result Result),
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorUserID, actorRole, ok := requireProjectActor(writer, request)
		if !ok {
			return
		}
		payload, err := decodePayload(request)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		result, err := operation(request.Context(), actorUserID, actorRole, payload)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		writeSuccess(writer, result)
	}
}

// MountProjectRoutes registers projects REST routes.
func MountProjectRoutes(router chi.Router, service ProjectService) {
	if service == nil {
		return
	}
	router.Get("/api/v1/projects", listProjectsHandler(service))
	router.Post("/api/v1/projects", createProjectHandler(service))
	router.Get("/api/v1/projects/default", getDefaultProjectHandler(service))
	router.Patch("/api/v1/projects/default", setDefaultProjectHandler(service))
}

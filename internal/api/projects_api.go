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
	defaultProjectListLimit    = 500
	minProjectListLimit        = 1
	maxProjectListLimit        = 1000
	defaultProjectListOffset   = 0
	defaultProjectMemberLimit  = 500
	maxProjectMemberLimit      = 1000
	defaultProjectMemberOffset = 0
	defaultProjectAuditLimit   = 200
	maxProjectAuditLimit       = 1000
	defaultProjectAuditOffset  = 0
)

var (
	errInvalidProjectQueryParam = errors.New("invalid project query parameter")
	errInvalidProjectActor      = errors.New("invalid project actor")
	errProjectNameRequired      = errors.New("name must not be blank")
	errProjectMemberRoleInvalid = errors.New("role must be one of owner/editor/viewer")
	errProjectMemberUserInvalid = errors.New("user_id must be a valid UUID")
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

// ProjectMemberListRouteRequest captures member-list filters.
type ProjectMemberListRouteRequest struct {
	ActorUserID    uuid.UUID
	ActorRole      models.UserRole
	ProjectID      string
	IncludeRevoked bool
	Limit          int
	Offset         int
}

// ProjectMemberCreateRouteRequest captures member-create payload.
type ProjectMemberCreateRouteRequest struct {
	ActorUserID uuid.UUID
	ActorRole   models.UserRole
	ProjectID   string
	Payload     models.ProjectMemberCreateRequest
}

// ProjectMemberUpdateRouteRequest captures member-update payload.
type ProjectMemberUpdateRouteRequest struct {
	ActorUserID uuid.UUID
	ActorRole   models.UserRole
	ProjectID   string
	UserID      uuid.UUID
	Payload     models.ProjectMemberUpdateRequest
}

// ProjectMemberDeleteRouteRequest captures member-delete payload.
type ProjectMemberDeleteRouteRequest struct {
	ActorUserID uuid.UUID
	ActorRole   models.UserRole
	ProjectID   string
	UserID      uuid.UUID
}

// ProjectAuditListRouteRequest captures audit-list filters.
type ProjectAuditListRouteRequest struct {
	ActorUserID uuid.UUID
	ActorRole   models.UserRole
	ProjectID   string
	Limit       int
	Offset      int
}

// ProjectService captures project route behavior used by REST handlers.
type ProjectService interface {
	ListProjects(ctx context.Context, request ProjectListRouteRequest) ([]models.ProjectRecord, error)
	CreateProject(ctx context.Context, request ProjectCreateRouteRequest) (*models.ProjectRecord, error)
	GetDefaultProjectID(ctx context.Context, actorUserID uuid.UUID) (*string, error)
	SetDefaultProjectID(ctx context.Context, request ProjectDefaultUpdateRouteRequest) (string, error)
	ListProjectMembers(ctx context.Context, request ProjectMemberListRouteRequest) ([]models.ProjectMemberRecord, error)
	AddProjectMember(ctx context.Context, request ProjectMemberCreateRouteRequest) (*models.ProjectMemberRecord, error)
	UpdateProjectMember(ctx context.Context, request ProjectMemberUpdateRouteRequest) (*models.ProjectMemberRecord, error)
	RemoveProjectMember(ctx context.Context, request ProjectMemberDeleteRouteRequest) error
	ListProjectAuditEvents(ctx context.Context, request ProjectAuditListRouteRequest) ([]models.ProjectAuditEventRecord, error)
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

func parseProjectMemberListQuery(request *http.Request) (includeRevoked bool, limit int, offset int, err error) {
	includeRevoked, err = parseProjectIncludeArchivedQuery(request.URL.Query().Get("include_revoked"))
	if err != nil {
		return false, 0, 0, errInvalidProjectQueryParam
	}
	limit, err = parseBoundedIntQueryParam(
		request.URL.Query().Get("limit"),
		defaultProjectMemberLimit,
		minProjectListLimit,
		maxProjectMemberLimit,
	)
	if err != nil {
		return false, 0, 0, errInvalidProjectQueryParam
	}
	offset, err = parseBoundedIntQueryParam(
		request.URL.Query().Get("offset"),
		defaultProjectMemberOffset,
		defaultProjectMemberOffset,
		int(^uint(0)>>1),
	)
	if err != nil {
		return false, 0, 0, errInvalidProjectQueryParam
	}
	return includeRevoked, limit, offset, nil
}

func parseProjectAuditListQuery(request *http.Request) (limit int, offset int, err error) {
	limit, err = parseBoundedIntQueryParam(
		request.URL.Query().Get("limit"),
		defaultProjectAuditLimit,
		minProjectListLimit,
		maxProjectAuditLimit,
	)
	if err != nil {
		return 0, 0, errInvalidProjectQueryParam
	}
	offset, err = parseBoundedIntQueryParam(
		request.URL.Query().Get("offset"),
		defaultProjectAuditOffset,
		defaultProjectAuditOffset,
		int(^uint(0)>>1),
	)
	if err != nil {
		return 0, 0, errInvalidProjectQueryParam
	}
	return limit, offset, nil
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
		errors.Is(err, errProjectNameRequired),
		errors.Is(err, errProjectMemberRoleInvalid),
		errors.Is(err, errProjectMemberUserInvalid),
		errors.Is(err, projects.ErrProjectOwnerMembershipImmutable),
		errors.Is(err, projects.ErrProjectOwnerRoleNotAssignable):
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"detail": err.Error()})
	case errors.Is(err, projects.ErrProjectWriteForbidden),
		errors.Is(err, projects.ErrProjectMemberManagementForbidden),
		errors.Is(err, projects.ErrProjectAuditForbidden):
		writeJSON(writer, http.StatusForbidden, map[string]string{"detail": err.Error()})
	case errors.Is(err, projects.ErrProjectNotFound),
		errors.Is(err, projects.ErrUserNotFound),
		errors.Is(err, projects.ErrProjectMemberNotFound):
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

func decodeProjectMemberCreatePayload(request *http.Request) (models.ProjectMemberCreateRequest, error) {
	payload := models.ProjectMemberCreateRequest{}
	if err := decodeChatPayload(request, &payload); err != nil {
		return models.ProjectMemberCreateRequest{}, err
	}
	if payload.UserID == uuid.Nil {
		return models.ProjectMemberCreateRequest{}, errProjectMemberUserInvalid
	}
	role, err := models.ParseProjectMemberRole(strings.TrimSpace(string(payload.Role)))
	if err != nil {
		return models.ProjectMemberCreateRequest{}, errProjectMemberRoleInvalid
	}
	payload.Role = role
	return payload, nil
}

func decodeProjectMemberUpdatePayload(request *http.Request) (models.ProjectMemberUpdateRequest, error) {
	payload := models.ProjectMemberUpdateRequest{}
	if err := decodeChatPayload(request, &payload); err != nil {
		return models.ProjectMemberUpdateRequest{}, err
	}
	role, err := models.ParseProjectMemberRole(strings.TrimSpace(string(payload.Role)))
	if err != nil {
		return models.ProjectMemberUpdateRequest{}, errProjectMemberRoleInvalid
	}
	payload.Role = role
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

func projectIDFromRoute(request *http.Request) (string, bool) {
	projectID := strings.TrimSpace(chi.URLParam(request, "project_id"))
	if projectID == "" {
		return "", false
	}
	return projectID, true
}

func userIDFromRoute(request *http.Request) (uuid.UUID, bool) {
	userIDText := strings.TrimSpace(chi.URLParam(request, "user_id"))
	if userIDText == "" {
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(userIDText)
	if err != nil {
		return uuid.Nil, false
	}
	return userID, true
}

func listProjectMembersHandler(service ProjectService) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorUserID, actorRole, ok := requireProjectActor(writer, request)
		if !ok {
			return
		}
		projectID, ok := projectIDFromRoute(request)
		if !ok {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid project_id"})
			return
		}
		includeRevoked, limit, offset, err := parseProjectMemberListQuery(request)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid query parameters"})
			return
		}
		records, err := service.ListProjectMembers(
			request.Context(),
			ProjectMemberListRouteRequest{
				ActorUserID:    actorUserID,
				ActorRole:      actorRole,
				ProjectID:      projectID,
				IncludeRevoked: includeRevoked,
				Limit:          limit,
				Offset:         offset,
			},
		)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, records)
	}
}

func createProjectMemberHandler(service ProjectService) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorUserID, actorRole, ok := requireProjectActor(writer, request)
		if !ok {
			return
		}
		projectID, ok := projectIDFromRoute(request)
		if !ok {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid project_id"})
			return
		}
		payload, err := decodeProjectMemberCreatePayload(request)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		created, err := service.AddProjectMember(
			request.Context(),
			ProjectMemberCreateRouteRequest{
				ActorUserID: actorUserID,
				ActorRole:   actorRole,
				ProjectID:   projectID,
				Payload:     payload,
			},
		)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusCreated, created)
	}
}

func updateProjectMemberHandler(service ProjectService) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorUserID, actorRole, ok := requireProjectActor(writer, request)
		if !ok {
			return
		}
		projectID, ok := projectIDFromRoute(request)
		if !ok {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid project_id"})
			return
		}
		userID, ok := userIDFromRoute(request)
		if !ok {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid user_id"})
			return
		}
		payload, err := decodeProjectMemberUpdatePayload(request)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		updated, err := service.UpdateProjectMember(
			request.Context(),
			ProjectMemberUpdateRouteRequest{
				ActorUserID: actorUserID,
				ActorRole:   actorRole,
				ProjectID:   projectID,
				UserID:      userID,
				Payload:     payload,
			},
		)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, updated)
	}
}

func deleteProjectMemberHandler(service ProjectService) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorUserID, actorRole, ok := requireProjectActor(writer, request)
		if !ok {
			return
		}
		projectID, ok := projectIDFromRoute(request)
		if !ok {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid project_id"})
			return
		}
		userID, ok := userIDFromRoute(request)
		if !ok {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid user_id"})
			return
		}
		err := service.RemoveProjectMember(
			request.Context(),
			ProjectMemberDeleteRouteRequest{
				ActorUserID: actorUserID,
				ActorRole:   actorRole,
				ProjectID:   projectID,
				UserID:      userID,
			},
		)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]bool{"removed": true})
	}
}

func listProjectAuditEventsHandler(service ProjectService) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorUserID, actorRole, ok := requireProjectActor(writer, request)
		if !ok {
			return
		}
		projectID, ok := projectIDFromRoute(request)
		if !ok {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid project_id"})
			return
		}
		limit, offset, err := parseProjectAuditListQuery(request)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid query parameters"})
			return
		}
		events, err := service.ListProjectAuditEvents(
			request.Context(),
			ProjectAuditListRouteRequest{
				ActorUserID: actorUserID,
				ActorRole:   actorRole,
				ProjectID:   projectID,
				Limit:       limit,
				Offset:      offset,
			},
		)
		if err != nil {
			writeProjectRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, events)
	}
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
	router.Get("/api/v1/projects/{project_id}/members", listProjectMembersHandler(service))
	router.Post("/api/v1/projects/{project_id}/members", createProjectMemberHandler(service))
	router.Patch("/api/v1/projects/{project_id}/members/{user_id}", updateProjectMemberHandler(service))
	router.Delete("/api/v1/projects/{project_id}/members/{user_id}", deleteProjectMemberHandler(service))
	router.Get("/api/v1/projects/{project_id}/audit-events", listProjectAuditEventsHandler(service))
}

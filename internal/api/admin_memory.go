package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"engram/internal/admin"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AdminActor identifies the authenticated admin actor for memory-admin operations.
type AdminActor struct {
	UserID uuid.UUID
	Role   string
}

// RequireAdminActor resolves and authorizes the current request actor.
type RequireAdminActor func(request *http.Request) (AdminActor, error)

// MemoryAdminService captures service methods required by memory-admin HTTP handlers.
type MemoryAdminService interface {
	ListSessions(ctx context.Context, request admin.MemoryAdminListRequest) ([]models.AdminChatSessionRecord, error)
	DeleteSession(ctx context.Context, sessionID, actorUserID uuid.UUID, payload admin.SessionDeleteRequest) (admin.SessionDeleteResponse, error)
	RestoreSession(ctx context.Context, sessionID uuid.UUID) (admin.SessionRestoreResponse, error)

	ListEngrams(ctx context.Context, request admin.MemoryAdminEngramListRequest) ([]models.AdminEngramRecord, error)
	GetEngram(ctx context.Context, engramID uuid.UUID, includeDeleted bool) (*models.AdminEngramRecord, error)
	UpdateEngram(ctx context.Context, engramID, actorUserID uuid.UUID, payload admin.EngramUpdateRequest) (*models.AdminEngramRecord, error)
	MoveEngram(ctx context.Context, engramID uuid.UUID, actor admin.WriteActor, payload admin.EngramMoveRequest) (*models.AdminEngramRecord, error)
	DeleteEngram(ctx context.Context, engramID, actorUserID uuid.UUID, payload admin.EngramDeleteRequest) (admin.EngramDeleteResponse, error)
	RestoreEngram(ctx context.Context, engramID uuid.UUID) (admin.EngramRestoreResponse, error)
	RefreshEngramFreshness(ctx context.Context, request admin.EngramFreshnessRefreshRequest) (admin.EngramFreshnessRefreshResponse, error)
	RefreshEngramConsolidationSuggestions(
		ctx context.Context,
		request admin.EngramConsolidationSuggestionRefreshRequest,
	) (admin.EngramConsolidationSuggestionRefreshResponse, error)
	ListEngramConsolidationSuggestions(
		ctx context.Context,
		request admin.EngramConsolidationSuggestionListRequest,
	) ([]models.EngramConsolidationSuggestion, error)
	ActionEngramConsolidationSuggestion(
		ctx context.Context,
		suggestionID uuid.UUID,
		actorUserID uuid.UUID,
		request admin.EngramConsolidationSuggestionActionRequest,
	) (*models.EngramConsolidationSuggestion, error)
	RefreshEngramContradictionAlerts(
		ctx context.Context,
		request admin.EngramContradictionAlertRefreshRequest,
	) (admin.EngramContradictionAlertRefreshResponse, error)
	ListEngramContradictionAlerts(
		ctx context.Context,
		request admin.EngramContradictionAlertListRequest,
	) ([]models.EngramContradictionAlert, error)
	ResolveEngramContradictionAlert(
		ctx context.Context,
		alertID uuid.UUID,
		actorUserID uuid.UUID,
		request admin.EngramContradictionAlertResolveRequest,
	) (*models.EngramContradictionAlert, error)
	ListMemoryCurationSuggestions(
		ctx context.Context,
		request admin.MemoryCurationSuggestionListRequest,
	) ([]models.MemoryCurationSuggestion, error)
	ActionMemoryCurationSuggestion(
		ctx context.Context,
		suggestionID uuid.UUID,
		actorUserID uuid.UUID,
		request admin.MemoryCurationSuggestionActionRequest,
	) (*models.MemoryCurationSuggestion, error)

	ListCollections(ctx context.Context, request admin.MemoryAdminListRequest) ([]models.EngramCollectionRecord, error)
	CreateCollection(ctx context.Context, actorUserID uuid.UUID, actorRole string, payload admin.CollectionCreateRequest) (*models.EngramCollectionRecord, error)
	UpdateCollection(ctx context.Context, collectionID uuid.UUID, payload admin.CollectionUpdateRequest) (*models.EngramCollectionRecord, error)
	DeleteCollection(ctx context.Context, collectionID, actorUserID uuid.UUID, payload admin.CollectionDeleteRequest) (admin.CollectionDeleteResponse, error)
	AddCollectionItems(ctx context.Context, collectionID, actorUserID uuid.UUID, payload admin.CollectionItemsUpdateRequest) (admin.CollectionItemsAddResponse, error)
	RemoveCollectionItem(ctx context.Context, collectionID, engramID uuid.UUID) (admin.CollectionItemRemoveResponse, error)
}

// MountMemoryAdminRoutes registers /api/v1/admin/memory routes on the provided router.
func MountMemoryAdminRoutes(router chi.Router, service MemoryAdminService, requireAdminActor RequireAdminActor) {
	router.Route("/api/v1/admin/memory", func(memory chi.Router) {
		mountMemoryAdminSessionRoutes(memory, service, requireAdminActor)
		mountMemoryAdminEngramRoutes(memory, service, requireAdminActor)
		mountMemoryAdminCollectionRoutes(memory, service, requireAdminActor)
	})
}

func writeServiceCall(writer http.ResponseWriter, successStatus int, execute func() (any, error)) {
	response, err := execute()
	if err != nil {
		writeServiceError(writer, err)
		return
	}
	writeJSON(writer, successStatus, response)
}

func parseMemoryAdminListRequest(
	writer http.ResponseWriter,
	request *http.Request,
	includeOwnerUserFilter bool,
) (admin.MemoryAdminListRequest, bool) {
	parsed := admin.MemoryAdminListRequest{
		ProjectID: optionalTrimmedString(request.URL.Query().Get("project_id")),
	}
	if includeOwnerUserFilter {
		ownerUserID, ok := parseOptionalUUIDQuery(writer, request, "owner_user_id")
		if !ok {
			return admin.MemoryAdminListRequest{}, false
		}
		parsed.OwnerUserID = ownerUserID
	}
	includeDeleted, ok := parseOptionalBoolQuery(writer, request, "include_deleted", false)
	if !ok {
		return admin.MemoryAdminListRequest{}, false
	}
	parsed.IncludeDeleted = includeDeleted
	limit, ok := parseOptionalIntQuery(writer, request, "limit", intQuerySpec{Default: 200, Min: 1, Max: 1000})
	if !ok {
		return admin.MemoryAdminListRequest{}, false
	}
	parsed.Limit = limit
	offset, ok := parseOptionalIntQuery(writer, request, "offset", intQuerySpec{Default: 0, Min: 0, Max: 1_000_000})
	if !ok {
		return admin.MemoryAdminListRequest{}, false
	}
	parsed.Offset = offset
	return parsed, true
}

func parseMemoryAdminEngramListRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (admin.MemoryAdminEngramListRequest, bool) {
	base, ok := parseMemoryAdminListRequest(writer, request, false)
	if !ok {
		return admin.MemoryAdminEngramListRequest{}, false
	}
	sessionID, ok := parseOptionalUUIDQuery(writer, request, "session_id")
	if !ok {
		return admin.MemoryAdminEngramListRequest{}, false
	}
	return admin.MemoryAdminEngramListRequest{
		MemoryAdminListRequest: base,
		SessionID:              sessionID,
		QueryText:              optionalTrimmedString(request.URL.Query().Get("q")),
	}, true
}

func parseMemoryAdminConsolidationSuggestionListRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (admin.EngramConsolidationSuggestionListRequest, bool) {
	projectID, status, limit, offset, ok := parseMemoryAdminProjectStatusListRequest(
		writer,
		request,
		func(w http.ResponseWriter, r *http.Request, key string) (*models.ConsolidationSuggestionStatus, bool) {
			return parseOptionalStatusQuery(w, r, key, models.ParseConsolidationSuggestionStatus)
		},
	)
	if !ok {
		return admin.EngramConsolidationSuggestionListRequest{}, false
	}
	return admin.EngramConsolidationSuggestionListRequest{
		ProjectID: projectID,
		Status:    status,
		Limit:     limit,
		Offset:    offset,
	}, true
}

func parseMemoryAdminCurationSuggestionListRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (admin.MemoryCurationSuggestionListRequest, bool) {
	base, ok := parseMemoryAdminListRequest(writer, request, false)
	if !ok {
		return admin.MemoryCurationSuggestionListRequest{}, false
	}
	sessionID, ok := parseOptionalUUIDQuery(writer, request, "session_id")
	if !ok {
		return admin.MemoryCurationSuggestionListRequest{}, false
	}
	suggestionType, ok := parseOptionalStatusQuery(
		writer,
		request,
		"suggestion_type",
		models.ParseMemoryCurationSuggestionType,
	)
	if !ok {
		return admin.MemoryCurationSuggestionListRequest{}, false
	}
	status, ok := parseOptionalStatusQuery(
		writer,
		request,
		"status",
		models.ParseMemoryCurationSuggestionStatus,
	)
	if !ok {
		return admin.MemoryCurationSuggestionListRequest{}, false
	}
	return admin.MemoryCurationSuggestionListRequest{
		ProjectID:      base.ProjectID,
		SessionID:      sessionID,
		SuggestionType: suggestionType,
		Status:         status,
		Limit:          base.Limit,
		Offset:         base.Offset,
	}, true
}

func parseMemoryAdminProjectStatusListRequest[Status any](
	writer http.ResponseWriter,
	request *http.Request,
	parseStatus func(http.ResponseWriter, *http.Request, string) (*Status, bool),
) (*string, *Status, int, int, bool) {
	base, ok := parseMemoryAdminListRequest(writer, request, false)
	if !ok {
		return nil, nil, 0, 0, false
	}
	status, ok := parseStatus(writer, request, "status")
	if !ok {
		return nil, nil, 0, 0, false
	}
	return base.ProjectID, status, base.Limit, base.Offset, true
}

func parseOptionalStatusQuery[Status any](
	writer http.ResponseWriter,
	request *http.Request,
	key string,
	parseStatusValue func(string) (Status, error),
) (*Status, bool) {
	return parseOptionalQueryValue(writer, request, key, parseStatusValue)
}

func requireActor(writer http.ResponseWriter, request *http.Request, requireAdminActor RequireAdminActor) (AdminActor, bool) {
	if requireAdminActor == nil {
		writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "forbidden"})
		return AdminActor{}, false
	}
	actor, err := requireAdminActor(request)
	if err != nil {
		writeJSON(writer, http.StatusForbidden, map[string]string{"detail": "forbidden"})
		return AdminActor{}, false
	}
	return actor, true
}

func parsePathUUID(writer http.ResponseWriter, request *http.Request, param string) (uuid.UUID, bool) {
	return parseRequiredUUID(writer, chi.URLParam(request, param), param)
}

func parseOptionalUUIDQuery(writer http.ResponseWriter, request *http.Request, key string) (*uuid.UUID, bool) {
	value := strings.TrimSpace(request.URL.Query().Get(key))
	if value == "" {
		return nil, true
	}
	parsed, ok := parseRequiredUUID(writer, value, key)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func parseOptionalBoolQuery(writer http.ResponseWriter, request *http.Request, key string, defaultValue bool) (bool, bool) {
	parsed, ok := parseOptionalQueryValue(writer, request, key, strconv.ParseBool)
	if !ok {
		return false, false
	}
	if parsed == nil {
		return defaultValue, true
	}
	return *parsed, true
}

type intQuerySpec struct {
	Default int
	Min     int
	Max     int
}

func parseOptionalIntQuery(writer http.ResponseWriter, request *http.Request, key string, spec intQuerySpec) (int, bool) {
	value := strings.TrimSpace(request.URL.Query().Get(key))
	if value == "" {
		return spec.Default, true
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		writeInvalidParameter(writer, key)
		return 0, false
	}
	if parsed < spec.Min {
		writeInvalidParameter(writer, key)
		return 0, false
	}
	if parsed > spec.Max {
		writeInvalidParameter(writer, key)
		return 0, false
	}
	return parsed, true
}

func parseOptionalQueryValue[T any](
	writer http.ResponseWriter,
	request *http.Request,
	key string,
	parse func(string) (T, error),
) (*T, bool) {
	value := strings.TrimSpace(request.URL.Query().Get(key))
	if value == "" {
		return nil, true
	}
	parsed, err := parse(value)
	if err != nil {
		writeInvalidParameter(writer, key)
		return nil, false
	}
	return &parsed, true
}

func parseRequiredUUID(writer http.ResponseWriter, rawValue string, key string) (uuid.UUID, bool) {
	parsed, err := uuid.Parse(strings.TrimSpace(rawValue))
	if err != nil {
		writeInvalidParameter(writer, key)
		return uuid.Nil, false
	}
	return parsed, true
}

func writeInvalidParameter(writer http.ResponseWriter, key string) {
	writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid " + key})
}

func optionalTrimmedString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func decodeJSONAllowEmpty(writer http.ResponseWriter, request *http.Request, destination any) bool {
	if request.Body == nil {
		return true
	}
	defer request.Body.Close()

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid json body"})
		return false
	}
	return true
}

func writeServiceError(writer http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	detail := "internal error"
	switch {
	case errors.Is(err, admin.ErrSessionNotFound),
		errors.Is(err, admin.ErrEngramNotFound),
		errors.Is(err, admin.ErrCollectionNotFound),
		errors.Is(err, admin.ErrConsolidationSuggestionNotFound),
		errors.Is(err, admin.ErrContradictionAlertNotFound),
		errors.Is(err, admin.ErrMemoryCurationSuggestionNotFound):
		statusCode = http.StatusNotFound
		detail = err.Error()
	case errors.Is(err, admin.ErrProjectIDRequired):
		statusCode = http.StatusBadRequest
		detail = err.Error()
	case errors.Is(err, admin.ErrConsolidationMinGroupSizeInvalid):
		statusCode = http.StatusBadRequest
		detail = err.Error()
	case errors.Is(err, admin.ErrConsolidationSuggestionActionInvalid),
		errors.Is(err, admin.ErrContradictionAlertResolveStatusInvalid),
		errors.Is(err, admin.ErrMemoryCurationSuggestionActionInvalid),
		errors.Is(err, admin.ErrMemoryCurationSuggestionPayloadInvalid),
		errors.Is(err, admin.ErrMemoryCurationSuggestionApplyUnsupported):
		statusCode = http.StatusBadRequest
		detail = err.Error()
	case errors.Is(err, admin.ErrEngramStale),
		errors.Is(err, admin.ErrCollectionStale),
		errors.Is(err, repository.ErrCollectionNameExists):
		statusCode = http.StatusConflict
		detail = err.Error()
	}
	writeJSON(writer, statusCode, map[string]string{"detail": detail})
}

package api

import (
	"errors"
	"net/http"
	"strings"

	"engram/internal/models"

	"github.com/google/uuid"
)

const (
	defaultEngramListLimit   = 25
	defaultEngramQueryTopK   = 5
	defaultEngramSourceLimit = 100
)

var (
	// ErrProjectIDRequiredWhenNoDefaultProject matches Python parity when write project id cannot be resolved.
	ErrProjectIDRequiredWhenNoDefaultProject = errors.New("project_id is required when no default project is configured")
	// ErrDefaultProjectNotAccessible matches Python parity when caller's default project is not visible.
	ErrDefaultProjectNotAccessible = errors.New("default project is not accessible; set a valid default project first")
	// ErrProjectWriteForbidden indicates caller lacks write access to the resolved project.
	ErrProjectWriteForbidden = errors.New("project is not writable by actor")
)

// SessionProjectResolution carries project-resolution output for write routes.
type SessionProjectResolution struct {
	ProjectID          string
	UsedDefaultProject bool
}

// SessionEngramFeedbackInput captures feedback route payload + actor context.
type SessionEngramFeedbackInput struct {
	EngramID     uuid.UUID
	ActorUserID  uuid.UUID
	FeedbackType models.EngramFeedbackType
	Note         *string
}

func (dependencies sessionAuthDependencies) handleCreateEngram(writer http.ResponseWriter, request *http.Request) {
	actor, ok := dependencies.requireAuthenticatedAPIActor(writer, request)
	if !ok {
		return
	}
	if !dependencies.hasCreateEngramDependencies() {
		writeSessionUserDependenciesError(writer)
		return
	}

	payload, ok := decodeCreateEngramRequest(writer, request)
	if !ok {
		return
	}
	resolution, ok := dependencies.resolveWriteProjectID(writer, request, actor, payload.ProjectID)
	if !ok {
		return
	}
	payload.ProjectID = resolution.ProjectID

	created, err := dependencies.createEngram(request.Context(), payload, actor.UserID)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	applyEngramCreateProjectResolution(created, resolution)
	writeJSON(writer, http.StatusOK, created)
}

func (dependencies sessionAuthDependencies) handleListEngrams(writer http.ResponseWriter, request *http.Request) {
	actor, ok := dependencies.requireAuthenticatedAPIActor(writer, request)
	if !ok {
		return
	}
	if dependencies.listEngrams == nil {
		writeSessionUserDependenciesError(writer)
		return
	}

	limit, ok := parseOptionalIntQuery(
		writer,
		request,
		"limit",
		intQuerySpec{Default: defaultEngramListLimit, Min: 1, Max: 200},
	)
	if !ok {
		return
	}
	offset, ok := parseOptionalIntQuery(
		writer,
		request,
		"offset",
		intQuerySpec{Default: 0, Min: 0, Max: 1_000_000},
	)
	if !ok {
		return
	}
	projectID := optionalTrimmedString(request.URL.Query().Get("project_id"))
	records, err := dependencies.listEngrams(request.Context(), projectID, limit, offset, actor.UserID)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	writeJSON(writer, http.StatusOK, records)
}

func (dependencies sessionAuthDependencies) handleQueryEngrams(writer http.ResponseWriter, request *http.Request) {
	actor, ok := dependencies.requireAuthenticatedAPIActor(writer, request)
	if !ok {
		return
	}
	if dependencies.queryEngrams == nil {
		writeSessionUserDependenciesError(writer)
		return
	}

	payload, ok := decodeQueryEngramsRequest(writer, request)
	if !ok {
		return
	}
	results, err := dependencies.queryEngrams(request.Context(), payload, actor.UserID)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	writeJSON(writer, http.StatusOK, results)
}

func (dependencies sessionAuthDependencies) handleListEngramSources(writer http.ResponseWriter, request *http.Request) {
	actor, ok := dependencies.requireAuthenticatedAPIActor(writer, request)
	if !ok {
		return
	}
	if !dependencies.hasEngramReadDependencies() {
		writeSessionUserDependenciesError(writer)
		return
	}

	engramID, ok := parsePathUUID(writer, request, "engram_id")
	if !ok {
		return
	}
	limit, ok := parseOptionalIntQuery(
		writer,
		request,
		"limit",
		intQuerySpec{Default: defaultEngramSourceLimit, Min: 1, Max: 500},
	)
	if !ok {
		return
	}
	if !dependencies.ensureVisibleEngram(writer, request, engramID, actor.UserID) {
		return
	}

	sources, err := dependencies.getEngramSources(request.Context(), engramID, limit, actor.UserID)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	writeJSON(writer, http.StatusOK, sources)
}

func (dependencies sessionAuthDependencies) handleRehydrateEngram(writer http.ResponseWriter, request *http.Request) {
	actor, ok := dependencies.requireAuthenticatedAPIActor(writer, request)
	if !ok {
		return
	}
	if dependencies.getRehydrationBundle == nil {
		writeSessionUserDependenciesError(writer)
		return
	}

	engramID, ok := parsePathUUID(writer, request, "engram_id")
	if !ok {
		return
	}
	bundle, err := dependencies.getRehydrationBundle(request.Context(), engramID, actor.UserID)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	if bundle == nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "Engram not found"})
		return
	}
	writeJSON(writer, http.StatusOK, bundle)
}

func (dependencies sessionAuthDependencies) handleSubmitEngramFeedback(
	writer http.ResponseWriter,
	request *http.Request,
) {
	actor, ok := dependencies.requireAuthenticatedAPIActor(writer, request)
	if !ok {
		return
	}
	if dependencies.submitEngramFeedback == nil {
		writeSessionUserDependenciesError(writer)
		return
	}
	engramID, ok := parsePathUUID(writer, request, "engram_id")
	if !ok {
		return
	}
	payload, ok := decodeEngramFeedbackRequest(writer, request)
	if !ok {
		return
	}
	record, err := dependencies.submitEngramFeedback(
		request.Context(),
		SessionEngramFeedbackInput{
			EngramID:     engramID,
			ActorUserID:  actor.UserID,
			FeedbackType: payload.FeedbackType,
			Note:         payload.Note,
		},
	)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	if record == nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "Engram not found"})
		return
	}
	writeJSON(writer, http.StatusOK, record)
}

func (dependencies sessionAuthDependencies) hasCreateEngramDependencies() bool {
	return dependencies.createEngram != nil && dependencies.resolveProjectIDForWrite != nil
}

func (dependencies sessionAuthDependencies) hasEngramReadDependencies() bool {
	return dependencies.getRehydrationBundle != nil && dependencies.getEngramSources != nil
}

func (dependencies sessionAuthDependencies) resolveWriteProjectID(
	writer http.ResponseWriter,
	request *http.Request,
	actor *models.UserAuthRecord,
	projectID string,
) (SessionProjectResolution, bool) {
	resolution, err := dependencies.resolveProjectIDForWrite(
		request.Context(),
		actor.UserID,
		actor.Role,
		projectID,
	)
	if err == nil {
		return resolution, true
	}

	statusCode, detail, mapped := mapProjectResolutionError(err)
	if !mapped {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return SessionProjectResolution{}, false
	}
	writeJSON(writer, statusCode, map[string]string{"detail": detail})
	return SessionProjectResolution{}, false
}

func (dependencies sessionAuthDependencies) ensureVisibleEngram(
	writer http.ResponseWriter,
	request *http.Request,
	engramID uuid.UUID,
	actorUserID uuid.UUID,
) bool {
	bundle, err := dependencies.getRehydrationBundle(request.Context(), engramID, actorUserID)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return false
	}
	if bundle == nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "Engram not found"})
		return false
	}
	return true
}

func decodeCreateEngramRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (models.MemoryEngramCreate, bool) {
	payload := models.MemoryEngramCreate{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return models.MemoryEngramCreate{}, false
	}
	normalizeCreateEngramPayload(&payload)
	if !isValidCreateEngramPayload(payload) {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "title and detailed_summary_markdown are required"})
		return models.MemoryEngramCreate{}, false
	}
	return payload, true
}

func decodeQueryEngramsRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (models.EngramQueryRequest, bool) {
	payload := models.EngramQueryRequest{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return models.EngramQueryRequest{}, false
	}
	payload.Query = strings.TrimSpace(payload.Query)
	if payload.Query == "" {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "query is required"})
		return models.EngramQueryRequest{}, false
	}
	if payload.TopK == 0 {
		payload.TopK = defaultEngramQueryTopK
	}
	if payload.TopK < 1 || payload.TopK > 50 {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid top_k"})
		return models.EngramQueryRequest{}, false
	}
	payload.ProjectID = normalizeOptionalProjectID(payload.ProjectID)
	if payload.Tags == nil {
		payload.Tags = []string{}
	}
	if payload.Keywords == nil {
		payload.Keywords = []string{}
	}
	return payload, true
}

func decodeEngramFeedbackRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (SessionEngramFeedbackInput, bool) {
	payload := models.EngramFeedbackCreateRequest{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return SessionEngramFeedbackInput{}, false
	}
	parsedType, err := models.ParseEngramFeedbackType(payload.FeedbackType)
	if err != nil {
		writeJSON(
			writer,
			http.StatusBadRequest,
			map[string]string{"detail": "feedback_type must be one of: useful, contradiction"},
		)
		return SessionEngramFeedbackInput{}, false
	}
	return SessionEngramFeedbackInput{
		FeedbackType: parsedType,
		Note:         payload.Note,
	}, true
}

func normalizeCreateEngramPayload(payload *models.MemoryEngramCreate) {
	payload.ProjectID = strings.TrimSpace(payload.ProjectID)
	payload.Title = strings.TrimSpace(payload.Title)
	payload.Abstract = strings.TrimSpace(payload.Abstract)
	payload.DetailedSummaryMarkdown = strings.TrimSpace(payload.DetailedSummaryMarkdown)
	payload.VisibilityScope = normalizeEngramVisibilityScope(payload.VisibilityScope)
	if payload.Decisions == nil {
		payload.Decisions = []models.Decision{}
	}
	if payload.Assumptions == nil {
		payload.Assumptions = []string{}
	}
	if payload.OpenQuestions == nil {
		payload.OpenQuestions = []string{}
	}
	if payload.Claims == nil {
		payload.Claims = []models.Claim{}
	}
	if payload.Tags == nil {
		payload.Tags = []string{}
	}
	if payload.Keywords == nil {
		payload.Keywords = []string{}
	}
	if payload.Artifacts == nil {
		payload.Artifacts = []models.ArtifactIn{}
	}
}

func isValidCreateEngramPayload(payload models.MemoryEngramCreate) bool {
	return payload.Title != "" && payload.DetailedSummaryMarkdown != ""
}

func normalizeEngramVisibilityScope(scope string) string {
	trimmed := strings.TrimSpace(scope)
	if trimmed == "" {
		return "private"
	}
	return trimmed
}

func normalizeOptionalProjectID(projectID *string) *string {
	if projectID == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*projectID)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func applyEngramCreateProjectResolution(
	response *models.EngramCreateResponse,
	resolution SessionProjectResolution,
) {
	if response == nil {
		return
	}
	resolvedProjectID := resolution.ProjectID
	response.ResolvedProjectID = &resolvedProjectID
	response.UsedDefaultProject = resolution.UsedDefaultProject
}

func mapProjectResolutionError(err error) (int, string, bool) {
	switch {
	case errors.Is(err, ErrProjectIDRequiredWhenNoDefaultProject):
		return http.StatusUnprocessableEntity, ErrProjectIDRequiredWhenNoDefaultProject.Error(), true
	case errors.Is(err, ErrDefaultProjectNotAccessible):
		return http.StatusUnprocessableEntity, ErrDefaultProjectNotAccessible.Error(), true
	case errors.Is(err, ErrProjectWriteForbidden):
		return http.StatusForbidden, ErrProjectWriteForbidden.Error(), true
	default:
		return 0, "", false
	}
}

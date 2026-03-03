package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

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
	EngramID         uuid.UUID
	SessionID        *uuid.UUID
	ActorUserID      uuid.UUID
	FeedbackType     models.EngramFeedbackType
	IntegrationDepth *models.EngramFeedbackIntegrationDepth
	Note             *string
	RelevanceScore   *int
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
			EngramID:         engramID,
			SessionID:        payload.SessionID,
			ActorUserID:      actor.UserID,
			FeedbackType:     payload.FeedbackType,
			IntegrationDepth: payload.IntegrationDepth,
			Note:             payload.Note,
			RelevanceScore:   payload.RelevanceScore,
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
	normalizeQueryEngramsPayload(&payload)
	if detail := validateQueryEngramsPayload(payload); detail != "" {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": detail})
		return models.EngramQueryRequest{}, false
	}
	return payload, true
}

func normalizeQueryEngramsPayload(payload *models.EngramQueryRequest) {
	payload.Query = strings.TrimSpace(payload.Query)
	normalizeQueryTopK(payload)
	normalizeQueryRelationType(payload)
	normalizeQueryTraceDepth(payload)
	normalizeQueryScope(payload)
}

func normalizeQueryTopK(payload *models.EngramQueryRequest) {
	if payload.TopK != 0 {
		return
	}
	payload.TopK = defaultEngramQueryTopK
}

func normalizeQueryRelationType(payload *models.EngramQueryRequest) {
	if payload.RelationType == nil {
		return
	}
	relation := strings.TrimSpace(string(*payload.RelationType))
	if relation == "" {
		payload.RelationType = nil
		return
	}
	typed := models.EngramLinkRelationType(relation)
	payload.RelationType = &typed
}

func normalizeQueryTraceDepth(payload *models.EngramQueryRequest) {
	if payload.RelationType == nil || payload.TraceDepth != nil {
		return
	}
	defaultDepth := 1
	payload.TraceDepth = &defaultDepth
}

func normalizeQueryScope(payload *models.EngramQueryRequest) {
	payload.ProjectID = normalizeOptionalProjectID(payload.ProjectID)
	payload.Tags = normalizeStringSlice(payload.Tags)
	payload.Keywords = normalizeStringSlice(payload.Keywords)
}

func normalizeStringSlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func validateQueryEngramsPayload(payload models.EngramQueryRequest) string {
	for _, rule := range queryEngramValidationRules(payload) {
		if rule.invalid {
			return rule.detail
		}
	}
	return invalidQueryTemporalWindowDetail(payload)
}

type queryEngramValidationRule struct {
	detail  string
	invalid bool
}

func queryEngramValidationRules(payload models.EngramQueryRequest) []queryEngramValidationRule {
	return []queryEngramValidationRule{
		{detail: "query is required", invalid: payload.Query == ""},
		{detail: "invalid top_k", invalid: payload.TopK < 1 || payload.TopK > 50},
		{detail: "invalid useful_count_min", invalid: invalidUsefulCountMin(payload.UsefulCountMin)},
		{detail: "invalid access_count_min", invalid: invalidAccessCountMin(payload.AccessCountMin)},
		{detail: "invalid feedback_count_min", invalid: invalidFeedbackCountMin(payload.FeedbackCountMin)},
		{
			detail:  "invalid contradiction_count_max",
			invalid: invalidContradictionCountMax(payload.ContradictionCountMax),
		},
		{detail: "invalid freshness_score_min", invalid: invalidFreshnessScoreMin(payload.FreshnessScoreMin)},
		{
			detail:  "invalid avg_relevance_feedback_min",
			invalid: invalidAvgRelevanceFeedbackMin(payload.AvgRelevanceFeedbackMin),
		},
		{
			detail:  "invalid source_session_quality_min",
			invalid: invalidSourceSessionQualityMin(payload.SourceSessionQualityMin),
		},
		{detail: "invalid relation_type", invalid: invalidRelationType(payload.RelationType)},
		{detail: "invalid trace_depth", invalid: invalidTraceDepth(payload.TraceDepth)},
		{
			detail:  "invalid trace_depth",
			invalid: relationTypeTraceDepthConflict(payload.RelationType, payload.TraceDepth),
		},
	}
}

func invalidUsefulCountMin(value *int) bool {
	return value != nil && *value < 0
}

func invalidAccessCountMin(value *int) bool {
	return value != nil && *value < 0
}

func invalidFeedbackCountMin(value *int) bool {
	return value != nil && *value < 0
}

func invalidContradictionCountMax(value *int) bool {
	return value != nil && *value < 0
}

func invalidFreshnessScoreMin(value *float64) bool {
	return invalidBoundedUnitInterval(value)
}

func invalidAvgRelevanceFeedbackMin(value *float64) bool {
	return invalidBoundedUnitInterval(value)
}

func invalidSourceSessionQualityMin(value *float64) bool {
	return invalidBoundedUnitInterval(value)
}

func invalidBoundedUnitInterval(value *float64) bool {
	if value == nil {
		return false
	}
	if *value < 0 {
		return true
	}
	return *value > 1
}

func hasInvalidTemporalWindow(after *time.Time, before *time.Time) bool {
	if after == nil || before == nil {
		return false
	}
	return after.After(*before)
}

func invalidRelationType(value *models.EngramLinkRelationType) bool {
	if value == nil {
		return false
	}
	_, err := models.ParseEngramLinkRelationType(strings.TrimSpace(string(*value)))
	return err != nil
}

func invalidTraceDepth(value *int) bool {
	if value == nil {
		return false
	}
	if *value < 0 {
		return true
	}
	return *value > 1
}

func relationTypeTraceDepthConflict(
	relationType *models.EngramLinkRelationType,
	traceDepth *int,
) bool {
	return relationType != nil && traceDepth != nil && *traceDepth == 0
}

type queryTemporalWindowSpec struct {
	after         *time.Time
	before        *time.Time
	invalidDetail string
}

func invalidQueryTemporalWindowDetail(payload models.EngramQueryRequest) string {
	specs := []queryTemporalWindowSpec{
		{
			after:         payload.CreatedAfter,
			before:        payload.CreatedBefore,
			invalidDetail: "invalid created_at window",
		},
		{
			after:         payload.LastAccessedAfter,
			before:        payload.LastAccessedBefore,
			invalidDetail: "invalid last_accessed window",
		},
		{
			after:         payload.FreshnessComputedAfter,
			before:        payload.FreshnessComputedBefore,
			invalidDetail: "invalid freshness_computed window",
		},
	}
	for _, spec := range specs {
		if hasInvalidTemporalWindow(spec.after, spec.before) {
			return spec.invalidDetail
		}
	}
	return ""
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
	relevanceScore, ok := normalizeOptionalFeedbackRelevanceScore(payload.RelevanceScore)
	if !ok {
		writeJSON(
			writer,
			http.StatusBadRequest,
			map[string]string{"detail": "relevance_score must be an integer between 1 and 5"},
		)
		return SessionEngramFeedbackInput{}, false
	}
	integrationDepth, ok := parseOptionalFeedbackIntegrationDepth(payload.IntegrationDepth)
	if !ok {
		writeJSON(
			writer,
			http.StatusBadRequest,
			map[string]string{
				"detail": "integration_depth must be one of: mentioned, elaborated, contradicted, ignored",
			},
		)
		return SessionEngramFeedbackInput{}, false
	}
	sessionID, ok := parseOptionalFeedbackSessionID(payload.SessionID)
	if !ok {
		writeJSON(
			writer,
			http.StatusBadRequest,
			map[string]string{"detail": "session_id must be a valid uuid when provided"},
		)
		return SessionEngramFeedbackInput{}, false
	}
	return SessionEngramFeedbackInput{
		FeedbackType:     parsedType,
		IntegrationDepth: integrationDepth,
		Note:             normalizeOptionalTrimmedString(payload.Note),
		RelevanceScore:   relevanceScore,
		SessionID:        sessionID,
	}, true
}

func normalizeOptionalFeedbackRelevanceScore(
	relevanceScore *int,
) (*int, bool) {
	if relevanceScore == nil {
		return nil, true
	}
	if *relevanceScore < 1 || *relevanceScore > 5 {
		return nil, false
	}
	normalized := *relevanceScore
	return &normalized, true
}

func parseOptionalFeedbackIntegrationDepth(
	integrationDepth *string,
) (*models.EngramFeedbackIntegrationDepth, bool) {
	if integrationDepth == nil {
		return nil, true
	}
	parsed, err := models.ParseEngramFeedbackIntegrationDepth(*integrationDepth)
	if err != nil {
		return nil, false
	}
	normalized := parsed
	return &normalized, true
}

func parseOptionalFeedbackSessionID(sessionID *string) (*uuid.UUID, bool) {
	if sessionID == nil {
		return nil, true
	}
	trimmed := strings.TrimSpace(*sessionID)
	if trimmed == "" {
		return nil, false
	}
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, false
	}
	return &parsed, true
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
	return normalizeOptionalTrimmedString(projectID)
}

func normalizeOptionalTrimmedString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
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

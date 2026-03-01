package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

const (
	defaultEngramLinkListLimit          = 50
	defaultEngramLinkSuggestLimit       = 5
	defaultEngramLinkSuggestMaxCandiate = 20
	defaultEngramTraceMaxDepth          = 2
	defaultEngramTraceMaxNeighbors      = 20
	defaultEngramLinkHygieneLimit       = 500
	defaultEngramLinkHygieneStaleDays   = 120
	defaultEngramLinkHygieneLowValue    = 0.25
)

// SessionEngramLinkCreateInput captures authenticated create-link route inputs.
type SessionEngramLinkCreateInput struct {
	SourceEngramID   uuid.UUID
	TargetEngramID   uuid.UUID
	RelationType     models.EngramLinkRelationType
	Weight           float64
	TemporalWeight   float64
	Confidence       float64
	Origin           models.EngramLinkOrigin
	Status           models.EngramLinkStatus
	EvidenceJSON     map[string]any
	CreatedByUserID  uuid.UUID
	ActorUserID      uuid.UUID
	LastReinforcedAt *time.Time
}

// SessionEngramLinkListInput captures authenticated list-link route inputs.
type SessionEngramLinkListInput struct {
	SourceEngramID  uuid.UUID
	ActorUserID     uuid.UUID
	RelationType    *models.EngramLinkRelationType
	IncludeArchived bool
	Limit           int
	Offset          int
}

// SessionEngramLinkUpdateInput captures authenticated update-link route inputs.
type SessionEngramLinkUpdateInput struct {
	LinkID           uuid.UUID
	ActorUserID      uuid.UUID
	Weight           *float64
	TemporalWeight   *float64
	Confidence       *float64
	Status           *models.EngramLinkStatus
	EvidenceJSON     *map[string]any
	LastReinforcedAt *time.Time
}

// SessionEngramLinkArchiveInput captures authenticated archive-link route inputs.
type SessionEngramLinkArchiveInput struct {
	LinkID      uuid.UUID
	ActorUserID uuid.UUID
}

// SessionEngramLinkSuggestInput captures authenticated suggestion route inputs.
type SessionEngramLinkSuggestInput struct {
	SourceEngramID  uuid.UUID
	ActorUserID     uuid.UUID
	Limit           int
	MaxCandidates   int
	MinimumScore    float64
	IncludeArchived bool
}

// SessionEngramTraceInput captures authenticated trace route inputs.
type SessionEngramTraceInput struct {
	RootEngramID    uuid.UUID
	ActorUserID     uuid.UUID
	MaxDepth        int
	MaxNeighbors    int
	IncludeArchived bool
}

// SessionEngramLinkHygieneInput captures authenticated hygiene recommendation inputs.
type SessionEngramLinkHygieneInput struct {
	SourceEngramID    uuid.UUID
	ActorUserID       uuid.UUID
	IncludeArchived   bool
	Limit             int
	StaleAfterDays    int
	LowValueThreshold float64
}

type createEngramLinkPayload struct {
	TargetEngramID   uuid.UUID      `json:"target_engram_id"`
	RelationType     string         `json:"relation_type"`
	Weight           *float64       `json:"weight,omitempty"`
	TemporalWeight   *float64       `json:"temporal_weight,omitempty"`
	Confidence       *float64       `json:"confidence,omitempty"`
	Origin           string         `json:"origin"`
	Status           string         `json:"status"`
	EvidenceJSON     map[string]any `json:"evidence_json,omitempty"`
	LastReinforcedAt *time.Time     `json:"last_reinforced_at,omitempty"`
}

type updateEngramLinkPayload struct {
	Weight           *float64        `json:"weight,omitempty"`
	TemporalWeight   *float64        `json:"temporal_weight,omitempty"`
	Confidence       *float64        `json:"confidence,omitempty"`
	Status           *string         `json:"status,omitempty"`
	EvidenceJSON     *map[string]any `json:"evidence_json,omitempty"`
	LastReinforcedAt *time.Time      `json:"last_reinforced_at,omitempty"`
}

type suggestEngramLinkPayload struct {
	Limit           int      `json:"limit"`
	MaxCandidates   int      `json:"max_candidates"`
	MinimumScore    *float64 `json:"minimum_score,omitempty"`
	IncludeArchived *bool    `json:"include_archived,omitempty"`
}

type traceEngramPayload struct {
	MaxDepth        int   `json:"max_depth"`
	MaxNeighbors    int   `json:"max_neighbors"`
	IncludeArchived *bool `json:"include_archived,omitempty"`
}

type hygieneEngramLinkPayload struct {
	IncludeArchived   *bool    `json:"include_archived,omitempty"`
	Limit             int      `json:"limit"`
	StaleAfterDays    int      `json:"stale_after_days"`
	LowValueThreshold *float64 `json:"low_value_threshold,omitempty"`
}

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
	relationType, ok := parseOptionalEngramLinkRelationTypeQuery(writer, request)
	if !ok {
		return
	}
	includeArchived, ok := parseOptionalBoolQuery(writer, request, "include_archived", false)
	if !ok {
		return
	}
	limit, ok := parseOptionalIntQuery(
		writer,
		request,
		"limit",
		intQuerySpec{Default: defaultEngramLinkListLimit, Min: 1, Max: 200},
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
	links, err := dependencies.listEngramLinks(
		request.Context(),
		SessionEngramLinkListInput{
			SourceEngramID:  sourceEngramID,
			ActorUserID:     actor.UserID,
			RelationType:    relationType,
			IncludeArchived: includeArchived,
			Limit:           limit,
			Offset:          offset,
		},
	)
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

func decodeCreateEngramLinkInput(
	writer http.ResponseWriter,
	request *http.Request,
	sourceEngramID uuid.UUID,
	actorUserID uuid.UUID,
) (SessionEngramLinkCreateInput, bool) {
	payload := createEngramLinkPayload{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return SessionEngramLinkCreateInput{}, false
	}
	if payload.TargetEngramID == uuid.Nil {
		writeInvalidParameter(writer, "target_engram_id")
		return SessionEngramLinkCreateInput{}, false
	}
	relationType := models.EngramLinkRelationRelatedTo
	if trimmed := strings.TrimSpace(payload.RelationType); trimmed != "" {
		parsed, err := models.ParseEngramLinkRelationType(trimmed)
		if err != nil {
			writeInvalidParameter(writer, "relation_type")
			return SessionEngramLinkCreateInput{}, false
		}
		relationType = parsed
	}
	origin := models.EngramLinkOriginManual
	if trimmed := strings.TrimSpace(payload.Origin); trimmed != "" {
		parsed, err := models.ParseEngramLinkOrigin(trimmed)
		if err != nil {
			writeInvalidParameter(writer, "origin")
			return SessionEngramLinkCreateInput{}, false
		}
		origin = parsed
	}
	status := models.EngramLinkStatusActive
	if trimmed := strings.TrimSpace(payload.Status); trimmed != "" {
		parsed, err := models.ParseEngramLinkStatus(trimmed)
		if err != nil {
			writeInvalidParameter(writer, "status")
			return SessionEngramLinkCreateInput{}, false
		}
		status = parsed
	}
	weight, ok := parseScoreWithDefault(writer, "weight", payload.Weight, 0.6)
	if !ok {
		return SessionEngramLinkCreateInput{}, false
	}
	temporalWeight, ok := parseScoreWithDefault(writer, "temporal_weight", payload.TemporalWeight, 0.5)
	if !ok {
		return SessionEngramLinkCreateInput{}, false
	}
	confidence, ok := parseScoreWithDefault(writer, "confidence", payload.Confidence, 0.5)
	if !ok {
		return SessionEngramLinkCreateInput{}, false
	}
	return SessionEngramLinkCreateInput{
		SourceEngramID:   sourceEngramID,
		TargetEngramID:   payload.TargetEngramID,
		RelationType:     relationType,
		Weight:           weight,
		TemporalWeight:   temporalWeight,
		Confidence:       confidence,
		Origin:           origin,
		Status:           status,
		EvidenceJSON:     emptyMap(payload.EvidenceJSON),
		CreatedByUserID:  actorUserID,
		ActorUserID:      actorUserID,
		LastReinforcedAt: payload.LastReinforcedAt,
	}, true
}

func decodeUpdateEngramLinkInput(
	writer http.ResponseWriter,
	request *http.Request,
	linkID uuid.UUID,
	actorUserID uuid.UUID,
) (SessionEngramLinkUpdateInput, bool) {
	payload := updateEngramLinkPayload{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return SessionEngramLinkUpdateInput{}, false
	}
	if !hasEngramLinkUpdateField(payload) {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "No update fields provided"})
		return SessionEngramLinkUpdateInput{}, false
	}
	if !isNilOrScore("weight", payload.Weight, writer) {
		return SessionEngramLinkUpdateInput{}, false
	}
	if !isNilOrScore("temporal_weight", payload.TemporalWeight, writer) {
		return SessionEngramLinkUpdateInput{}, false
	}
	if !isNilOrScore("confidence", payload.Confidence, writer) {
		return SessionEngramLinkUpdateInput{}, false
	}
	var status *models.EngramLinkStatus
	if payload.Status != nil {
		parsedStatus, err := models.ParseEngramLinkStatus(strings.TrimSpace(*payload.Status))
		if err != nil {
			writeInvalidParameter(writer, "status")
			return SessionEngramLinkUpdateInput{}, false
		}
		status = &parsedStatus
	}
	return SessionEngramLinkUpdateInput{
		LinkID:           linkID,
		ActorUserID:      actorUserID,
		Weight:           payload.Weight,
		TemporalWeight:   payload.TemporalWeight,
		Confidence:       payload.Confidence,
		Status:           status,
		EvidenceJSON:     payload.EvidenceJSON,
		LastReinforcedAt: payload.LastReinforcedAt,
	}, true
}

func hasEngramLinkUpdateField(payload updateEngramLinkPayload) bool {
	return payload.Weight != nil ||
		payload.TemporalWeight != nil ||
		payload.Confidence != nil ||
		payload.Status != nil ||
		payload.EvidenceJSON != nil ||
		payload.LastReinforcedAt != nil
}

func decodeSuggestEngramLinksInput(
	writer http.ResponseWriter,
	request *http.Request,
	sourceEngramID uuid.UUID,
	actorUserID uuid.UUID,
) (SessionEngramLinkSuggestInput, bool) {
	payload := suggestEngramLinkPayload{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return SessionEngramLinkSuggestInput{}, false
	}
	limit := payload.Limit
	if limit == 0 {
		limit = defaultEngramLinkSuggestLimit
	}
	if limit < 1 || limit > 50 {
		writeInvalidParameter(writer, "limit")
		return SessionEngramLinkSuggestInput{}, false
	}
	maxCandidates := payload.MaxCandidates
	if maxCandidates == 0 {
		maxCandidates = defaultEngramLinkSuggestMaxCandiate
	}
	if maxCandidates < 1 || maxCandidates > 100 {
		writeInvalidParameter(writer, "max_candidates")
		return SessionEngramLinkSuggestInput{}, false
	}
	if maxCandidates < limit {
		maxCandidates = limit
	}
	minimumScore := 0.0
	if payload.MinimumScore != nil {
		minimumScore = *payload.MinimumScore
	}
	if minimumScore < 0 || minimumScore > 1 {
		writeInvalidParameter(writer, "minimum_score")
		return SessionEngramLinkSuggestInput{}, false
	}
	includeArchived := false
	if payload.IncludeArchived != nil {
		includeArchived = *payload.IncludeArchived
	}
	return SessionEngramLinkSuggestInput{
		SourceEngramID:  sourceEngramID,
		ActorUserID:     actorUserID,
		Limit:           limit,
		MaxCandidates:   maxCandidates,
		MinimumScore:    minimumScore,
		IncludeArchived: includeArchived,
	}, true
}

func decodeTraceEngramLinksInput(
	writer http.ResponseWriter,
	request *http.Request,
	rootEngramID uuid.UUID,
	actorUserID uuid.UUID,
) (SessionEngramTraceInput, bool) {
	payload := traceEngramPayload{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return SessionEngramTraceInput{}, false
	}
	maxDepth := payload.MaxDepth
	if maxDepth == 0 {
		maxDepth = defaultEngramTraceMaxDepth
	}
	if maxDepth < 1 || maxDepth > 6 {
		writeInvalidParameter(writer, "max_depth")
		return SessionEngramTraceInput{}, false
	}
	maxNeighbors := payload.MaxNeighbors
	if maxNeighbors == 0 {
		maxNeighbors = defaultEngramTraceMaxNeighbors
	}
	if maxNeighbors < 1 || maxNeighbors > 200 {
		writeInvalidParameter(writer, "max_neighbors")
		return SessionEngramTraceInput{}, false
	}
	includeArchived := false
	if payload.IncludeArchived != nil {
		includeArchived = *payload.IncludeArchived
	}
	return SessionEngramTraceInput{
		RootEngramID:    rootEngramID,
		ActorUserID:     actorUserID,
		MaxDepth:        maxDepth,
		MaxNeighbors:    maxNeighbors,
		IncludeArchived: includeArchived,
	}, true
}

func decodeHygieneEngramLinksInput(
	writer http.ResponseWriter,
	request *http.Request,
	sourceEngramID uuid.UUID,
	actorUserID uuid.UUID,
) (SessionEngramLinkHygieneInput, bool) {
	payload := hygieneEngramLinkPayload{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return SessionEngramLinkHygieneInput{}, false
	}
	includeArchived := parseHygieneIncludeArchived(payload)
	limit, ok := parseHygieneLimit(writer, payload.Limit)
	if !ok {
		return SessionEngramLinkHygieneInput{}, false
	}
	staleAfterDays, ok := parseHygieneStaleAfterDays(writer, payload.StaleAfterDays)
	if !ok {
		return SessionEngramLinkHygieneInput{}, false
	}
	lowValueThreshold, ok := parseHygieneLowValueThreshold(writer, payload.LowValueThreshold)
	if !ok {
		return SessionEngramLinkHygieneInput{}, false
	}
	return SessionEngramLinkHygieneInput{
		SourceEngramID:    sourceEngramID,
		ActorUserID:       actorUserID,
		IncludeArchived:   includeArchived,
		Limit:             limit,
		StaleAfterDays:    staleAfterDays,
		LowValueThreshold: lowValueThreshold,
	}, true
}

func parseHygieneIncludeArchived(payload hygieneEngramLinkPayload) bool {
	if payload.IncludeArchived == nil {
		return false
	}
	return *payload.IncludeArchived
}

func parseHygieneLimit(writer http.ResponseWriter, value int) (int, bool) {
	limit := value
	if limit == 0 {
		limit = defaultEngramLinkHygieneLimit
	}
	if limit < 1 || limit > 1000 {
		writeInvalidParameter(writer, "limit")
		return 0, false
	}
	return limit, true
}

func parseHygieneStaleAfterDays(writer http.ResponseWriter, value int) (int, bool) {
	staleAfterDays := value
	if staleAfterDays == 0 {
		staleAfterDays = defaultEngramLinkHygieneStaleDays
	}
	if staleAfterDays < 7 || staleAfterDays > 3650 {
		writeInvalidParameter(writer, "stale_after_days")
		return 0, false
	}
	return staleAfterDays, true
}

func parseHygieneLowValueThreshold(
	writer http.ResponseWriter,
	value *float64,
) (float64, bool) {
	lowValueThreshold := defaultEngramLinkHygieneLowValue
	if value != nil {
		lowValueThreshold = *value
	}
	if lowValueThreshold <= 0 || lowValueThreshold > 1 {
		writeInvalidParameter(writer, "low_value_threshold")
		return 0, false
	}
	return lowValueThreshold, true
}

func parseOptionalEngramLinkRelationTypeQuery(
	writer http.ResponseWriter,
	request *http.Request,
) (*models.EngramLinkRelationType, bool) {
	value := optionalTrimmedString(request.URL.Query().Get("relation_type"))
	if value == nil {
		return nil, true
	}
	parsed, err := models.ParseEngramLinkRelationType(*value)
	if err != nil {
		writeInvalidParameter(writer, "relation_type")
		return nil, false
	}
	return &parsed, true
}

func parseScoreWithDefault(
	writer http.ResponseWriter,
	field string,
	value *float64,
	defaultValue float64,
) (float64, bool) {
	if value == nil {
		return defaultValue, true
	}
	if *value < 0 || *value > 1 {
		writeInvalidParameter(writer, field)
		return 0, false
	}
	return *value, true
}

func isNilOrScore(field string, value *float64, writer http.ResponseWriter) bool {
	if value == nil {
		return true
	}
	if *value < 0 || *value > 1 {
		writeInvalidParameter(writer, field)
		return false
	}
	return true
}

func writeEngramLinkServiceError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrEngramLinkExists):
		writeJSON(writer, http.StatusConflict, map[string]string{"detail": repository.ErrEngramLinkExists.Error()})
	default:
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
	}
}

func emptyMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

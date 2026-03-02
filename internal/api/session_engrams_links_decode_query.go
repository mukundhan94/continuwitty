package api

import (
	"net/http"

	"engram/internal/models"

	"github.com/google/uuid"
)

func decodeListEngramLinksInput(
	writer http.ResponseWriter,
	request *http.Request,
	sourceEngramID uuid.UUID,
	actorUserID uuid.UUID,
) (SessionEngramLinkListInput, bool) {
	relationType, ok := parseOptionalEngramLinkRelationTypeQuery(writer, request)
	if !ok {
		return SessionEngramLinkListInput{}, false
	}
	includeArchived, ok := parseOptionalBoolQuery(writer, request, "include_archived", false)
	if !ok {
		return SessionEngramLinkListInput{}, false
	}
	limit, ok := parseOptionalIntQuery(
		writer,
		request,
		"limit",
		intQuerySpec{Default: defaultEngramLinkListLimit, Min: 1, Max: 200},
	)
	if !ok {
		return SessionEngramLinkListInput{}, false
	}
	offset, ok := parseOptionalIntQuery(
		writer,
		request,
		"offset",
		intQuerySpec{Default: 0, Min: 0, Max: 1_000_000},
	)
	if !ok {
		return SessionEngramLinkListInput{}, false
	}
	return SessionEngramLinkListInput{
		SourceEngramID:  sourceEngramID,
		ActorUserID:     actorUserID,
		RelationType:    relationType,
		IncludeArchived: includeArchived,
		Limit:           limit,
		Offset:          offset,
	}, true
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
	limit, ok := parseBoundedIntOrDefault(
		writer,
		"limit",
		payload.Limit,
		intQuerySpec{Default: defaultEngramLinkSuggestLimit, Min: 1, Max: 50},
	)
	if !ok {
		return SessionEngramLinkSuggestInput{}, false
	}
	maxCandidates, ok := parseSuggestedLinkMaxCandidates(writer, payload.MaxCandidates, limit)
	if !ok {
		return SessionEngramLinkSuggestInput{}, false
	}
	minimumScore, ok := parseSuggestedLinkMinimumScore(writer, payload.MinimumScore)
	if !ok {
		return SessionEngramLinkSuggestInput{}, false
	}
	return SessionEngramLinkSuggestInput{
		SourceEngramID:  sourceEngramID,
		ActorUserID:     actorUserID,
		Limit:           limit,
		MaxCandidates:   maxCandidates,
		MinimumScore:    minimumScore,
		IncludeArchived: boolFromPointer(payload.IncludeArchived),
	}, true
}

func parseSuggestedLinkMaxCandidates(writer http.ResponseWriter, value int, minimum int) (int, bool) {
	maxCandidates, ok := parseBoundedIntOrDefault(
		writer,
		"max_candidates",
		value,
		intQuerySpec{Default: defaultEngramLinkSuggestMaxCandiate, Min: 1, Max: 100},
	)
	if !ok {
		return 0, false
	}
	if maxCandidates < minimum {
		return minimum, true
	}
	return maxCandidates, true
}

func parseSuggestedLinkMinimumScore(writer http.ResponseWriter, value *float64) (float64, bool) {
	if value == nil {
		return 0, true
	}
	if *value < 0 || *value > 1 {
		writeInvalidParameter(writer, "minimum_score")
		return 0, false
	}
	return *value, true
}

func boolFromPointer(value *bool) bool {
	return value != nil && *value
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
	maxDepth, ok := parseBoundedIntOrDefault(
		writer,
		"max_depth",
		payload.MaxDepth,
		intQuerySpec{Default: defaultEngramTraceMaxDepth, Min: 1, Max: 6},
	)
	if !ok {
		return SessionEngramTraceInput{}, false
	}
	maxNeighbors, ok := parseBoundedIntOrDefault(
		writer,
		"max_neighbors",
		payload.MaxNeighbors,
		intQuerySpec{Default: defaultEngramTraceMaxNeighbors, Min: 1, Max: 200},
	)
	if !ok {
		return SessionEngramTraceInput{}, false
	}
	return SessionEngramTraceInput{
		RootEngramID:    rootEngramID,
		ActorUserID:     actorUserID,
		MaxDepth:        maxDepth,
		MaxNeighbors:    maxNeighbors,
		IncludeArchived: boolFromPointer(payload.IncludeArchived),
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
	limit, ok := parseBoundedIntOrDefault(
		writer,
		"limit",
		payload.Limit,
		intQuerySpec{Default: defaultEngramLinkHygieneLimit, Min: 1, Max: 1000},
	)
	if !ok {
		return SessionEngramLinkHygieneInput{}, false
	}
	staleAfterDays, ok := parseBoundedIntOrDefault(
		writer,
		"stale_after_days",
		payload.StaleAfterDays,
		intQuerySpec{Default: defaultEngramLinkHygieneStaleDays, Min: 7, Max: 3650},
	)
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
		IncludeArchived:   boolFromPointer(payload.IncludeArchived),
		Limit:             limit,
		StaleAfterDays:    staleAfterDays,
		LowValueThreshold: lowValueThreshold,
	}, true
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

func parseBoundedIntOrDefault(
	writer http.ResponseWriter,
	field string,
	value int,
	spec intQuerySpec,
) (int, bool) {
	resolved := value
	if resolved == 0 {
		resolved = spec.Default
	}
	if resolved < spec.Min || resolved > spec.Max {
		writeInvalidParameter(writer, field)
		return 0, false
	}
	return resolved, true
}

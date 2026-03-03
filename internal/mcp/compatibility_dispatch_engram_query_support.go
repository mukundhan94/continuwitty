package mcp

import (
	"context"
	"strings"
	"time"

	"engram/internal/models"
)

func (service *CompatibilityService) dispatchEngramQueryTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramQuery == nil {
		return nil, false, nil
	}
	request, dispatchErr := buildEngramQueryDispatchRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	results, err := service.engramQuery.QueryEngrams(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"results": results}, true, nil
}

func buildEngramQueryDispatchRequest(
	actor Actor,
	params map[string]any,
) (EngramQueryDispatchRequest, *toolDispatchError) {
	query, dispatchErr := parseRequiredQueryParam(params)
	if dispatchErr != nil {
		return EngramQueryDispatchRequest{}, dispatchErr
	}
	payload, dispatchErr := parseEngramQueryPayload(params)
	if dispatchErr != nil {
		return EngramQueryDispatchRequest{}, dispatchErr
	}
	return EngramQueryDispatchRequest{
		ActorUserID: actor.UserID,
		Payload:     payload.withQuery(query),
	}, nil
}

type engramQueryPayloadParts struct {
	topK                    int
	projectID               *string
	tags                    []string
	keywords                []string
	createdAfter            *time.Time
	createdBefore           *time.Time
	distanceMin             *float64
	distanceMax             *float64
	usefulCountMin          *int
	usefulCountMax          *int
	accessCountMin          *int
	accessCountMax          *int
	feedbackCountMin        *int
	feedbackCountMax        *int
	contradictionCountMin   *int
	contradictionCountMax   *int
	contradictionRatioMin   *float64
	contradictionRatioMax   *float64
	freshnessScoreMin       *float64
	freshnessScoreMax       *float64
	usefulFeedbackRatioMin  *float64
	usefulFeedbackRatioMax  *float64
	avgRelevanceFeedbackMin *float64
	avgRelevanceFeedbackMax *float64
	sourceSessionQualityMin *float64
	sourceSessionQualityMax *float64
	compositeRankScoreMin   *float64
	compositeRankScoreMax   *float64
	lastAccessedAfter       *time.Time
	lastAccessedBefore      *time.Time
	freshnessComputedAfter  *time.Time
	freshnessComputedBefore *time.Time
	relationType            *models.EngramLinkRelationType
	traceDepth              *int
}

func parseRequiredQueryParam(params map[string]any) (string, *toolDispatchError) {
	query, ok := requiredStringParam(params, "query")
	if !ok {
		return "", invalidParamError("query")
	}
	return query, nil
}

func parseEngramQueryPayload(params map[string]any) (engramQueryPayloadParts, *toolDispatchError) {
	topK, dispatchErr := parseEngramQueryTopKParam(params)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	tags, dispatchErr := parseOptionalParam(params, "tags", optionalStringArrayParam)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	keywords, dispatchErr := parseOptionalParam(params, "keywords", optionalStringArrayParam)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	temporalParts, dispatchErr := parseEngramQueryTemporalParts(params)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	distanceMin, dispatchErr := parseEngramQueryDistanceMin(params)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	distanceMax, dispatchErr := parseEngramQueryDistanceMax(params)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	if hasInvalidScoreWindow(distanceMin, distanceMax) {
		return engramQueryPayloadParts{}, invalidParamError("distance_min")
	}
	engagementParts, dispatchErr := parseEngramQueryEngagementParts(params)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	traceParts, dispatchErr := parseEngramQueryTraceParts(params)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	return engramQueryPayloadParts{
		topK:                    topK,
		projectID:               optionalProjectIDParam(params, "project_id"),
		tags:                    tags,
		keywords:                keywords,
		createdAfter:            temporalParts.createdAfter,
		createdBefore:           temporalParts.createdBefore,
		distanceMin:             distanceMin,
		distanceMax:             distanceMax,
		usefulCountMin:          engagementParts.usefulCountMin,
		usefulCountMax:          engagementParts.usefulCountMax,
		accessCountMin:          engagementParts.accessCountMin,
		accessCountMax:          engagementParts.accessCountMax,
		feedbackCountMin:        engagementParts.feedbackCountMin,
		feedbackCountMax:        engagementParts.feedbackCountMax,
		contradictionCountMin:   engagementParts.contradictionCountMin,
		contradictionCountMax:   engagementParts.contradictionCountMax,
		contradictionRatioMin:   engagementParts.contradictionRatioMin,
		contradictionRatioMax:   engagementParts.contradictionRatioMax,
		freshnessScoreMin:       engagementParts.freshnessScoreMin,
		freshnessScoreMax:       engagementParts.freshnessScoreMax,
		usefulFeedbackRatioMin:  engagementParts.usefulFeedbackRatioMin,
		usefulFeedbackRatioMax:  engagementParts.usefulFeedbackRatioMax,
		avgRelevanceFeedbackMin: engagementParts.avgRelevanceFeedbackMin,
		avgRelevanceFeedbackMax: engagementParts.avgRelevanceFeedbackMax,
		sourceSessionQualityMin: engagementParts.sourceSessionQualityMin,
		sourceSessionQualityMax: engagementParts.sourceSessionQualityMax,
		compositeRankScoreMin:   engagementParts.compositeRankScoreMin,
		compositeRankScoreMax:   engagementParts.compositeRankScoreMax,
		lastAccessedAfter:       temporalParts.lastAccessedAfter,
		lastAccessedBefore:      temporalParts.lastAccessedBefore,
		freshnessComputedAfter:  temporalParts.freshnessComputedAfter,
		freshnessComputedBefore: temporalParts.freshnessComputedBefore,
		relationType:            traceParts.relationType,
		traceDepth:              traceParts.traceDepth,
	}, nil
}

type engramQueryTemporalParts struct {
	createdAfter            *time.Time
	createdBefore           *time.Time
	lastAccessedAfter       *time.Time
	lastAccessedBefore      *time.Time
	freshnessComputedAfter  *time.Time
	freshnessComputedBefore *time.Time
}

func parseEngramQueryTemporalParts(params map[string]any) (engramQueryTemporalParts, *toolDispatchError) {
	createdAfter, dispatchErr := parseOptionalParam(params, "created_after", optionalRFC3339TimeParam)
	if dispatchErr != nil {
		return engramQueryTemporalParts{}, dispatchErr
	}
	createdBefore, dispatchErr := parseOptionalParam(params, "created_before", optionalRFC3339TimeParam)
	if dispatchErr != nil {
		return engramQueryTemporalParts{}, dispatchErr
	}
	lastAccessedAfter, dispatchErr := parseOptionalParam(params, "last_accessed_after", optionalRFC3339TimeParam)
	if dispatchErr != nil {
		return engramQueryTemporalParts{}, dispatchErr
	}
	lastAccessedBefore, dispatchErr := parseOptionalParam(params, "last_accessed_before", optionalRFC3339TimeParam)
	if dispatchErr != nil {
		return engramQueryTemporalParts{}, dispatchErr
	}
	freshnessComputedAfter, dispatchErr := parseOptionalParam(params, "freshness_computed_after", optionalRFC3339TimeParam)
	if dispatchErr != nil {
		return engramQueryTemporalParts{}, dispatchErr
	}
	freshnessComputedBefore, dispatchErr := parseOptionalParam(params, "freshness_computed_before", optionalRFC3339TimeParam)
	if dispatchErr != nil {
		return engramQueryTemporalParts{}, dispatchErr
	}
	parts := engramQueryTemporalParts{
		createdAfter:            createdAfter,
		createdBefore:           createdBefore,
		lastAccessedAfter:       lastAccessedAfter,
		lastAccessedBefore:      lastAccessedBefore,
		freshnessComputedAfter:  freshnessComputedAfter,
		freshnessComputedBefore: freshnessComputedBefore,
	}
	dispatchErr = validateTemporalWindows(temporalWindowSpecs(parts))
	if dispatchErr != nil {
		return engramQueryTemporalParts{}, dispatchErr
	}
	return parts, nil
}

func temporalWindowSpecs(parts engramQueryTemporalParts) []temporalWindowSpec {
	return []temporalWindowSpec{
		{after: parts.createdAfter, before: parts.createdBefore, afterParam: "created_after"},
		{after: parts.lastAccessedAfter, before: parts.lastAccessedBefore, afterParam: "last_accessed_after"},
		{
			after:      parts.freshnessComputedAfter,
			before:     parts.freshnessComputedBefore,
			afterParam: "freshness_computed_after",
		},
	}
}

type engramQueryEngagementParts struct {
	usefulCountMin          *int
	usefulCountMax          *int
	accessCountMin          *int
	accessCountMax          *int
	feedbackCountMin        *int
	feedbackCountMax        *int
	contradictionCountMin   *int
	contradictionCountMax   *int
	contradictionRatioMin   *float64
	contradictionRatioMax   *float64
	freshnessScoreMin       *float64
	freshnessScoreMax       *float64
	usefulFeedbackRatioMin  *float64
	usefulFeedbackRatioMax  *float64
	avgRelevanceFeedbackMin *float64
	avgRelevanceFeedbackMax *float64
	sourceSessionQualityMin *float64
	sourceSessionQualityMax *float64
	compositeRankScoreMin   *float64
	compositeRankScoreMax   *float64
}

func parseEngramQueryEngagementParts(params map[string]any) (engramQueryEngagementParts, *toolDispatchError) {
	integerParts, dispatchErr := parseEngramQueryIntegerEngagementParts(params)
	if dispatchErr != nil {
		return engramQueryEngagementParts{}, dispatchErr
	}
	scoreParts, dispatchErr := parseEngramQueryScoreEngagementParts(params)
	if dispatchErr != nil {
		return engramQueryEngagementParts{}, dispatchErr
	}
	parts := engramQueryEngagementParts{
		usefulCountMin:          integerParts.usefulCountMin,
		usefulCountMax:          integerParts.usefulCountMax,
		accessCountMin:          integerParts.accessCountMin,
		accessCountMax:          integerParts.accessCountMax,
		feedbackCountMin:        integerParts.feedbackCountMin,
		feedbackCountMax:        integerParts.feedbackCountMax,
		contradictionCountMin:   integerParts.contradictionCountMin,
		contradictionCountMax:   integerParts.contradictionCountMax,
		contradictionRatioMin:   scoreParts.contradictionRatioMin,
		contradictionRatioMax:   scoreParts.contradictionRatioMax,
		freshnessScoreMin:       scoreParts.freshnessScoreMin,
		freshnessScoreMax:       scoreParts.freshnessScoreMax,
		usefulFeedbackRatioMin:  scoreParts.usefulFeedbackRatioMin,
		usefulFeedbackRatioMax:  scoreParts.usefulFeedbackRatioMax,
		avgRelevanceFeedbackMin: scoreParts.avgRelevanceFeedbackMin,
		avgRelevanceFeedbackMax: scoreParts.avgRelevanceFeedbackMax,
		sourceSessionQualityMin: scoreParts.sourceSessionQualityMin,
		sourceSessionQualityMax: scoreParts.sourceSessionQualityMax,
		compositeRankScoreMin:   scoreParts.compositeRankScoreMin,
		compositeRankScoreMax:   scoreParts.compositeRankScoreMax,
	}
	if dispatchErr := invalidQueryEngagementWindowError(parts); dispatchErr != nil {
		return engramQueryEngagementParts{}, dispatchErr
	}
	return parts, nil
}

type engramQueryIntegerEngagementParts struct {
	usefulCountMin        *int
	usefulCountMax        *int
	accessCountMin        *int
	accessCountMax        *int
	feedbackCountMin      *int
	feedbackCountMax      *int
	contradictionCountMin *int
	contradictionCountMax *int
}

func invalidQueryEngagementWindowError(
	parts engramQueryEngagementParts,
) *toolDispatchError {
	if hasInvalidIntWindow(parts.usefulCountMin, parts.usefulCountMax) {
		return invalidParamError("useful_count_min")
	}
	if hasInvalidIntWindow(parts.accessCountMin, parts.accessCountMax) {
		return invalidParamError("access_count_min")
	}
	if hasInvalidIntWindow(parts.feedbackCountMin, parts.feedbackCountMax) {
		return invalidParamError("feedback_count_min")
	}
	if hasInvalidIntWindow(parts.contradictionCountMin, parts.contradictionCountMax) {
		return invalidParamError("contradiction_count_min")
	}
	if hasInvalidScoreWindow(parts.contradictionRatioMin, parts.contradictionRatioMax) {
		return invalidParamError("contradiction_feedback_ratio_min")
	}
	if hasInvalidScoreWindow(parts.freshnessScoreMin, parts.freshnessScoreMax) {
		return invalidParamError("freshness_score_min")
	}
	if hasInvalidScoreWindow(parts.usefulFeedbackRatioMin, parts.usefulFeedbackRatioMax) {
		return invalidParamError("useful_feedback_ratio_min")
	}
	if hasInvalidScoreWindow(parts.avgRelevanceFeedbackMin, parts.avgRelevanceFeedbackMax) {
		return invalidParamError("avg_relevance_feedback_min")
	}
	if hasInvalidScoreWindow(parts.sourceSessionQualityMin, parts.sourceSessionQualityMax) {
		return invalidParamError("source_session_quality_min")
	}
	if hasInvalidScoreWindow(parts.compositeRankScoreMin, parts.compositeRankScoreMax) {
		return invalidParamError("composite_rank_score_min")
	}
	return nil
}

func hasInvalidIntWindow(minValue *int, maxValue *int) bool {
	if minValue == nil || maxValue == nil {
		return false
	}
	return *minValue > *maxValue
}

func hasInvalidScoreWindow(minValue *float64, maxValue *float64) bool {
	if minValue == nil || maxValue == nil {
		return false
	}
	return *minValue > *maxValue
}

func parseEngramQueryIntegerEngagementParts(
	params map[string]any,
) (engramQueryIntegerEngagementParts, *toolDispatchError) {
	usefulCountMin, dispatchErr := parseEngramQueryNonNegativeIntPointer(params, "useful_count_min")
	if dispatchErr != nil {
		return engramQueryIntegerEngagementParts{}, dispatchErr
	}
	usefulCountMax, dispatchErr := parseEngramQueryNonNegativeIntPointer(params, "useful_count_max")
	if dispatchErr != nil {
		return engramQueryIntegerEngagementParts{}, dispatchErr
	}
	accessCountMin, dispatchErr := parseEngramQueryNonNegativeIntPointer(params, "access_count_min")
	if dispatchErr != nil {
		return engramQueryIntegerEngagementParts{}, dispatchErr
	}
	accessCountMax, dispatchErr := parseEngramQueryNonNegativeIntPointer(params, "access_count_max")
	if dispatchErr != nil {
		return engramQueryIntegerEngagementParts{}, dispatchErr
	}
	feedbackCountMin, dispatchErr := parseEngramQueryNonNegativeIntPointer(params, "feedback_count_min")
	if dispatchErr != nil {
		return engramQueryIntegerEngagementParts{}, dispatchErr
	}
	feedbackCountMax, dispatchErr := parseEngramQueryNonNegativeIntPointer(params, "feedback_count_max")
	if dispatchErr != nil {
		return engramQueryIntegerEngagementParts{}, dispatchErr
	}
	contradictionCountMin, dispatchErr := parseEngramQueryNonNegativeIntPointer(
		params,
		"contradiction_count_min",
	)
	if dispatchErr != nil {
		return engramQueryIntegerEngagementParts{}, dispatchErr
	}
	contradictionCountMax, dispatchErr := parseEngramQueryNonNegativeIntPointer(
		params,
		"contradiction_count_max",
	)
	if dispatchErr != nil {
		return engramQueryIntegerEngagementParts{}, dispatchErr
	}
	return engramQueryIntegerEngagementParts{
		usefulCountMin:        usefulCountMin,
		usefulCountMax:        usefulCountMax,
		accessCountMin:        accessCountMin,
		accessCountMax:        accessCountMax,
		feedbackCountMin:      feedbackCountMin,
		feedbackCountMax:      feedbackCountMax,
		contradictionCountMin: contradictionCountMin,
		contradictionCountMax: contradictionCountMax,
	}, nil
}

type engramQueryScoreEngagementParts struct {
	contradictionRatioMin   *float64
	contradictionRatioMax   *float64
	freshnessScoreMin       *float64
	freshnessScoreMax       *float64
	usefulFeedbackRatioMin  *float64
	usefulFeedbackRatioMax  *float64
	avgRelevanceFeedbackMin *float64
	avgRelevanceFeedbackMax *float64
	sourceSessionQualityMin *float64
	sourceSessionQualityMax *float64
	compositeRankScoreMin   *float64
	compositeRankScoreMax   *float64
}

func parseEngramQueryScoreEngagementParts(
	params map[string]any,
) (engramQueryScoreEngagementParts, *toolDispatchError) {
	contradictionRatioMax, dispatchErr := parseEngramQueryContradictionFeedbackRatioMax(params)
	if dispatchErr != nil {
		return engramQueryScoreEngagementParts{}, dispatchErr
	}
	contradictionRatioMin, dispatchErr := parseEngramQueryContradictionFeedbackRatioMin(params)
	if dispatchErr != nil {
		return engramQueryScoreEngagementParts{}, dispatchErr
	}
	freshnessScoreMin, dispatchErr := parseEngramQueryFreshnessScoreMin(params)
	if dispatchErr != nil {
		return engramQueryScoreEngagementParts{}, dispatchErr
	}
	freshnessScoreMax, dispatchErr := parseEngramQueryFreshnessScoreMax(params)
	if dispatchErr != nil {
		return engramQueryScoreEngagementParts{}, dispatchErr
	}
	usefulFeedbackRatioMin, dispatchErr := parseEngramQueryUsefulFeedbackRatioMin(params)
	if dispatchErr != nil {
		return engramQueryScoreEngagementParts{}, dispatchErr
	}
	usefulFeedbackRatioMax, dispatchErr := parseEngramQueryUsefulFeedbackRatioMax(params)
	if dispatchErr != nil {
		return engramQueryScoreEngagementParts{}, dispatchErr
	}
	avgRelevanceFeedbackMin, dispatchErr := parseEngramQueryAvgRelevanceFeedbackMin(params)
	if dispatchErr != nil {
		return engramQueryScoreEngagementParts{}, dispatchErr
	}
	avgRelevanceFeedbackMax, dispatchErr := parseEngramQueryAvgRelevanceFeedbackMax(params)
	if dispatchErr != nil {
		return engramQueryScoreEngagementParts{}, dispatchErr
	}
	sourceSessionQualityMin, dispatchErr := parseEngramQuerySourceSessionQualityMin(params)
	if dispatchErr != nil {
		return engramQueryScoreEngagementParts{}, dispatchErr
	}
	sourceSessionQualityMax, dispatchErr := parseEngramQuerySourceSessionQualityMax(params)
	if dispatchErr != nil {
		return engramQueryScoreEngagementParts{}, dispatchErr
	}
	compositeRankScoreMin, dispatchErr := parseEngramQueryCompositeRankScoreMin(params)
	if dispatchErr != nil {
		return engramQueryScoreEngagementParts{}, dispatchErr
	}
	compositeRankScoreMax, dispatchErr := parseEngramQueryCompositeRankScoreMax(params)
	if dispatchErr != nil {
		return engramQueryScoreEngagementParts{}, dispatchErr
	}
	return engramQueryScoreEngagementParts{
		contradictionRatioMin:   contradictionRatioMin,
		contradictionRatioMax:   contradictionRatioMax,
		freshnessScoreMin:       freshnessScoreMin,
		freshnessScoreMax:       freshnessScoreMax,
		usefulFeedbackRatioMin:  usefulFeedbackRatioMin,
		usefulFeedbackRatioMax:  usefulFeedbackRatioMax,
		avgRelevanceFeedbackMin: avgRelevanceFeedbackMin,
		avgRelevanceFeedbackMax: avgRelevanceFeedbackMax,
		sourceSessionQualityMin: sourceSessionQualityMin,
		sourceSessionQualityMax: sourceSessionQualityMax,
		compositeRankScoreMin:   compositeRankScoreMin,
		compositeRankScoreMax:   compositeRankScoreMax,
	}, nil
}

type engramQueryTraceParts struct {
	relationType *models.EngramLinkRelationType
	traceDepth   *int
}

func parseEngramQueryTraceParts(params map[string]any) (engramQueryTraceParts, *toolDispatchError) {
	relationType, ok := optionalEngramLinkRelationTypePointer(params, engramLinkParamKey("relation_type"))
	if !ok {
		return engramQueryTraceParts{}, invalidParamError("relation_type")
	}
	traceDepth, dispatchErr := parseTraceDepthParam(params, relationType)
	if dispatchErr != nil {
		return engramQueryTraceParts{}, dispatchErr
	}
	return engramQueryTraceParts{relationType: relationType, traceDepth: traceDepth}, nil
}

func parseTraceDepthParam(
	params map[string]any,
	relationType *models.EngramLinkRelationType,
) (*int, *toolDispatchError) {
	value, ok := optionalIntPointerParam(params, "trace_depth")
	if !ok {
		return nil, invalidParamError("trace_depth")
	}
	if value == nil {
		if relationType == nil {
			return nil, nil
		}
		defaultDepth := 1
		return &defaultDepth, nil
	}
	if *value < 0 || *value > 1 {
		return nil, invalidParamError("trace_depth")
	}
	if relationType != nil && *value == 0 {
		return nil, invalidParamError("trace_depth")
	}
	return value, nil
}

func parseOptionalParam[T any](
	params map[string]any,
	key string,
	parser func(map[string]any, string) (T, bool),
) (T, *toolDispatchError) {
	parsed, ok := parser(params, key)
	if !ok {
		var zero T
		return zero, invalidParamError(key)
	}
	return parsed, nil
}

func (parts engramQueryPayloadParts) withQuery(query string) models.EngramQueryRequest {
	return models.EngramQueryRequest{
		Query:                   query,
		TopK:                    parts.topK,
		ProjectID:               parts.projectID,
		Tags:                    parts.tags,
		Keywords:                parts.keywords,
		CreatedAfter:            parts.createdAfter,
		CreatedBefore:           parts.createdBefore,
		DistanceMin:             parts.distanceMin,
		DistanceMax:             parts.distanceMax,
		UsefulCountMin:          parts.usefulCountMin,
		UsefulCountMax:          parts.usefulCountMax,
		AccessCountMin:          parts.accessCountMin,
		AccessCountMax:          parts.accessCountMax,
		FeedbackCountMin:        parts.feedbackCountMin,
		FeedbackCountMax:        parts.feedbackCountMax,
		ContradictionCountMin:   parts.contradictionCountMin,
		ContradictionCountMax:   parts.contradictionCountMax,
		ContradictionRatioMin:   parts.contradictionRatioMin,
		ContradictionRatioMax:   parts.contradictionRatioMax,
		FreshnessScoreMin:       parts.freshnessScoreMin,
		FreshnessScoreMax:       parts.freshnessScoreMax,
		UsefulFeedbackRatioMin:  parts.usefulFeedbackRatioMin,
		UsefulFeedbackRatioMax:  parts.usefulFeedbackRatioMax,
		AvgRelevanceFeedbackMin: parts.avgRelevanceFeedbackMin,
		AvgRelevanceFeedbackMax: parts.avgRelevanceFeedbackMax,
		SourceSessionQualityMin: parts.sourceSessionQualityMin,
		SourceSessionQualityMax: parts.sourceSessionQualityMax,
		CompositeRankScoreMin:   parts.compositeRankScoreMin,
		CompositeRankScoreMax:   parts.compositeRankScoreMax,
		LastAccessedAfter:       parts.lastAccessedAfter,
		LastAccessedBefore:      parts.lastAccessedBefore,
		FreshnessComputedAfter:  parts.freshnessComputedAfter,
		FreshnessComputedBefore: parts.freshnessComputedBefore,
		RelationType:            parts.relationType,
		TraceDepth:              parts.traceDepth,
	}
}

type temporalWindowSpec struct {
	after      *time.Time
	before     *time.Time
	afterParam string
}

func validateTemporalWindows(specs []temporalWindowSpec) *toolDispatchError {
	for _, spec := range specs {
		if spec.after == nil || spec.before == nil {
			continue
		}
		if spec.after.After(*spec.before) {
			return invalidParamError(spec.afterParam)
		}
	}
	return nil
}

func parseEngramQueryTopKParam(params map[string]any) (int, *toolDispatchError) {
	topK, ok := optionalIntParam(params, "top_k", 5)
	if !ok {
		return 0, invalidParamError("top_k")
	}
	if topK < 1 {
		return 0, invalidParamError("top_k")
	}
	if topK > 50 {
		return 0, invalidParamError("top_k")
	}
	return topK, nil
}

func parseEngramQueryDistanceMax(params map[string]any) (*float64, *toolDispatchError) {
	rawValue, found := optionalParamValue(params, "distance_max")
	if !found {
		return nil, nil
	}
	parsed, ok := parseFloatValue(rawValue)
	if !ok {
		return nil, invalidParamError("distance_max")
	}
	if parsed < 0 {
		return nil, invalidParamError("distance_max")
	}
	copy := parsed
	return &copy, nil
}

func parseEngramQueryDistanceMin(params map[string]any) (*float64, *toolDispatchError) {
	rawValue, found := optionalParamValue(params, "distance_min")
	if !found {
		return nil, nil
	}
	parsed, ok := parseFloatValue(rawValue)
	if !ok {
		return nil, invalidParamError("distance_min")
	}
	if parsed < 0 {
		return nil, invalidParamError("distance_min")
	}
	copy := parsed
	return &copy, nil
}

func optionalStringArrayParam(params map[string]any, key string) ([]string, bool) {
	value, found := optionalParamValue(params, key)
	if !found {
		return []string{}, true
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), true
	case []any:
		return stringArrayFromAnySlice(typed)
	default:
		return nil, false
	}
}

func stringArrayFromAnySlice(values []any) ([]string, bool) {
	result := make([]string, 0, len(values))
	for _, item := range values {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		result = append(result, text)
	}
	return result, true
}

func optionalRFC3339TimeParam(params map[string]any, key string) (*time.Time, bool) {
	value, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	text, ok := value.(string)
	if !ok {
		return nil, false
	}
	return parseRFC3339Pointer(text)
}

func parseRFC3339Pointer(raw string) (*time.Time, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func parseEngramQueryNonNegativeIntPointer(
	params map[string]any,
	paramName string,
) (*int, *toolDispatchError) {
	value, ok := optionalIntPointerParam(params, paramName)
	if !ok {
		return nil, invalidParamError(paramName)
	}
	if value != nil && *value < 0 {
		return nil, invalidParamError(paramName)
	}
	return value, nil
}

func parseEngramQueryFreshnessScoreMin(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "freshness_score_min")
}

func parseEngramQueryFreshnessScoreMax(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "freshness_score_max")
}

func parseEngramQueryContradictionFeedbackRatioMax(
	params map[string]any,
) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "contradiction_feedback_ratio_max")
}

func parseEngramQueryContradictionFeedbackRatioMin(
	params map[string]any,
) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "contradiction_feedback_ratio_min")
}

func parseEngramQueryUsefulFeedbackRatioMin(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "useful_feedback_ratio_min")
}

func parseEngramQueryUsefulFeedbackRatioMax(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "useful_feedback_ratio_max")
}

func parseEngramQueryAvgRelevanceFeedbackMin(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "avg_relevance_feedback_min")
}

func parseEngramQueryAvgRelevanceFeedbackMax(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "avg_relevance_feedback_max")
}

func parseEngramQuerySourceSessionQualityMin(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "source_session_quality_min")
}

func parseEngramQuerySourceSessionQualityMax(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "source_session_quality_max")
}

func parseEngramQueryCompositeRankScoreMin(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "composite_rank_score_min")
}

func parseEngramQueryCompositeRankScoreMax(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "composite_rank_score_max")
}

func parseEngramQueryBoundedScoreMin(
	params map[string]any,
	key string,
) (*float64, *toolDispatchError) {
	rawValue, found := optionalParamValue(params, key)
	if !found {
		return nil, nil
	}
	parsed, ok := parseFloatValue(rawValue)
	if !ok {
		return nil, invalidParamError(key)
	}
	if parsed < 0 {
		return nil, invalidParamError(key)
	}
	if parsed > 1 {
		return nil, invalidParamError(key)
	}
	copy := parsed
	return &copy, nil
}

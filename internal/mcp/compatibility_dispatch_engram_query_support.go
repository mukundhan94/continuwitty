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
	topK                     int
	projectID                *string
	tags                     []string
	keywords                 []string
	createdAfter             *time.Time
	createdBefore            *time.Time
	distanceMin              *float64
	distanceMax              *float64
	usefulCountMin           *int
	usefulCountMax           *int
	accessCountMin           *int
	accessCountMax           *int
	feedbackCountMin         *int
	feedbackCountMax         *int
	contradictionCountMin    *int
	contradictionCountMax    *int
	contradictionRatioMin    *float64
	contradictionRatioMax    *float64
	freshnessScoreMin        *float64
	freshnessScoreMax        *float64
	usefulFeedbackRatioMin   *float64
	usefulFeedbackRatioMax   *float64
	avgRelevanceFeedbackMin  *float64
	avgRelevanceFeedbackMax  *float64
	sourceSessionQualityMin  *float64
	sourceSessionQualityMax  *float64
	denseScoreMin            *float64
	denseScoreMax            *float64
	lexicalOverlapScoreMin   *float64
	lexicalOverlapScoreMax   *float64
	feedbackSignalScoreMin   *float64
	feedbackSignalScoreMax   *float64
	engagementSignalScoreMin *float64
	engagementSignalScoreMax *float64
	freshnessSignalScoreMin  *float64
	freshnessSignalScoreMax  *float64
	authoritySignalScoreMin  *float64
	authoritySignalScoreMax  *float64
	compositeRankScoreMin    *float64
	compositeRankScoreMax    *float64
	lastAccessedAfter        *time.Time
	lastAccessedBefore       *time.Time
	freshnessComputedAfter   *time.Time
	freshnessComputedBefore  *time.Time
	relationType             *models.EngramLinkRelationType
	traceDepth               *int
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
	scopeParts, dispatchErr := parseEngramQueryScopeParts(params)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	temporalParts, dispatchErr := parseEngramQueryTemporalParts(params)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
	}
	distanceParts, dispatchErr := parseEngramQueryDistanceParts(params)
	if dispatchErr != nil {
		return engramQueryPayloadParts{}, dispatchErr
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
		topK:                     topK,
		projectID:                scopeParts.projectID,
		tags:                     scopeParts.tags,
		keywords:                 scopeParts.keywords,
		createdAfter:             temporalParts.createdAfter,
		createdBefore:            temporalParts.createdBefore,
		distanceMin:              distanceParts.distanceMin,
		distanceMax:              distanceParts.distanceMax,
		usefulCountMin:           engagementParts.usefulCountMin,
		usefulCountMax:           engagementParts.usefulCountMax,
		accessCountMin:           engagementParts.accessCountMin,
		accessCountMax:           engagementParts.accessCountMax,
		feedbackCountMin:         engagementParts.feedbackCountMin,
		feedbackCountMax:         engagementParts.feedbackCountMax,
		contradictionCountMin:    engagementParts.contradictionCountMin,
		contradictionCountMax:    engagementParts.contradictionCountMax,
		contradictionRatioMin:    engagementParts.contradictionRatioMin,
		contradictionRatioMax:    engagementParts.contradictionRatioMax,
		freshnessScoreMin:        engagementParts.freshnessScoreMin,
		freshnessScoreMax:        engagementParts.freshnessScoreMax,
		usefulFeedbackRatioMin:   engagementParts.usefulFeedbackRatioMin,
		usefulFeedbackRatioMax:   engagementParts.usefulFeedbackRatioMax,
		avgRelevanceFeedbackMin:  engagementParts.avgRelevanceFeedbackMin,
		avgRelevanceFeedbackMax:  engagementParts.avgRelevanceFeedbackMax,
		sourceSessionQualityMin:  engagementParts.sourceSessionQualityMin,
		sourceSessionQualityMax:  engagementParts.sourceSessionQualityMax,
		denseScoreMin:            engagementParts.denseScoreMin,
		denseScoreMax:            engagementParts.denseScoreMax,
		lexicalOverlapScoreMin:   engagementParts.lexicalOverlapScoreMin,
		lexicalOverlapScoreMax:   engagementParts.lexicalOverlapScoreMax,
		feedbackSignalScoreMin:   engagementParts.feedbackSignalScoreMin,
		feedbackSignalScoreMax:   engagementParts.feedbackSignalScoreMax,
		engagementSignalScoreMin: engagementParts.engagementSignalScoreMin,
		engagementSignalScoreMax: engagementParts.engagementSignalScoreMax,
		freshnessSignalScoreMin:  engagementParts.freshnessSignalScoreMin,
		freshnessSignalScoreMax:  engagementParts.freshnessSignalScoreMax,
		authoritySignalScoreMin:  engagementParts.authoritySignalScoreMin,
		authoritySignalScoreMax:  engagementParts.authoritySignalScoreMax,
		compositeRankScoreMin:    engagementParts.compositeRankScoreMin,
		compositeRankScoreMax:    engagementParts.compositeRankScoreMax,
		lastAccessedAfter:        temporalParts.lastAccessedAfter,
		lastAccessedBefore:       temporalParts.lastAccessedBefore,
		freshnessComputedAfter:   temporalParts.freshnessComputedAfter,
		freshnessComputedBefore:  temporalParts.freshnessComputedBefore,
		relationType:             traceParts.relationType,
		traceDepth:               traceParts.traceDepth,
	}, nil
}

type engramQueryScopeParts struct {
	projectID *string
	tags      []string
	keywords  []string
}

func parseEngramQueryScopeParts(params map[string]any) (engramQueryScopeParts, *toolDispatchError) {
	tags, dispatchErr := parseOptionalParam(params, "tags", optionalStringArrayParam)
	if dispatchErr != nil {
		return engramQueryScopeParts{}, dispatchErr
	}
	keywords, dispatchErr := parseOptionalParam(params, "keywords", optionalStringArrayParam)
	if dispatchErr != nil {
		return engramQueryScopeParts{}, dispatchErr
	}
	return engramQueryScopeParts{
		projectID: optionalProjectIDParam(params, "project_id"),
		tags:      tags,
		keywords:  keywords,
	}, nil
}

type engramQueryDistanceParts struct {
	distanceMin *float64
	distanceMax *float64
}

func parseEngramQueryDistanceParts(params map[string]any) (engramQueryDistanceParts, *toolDispatchError) {
	distanceMin, dispatchErr := parseEngramQueryFloatWithValidator(
		params,
		"distance_min",
		func(value float64) bool { return value >= 0 },
	)
	if dispatchErr != nil {
		return engramQueryDistanceParts{}, dispatchErr
	}
	distanceMax, dispatchErr := parseEngramQueryFloatWithValidator(
		params,
		"distance_max",
		func(value float64) bool { return value >= 0 },
	)
	if dispatchErr != nil {
		return engramQueryDistanceParts{}, dispatchErr
	}
	if hasInvalidScoreWindow(distanceMin, distanceMax) {
		return engramQueryDistanceParts{}, invalidParamError("distance_min")
	}
	return engramQueryDistanceParts{distanceMin: distanceMin, distanceMax: distanceMax}, nil
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
	usefulCountMin           *int
	usefulCountMax           *int
	accessCountMin           *int
	accessCountMax           *int
	feedbackCountMin         *int
	feedbackCountMax         *int
	contradictionCountMin    *int
	contradictionCountMax    *int
	contradictionRatioMin    *float64
	contradictionRatioMax    *float64
	freshnessScoreMin        *float64
	freshnessScoreMax        *float64
	usefulFeedbackRatioMin   *float64
	usefulFeedbackRatioMax   *float64
	avgRelevanceFeedbackMin  *float64
	avgRelevanceFeedbackMax  *float64
	sourceSessionQualityMin  *float64
	sourceSessionQualityMax  *float64
	denseScoreMin            *float64
	denseScoreMax            *float64
	lexicalOverlapScoreMin   *float64
	lexicalOverlapScoreMax   *float64
	feedbackSignalScoreMin   *float64
	feedbackSignalScoreMax   *float64
	engagementSignalScoreMin *float64
	engagementSignalScoreMax *float64
	freshnessSignalScoreMin  *float64
	freshnessSignalScoreMax  *float64
	authoritySignalScoreMin  *float64
	authoritySignalScoreMax  *float64
	compositeRankScoreMin    *float64
	compositeRankScoreMax    *float64
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
		usefulCountMin:           integerParts.usefulCountMin,
		usefulCountMax:           integerParts.usefulCountMax,
		accessCountMin:           integerParts.accessCountMin,
		accessCountMax:           integerParts.accessCountMax,
		feedbackCountMin:         integerParts.feedbackCountMin,
		feedbackCountMax:         integerParts.feedbackCountMax,
		contradictionCountMin:    integerParts.contradictionCountMin,
		contradictionCountMax:    integerParts.contradictionCountMax,
		contradictionRatioMin:    scoreParts.contradictionRatioMin,
		contradictionRatioMax:    scoreParts.contradictionRatioMax,
		freshnessScoreMin:        scoreParts.freshnessScoreMin,
		freshnessScoreMax:        scoreParts.freshnessScoreMax,
		usefulFeedbackRatioMin:   scoreParts.usefulFeedbackRatioMin,
		usefulFeedbackRatioMax:   scoreParts.usefulFeedbackRatioMax,
		avgRelevanceFeedbackMin:  scoreParts.avgRelevanceFeedbackMin,
		avgRelevanceFeedbackMax:  scoreParts.avgRelevanceFeedbackMax,
		sourceSessionQualityMin:  scoreParts.sourceSessionQualityMin,
		sourceSessionQualityMax:  scoreParts.sourceSessionQualityMax,
		denseScoreMin:            scoreParts.denseScoreMin,
		denseScoreMax:            scoreParts.denseScoreMax,
		lexicalOverlapScoreMin:   scoreParts.lexicalOverlapScoreMin,
		lexicalOverlapScoreMax:   scoreParts.lexicalOverlapScoreMax,
		feedbackSignalScoreMin:   scoreParts.feedbackSignalScoreMin,
		feedbackSignalScoreMax:   scoreParts.feedbackSignalScoreMax,
		engagementSignalScoreMin: scoreParts.engagementSignalScoreMin,
		engagementSignalScoreMax: scoreParts.engagementSignalScoreMax,
		freshnessSignalScoreMin:  scoreParts.freshnessSignalScoreMin,
		freshnessSignalScoreMax:  scoreParts.freshnessSignalScoreMax,
		authoritySignalScoreMin:  scoreParts.authoritySignalScoreMin,
		authoritySignalScoreMax:  scoreParts.authoritySignalScoreMax,
		compositeRankScoreMin:    scoreParts.compositeRankScoreMin,
		compositeRankScoreMax:    scoreParts.compositeRankScoreMax,
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
	for _, check := range engramQueryEngagementWindowChecks(parts) {
		if check.invalid {
			return invalidParamError(check.param)
		}
	}
	return nil
}

type engramQueryWindowCheck struct {
	param   string
	invalid bool
}

func engramQueryEngagementWindowChecks(parts engramQueryEngagementParts) []engramQueryWindowCheck {
	return []engramQueryWindowCheck{
		{param: "useful_count_min", invalid: hasInvalidIntWindow(parts.usefulCountMin, parts.usefulCountMax)},
		{param: "access_count_min", invalid: hasInvalidIntWindow(parts.accessCountMin, parts.accessCountMax)},
		{param: "feedback_count_min", invalid: hasInvalidIntWindow(parts.feedbackCountMin, parts.feedbackCountMax)},
		{
			param:   "contradiction_count_min",
			invalid: hasInvalidIntWindow(parts.contradictionCountMin, parts.contradictionCountMax),
		},
		{
			param:   "contradiction_feedback_ratio_min",
			invalid: hasInvalidScoreWindow(parts.contradictionRatioMin, parts.contradictionRatioMax),
		},
		{param: "freshness_score_min", invalid: hasInvalidScoreWindow(parts.freshnessScoreMin, parts.freshnessScoreMax)},
		{
			param:   "useful_feedback_ratio_min",
			invalid: hasInvalidScoreWindow(parts.usefulFeedbackRatioMin, parts.usefulFeedbackRatioMax),
		},
		{
			param:   "avg_relevance_feedback_min",
			invalid: hasInvalidScoreWindow(parts.avgRelevanceFeedbackMin, parts.avgRelevanceFeedbackMax),
		},
		{
			param:   "source_session_quality_min",
			invalid: hasInvalidScoreWindow(parts.sourceSessionQualityMin, parts.sourceSessionQualityMax),
		},
		{param: "dense_score_min", invalid: hasInvalidScoreWindow(parts.denseScoreMin, parts.denseScoreMax)},
		{
			param:   "lexical_overlap_score_min",
			invalid: hasInvalidScoreWindow(parts.lexicalOverlapScoreMin, parts.lexicalOverlapScoreMax),
		},
		{
			param:   "feedback_signal_score_min",
			invalid: hasInvalidScoreWindow(parts.feedbackSignalScoreMin, parts.feedbackSignalScoreMax),
		},
		{
			param:   "engagement_signal_score_min",
			invalid: hasInvalidScoreWindow(parts.engagementSignalScoreMin, parts.engagementSignalScoreMax),
		},
		{
			param:   "freshness_signal_score_min",
			invalid: hasInvalidScoreWindow(parts.freshnessSignalScoreMin, parts.freshnessSignalScoreMax),
		},
		{
			param:   "authority_signal_score_min",
			invalid: hasInvalidScoreWindow(parts.authoritySignalScoreMin, parts.authoritySignalScoreMax),
		},
		{
			param:   "composite_rank_score_min",
			invalid: hasInvalidScoreWindow(parts.compositeRankScoreMin, parts.compositeRankScoreMax),
		},
	}
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
	return parseFieldParserSpecs(params, integerEngagementFieldSpecs())
}

type integerEngagementFieldSpec struct {
	parse  func(map[string]any) (*int, *toolDispatchError)
	assign func(*engramQueryIntegerEngagementParts, *int)
}

func integerEngagementFieldSpecs() []integerEngagementFieldSpec {
	return []integerEngagementFieldSpec{
		{
			parse: nonNegativeIntFieldParser("useful_count_min"),
			assign: func(parts *engramQueryIntegerEngagementParts, value *int) {
				parts.usefulCountMin = value
			},
		},
		{
			parse: nonNegativeIntFieldParser("useful_count_max"),
			assign: func(parts *engramQueryIntegerEngagementParts, value *int) {
				parts.usefulCountMax = value
			},
		},
		{
			parse: nonNegativeIntFieldParser("access_count_min"),
			assign: func(parts *engramQueryIntegerEngagementParts, value *int) {
				parts.accessCountMin = value
			},
		},
		{
			parse: nonNegativeIntFieldParser("access_count_max"),
			assign: func(parts *engramQueryIntegerEngagementParts, value *int) {
				parts.accessCountMax = value
			},
		},
		{
			parse: nonNegativeIntFieldParser("feedback_count_min"),
			assign: func(parts *engramQueryIntegerEngagementParts, value *int) {
				parts.feedbackCountMin = value
			},
		},
		{
			parse: nonNegativeIntFieldParser("feedback_count_max"),
			assign: func(parts *engramQueryIntegerEngagementParts, value *int) {
				parts.feedbackCountMax = value
			},
		},
		{
			parse: nonNegativeIntFieldParser("contradiction_count_min"),
			assign: func(parts *engramQueryIntegerEngagementParts, value *int) {
				parts.contradictionCountMin = value
			},
		},
		{
			parse: nonNegativeIntFieldParser("contradiction_count_max"),
			assign: func(parts *engramQueryIntegerEngagementParts, value *int) {
				parts.contradictionCountMax = value
			},
		},
	}
}

func nonNegativeIntFieldParser(paramName string) func(map[string]any) (*int, *toolDispatchError) {
	return func(params map[string]any) (*int, *toolDispatchError) {
		return parseEngramQueryNonNegativeIntPointer(params, paramName)
	}
}

type engramQueryScoreEngagementParts struct {
	contradictionRatioMin    *float64
	contradictionRatioMax    *float64
	freshnessScoreMin        *float64
	freshnessScoreMax        *float64
	usefulFeedbackRatioMin   *float64
	usefulFeedbackRatioMax   *float64
	avgRelevanceFeedbackMin  *float64
	avgRelevanceFeedbackMax  *float64
	sourceSessionQualityMin  *float64
	sourceSessionQualityMax  *float64
	denseScoreMin            *float64
	denseScoreMax            *float64
	lexicalOverlapScoreMin   *float64
	lexicalOverlapScoreMax   *float64
	feedbackSignalScoreMin   *float64
	feedbackSignalScoreMax   *float64
	engagementSignalScoreMin *float64
	engagementSignalScoreMax *float64
	freshnessSignalScoreMin  *float64
	freshnessSignalScoreMax  *float64
	authoritySignalScoreMin  *float64
	authoritySignalScoreMax  *float64
	compositeRankScoreMin    *float64
	compositeRankScoreMax    *float64
}

func parseEngramQueryScoreEngagementParts(
	params map[string]any,
) (engramQueryScoreEngagementParts, *toolDispatchError) {
	return parseFieldParserSpecs(params, scoreEngagementFieldSpecs())
}

type scoreEngagementFieldSpec struct {
	parse  func(map[string]any) (*float64, *toolDispatchError)
	assign func(*engramQueryScoreEngagementParts, *float64)
}

var scoreEngagementFieldSpecsFixture = []scoreEngagementFieldSpec{
	{
		parse: parseEngramQueryContradictionFeedbackRatioMin,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.contradictionRatioMin = value
		},
	},
	{
		parse: parseEngramQueryContradictionFeedbackRatioMax,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.contradictionRatioMax = value
		},
	},
	{
		parse: parseEngramQueryFreshnessScoreMin,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.freshnessScoreMin = value
		},
	},
	{
		parse: parseEngramQueryFreshnessScoreMax,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.freshnessScoreMax = value
		},
	},
	{
		parse: parseEngramQueryUsefulFeedbackRatioMin,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.usefulFeedbackRatioMin = value
		},
	},
	{
		parse: parseEngramQueryUsefulFeedbackRatioMax,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.usefulFeedbackRatioMax = value
		},
	},
	{
		parse: parseEngramQueryAvgRelevanceFeedbackMin,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.avgRelevanceFeedbackMin = value
		},
	},
	{
		parse: parseEngramQueryAvgRelevanceFeedbackMax,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.avgRelevanceFeedbackMax = value
		},
	},
	{
		parse: parseEngramQuerySourceSessionQualityMin,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.sourceSessionQualityMin = value
		},
	},
	{
		parse: parseEngramQuerySourceSessionQualityMax,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.sourceSessionQualityMax = value
		},
	},
	{
		parse: parseEngramQueryDenseScoreMin,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.denseScoreMin = value
		},
	},
	{
		parse: parseEngramQueryDenseScoreMax,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.denseScoreMax = value
		},
	},
	{
		parse: parseEngramQueryLexicalOverlapScoreMin,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.lexicalOverlapScoreMin = value
		},
	},
	{
		parse: parseEngramQueryLexicalOverlapScoreMax,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.lexicalOverlapScoreMax = value
		},
	},
	{
		parse: parseEngramQueryFeedbackSignalScoreMin,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.feedbackSignalScoreMin = value
		},
	},
	{
		parse: parseEngramQueryFeedbackSignalScoreMax,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.feedbackSignalScoreMax = value
		},
	},
	{
		parse: parseEngramQueryEngagementSignalScoreMin,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.engagementSignalScoreMin = value
		},
	},
	{
		parse: parseEngramQueryEngagementSignalScoreMax,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.engagementSignalScoreMax = value
		},
	},
	{
		parse: parseEngramQueryFreshnessSignalScoreMin,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.freshnessSignalScoreMin = value
		},
	},
	{
		parse: parseEngramQueryFreshnessSignalScoreMax,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.freshnessSignalScoreMax = value
		},
	},
	{
		parse: parseEngramQueryAuthoritySignalScoreMin,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.authoritySignalScoreMin = value
		},
	},
	{
		parse: parseEngramQueryAuthoritySignalScoreMax,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.authoritySignalScoreMax = value
		},
	},
	{
		parse: parseEngramQueryCompositeRankScoreMin,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.compositeRankScoreMin = value
		},
	},
	{
		parse: parseEngramQueryCompositeRankScoreMax,
		assign: func(parts *engramQueryScoreEngagementParts, value *float64) {
			parts.compositeRankScoreMax = value
		},
	},
}

func scoreEngagementFieldSpecs() []scoreEngagementFieldSpec {
	return append([]scoreEngagementFieldSpec(nil), scoreEngagementFieldSpecsFixture...)
}

type fieldParserSpec[T any, V any] interface {
	parseField(map[string]any) (V, *toolDispatchError)
	assignField(*T, V)
}

func parseFieldParserSpecs[T any, V any, S interface {
	parseField(map[string]any) (V, *toolDispatchError)
	assignField(*T, V)
}](params map[string]any, specs []S) (T, *toolDispatchError) {
	var parts T
	for _, spec := range specs {
		value, dispatchErr := spec.parseField(params)
		if dispatchErr != nil {
			var zero T
			return zero, dispatchErr
		}
		spec.assignField(&parts, value)
	}
	return parts, nil
}

func (spec integerEngagementFieldSpec) parseField(params map[string]any) (*int, *toolDispatchError) {
	return spec.parse(params)
}

func (spec integerEngagementFieldSpec) assignField(
	parts *engramQueryIntegerEngagementParts,
	value *int,
) {
	spec.assign(parts, value)
}

func (spec scoreEngagementFieldSpec) parseField(params map[string]any) (*float64, *toolDispatchError) {
	return spec.parse(params)
}

func (spec scoreEngagementFieldSpec) assignField(
	parts *engramQueryScoreEngagementParts,
	value *float64,
) {
	spec.assign(parts, value)
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
		Query:                    query,
		TopK:                     parts.topK,
		ProjectID:                parts.projectID,
		Tags:                     parts.tags,
		Keywords:                 parts.keywords,
		CreatedAfter:             parts.createdAfter,
		CreatedBefore:            parts.createdBefore,
		DistanceMin:              parts.distanceMin,
		DistanceMax:              parts.distanceMax,
		UsefulCountMin:           parts.usefulCountMin,
		UsefulCountMax:           parts.usefulCountMax,
		AccessCountMin:           parts.accessCountMin,
		AccessCountMax:           parts.accessCountMax,
		FeedbackCountMin:         parts.feedbackCountMin,
		FeedbackCountMax:         parts.feedbackCountMax,
		ContradictionCountMin:    parts.contradictionCountMin,
		ContradictionCountMax:    parts.contradictionCountMax,
		ContradictionRatioMin:    parts.contradictionRatioMin,
		ContradictionRatioMax:    parts.contradictionRatioMax,
		FreshnessScoreMin:        parts.freshnessScoreMin,
		FreshnessScoreMax:        parts.freshnessScoreMax,
		UsefulFeedbackRatioMin:   parts.usefulFeedbackRatioMin,
		UsefulFeedbackRatioMax:   parts.usefulFeedbackRatioMax,
		AvgRelevanceFeedbackMin:  parts.avgRelevanceFeedbackMin,
		AvgRelevanceFeedbackMax:  parts.avgRelevanceFeedbackMax,
		SourceSessionQualityMin:  parts.sourceSessionQualityMin,
		SourceSessionQualityMax:  parts.sourceSessionQualityMax,
		DenseScoreMin:            parts.denseScoreMin,
		DenseScoreMax:            parts.denseScoreMax,
		LexicalOverlapScoreMin:   parts.lexicalOverlapScoreMin,
		LexicalOverlapScoreMax:   parts.lexicalOverlapScoreMax,
		FeedbackSignalScoreMin:   parts.feedbackSignalScoreMin,
		FeedbackSignalScoreMax:   parts.feedbackSignalScoreMax,
		EngagementSignalScoreMin: parts.engagementSignalScoreMin,
		EngagementSignalScoreMax: parts.engagementSignalScoreMax,
		FreshnessSignalScoreMin:  parts.freshnessSignalScoreMin,
		FreshnessSignalScoreMax:  parts.freshnessSignalScoreMax,
		AuthoritySignalScoreMin:  parts.authoritySignalScoreMin,
		AuthoritySignalScoreMax:  parts.authoritySignalScoreMax,
		CompositeRankScoreMin:    parts.compositeRankScoreMin,
		CompositeRankScoreMax:    parts.compositeRankScoreMax,
		LastAccessedAfter:        parts.lastAccessedAfter,
		LastAccessedBefore:       parts.lastAccessedBefore,
		FreshnessComputedAfter:   parts.freshnessComputedAfter,
		FreshnessComputedBefore:  parts.freshnessComputedBefore,
		RelationType:             parts.relationType,
		TraceDepth:               parts.traceDepth,
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

func parseEngramQueryDenseScoreMin(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "dense_score_min")
}

func parseEngramQueryDenseScoreMax(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "dense_score_max")
}

func parseEngramQueryLexicalOverlapScoreMin(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "lexical_overlap_score_min")
}

func parseEngramQueryLexicalOverlapScoreMax(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "lexical_overlap_score_max")
}

func parseEngramQueryFeedbackSignalScoreMin(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "feedback_signal_score_min")
}

func parseEngramQueryFeedbackSignalScoreMax(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "feedback_signal_score_max")
}

func parseEngramQueryEngagementSignalScoreMin(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "engagement_signal_score_min")
}

func parseEngramQueryEngagementSignalScoreMax(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "engagement_signal_score_max")
}

func parseEngramQueryFreshnessSignalScoreMin(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "freshness_signal_score_min")
}

func parseEngramQueryFreshnessSignalScoreMax(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "freshness_signal_score_max")
}

func parseEngramQueryAuthoritySignalScoreMin(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "authority_signal_score_min")
}

func parseEngramQueryAuthoritySignalScoreMax(params map[string]any) (*float64, *toolDispatchError) {
	return parseEngramQueryBoundedScoreMin(params, "authority_signal_score_max")
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
	return parseEngramQueryFloatWithValidator(
		params,
		key,
		func(value float64) bool { return value >= 0 && value <= 1 },
	)
}

func parseEngramQueryFloatWithValidator(
	params map[string]any,
	key string,
	validator func(float64) bool,
) (*float64, *toolDispatchError) {
	rawValue, found := optionalParamValue(params, key)
	if !found {
		return nil, nil
	}
	parsed, ok := parseFloatValue(rawValue)
	if !ok {
		return nil, invalidParamError(key)
	}
	if !validator(parsed) {
		return nil, invalidParamError(key)
	}
	copy := parsed
	return &copy, nil
}

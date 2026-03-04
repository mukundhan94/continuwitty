package api

import (
	"strings"
	"time"

	"engram/internal/models"
)

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
	if detail := invalidQueryTemporalWindowDetail(payload); detail != "" {
		return detail
	}
	return invalidQueryNumericWindowDetail(payload)
}

type queryEngramValidationRule struct {
	detail  string
	invalid bool
}

func queryEngramValidationRules(payload models.EngramQueryRequest) []queryEngramValidationRule {
	rules := make([]queryEngramValidationRule, 0, 40)
	rules = append(rules, queryEngramCoreValidationRules(payload)...)
	rules = append(rules, queryEngramCountValidationRules(payload)...)
	rules = append(rules, queryEngramRatioValidationRules(payload)...)
	rules = append(rules, queryEngramRerankValidationRules(payload)...)
	rules = append(rules, queryEngramTraceValidationRules(payload)...)
	return rules
}

type nonNegativeIntSpec struct {
	value *int
}

func hasInvalidNonNegativeInt(spec nonNegativeIntSpec) bool {
	return spec.value != nil && *spec.value < 0
}

type minFloatSpec struct {
	value *float64
	min   float64
}

func hasInvalidMinFloat(spec minFloatSpec) bool {
	return spec.value != nil && *spec.value < spec.min
}

type unitIntervalSpec struct {
	value *float64
}

func hasInvalidUnitInterval(spec unitIntervalSpec) bool {
	if spec.value == nil {
		return false
	}
	if *spec.value < 0 {
		return true
	}
	return *spec.value > 1
}

func queryEngramCoreValidationRules(payload models.EngramQueryRequest) []queryEngramValidationRule {
	return []queryEngramValidationRule{
		{detail: "query is required", invalid: payload.Query == ""},
		{detail: "invalid top_k", invalid: payload.TopK < 1 || payload.TopK > 50},
		{
			detail: "invalid distance_min",
			invalid: hasInvalidMinFloat(
				minFloatSpec{value: payload.DistanceMin, min: 0},
			),
		},
		{
			detail: "invalid distance_max",
			invalid: hasInvalidMinFloat(
				minFloatSpec{value: payload.DistanceMax, min: 0},
			),
		},
	}
}

func queryEngramCountValidationRules(payload models.EngramQueryRequest) []queryEngramValidationRule {
	return []queryEngramValidationRule{
		{
			detail: "invalid useful_count_min",
			invalid: hasInvalidNonNegativeInt(
				nonNegativeIntSpec{value: payload.UsefulCountMin},
			),
		},
		{
			detail: "invalid useful_count_max",
			invalid: hasInvalidNonNegativeInt(
				nonNegativeIntSpec{value: payload.UsefulCountMax},
			),
		},
		{
			detail: "invalid access_count_min",
			invalid: hasInvalidNonNegativeInt(
				nonNegativeIntSpec{value: payload.AccessCountMin},
			),
		},
		{
			detail: "invalid access_count_max",
			invalid: hasInvalidNonNegativeInt(
				nonNegativeIntSpec{value: payload.AccessCountMax},
			),
		},
		{
			detail: "invalid feedback_count_min",
			invalid: hasInvalidNonNegativeInt(
				nonNegativeIntSpec{value: payload.FeedbackCountMin},
			),
		},
		{
			detail: "invalid feedback_count_max",
			invalid: hasInvalidNonNegativeInt(
				nonNegativeIntSpec{value: payload.FeedbackCountMax},
			),
		},
		{
			detail: "invalid contradiction_count_min",
			invalid: hasInvalidNonNegativeInt(
				nonNegativeIntSpec{value: payload.ContradictionCountMin},
			),
		},
		{
			detail: "invalid contradiction_count_max",
			invalid: hasInvalidNonNegativeInt(
				nonNegativeIntSpec{value: payload.ContradictionCountMax},
			),
		},
	}
}

func queryEngramRatioValidationRules(payload models.EngramQueryRequest) []queryEngramValidationRule {
	return []queryEngramValidationRule{
		{
			detail: "invalid contradiction_feedback_ratio_min",
			invalid: hasInvalidUnitInterval(
				unitIntervalSpec{value: payload.ContradictionRatioMin},
			),
		},
		{
			detail: "invalid contradiction_feedback_ratio_max",
			invalid: hasInvalidUnitInterval(
				unitIntervalSpec{value: payload.ContradictionRatioMax},
			),
		},
		{
			detail: "invalid freshness_score_min",
			invalid: hasInvalidUnitInterval(
				unitIntervalSpec{value: payload.FreshnessScoreMin},
			),
		},
		{
			detail: "invalid freshness_score_max",
			invalid: hasInvalidUnitInterval(
				unitIntervalSpec{value: payload.FreshnessScoreMax},
			),
		},
		{
			detail: "invalid useful_feedback_ratio_min",
			invalid: hasInvalidUnitInterval(
				unitIntervalSpec{value: payload.UsefulFeedbackRatioMin},
			),
		},
		{
			detail: "invalid useful_feedback_ratio_max",
			invalid: hasInvalidUnitInterval(
				unitIntervalSpec{value: payload.UsefulFeedbackRatioMax},
			),
		},
		{
			detail: "invalid avg_relevance_feedback_min",
			invalid: hasInvalidUnitInterval(
				unitIntervalSpec{value: payload.AvgRelevanceFeedbackMin},
			),
		},
		{
			detail: "invalid avg_relevance_feedback_max",
			invalid: hasInvalidUnitInterval(
				unitIntervalSpec{value: payload.AvgRelevanceFeedbackMax},
			),
		},
		{
			detail: "invalid source_session_quality_min",
			invalid: hasInvalidUnitInterval(
				unitIntervalSpec{value: payload.SourceSessionQualityMin},
			),
		},
		{
			detail: "invalid source_session_quality_max",
			invalid: hasInvalidUnitInterval(
				unitIntervalSpec{value: payload.SourceSessionQualityMax},
			),
		},
	}
}

func queryEngramRerankValidationRules(payload models.EngramQueryRequest) []queryEngramValidationRule {
	return []queryEngramValidationRule{
		unitIntervalValidationRule("invalid dense_score_min", payload.DenseScoreMin),
		unitIntervalValidationRule("invalid dense_score_max", payload.DenseScoreMax),
		unitIntervalValidationRule("invalid lexical_overlap_score_min", payload.LexicalOverlapScoreMin),
		unitIntervalValidationRule("invalid lexical_overlap_score_max", payload.LexicalOverlapScoreMax),
		unitIntervalValidationRule("invalid feedback_signal_score_min", payload.FeedbackSignalScoreMin),
		unitIntervalValidationRule("invalid feedback_signal_score_max", payload.FeedbackSignalScoreMax),
		unitIntervalValidationRule("invalid engagement_signal_score_min", payload.EngagementSignalScoreMin),
		unitIntervalValidationRule("invalid engagement_signal_score_max", payload.EngagementSignalScoreMax),
		unitIntervalValidationRule("invalid freshness_signal_score_min", payload.FreshnessSignalScoreMin),
		unitIntervalValidationRule("invalid freshness_signal_score_max", payload.FreshnessSignalScoreMax),
		unitIntervalValidationRule("invalid authority_signal_score_min", payload.AuthoritySignalScoreMin),
		unitIntervalValidationRule("invalid authority_signal_score_max", payload.AuthoritySignalScoreMax),
		unitIntervalValidationRule("invalid composite_rank_score_min", payload.CompositeRankScoreMin),
		unitIntervalValidationRule("invalid composite_rank_score_max", payload.CompositeRankScoreMax),
	}
}

func unitIntervalValidationRule(detail string, value *float64) queryEngramValidationRule {
	return queryEngramValidationRule{
		detail:  detail,
		invalid: hasInvalidUnitInterval(unitIntervalSpec{value: value}),
	}
}

func queryEngramTraceValidationRules(payload models.EngramQueryRequest) []queryEngramValidationRule {
	return []queryEngramValidationRule{
		{
			detail: "invalid relation_type",
			invalid: hasInvalidRelationType(
				relationTypeValidationSpec{value: payload.RelationType},
			),
		},
		{
			detail: "invalid trace_depth",
			invalid: hasInvalidTraceDepth(
				traceDepthValidationSpec{value: payload.TraceDepth},
			),
		},
		{
			detail: "invalid trace_depth",
			invalid: hasRelationTypeTraceDepthConflict(
				relationTraceConflictSpec{
					relationType: payload.RelationType,
					traceDepth:   payload.TraceDepth,
				},
			),
		},
	}
}

type queryTemporalWindowSpec struct {
	after         *time.Time
	before        *time.Time
	invalidDetail string
}

type temporalWindowValidationSpec struct {
	after  *time.Time
	before *time.Time
}

func hasInvalidTemporalWindow(spec temporalWindowValidationSpec) bool {
	if spec.after == nil || spec.before == nil {
		return false
	}
	return spec.after.After(*spec.before)
}

type intWindowValidationSpec struct {
	minValue *int
	maxValue *int
}

func hasInvalidIntWindow(spec intWindowValidationSpec) bool {
	if spec.minValue == nil || spec.maxValue == nil {
		return false
	}
	return *spec.minValue > *spec.maxValue
}

type scoreWindowValidationSpec struct {
	minValue *float64
	maxValue *float64
}

func hasInvalidScoreWindow(spec scoreWindowValidationSpec) bool {
	if spec.minValue == nil || spec.maxValue == nil {
		return false
	}
	return *spec.minValue > *spec.maxValue
}

type relationTypeValidationSpec struct {
	value *models.EngramLinkRelationType
}

func hasInvalidRelationType(spec relationTypeValidationSpec) bool {
	if spec.value == nil {
		return false
	}
	_, err := models.ParseEngramLinkRelationType(strings.TrimSpace(string(*spec.value)))
	return err != nil
}

type traceDepthValidationSpec struct {
	value *int
}

func hasInvalidTraceDepth(spec traceDepthValidationSpec) bool {
	if spec.value == nil {
		return false
	}
	if *spec.value < 0 {
		return true
	}
	return *spec.value > 1
}

type relationTraceConflictSpec struct {
	relationType *models.EngramLinkRelationType
	traceDepth   *int
}

func hasRelationTypeTraceDepthConflict(spec relationTraceConflictSpec) bool {
	return spec.relationType != nil && spec.traceDepth != nil && *spec.traceDepth == 0
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
		if hasInvalidTemporalWindow(
			temporalWindowValidationSpec{
				after:  spec.after,
				before: spec.before,
			},
		) {
			return spec.invalidDetail
		}
	}
	return ""
}

type queryNumericWindowSpec struct {
	detail  string
	invalid bool
}

func invalidQueryNumericWindowDetail(payload models.EngramQueryRequest) string {
	for _, spec := range queryNumericWindowSpecs(payload) {
		if spec.invalid {
			return spec.detail
		}
	}
	return ""
}

func queryNumericWindowSpecs(payload models.EngramQueryRequest) []queryNumericWindowSpec {
	return []queryNumericWindowSpec{
		intWindowRule("invalid useful_count window", payload.UsefulCountMin, payload.UsefulCountMax),
		intWindowRule("invalid access_count window", payload.AccessCountMin, payload.AccessCountMax),
		scoreWindowRule("invalid distance window", payload.DistanceMin, payload.DistanceMax),
		intWindowRule("invalid feedback_count window", payload.FeedbackCountMin, payload.FeedbackCountMax),
		intWindowRule("invalid contradiction_count window", payload.ContradictionCountMin, payload.ContradictionCountMax),
		scoreWindowRule("invalid contradiction_feedback_ratio window", payload.ContradictionRatioMin, payload.ContradictionRatioMax),
		scoreWindowRule("invalid freshness_score window", payload.FreshnessScoreMin, payload.FreshnessScoreMax),
		scoreWindowRule("invalid useful_feedback_ratio window", payload.UsefulFeedbackRatioMin, payload.UsefulFeedbackRatioMax),
		scoreWindowRule("invalid avg_relevance_feedback window", payload.AvgRelevanceFeedbackMin, payload.AvgRelevanceFeedbackMax),
		scoreWindowRule("invalid source_session_quality window", payload.SourceSessionQualityMin, payload.SourceSessionQualityMax),
		scoreWindowRule("invalid dense_score window", payload.DenseScoreMin, payload.DenseScoreMax),
		scoreWindowRule("invalid lexical_overlap_score window", payload.LexicalOverlapScoreMin, payload.LexicalOverlapScoreMax),
		scoreWindowRule("invalid feedback_signal_score window", payload.FeedbackSignalScoreMin, payload.FeedbackSignalScoreMax),
		scoreWindowRule("invalid engagement_signal_score window", payload.EngagementSignalScoreMin, payload.EngagementSignalScoreMax),
		scoreWindowRule("invalid freshness_signal_score window", payload.FreshnessSignalScoreMin, payload.FreshnessSignalScoreMax),
		scoreWindowRule("invalid authority_signal_score window", payload.AuthoritySignalScoreMin, payload.AuthoritySignalScoreMax),
		scoreWindowRule("invalid composite_rank_score window", payload.CompositeRankScoreMin, payload.CompositeRankScoreMax),
	}
}

func intWindowRule(detail string, minValue *int, maxValue *int) queryNumericWindowSpec {
	return queryNumericWindowSpec{
		detail: detail,
		invalid: hasInvalidIntWindow(
			intWindowValidationSpec{minValue: minValue, maxValue: maxValue},
		),
	}
}

func scoreWindowRule(detail string, minValue *float64, maxValue *float64) queryNumericWindowSpec {
	return queryNumericWindowSpec{
		detail: detail,
		invalid: hasInvalidScoreWindow(
			scoreWindowValidationSpec{minValue: minValue, maxValue: maxValue},
		),
	}
}

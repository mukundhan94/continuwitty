package api

import (
	"testing"
	"time"

	"engram/internal/models"
)

func assertTemporalQueryRequest(t *testing.T, request models.EngramQueryRequest) {
	t.Helper()
	assertTemporalQueryCoreFields(t, request)
	assertTemporalQueryCountAndDistanceFields(t, request)
	assertTemporalQueryRatioFields(t, request)
	assertTemporalQueryRerankFields(t, request)
	assertTemporalQueryTraceFields(t, request)
}

func assertTemporalQueryCoreFields(t *testing.T, request models.EngramQueryRequest) {
	t.Helper()
	requireEqual(t, "durable memory", request.Query)
	requireEqual(t, 5, request.TopK)
	requireEqual(t, 1, requireIntPointer(t, request.UsefulCountMin, "useful_count_min"))
	requireEqual(t, 8, requireIntPointer(t, request.UsefulCountMax, "useful_count_max"))
	requireEqual(t, 2, requireIntPointer(t, request.AccessCountMin, "access_count_min"))
	requireEqual(t, 20, requireIntPointer(t, request.AccessCountMax, "access_count_max"))
	requireEqual(t, 0.1, requireFloat64Pointer(t, request.DistanceMin, "distance_min"))
	requireEqual(t, 0.45, requireFloat64Pointer(t, request.DistanceMax, "distance_max"))
	requireEqual(t, 4, requireIntPointer(t, request.FeedbackCountMin, "feedback_count_min"))
	requireEqual(t, 10, requireIntPointer(t, request.FeedbackCountMax, "feedback_count_max"))
	requireEqual(t, 1, requireIntPointer(t, request.ContradictionCountMin, "contradiction_count_min"))
	requireEqual(t, 3, requireIntPointer(t, request.ContradictionCountMax, "contradiction_count_max"))
}

func assertTemporalQueryCountAndDistanceFields(t *testing.T, request models.EngramQueryRequest) {
	t.Helper()
	requireEqual(t, 0.1, requireFloat64Pointer(t, request.ContradictionRatioMin, "contradiction_feedback_ratio_min"))
	requireEqual(t, 0.3, requireFloat64Pointer(t, request.ContradictionRatioMax, "contradiction_feedback_ratio_max"))
	requireEqual(t, 0.4, requireFloat64Pointer(t, request.FreshnessScoreMin, "freshness_score_min"))
	requireEqual(t, 0.9, requireFloat64Pointer(t, request.FreshnessScoreMax, "freshness_score_max"))
	requireEqual(t, 0.8, requireFloat64Pointer(t, request.UsefulFeedbackRatioMin, "useful_feedback_ratio_min"))
	requireEqual(t, 0.95, requireFloat64Pointer(t, request.UsefulFeedbackRatioMax, "useful_feedback_ratio_max"))
	requireEqual(t, 0.55, requireFloat64Pointer(t, request.AvgRelevanceFeedbackMin, "avg_relevance_feedback_min"))
	requireEqual(t, 0.9, requireFloat64Pointer(t, request.AvgRelevanceFeedbackMax, "avg_relevance_feedback_max"))
	requireEqual(t, 0.7, requireFloat64Pointer(t, request.SourceSessionQualityMin, "source_session_quality_min"))
	requireEqual(t, 0.9, requireFloat64Pointer(t, request.SourceSessionQualityMax, "source_session_quality_max"))
}

func assertTemporalQueryRatioFields(t *testing.T, request models.EngramQueryRequest) {
	t.Helper()
	requireEqual(t, 0.72, requireFloat64Pointer(t, request.DenseScoreMin, "dense_score_min"))
	requireEqual(t, 0.96, requireFloat64Pointer(t, request.DenseScoreMax, "dense_score_max"))
	requireEqual(t, 0.65, requireFloat64Pointer(t, request.LexicalOverlapScoreMin, "lexical_overlap_score_min"))
	requireEqual(t, 0.98, requireFloat64Pointer(t, request.LexicalOverlapScoreMax, "lexical_overlap_score_max"))
	requireEqual(t, 0.55, requireFloat64Pointer(t, request.FeedbackSignalScoreMin, "feedback_signal_score_min"))
	requireEqual(t, 0.94, requireFloat64Pointer(t, request.FeedbackSignalScoreMax, "feedback_signal_score_max"))
}

func assertTemporalQueryRerankFields(t *testing.T, request models.EngramQueryRequest) {
	t.Helper()
	requireEqual(t, 0.45, requireFloat64Pointer(t, request.EngagementSignalScoreMin, "engagement_signal_score_min"))
	requireEqual(t, 0.92, requireFloat64Pointer(t, request.EngagementSignalScoreMax, "engagement_signal_score_max"))
	requireEqual(t, 0.41, requireFloat64Pointer(t, request.FreshnessSignalScoreMin, "freshness_signal_score_min"))
	requireEqual(t, 0.97, requireFloat64Pointer(t, request.FreshnessSignalScoreMax, "freshness_signal_score_max"))
	requireEqual(t, 0.38, requireFloat64Pointer(t, request.AuthoritySignalScoreMin, "authority_signal_score_min"))
	requireEqual(t, 0.96, requireFloat64Pointer(t, request.AuthoritySignalScoreMax, "authority_signal_score_max"))
	requireEqual(t, 0.75, requireFloat64Pointer(t, request.CompositeRankScoreMin, "composite_rank_score_min"))
	requireEqual(t, 0.95, requireFloat64Pointer(t, request.CompositeRankScoreMax, "composite_rank_score_max"))
}

func assertTemporalQueryTraceFields(t *testing.T, request models.EngramQueryRequest) {
	t.Helper()
	requireTimeWindowPresent(t, request.LastAccessedAfter, request.LastAccessedBefore, "last_accessed")
	requireTimeWindowPresent(t, request.FreshnessComputedAfter, request.FreshnessComputedBefore, "freshness_computed")
	requireEqual(t, models.EngramLinkRelationSupports, requireRelationTypePointer(t, request.RelationType, "relation_type"))
	requireEqual(t, 1, requireIntPointer(t, request.TraceDepth, "trace_depth"))
}

func requireIntPointer(t *testing.T, value *int, fieldName string) int {
	t.Helper()
	if value == nil {
		t.Fatalf("expected %s to be parsed", fieldName)
	}
	return *value
}

func requireFloat64Pointer(t *testing.T, value *float64, fieldName string) float64 {
	t.Helper()
	if value == nil {
		t.Fatalf("expected %s to be parsed", fieldName)
	}
	return *value
}

func requireRelationTypePointer(
	t *testing.T,
	value *models.EngramLinkRelationType,
	fieldName string,
) models.EngramLinkRelationType {
	t.Helper()
	if value == nil {
		t.Fatalf("expected %s to be parsed", fieldName)
	}
	return *value
}

func requireTimeWindowPresent(
	t *testing.T,
	after *time.Time,
	before *time.Time,
	fieldName string,
) {
	t.Helper()
	if after == nil || before == nil {
		t.Fatalf("expected %s window to be parsed", fieldName)
	}
}

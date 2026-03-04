package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMapEngramQueryResultIncludesFeedbackQualityDiagnostics(t *testing.T) {
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000341")
	row := map[string]any{
		"engram_id":                    uuid.MustParse("00000000-0000-0000-0000-000000000342"),
		"project_id":                   "engram-vault",
		"title":                        "Feedback calibrated memory",
		"abstract":                     "Maintains useful and relevance diagnostics",
		"created_at":                   time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		"tags":                         []string{"ops"},
		"keywords":                     []string{"diagnostics"},
		"owner_user_id":                &ownerUserID,
		"visibility_scope":             "private",
		"access_count":                 5,
		"freshness_score":              0.91,
		"feedback_count":               4,
		"useful_count":                 3,
		"avg_relevance_feedback":       0.83,
		"contradiction_count":          1,
		"source_session_quality_score": 0.76,
		"composite_rank_score":         0.81,
		"dense_score":                  0.82,
		"lexical_overlap_score":        0.9,
		"feedback_signal_score":        0.6,
		"engagement_signal_score":      0.55,
		"freshness_signal_score":       0.91,
		"authority_signal_score":       0.76,
		"rank_position":                2,
		"distance":                     0.22,
	}

	result := mapEngramQueryResult(row)
	requireEqual(t, 4, result.FeedbackCount)
	requireEqual(t, 3, result.UsefulCount)
	requireEqual(t, 0.83, result.AvgRelevanceFeedback)
	requireEqual(t, 0.75, result.UsefulFeedbackRatio)
	requireEqual(t, 0.25, result.ContradictionFeedbackRatio)
	requireEqual(t, 0.81, result.CompositeRankScore)
	requireEqual(t, 0.82, result.DenseScore)
	requireEqual(t, 0.9, result.LexicalOverlapScore)
	requireEqual(t, 0.6, result.FeedbackSignalScore)
	requireEqual(t, 0.55, result.EngagementSignalScore)
	requireEqual(t, 0.91, result.FreshnessSignalScore)
	requireEqual(t, 0.76, result.AuthoritySignalScore)
	requireEqual(t, 2, result.RankPosition)
}

func TestMapEngramQueryResultComputesRerankDiagnosticsFallbacks(t *testing.T) {
	row := map[string]any{
		"engram_id":                    uuid.MustParse("00000000-0000-0000-0000-000000000343"),
		"project_id":                   "engram-vault",
		"title":                        "Fallback diagnostics",
		"abstract":                     "Computes score fields when diagnostics are absent",
		"created_at":                   time.Date(2026, 3, 3, 0, 5, 0, 0, time.UTC),
		"tags":                         []string{},
		"keywords":                     []string{},
		"visibility_scope":             "private",
		"access_count":                 7,
		"freshness_score":              0.9,
		"feedback_count":               2,
		"useful_count":                 1,
		"avg_relevance_feedback":       0.8,
		"contradiction_count":          1,
		"source_session_quality_score": 0.7,
		"distance":                     0.25,
	}

	result := mapEngramQueryResult(row)
	requireEqual(t, denseDistanceScore(0.25), result.DenseScore)
	requireEqual(t, 0.0, result.LexicalOverlapScore)
	requireEqual(t, normalizeFeedbackScore(1, 1), result.FeedbackSignalScore)
	requireEqual(t, normalizeEngagementScore(7), result.EngagementSignalScore)
	requireEqual(t, 0.9, result.FreshnessSignalScore)
	requireEqual(t, 0.7, result.AuthoritySignalScore)
	requireEqual(
		t,
		combinedRankScore(
			rankScoreInput{
				distance:        0.25,
				lexicalOverlap:  0.0,
				feedbackScore:   normalizeFeedbackScore(1, 1),
				engagementScore: normalizeEngagementScore(7),
				freshnessScore:  0.9,
				authorityScore:  0.7,
			},
		),
		result.CompositeRankScore,
	)
}

func TestUsefulFeedbackRatioFromRowUsesNeutralFallbackWhenNoFeedback(t *testing.T) {
	row := map[string]any{
		"feedback_count": 0,
		"useful_count":   5,
	}
	requireEqual(t, 0.5, usefulFeedbackRatioFromRow(row))
}

func TestContradictionFeedbackRatioFromRowUsesLowRiskFallbackWhenNoFeedback(t *testing.T) {
	row := map[string]any{
		"feedback_count":      0,
		"contradiction_count": 5,
	}
	requireEqual(t, 0.0, contradictionFeedbackRatioFromRow(row))
}

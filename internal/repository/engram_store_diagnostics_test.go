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
		"distance":                     0.22,
	}

	result := mapEngramQueryResult(row)
	requireEqual(t, 4, result.FeedbackCount)
	requireEqual(t, 3, result.UsefulCount)
	requireEqual(t, 0.83, result.AvgRelevanceFeedback)
	requireEqual(t, 0.75, result.UsefulFeedbackRatio)
	requireEqual(t, 0.25, result.ContradictionFeedbackRatio)
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

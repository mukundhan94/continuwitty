package repository

import "testing"

func TestFilterByRankScoreBandsAppliesAllScoreBands(t *testing.T) {
	filtered := filterByRankScoreBands(buildRankScoreBandRows(), strictRankScoreBandFilters())

	requireEqual(t, 1, len(filtered))
	requireEqual(t, "pass", filtered[0]["engram_id"])
}

func TestFilterByRankScoreBandsWithoutFiltersReturnsAllRows(t *testing.T) {
	rows := []map[string]any{
		{"engram_id": "first", "distance": 0.3},
		{"engram_id": "second", "distance": 0.4},
	}

	filtered := filterByRankScoreBands(rows, rankScoreBandFilters{})

	requireEqual(t, len(rows), len(filtered))
	requireEqual(t, "first", filtered[0]["engram_id"])
	requireEqual(t, "second", filtered[1]["engram_id"])
}

func buildRankScoreBandRows() []map[string]any {
	return []map[string]any{
		{
			"engram_id":                  "pass",
			"dense_score":                0.72,
			"lexical_overlap_score":      0.93,
			"feedback_signal_score":      0.67,
			"engagement_signal_score":    0.58,
			"freshness_signal_score":     0.81,
			"authority_signal_score":     0.62,
			"composite_rank_score":       0.79,
			"distance":                   0.25,
			"retrieval_text":             "durable checkpoint",
			"title":                      "durable checkpoint",
			"useful_count":               4,
			"contradiction_count":        1,
			"access_count":               8,
			"freshness_score":            0.81,
			"source_session_quality":     0.62,
			"source_session_quality_raw": 0.62,
		},
		{
			"engram_id":               "fail-authority",
			"dense_score":             0.72,
			"lexical_overlap_score":   0.93,
			"feedback_signal_score":   0.67,
			"engagement_signal_score": 0.58,
			"freshness_signal_score":  0.81,
			"authority_signal_score":  0.88,
			"composite_rank_score":    0.79,
			"distance":                0.25,
			"retrieval_text":          "durable checkpoint",
			"title":                   "durable checkpoint",
			"useful_count":            4,
			"contradiction_count":     1,
			"access_count":            8,
			"freshness_score":         0.81,
		},
		{
			"engram_id":               "fail-dense",
			"dense_score":             0.42,
			"lexical_overlap_score":   0.93,
			"feedback_signal_score":   0.67,
			"engagement_signal_score": 0.58,
			"freshness_signal_score":  0.81,
			"authority_signal_score":  0.62,
			"composite_rank_score":    0.79,
			"distance":                0.25,
			"retrieval_text":          "durable checkpoint",
			"title":                   "durable checkpoint",
			"useful_count":            4,
			"contradiction_count":     1,
			"access_count":            8,
			"freshness_score":         0.81,
		},
	}
}

func strictRankScoreBandFilters() rankScoreBandFilters {
	denseMin := 0.6
	denseMax := 0.8
	lexicalMin := 0.9
	lexicalMax := 1.0
	feedbackMin := 0.6
	feedbackMax := 0.8
	engagementMin := 0.5
	engagementMax := 0.7
	freshnessMin := 0.75
	freshnessMax := 0.9
	authorityMin := 0.55
	authorityMax := 0.7
	compositeMin := 0.7
	compositeMax := 0.85
	return rankScoreBandFilters{
		denseScoreMin:          &denseMin,
		denseScoreMax:          &denseMax,
		lexicalOverlapScoreMin: &lexicalMin,
		lexicalOverlapScoreMax: &lexicalMax,
		feedbackSignalScoreMin: &feedbackMin,
		feedbackSignalScoreMax: &feedbackMax,
		engagementScoreMin:     &engagementMin,
		engagementScoreMax:     &engagementMax,
		freshnessScoreMin:      &freshnessMin,
		freshnessScoreMax:      &freshnessMax,
		authorityScoreMin:      &authorityMin,
		authorityScoreMax:      &authorityMax,
		compositeScoreMin:      &compositeMin,
		compositeScoreMax:      &compositeMax,
	}
}

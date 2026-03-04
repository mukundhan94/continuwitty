package api

var invalidQueryFilterCasesFixture = []invalidQueryFilterCase{
	{
		name: "invalid temporal window",
		body: map[string]any{
			"query":          "durable memory",
			"created_after":  "2026-03-01T00:00:00Z",
			"created_before": "2026-02-01T00:00:00Z",
		},
		expectedDetail: "invalid created_at window",
	},
	{
		name: "invalid trace depth",
		body: map[string]any{
			"query":         "durable memory",
			"relation_type": "supports",
			"trace_depth":   0,
		},
		expectedDetail: "invalid trace_depth",
	},
	{
		name: "invalid source session quality min",
		body: map[string]any{
			"query":                      "durable memory",
			"source_session_quality_min": 1.2,
		},
		expectedDetail: "invalid source_session_quality_min",
	},
	{
		name: "invalid average relevance feedback min",
		body: map[string]any{
			"query":                      "durable memory",
			"avg_relevance_feedback_min": -0.1,
		},
		expectedDetail: "invalid avg_relevance_feedback_min",
	},
	{
		name: "invalid contradiction count max",
		body: map[string]any{
			"query":                   "durable memory",
			"contradiction_count_max": -1,
		},
		expectedDetail: "invalid contradiction_count_max",
	},
	{
		name: "invalid contradiction count min",
		body: map[string]any{
			"query":                   "durable memory",
			"contradiction_count_min": -1,
		},
		expectedDetail: "invalid contradiction_count_min",
	},
	{
		name: "invalid contradiction feedback ratio max",
		body: map[string]any{
			"query":                            "durable memory",
			"contradiction_feedback_ratio_max": 1.2,
		},
		expectedDetail: "invalid contradiction_feedback_ratio_max",
	},
	{
		name: "invalid contradiction feedback ratio min",
		body: map[string]any{
			"query":                            "durable memory",
			"contradiction_feedback_ratio_min": 1.2,
		},
		expectedDetail: "invalid contradiction_feedback_ratio_min",
	},
	{
		name: "invalid feedback count min",
		body: map[string]any{
			"query":              "durable memory",
			"feedback_count_min": -1,
		},
		expectedDetail: "invalid feedback_count_min",
	},
	{
		name: "invalid feedback count max",
		body: map[string]any{
			"query":              "durable memory",
			"feedback_count_max": -1,
		},
		expectedDetail: "invalid feedback_count_max",
	},
	{
		name: "invalid useful count min",
		body: map[string]any{
			"query":            "durable memory",
			"useful_count_min": -1,
		},
		expectedDetail: "invalid useful_count_min",
	},
	{
		name: "invalid useful count max",
		body: map[string]any{
			"query":            "durable memory",
			"useful_count_max": -1,
		},
		expectedDetail: "invalid useful_count_max",
	},
	{
		name: "invalid useful count window",
		body: map[string]any{
			"query":            "durable memory",
			"useful_count_min": 5,
			"useful_count_max": 2,
		},
		expectedDetail: "invalid useful_count window",
	},
	{
		name: "invalid access count max",
		body: map[string]any{
			"query":            "durable memory",
			"access_count_max": -1,
		},
		expectedDetail: "invalid access_count_max",
	},
	{
		name: "invalid distance max",
		body: map[string]any{
			"query":        "durable memory",
			"distance_max": -0.1,
		},
		expectedDetail: "invalid distance_max",
	},
	{
		name: "invalid distance min",
		body: map[string]any{
			"query":        "durable memory",
			"distance_min": -0.1,
		},
		expectedDetail: "invalid distance_min",
	},
	{
		name: "invalid distance window",
		body: map[string]any{
			"query":        "durable memory",
			"distance_min": 0.8,
			"distance_max": 0.2,
		},
		expectedDetail: "invalid distance window",
	},
	{
		name: "invalid freshness score max",
		body: map[string]any{
			"query":               "durable memory",
			"freshness_score_max": 1.2,
		},
		expectedDetail: "invalid freshness_score_max",
	},
	{
		name: "invalid source session quality max",
		body: map[string]any{
			"query":                      "durable memory",
			"source_session_quality_max": 1.2,
		},
		expectedDetail: "invalid source_session_quality_max",
	},
	{
		name: "invalid dense score min",
		body: map[string]any{
			"query":           "durable memory",
			"dense_score_min": 1.2,
		},
		expectedDetail: "invalid dense_score_min",
	},
	{
		name: "invalid dense score max",
		body: map[string]any{
			"query":           "durable memory",
			"dense_score_max": 1.2,
		},
		expectedDetail: "invalid dense_score_max",
	},
	{
		name: "invalid dense score window",
		body: map[string]any{
			"query":           "durable memory",
			"dense_score_min": 0.9,
			"dense_score_max": 0.7,
		},
		expectedDetail: "invalid dense_score window",
	},
	{
		name: "invalid lexical overlap score min",
		body: map[string]any{
			"query":                     "durable memory",
			"lexical_overlap_score_min": 1.2,
		},
		expectedDetail: "invalid lexical_overlap_score_min",
	},
	{
		name: "invalid lexical overlap score max",
		body: map[string]any{
			"query":                     "durable memory",
			"lexical_overlap_score_max": 1.2,
		},
		expectedDetail: "invalid lexical_overlap_score_max",
	},
	{
		name: "invalid lexical overlap score window",
		body: map[string]any{
			"query":                     "durable memory",
			"lexical_overlap_score_min": 0.9,
			"lexical_overlap_score_max": 0.7,
		},
		expectedDetail: "invalid lexical_overlap_score window",
	},
	{
		name: "invalid feedback signal score min",
		body: map[string]any{
			"query":                     "durable memory",
			"feedback_signal_score_min": 1.2,
		},
		expectedDetail: "invalid feedback_signal_score_min",
	},
	{
		name: "invalid feedback signal score max",
		body: map[string]any{
			"query":                     "durable memory",
			"feedback_signal_score_max": 1.2,
		},
		expectedDetail: "invalid feedback_signal_score_max",
	},
	{
		name: "invalid feedback signal score window",
		body: map[string]any{
			"query":                     "durable memory",
			"feedback_signal_score_min": 0.9,
			"feedback_signal_score_max": 0.7,
		},
		expectedDetail: "invalid feedback_signal_score window",
	},
	{
		name: "invalid engagement signal score min",
		body: map[string]any{
			"query":                       "durable memory",
			"engagement_signal_score_min": 1.2,
		},
		expectedDetail: "invalid engagement_signal_score_min",
	},
	{
		name: "invalid engagement signal score max",
		body: map[string]any{
			"query":                       "durable memory",
			"engagement_signal_score_max": 1.2,
		},
		expectedDetail: "invalid engagement_signal_score_max",
	},
	{
		name: "invalid engagement signal score window",
		body: map[string]any{
			"query":                       "durable memory",
			"engagement_signal_score_min": 0.9,
			"engagement_signal_score_max": 0.7,
		},
		expectedDetail: "invalid engagement_signal_score window",
	},
	{
		name: "invalid freshness signal score min",
		body: map[string]any{
			"query":                      "durable memory",
			"freshness_signal_score_min": 1.2,
		},
		expectedDetail: "invalid freshness_signal_score_min",
	},
	{
		name: "invalid freshness signal score max",
		body: map[string]any{
			"query":                      "durable memory",
			"freshness_signal_score_max": 1.2,
		},
		expectedDetail: "invalid freshness_signal_score_max",
	},
	{
		name: "invalid freshness signal score window",
		body: map[string]any{
			"query":                      "durable memory",
			"freshness_signal_score_min": 0.9,
			"freshness_signal_score_max": 0.7,
		},
		expectedDetail: "invalid freshness_signal_score window",
	},
	{
		name: "invalid authority signal score min",
		body: map[string]any{
			"query":                      "durable memory",
			"authority_signal_score_min": 1.2,
		},
		expectedDetail: "invalid authority_signal_score_min",
	},
	{
		name: "invalid authority signal score max",
		body: map[string]any{
			"query":                      "durable memory",
			"authority_signal_score_max": 1.2,
		},
		expectedDetail: "invalid authority_signal_score_max",
	},
	{
		name: "invalid authority signal score window",
		body: map[string]any{
			"query":                      "durable memory",
			"authority_signal_score_min": 0.9,
			"authority_signal_score_max": 0.7,
		},
		expectedDetail: "invalid authority_signal_score window",
	},
	{
		name: "invalid composite rank score min",
		body: map[string]any{
			"query":                    "durable memory",
			"composite_rank_score_min": 1.2,
		},
		expectedDetail: "invalid composite_rank_score_min",
	},
	{
		name: "invalid composite rank score max",
		body: map[string]any{
			"query":                    "durable memory",
			"composite_rank_score_max": 1.2,
		},
		expectedDetail: "invalid composite_rank_score_max",
	},
	{
		name: "invalid composite rank score window",
		body: map[string]any{
			"query":                    "durable memory",
			"composite_rank_score_min": 0.9,
			"composite_rank_score_max": 0.7,
		},
		expectedDetail: "invalid composite_rank_score window",
	},
	{
		name: "invalid avg relevance feedback max",
		body: map[string]any{
			"query":                      "durable memory",
			"avg_relevance_feedback_max": 1.2,
		},
		expectedDetail: "invalid avg_relevance_feedback_max",
	},
	{
		name: "invalid useful feedback ratio max",
		body: map[string]any{
			"query":                     "durable memory",
			"useful_feedback_ratio_max": 1.2,
		},
		expectedDetail: "invalid useful_feedback_ratio_max",
	},
	{
		name: "invalid useful feedback ratio min",
		body: map[string]any{
			"query":                     "durable memory",
			"useful_feedback_ratio_min": 1.2,
		},
		expectedDetail: "invalid useful_feedback_ratio_min",
	},
}

func invalidQueryFilterCases() []invalidQueryFilterCase {
	return invalidQueryFilterCasesFixture
}

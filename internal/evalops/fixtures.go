package evalops

import "engram/internal/governance"

// DefaultSuiteDefinition returns the deterministic Phase-23 EvalOps baseline suite.
func DefaultSuiteDefinition() SuiteDefinition {
	cases := make([]EvalCase, 0, 8)
	cases = append(cases, continuityCases()...)
	cases = append(cases, citationTrustCases()...)
	cases = append(cases, memoryDriftCases()...)
	cases = append(cases, graphTraceCases()...)
	return SuiteDefinition{
		Version: governance.DefaultEvalSuiteVersion,
		Cases:   cases,
	}
}

func continuityCases() []EvalCase {
	return []EvalCase{
		{
			ID:         "continuity_carry_forward",
			Dimension:  DimensionContinuity,
			Prompt:     "Summarize the plan using memory from prior sessions.",
			OutputText: "From the previous session, LangGraph checkpoint resume stays the execution baseline for this project.",
			UsedEngramIDs: []string{
				"f7adf942-5fd2-4a64-a56a-af2e6f7cb340",
				"ca8acabd-3c6f-4fb1-89a5-3bf753e457f9",
			},
			RequiredTerms:      []string{"previous session", "langgraph", "checkpoint resume"},
			MinUsedEngramCount: 2,
		},
		{
			ID:         "continuity_cross_provider_alignment",
			Dimension:  DimensionContinuity,
			Prompt:     "Carry forward consensus between OpenAI and Anthropic threads.",
			OutputText: "OpenAI and Anthropic runs align on the same incident timeline and mitigation plan for checkout recovery.",
			UsedEngramIDs: []string{
				"42fcb930-f2ec-4f0b-bdc6-660fb2f01abf",
			},
			RequiredTerms:      []string{"same incident timeline", "mitigation plan"},
			MinUsedEngramCount: 1,
		},
	}
}

func citationTrustCases() []EvalCase {
	return []EvalCase{
		{
			ID:         "citation_trust_multi_source_alignment",
			Dimension:  DimensionCitationTrust,
			Prompt:     "Provide root-cause summary with supporting citations.",
			OutputText: "The rollback regression and DB saturation were confirmed by the incident postmortem and capacity report [1][2].",
			SourceReferences: []SourceReference{
				{URL: "https://example.com/postmortem", Title: "Incident postmortem"},
				{URL: "https://example.com/db-saturation", Title: "Capacity report"},
			},
			RequiredTerms:      []string{"rollback regression", "db saturation"},
			RequiredSourceURLs: []string{"https://example.com/postmortem", "https://example.com/db-saturation"},
			ForbiddenTerms:     []string{"unverified rumor"},
		},
		{
			ID:         "citation_trust_retention_claim",
			Dimension:  DimensionCitationTrust,
			Prompt:     "State the retention policy and include evidence.",
			OutputText: "Per the lifecycle runbook update [1], retention remains 30 days for session snapshots.",
			SourceReferences: []SourceReference{
				{URL: "https://example.com/lifecycle-runbook", Title: "Lifecycle runbook"},
			},
			RequiredTerms:      []string{"retention", "30 days", "runbook"},
			RequiredSourceURLs: []string{"https://example.com/lifecycle-runbook"},
			ForbiddenTerms:     []string{"guess", "speculation"},
		},
	}
}

func memoryDriftCases() []EvalCase {
	return []EvalCase{
		{
			ID:                  "memory_drift_policy_update_guard",
			Dimension:           DimensionMemoryDrift,
			Prompt:              "Describe the current retrieval policy.",
			OutputText:          "Current policy uses hybrid retrieval with sparse+dense scoring and a 30 day retention window.",
			RequiredTerms:       []string{"hybrid retrieval", "sparse+dense", "30 day retention"},
			ForbiddenTerms:      []string{"dense-only retrieval"},
			BaselineAnchorTerms: []string{"hybrid retrieval", "sparse+dense scoring", "30 day retention"},
			MinAnchorRecall:     0.67,
		},
		{
			ID:                  "memory_drift_provider_reliability_guard",
			Dimension:           DimensionMemoryDrift,
			Prompt:              "State provider reliability controls for transient failures.",
			OutputText:          "Provider fallback remains enabled with circuit breaker cooldown to protect continuity during transient failures.",
			RequiredTerms:       []string{"provider fallback", "circuit breaker", "transient failures"},
			ForbiddenTerms:      []string{"disable fallback"},
			BaselineAnchorTerms: []string{"provider fallback", "circuit breaker cooldown", "transient failures"},
			MinAnchorRecall:     0.67,
		},
	}
}

func graphTraceCases() []EvalCase {
	return []EvalCase{
		{
			ID:                     "graph_trace_provenance_chain_presence",
			Dimension:              DimensionGraphTrace,
			Prompt:                 "Explain the linked memory chain that informed this answer.",
			OutputText:             "This answer followed a linked chain from architecture notes to deployment runbook and incident timeline evidence.",
			UsedEngramIDs:          []string{"e1", "e2"},
			UsedEngramLinkIDs:      []string{"l1", "l2"},
			TraceTargetIDs:         []string{"engram-deploy-runbook", "engram-incident-timeline"},
			RequiredTerms:          []string{"linked chain", "runbook", "incident timeline"},
			MinUsedEngramLinkCount: 2,
			RequiredTraceTargets:   []string{"engram-deploy-runbook"},
		},
		{
			ID:                     "graph_trace_stale_suppression_signal",
			Dimension:              DimensionGraphTrace,
			Prompt:                 "Confirm stale links are suppressed from final response context.",
			OutputText:             "Stale low-confidence links were deprioritized while reinforced active links remained in the provenance chain.",
			UsedEngramLinkIDs:      []string{"l9"},
			TraceTargetIDs:         []string{"engram-active-target"},
			RequiredTerms:          []string{"stale", "deprioritized", "reinforced active links"},
			ForbiddenTerms:         []string{"stale link selected as primary"},
			MinUsedEngramLinkCount: 1,
			RequiredTraceTargets:   []string{"engram-active-target"},
		},
	}
}

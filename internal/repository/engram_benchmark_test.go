package repository

import (
	"fmt"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

var benchmarkScoreBandFilterRowCount int

func BenchmarkRerankByCombinedScore50Candidates(b *testing.B) {
	benchmarkRerankByCombinedScore(b, 50, 10)
}

func BenchmarkRerankByCombinedScore200Candidates(b *testing.B) {
	benchmarkRerankByCombinedScore(b, 200, 20)
}

func benchmarkRerankByCombinedScore(b *testing.B, candidateCount int, topK int) {
	rows := buildRerankBenchmarkRows(candidateCount)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rerankByCombinedScore(
			rerankRowsInput{
				rows:  rows,
				query: "durable checkpoint runbook recovery",
				topK:  topK,
			},
		)
	}
}

func BenchmarkBuildEngramQueryWhereComplexTemporalEngagementTrace(b *testing.B) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-00000000a001")
	relationType := models.EngramLinkRelationSupports
	traceDepth := 1
	projectID := "engram-vault"
	createdAfter := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	createdBefore := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	accessCountMin := 3
	freshnessScoreMin := 0.55
	lastAccessedAfter := time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC)
	lastAccessedBefore := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	freshnessComputedAfter := time.Date(2026, 2, 6, 0, 0, 0, 0, time.UTC)
	freshnessComputedBefore := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	request := models.EngramQueryRequest{
		Query:                   "durable checkpoint memory drift",
		TopK:                    12,
		ProjectID:               &projectID,
		Tags:                    []string{"memory", "continuity", "production"},
		Keywords:                []string{"checkpoint", "runbook", "drift"},
		CreatedAfter:            &createdAfter,
		CreatedBefore:           &createdBefore,
		AccessCountMin:          &accessCountMin,
		FreshnessScoreMin:       &freshnessScoreMin,
		LastAccessedAfter:       &lastAccessedAfter,
		LastAccessedBefore:      &lastAccessedBefore,
		FreshnessComputedAfter:  &freshnessComputedAfter,
		FreshnessComputedBefore: &freshnessComputedBefore,
		RelationType:            &relationType,
		TraceDepth:              &traceDepth,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = buildEngramQueryWhere(request, &actorUserID, "[0.1,0.2,0.3]")
	}
}

func BenchmarkBuildEngramQueryWhereComplexFilterMatrix(b *testing.B) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-00000000a002")
	requests := benchmarkComplexQueryRequests()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		request := requests[i%len(requests)]
		_, _ = buildEngramQueryWhere(request, &actorUserID, "[0.1,0.2,0.3]")
	}
}

func BenchmarkFilterByRankScoreBandsFullMatrix50Candidates(b *testing.B) {
	benchmarkFilterByRankScoreBandsFullMatrix(b, 50)
}

func BenchmarkFilterByRankScoreBandsFullMatrix200Candidates(b *testing.B) {
	benchmarkFilterByRankScoreBandsFullMatrix(b, 200)
}

func benchmarkFilterByRankScoreBandsFullMatrix(b *testing.B, candidateCount int) {
	rows := rerankByCombinedScore(
		rerankRowsInput{
			rows:  buildRerankBenchmarkRows(candidateCount),
			query: "durable checkpoint runbook recovery",
			topK:  candidateCount,
		},
	)
	denseMin := 0.3
	denseMax := 0.95
	lexicalMin := 0.2
	lexicalMax := 1.0
	feedbackMin := 0.2
	feedbackMax := 0.9
	engagementMin := 0.1
	engagementMax := 0.9
	freshnessMin := 0.2
	freshnessMax := 1.0
	authorityMin := 0.2
	authorityMax := 0.9
	compositeMin := 0.3
	compositeMax := 0.95

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filtered := filterByRankScoreBands(
			rows,
			&denseMin,
			&denseMax,
			&lexicalMin,
			&lexicalMax,
			&feedbackMin,
			&feedbackMax,
			&engagementMin,
			&engagementMax,
			&freshnessMin,
			&freshnessMax,
			&authorityMin,
			&authorityMax,
			&compositeMin,
			&compositeMax,
		)
		benchmarkScoreBandFilterRowCount = len(filtered)
	}
}

func buildRerankBenchmarkRows(candidateCount int) []map[string]any {
	rows := make([]map[string]any, 0, candidateCount)
	for index := 0; index < candidateCount; index++ {
		rows = append(
			rows,
			map[string]any{
				"engram_id":           benchmarkUUID(index),
				"project_id":          "engram-vault",
				"title":               fmt.Sprintf("Candidate %03d", index),
				"abstract":            "Durable checkpoint and incident recovery context.",
				"created_at":          time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC).Add(-time.Duration(index) * time.Minute),
				"tags":                []string{"durable", "memory"},
				"keywords":            []string{"checkpoint", "runbook"},
				"retrieval_text":      fmt.Sprintf("checkpoint runbook context %03d", index),
				"useful_count":        index % 11,
				"contradiction_count": index % 4,
				"access_count":        index % 35,
				"freshness_score":     float64((index%10)+1) / 10.0,
				"distance":            float64((index%20)+1) / 30.0,
			},
		)
	}
	return rows
}

func benchmarkComplexQueryRequests() []models.EngramQueryRequest {
	relationSupports := models.EngramLinkRelationSupports
	relationContradicts := models.EngramLinkRelationContradicts
	traceDepth := 1
	projectID := "engram-vault"
	createdAfter := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	createdBefore := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	lastAccessedAfter := time.Date(2026, 2, 7, 0, 0, 0, 0, time.UTC)
	lastAccessedBefore := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	freshnessComputedAfter := time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC)
	freshnessComputedBefore := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	highAccessCountMin := 10
	lowAccessCountMin := 2
	freshnessStrictMin := 0.75
	freshnessRelaxedMin := 0.25

	return []models.EngramQueryRequest{
		{
			Query:                   "checkpoint durability and runbook updates",
			TopK:                    10,
			ProjectID:               &projectID,
			Tags:                    []string{"memory", "durability"},
			Keywords:                []string{"checkpoint", "runbook"},
			CreatedAfter:            &createdAfter,
			CreatedBefore:           &createdBefore,
			AccessCountMin:          &highAccessCountMin,
			FreshnessScoreMin:       &freshnessStrictMin,
			LastAccessedAfter:       &lastAccessedAfter,
			LastAccessedBefore:      &lastAccessedBefore,
			FreshnessComputedAfter:  &freshnessComputedAfter,
			FreshnessComputedBefore: &freshnessComputedBefore,
			RelationType:            &relationSupports,
			TraceDepth:              &traceDepth,
		},
		{
			Query:                   "incident contradiction timeline",
			TopK:                    8,
			ProjectID:               &projectID,
			Tags:                    []string{"incident", "memory"},
			Keywords:                []string{"contradiction", "timeline"},
			CreatedAfter:            &createdAfter,
			CreatedBefore:           &createdBefore,
			AccessCountMin:          &lowAccessCountMin,
			FreshnessScoreMin:       &freshnessRelaxedMin,
			LastAccessedAfter:       &lastAccessedAfter,
			LastAccessedBefore:      &lastAccessedBefore,
			FreshnessComputedAfter:  &freshnessComputedAfter,
			FreshnessComputedBefore: &freshnessComputedBefore,
			RelationType:            &relationContradicts,
			TraceDepth:              &traceDepth,
		},
		{
			Query:                   "support links with recent access",
			TopK:                    6,
			Tags:                    []string{"trace"},
			Keywords:                []string{"support", "recent"},
			CreatedAfter:            &createdAfter,
			CreatedBefore:           &createdBefore,
			AccessCountMin:          &lowAccessCountMin,
			FreshnessScoreMin:       &freshnessStrictMin,
			LastAccessedAfter:       &lastAccessedAfter,
			LastAccessedBefore:      &lastAccessedBefore,
			FreshnessComputedAfter:  &freshnessComputedAfter,
			FreshnessComputedBefore: &freshnessComputedBefore,
			RelationType:            &relationSupports,
		},
	}
}

func benchmarkUUID(index int) uuid.UUID {
	value := fmt.Sprintf("00000000-0000-0000-0000-%012d", index+1)
	return uuid.MustParse(value)
}

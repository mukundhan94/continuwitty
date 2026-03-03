package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

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

func benchmarkUUID(index int) uuid.UUID {
	value := fmt.Sprintf("00000000-0000-0000-0000-%012d", index+1)
	return uuid.MustParse(value)
}

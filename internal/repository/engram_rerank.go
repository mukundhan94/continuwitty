package repository

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
)

var tokenPattern = regexp.MustCompile(`[a-z0-9]{2,}`)

type lexicalOverlapInput struct {
	query          string
	candidateParts []string
}

type rankScoreInput struct {
	distance        float64
	lexicalOverlap  float64
	feedbackScore   float64
	engagementScore float64
	freshnessScore  float64
}

type rerankRowsInput struct {
	rows  []map[string]any
	query string
	topK  int
}

const (
	denseRankWeight      = 0.55
	lexicalRankWeight    = 0.20
	feedbackRankWeight   = 0.10
	engagementRankWeight = 0.10
	freshnessRankWeight  = 0.05
)

func tokenize(text string) map[string]struct{} {
	matches := tokenPattern.FindAllString(strings.ToLower(text), -1)
	tokens := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		tokens[match] = struct{}{}
	}
	return tokens
}

func lexicalOverlapScore(input lexicalOverlapInput) float64 {
	queryTokens := tokenize(input.query)
	if len(queryTokens) == 0 {
		return 0
	}
	documentTokens := collectTokens(input.candidateParts)
	if len(documentTokens) == 0 {
		return 0
	}
	return float64(overlapCount(queryTokens, documentTokens)) / float64(len(queryTokens))
}

func collectTokens(parts []string) map[string]struct{} {
	tokens := make(map[string]struct{})
	for _, part := range parts {
		for token := range tokenize(part) {
			tokens[token] = struct{}{}
		}
	}
	return tokens
}

func overlapCount(source map[string]struct{}, target map[string]struct{}) int {
	overlap := 0
	for token := range source {
		if _, exists := target[token]; exists {
			overlap++
		}
	}
	return overlap
}

func combinedRankScore(input rankScoreInput) float64 {
	denseScore := 1.0 / (1.0 + math.Max(input.distance, 0))
	return (denseScore * denseRankWeight) +
		(input.lexicalOverlap * lexicalRankWeight) +
		(input.feedbackScore * feedbackRankWeight) +
		(input.engagementScore * engagementRankWeight) +
		(input.freshnessScore * freshnessRankWeight)
}

func normalizeFeedbackScore(usefulCount int, contradictionCount int) float64 {
	useful := max(usefulCount, 0)
	contradiction := max(contradictionCount, 0)
	total := useful + contradiction
	if total == 0 {
		return 0.5
	}
	raw := float64(useful-contradiction) / float64(total+2)
	return clamp01((raw + 1.0) / 2.0)
}

func normalizeEngagementScore(accessCount int) float64 {
	if accessCount <= 0 {
		return 0
	}
	const fullSignalAccessCount = 20.0
	return clamp01(math.Log1p(float64(accessCount)) / math.Log1p(fullSignalAccessCount))
}

func normalizeFreshnessScore(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0.5
	}
	return clamp01(value)
}

func candidateFreshnessScore(row map[string]any) float64 {
	raw, found := row["freshness_score"]
	if !found {
		return 1.0
	}
	return normalizeFreshnessScore(float64FromAny(raw))
}

func clamp01(value float64) float64 {
	return min(max(value, 0), 1)
}

func rerankByCombinedScore(input rerankRowsInput) []map[string]any {
	type rankedRow struct {
		score     float64
		createdAt time.Time
		row       map[string]any
	}

	ranked := make([]rankedRow, 0, len(input.rows))
	for _, row := range input.rows {
		lexicalScore := lexicalOverlapScore(
			lexicalOverlapInput{
				query: input.query,
				candidateParts: []string{
					stringFromAny(row["title"]),
					stringFromAny(row["abstract"]),
					stringFromAny(row["retrieval_text"]),
					strings.Join(stringSliceFromAny(row["tags"]), " "),
					strings.Join(stringSliceFromAny(row["keywords"]), " "),
				},
			},
		)
		ranked = append(
			ranked,
			rankedRow{
				score: combinedRankScore(
					rankScoreInput{
						distance:       float64FromAny(row["distance"]),
						lexicalOverlap: lexicalScore,
						feedbackScore: normalizeFeedbackScore(
							intFromAny(row["useful_count"]),
							intFromAny(row["contradiction_count"]),
						),
						engagementScore: normalizeEngagementScore(
							intFromAny(row["access_count"]),
						),
						freshnessScore: candidateFreshnessScore(row),
					},
				),
				createdAt: timeFromAny(row["created_at"]),
				row:       row,
			},
		)
	}

	sort.Slice(ranked, func(left, right int) bool {
		if ranked[left].score == ranked[right].score {
			return ranked[left].createdAt.After(ranked[right].createdAt)
		}
		return ranked[left].score > ranked[right].score
	})

	if input.topK <= 0 || input.topK > len(ranked) {
		input.topK = len(ranked)
	}
	trimmed := make([]map[string]any, 0, input.topK)
	for _, row := range ranked[:input.topK] {
		trimmed = append(trimmed, row.row)
	}
	return trimmed
}

package repository

import (
	"math"
	"strings"
	"testing"
	"time"

	"engram/internal/models"
)

func TestVectorLiteralFormat(t *testing.T) {
	literal := vectorLiteral([]float64{0.1, -0.2, 0})
	if !strings.HasPrefix(literal, "[") {
		t.Fatalf("expected literal to start with '['")
	}
	if !strings.HasSuffix(literal, "]") {
		t.Fatalf("expected literal to end with ']'")
	}
	if !strings.Contains(literal, ",") {
		t.Fatalf("expected literal to contain comma separated values")
	}
}

func TestBuildRetrievalTextUsesOverride(t *testing.T) {
	retrieval := "override retrieval"
	payload := models.MemoryEngramCreate{
		ProjectID:               "p1",
		Title:                   "T",
		Abstract:                "A",
		DetailedSummaryMarkdown: "D",
		RetrievalText:           &retrieval,
	}
	if got := buildRetrievalText(payload); got != retrieval {
		t.Fatalf("expected retrieval override, got %q", got)
	}
}

func TestBuildRetrievalTextFallbackComposesFields(t *testing.T) {
	payload := models.MemoryEngramCreate{
		ProjectID:               "p1",
		Title:                   "LangGraph Choice",
		Abstract:                "Selected for durable checkpoints",
		DetailedSummaryMarkdown: "details",
		OpenQuestions:           []string{"When to rerank?"},
	}
	value := buildRetrievalText(payload)
	if !strings.Contains(value, "LangGraph Choice") {
		t.Fatalf("expected title in retrieval text, got %q", value)
	}
	if !strings.Contains(value, "Selected for durable checkpoints") {
		t.Fatalf("expected abstract in retrieval text, got %q", value)
	}
	if !strings.Contains(value, "When to rerank?") {
		t.Fatalf("expected question in retrieval text, got %q", value)
	}
}

func TestLexicalOverlapScorePrefersMatchingTerms(t *testing.T) {
	score := lexicalOverlapScore(
		lexicalOverlapInput{
			query:          "durable checkpoint workflow",
			candidateParts: []string{"LangGraph enables durable checkpoint workflow execution"},
		},
	)
	if score <= 0.6 {
		t.Fatalf("expected overlap score > 0.6, got %v", score)
	}
}

func TestCombinedRankScoreUsesDenseAndLexicalSignals(t *testing.T) {
	baseline := combinedRankScore(rankScoreInput{distance: 0.8, lexicalOverlap: 0.0})
	lexicalBoosted := combinedRankScore(rankScoreInput{distance: 0.8, lexicalOverlap: 1.0})
	denseBoosted := combinedRankScore(rankScoreInput{distance: 0.1, lexicalOverlap: 0.0})
	if lexicalBoosted <= baseline {
		t.Fatalf("expected lexical boost (%v) > baseline (%v)", lexicalBoosted, baseline)
	}
	if denseBoosted <= baseline {
		t.Fatalf("expected dense boost (%v) > baseline (%v)", denseBoosted, baseline)
	}
}

func TestNormalizeEngagementScoreUsesLogScaling(t *testing.T) {
	none := normalizeEngagementScore(0)
	moderate := normalizeEngagementScore(5)
	high := normalizeEngagementScore(25)
	if none != 0 {
		t.Fatalf("expected 0-access score to be 0, got %v", none)
	}
	if moderate <= none {
		t.Fatalf("expected moderate engagement > none, got %v <= %v", moderate, none)
	}
	if high < 0.95 {
		t.Fatalf("expected high engagement to saturate near 1, got %v", high)
	}
}

func TestNormalizeFreshnessScoreClampsAndHandlesInvalidValues(t *testing.T) {
	if score := normalizeFreshnessScore(-1.2); score != 0 {
		t.Fatalf("expected negative freshness to clamp to 0, got %v", score)
	}
	if score := normalizeFreshnessScore(1.3); score != 1 {
		t.Fatalf("expected high freshness to clamp to 1, got %v", score)
	}
	if score := normalizeFreshnessScore(math.NaN()); score != 0.5 {
		t.Fatalf("expected NaN freshness fallback to 0.5, got %v", score)
	}
}

func TestPackCitationsDeduplicatesURLs(t *testing.T) {
	titleA1 := "A1"
	titleA2 := "A2"
	titleB := "B"
	citations := []models.RehydrationCitation{
		{
			URL:        "https://example.com/a",
			Title:      &titleA1,
			Snippet:    "first",
			CapturedAt: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			URL:        "https://example.com/a",
			Title:      &titleA2,
			Snippet:    "duplicate",
			CapturedAt: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			URL:        "https://example.com/b",
			Title:      &titleB,
			Snippet:    "second",
			CapturedAt: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
		},
	}
	packed := packCitations(
		citationPackInput{
			citations: citations,
			limit:     5,
		},
	)
	if len(packed) != 2 {
		t.Fatalf("expected 2 packed citations, got %d", len(packed))
	}
	if packed[0].URL != "https://example.com/a" {
		t.Fatalf("expected first citation url A, got %q", packed[0].URL)
	}
	if packed[1].URL != "https://example.com/b" {
		t.Fatalf("expected second citation url B, got %q", packed[1].URL)
	}
}

func TestResolveCompactSummaryUsesDetailedForGenericChatSnapshot(t *testing.T) {
	resolved := resolveCompactSummary(
		compactSummaryInput{
			abstract:                "Snapshot from active chat session.",
			detailedSummaryMarkdown: "# Chat Session Snapshot\n\n## ASSISTANT (ts)\nPrimary cause was DB CPU saturation; rollback did not help.",
			maxChars:                800,
		},
	)
	if !strings.Contains(resolved, "Primary cause was DB CPU saturation") {
		t.Fatalf("expected compact summary to use detailed excerpt, got %q", resolved)
	}
}

func TestExtractDetailedExcerptPrefersAssistantSection(t *testing.T) {
	excerpt := extractDetailedExcerpt(
		detailedExcerptInput{
			markdown: "# Chat Session Snapshot\n\n## USER (ts)\nWhat happened?\n\n## ASSISTANT (ts)\nDatabase saturation triggered payment latency.\n\n## USER (ts)\nThanks.",
			maxChars: 200,
		},
	)
	if !strings.HasPrefix(excerpt, "Database saturation triggered payment latency.") {
		t.Fatalf("expected assistant excerpt, got %q", excerpt)
	}
}

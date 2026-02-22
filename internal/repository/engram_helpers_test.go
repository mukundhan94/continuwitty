package repository

import (
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
		"durable checkpoint workflow",
		[]string{"LangGraph enables durable checkpoint workflow execution"},
	)
	if score <= 0.6 {
		t.Fatalf("expected overlap score > 0.6, got %v", score)
	}
}

func TestCombinedRankScoreUsesDenseAndLexicalSignals(t *testing.T) {
	weakDenseStrongLexical := combinedRankScore(0.8, 1.0)
	strongDenseWeakLexical := combinedRankScore(0.1, 0.0)
	if weakDenseStrongLexical <= 0 {
		t.Fatalf("expected weakDenseStrongLexical > 0, got %v", weakDenseStrongLexical)
	}
	if strongDenseWeakLexical <= weakDenseStrongLexical {
		t.Fatalf(
			"expected strongDenseWeakLexical (%v) > weakDenseStrongLexical (%v)",
			strongDenseWeakLexical,
			weakDenseStrongLexical,
		)
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
	packed := packCitations(citations, 5)
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
		"Snapshot from active chat session.",
		"# Chat Session Snapshot\n\n## ASSISTANT (ts)\nPrimary cause was DB CPU saturation; rollback did not help.",
		800,
	)
	if !strings.Contains(resolved, "Primary cause was DB CPU saturation") {
		t.Fatalf("expected compact summary to use detailed excerpt, got %q", resolved)
	}
}

func TestExtractDetailedExcerptPrefersAssistantSection(t *testing.T) {
	excerpt := extractDetailedExcerpt(
		"# Chat Session Snapshot\n\n## USER (ts)\nWhat happened?\n\n## ASSISTANT (ts)\nDatabase saturation triggered payment latency.\n\n## USER (ts)\nThanks.",
		200,
	)
	if !strings.HasPrefix(excerpt, "Database saturation triggered payment latency.") {
		t.Fatalf("expected assistant excerpt, got %q", excerpt)
	}
}

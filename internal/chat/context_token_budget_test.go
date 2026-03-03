package chat

import (
	"strings"
	"testing"
)

func TestNormalizeContextTokenBudgetOptionDefaultsAndBounds(t *testing.T) {
	defaulted := normalizeContextTokenBudgetOption(ChatContextRequest{})
	if defaulted.ContextTokenBudget == nil || *defaulted.ContextTokenBudget != defaultChatContextTokenBudget {
		t.Fatalf("expected default context token budget %d", defaultChatContextTokenBudget)
	}

	lowBudget := 1
	normalizedLow := normalizeContextTokenBudgetOption(
		ChatContextRequest{ContextTokenBudget: &lowBudget},
	)
	if normalizedLow.ContextTokenBudget == nil || *normalizedLow.ContextTokenBudget != minChatContextTokenBudget {
		t.Fatalf("expected bounded low token budget %d", minChatContextTokenBudget)
	}

	highBudget := 90000
	normalizedHigh := normalizeContextTokenBudgetOption(
		ChatContextRequest{ContextTokenBudget: &highBudget},
	)
	if normalizedHigh.ContextTokenBudget == nil || *normalizedHigh.ContextTokenBudget != maxChatContextTokenBudget {
		t.Fatalf("expected bounded high token budget %d", maxChatContextTokenBudget)
	}
}

func TestBuildBudgetedContextMarkdownWithoutTruncation(t *testing.T) {
	sections := []string{
		"# Engram Retrieval Context",
		"## Item 1\nSummary",
		"## Item 2\nSummary",
	}
	budgeted := buildBudgetedContextMarkdown(sections, 800)
	if budgeted.truncated {
		t.Fatalf("expected context to fit budget")
	}
	if budgeted.markdown != strings.Join(sections, "\n\n") {
		t.Fatalf("expected markdown to keep all sections")
	}
	if budgeted.tokenEstimate <= 0 {
		t.Fatalf("expected positive token estimate")
	}
}

func TestBuildBudgetedContextMarkdownWithTruncation(t *testing.T) {
	sections := []string{
		strings.Repeat("A", 160),
		strings.Repeat("B", 160),
	}
	budgeted := buildBudgetedContextMarkdown(sections, 20)
	if !budgeted.truncated {
		t.Fatalf("expected context truncation when budget is small")
	}
	if len(budgeted.markdown) > maxCharsForTokenBudget(20) {
		t.Fatalf("expected markdown to respect character budget")
	}
	if !strings.HasSuffix(budgeted.markdown, "...") {
		t.Fatalf("expected truncated markdown to end with ellipsis")
	}
	if budgeted.tokenEstimate <= 0 {
		t.Fatalf("expected positive token estimate")
	}
}

func TestJoinContextSectionsWithinCharBudgetStopsAtSectionBoundary(t *testing.T) {
	markdown, truncated := joinContextSectionsWithinCharBudget([]string{"12345", "67890", "abc"}, 12)
	if !truncated {
		t.Fatalf("expected truncation when final section exceeds budget")
	}
	if markdown != "12345\n\n67890" {
		t.Fatalf("unexpected markdown result %q", markdown)
	}
}

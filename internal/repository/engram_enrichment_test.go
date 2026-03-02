package repository

import (
	"strings"
	"testing"

	"engram/internal/models"
)

func TestEnrichEngramPayloadIfMissingDerivesMissingMetadata(t *testing.T) {
	payload := models.MemoryEngramCreate{
		ProjectID: "engram-vault",
		Title:     "Acceptance auto metadata seed",
		DetailedSummaryMarkdown: "## USER\nSummarize incident impact.\n\n## ASSISTANT\n" +
			"Outage blast radius dropped after rollback and queue drain while support prepared customer updates.",
		Abstract:        "",
		Tags:            []string{},
		Keywords:        []string{},
		VisibilityScope: "project",
	}

	resolved, report := enrichEngramPayloadIfMissing(payload, "mcp.engram.create_from_conversation")
	assertDerivedMetadata(t, resolved, report)
}

func TestEnrichEngramPayloadIfMissingPreservesExplicitMetadata(t *testing.T) {
	payload := models.MemoryEngramCreate{
		ProjectID:               "engram-vault",
		Title:                   "Explicit metadata",
		DetailedSummaryMarkdown: "Conversation body",
		Abstract:                "Manual abstract",
		Tags:                    []string{"manual-tag"},
		Keywords:                []string{"manual-keyword"},
		VisibilityScope:         "project",
	}

	resolved, report := enrichEngramPayloadIfMissing(payload, "mcp.engram.create_from_conversation")
	assertPreservedMetadata(t, resolved, report)
}

func assertDerivedMetadata(
	t *testing.T,
	resolved models.MemoryEngramCreate,
	report map[string]any,
) {
	t.Helper()
	if strings.TrimSpace(resolved.Abstract) == "" {
		t.Fatalf("expected derived abstract")
	}
	if len(resolved.Tags) == 0 {
		t.Fatalf("expected derived tags")
	}
	if len(resolved.Keywords) == 0 {
		t.Fatalf("expected derived keywords")
	}
	assertEnrichmentFlags(t, report, true)
	assertReportListType(t, report, "auto_tags")
	assertReportListType(t, report, "auto_keywords")
}

func assertPreservedMetadata(
	t *testing.T,
	resolved models.MemoryEngramCreate,
	report map[string]any,
) {
	t.Helper()
	if resolved.Abstract != "Manual abstract" {
		t.Fatalf("expected abstract to remain unchanged")
	}
	assertSingleValueSlice(t, resolved.Tags, "manual-tag", "tags")
	assertSingleValueSlice(t, resolved.Keywords, "manual-keyword", "keywords")
	assertEnrichmentFlags(t, report, false)
	assertEmptyReportList(t, report, "auto_tags")
	assertEmptyReportList(t, report, "auto_keywords")
}

func assertEnrichmentFlags(t *testing.T, report map[string]any, expected bool) {
	t.Helper()
	assertReportBoolean(t, report, "enrichment_applied", expected)
	assertReportBoolean(t, report, "abstract_derived", expected)
	assertReportBoolean(t, report, "tags_derived", expected)
	assertReportBoolean(t, report, "keywords_derived", expected)
}

func assertReportBoolean(t *testing.T, report map[string]any, key string, expected bool) {
	t.Helper()
	if report[key] != expected {
		t.Fatalf("expected %s=%v", key, expected)
	}
}

func assertSingleValueSlice(t *testing.T, values []string, expected string, field string) {
	t.Helper()
	if len(values) != 1 || values[0] != expected {
		t.Fatalf("expected %s to remain unchanged", field)
	}
}

func assertReportListType(t *testing.T, report map[string]any, key string) {
	t.Helper()
	if _, ok := report[key].([]string); !ok {
		t.Fatalf("expected %s []string report payload", key)
	}
}

func assertEmptyReportList(t *testing.T, report map[string]any, key string) {
	t.Helper()
	values, ok := report[key].([]string)
	if !ok || len(values) != 0 {
		t.Fatalf("expected empty %s report payload", key)
	}
}

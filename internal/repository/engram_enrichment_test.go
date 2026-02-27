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

	if strings.TrimSpace(resolved.Abstract) == "" {
		t.Fatalf("expected derived abstract")
	}
	if len(resolved.Tags) == 0 {
		t.Fatalf("expected derived tags")
	}
	if len(resolved.Keywords) == 0 {
		t.Fatalf("expected derived keywords")
	}
	if report["enrichment_applied"] != true {
		t.Fatalf("expected enrichment_applied=true")
	}
	if report["abstract_derived"] != true {
		t.Fatalf("expected abstract_derived=true")
	}
	if report["tags_derived"] != true {
		t.Fatalf("expected tags_derived=true")
	}
	if report["keywords_derived"] != true {
		t.Fatalf("expected keywords_derived=true")
	}
	if _, ok := report["auto_tags"].([]string); !ok {
		t.Fatalf("expected auto_tags []string report payload")
	}
	if _, ok := report["auto_keywords"].([]string); !ok {
		t.Fatalf("expected auto_keywords []string report payload")
	}
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

	if resolved.Abstract != "Manual abstract" {
		t.Fatalf("expected abstract to remain unchanged")
	}
	if len(resolved.Tags) != 1 || resolved.Tags[0] != "manual-tag" {
		t.Fatalf("expected tags to remain unchanged")
	}
	if len(resolved.Keywords) != 1 || resolved.Keywords[0] != "manual-keyword" {
		t.Fatalf("expected keywords to remain unchanged")
	}
	if report["enrichment_applied"] != false {
		t.Fatalf("expected enrichment_applied=false")
	}
	if report["abstract_derived"] != false {
		t.Fatalf("expected abstract_derived=false")
	}
	if report["tags_derived"] != false {
		t.Fatalf("expected tags_derived=false")
	}
	if report["keywords_derived"] != false {
		t.Fatalf("expected keywords_derived=false")
	}
	if autoTags, ok := report["auto_tags"].([]string); !ok || len(autoTags) != 0 {
		t.Fatalf("expected empty auto_tags report payload")
	}
	if autoKeywords, ok := report["auto_keywords"].([]string); !ok || len(autoKeywords) != 0 {
		t.Fatalf("expected empty auto_keywords report payload")
	}
}

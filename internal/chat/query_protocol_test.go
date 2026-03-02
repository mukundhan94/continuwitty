package chat

import "testing"

func TestParseCWQueryProtocolWithoutPrefixReturnsOriginalContent(t *testing.T) {
	content := "Summarize the latest retention updates."
	normalized, plan := ParseCWQueryProtocol(content)
	if normalized != content {
		t.Fatalf("expected content unchanged, got %q", normalized)
	}
	if plan != nil {
		t.Fatalf("expected no cw plan for non-prefixed query")
	}
}

func TestParseCWQueryProtocolBarePrefixDefaultsToAuto(t *testing.T) {
	normalized, plan := ParseCWQueryProtocol("cw>\nFind related linked engrams.")
	if normalized != "Find related linked engrams." {
		t.Fatalf("unexpected normalized content %q", normalized)
	}
	if plan == nil {
		t.Fatalf("expected cw plan")
	}
	if plan.Mode != "auto" {
		t.Fatalf("expected default mode auto, got %q", plan.Mode)
	}
	if plan.Project != "" {
		t.Fatalf("expected empty project, got %q", plan.Project)
	}
	if plan.CitationsRequired {
		t.Fatalf("expected citations_required false by default")
	}
}

func TestParseCWQueryProtocolIntentFormKeepsInlineBody(t *testing.T) {
	normalized, plan := ParseCWQueryProtocol("CW> retrieve summarize decisions\nwith citations")
	expected := "summarize decisions\nwith citations"
	if normalized != expected {
		t.Fatalf("expected normalized content %q, got %q", expected, normalized)
	}
	if plan == nil || plan.Mode != "retrieve" {
		t.Fatalf("expected retrieve mode plan, got %#v", plan)
	}
}

func TestParseCWQueryProtocolKeyValueForm(t *testing.T) {
	normalized, plan := ParseCWQueryProtocol("cw> mode=analyze project=engram-vault citations=required\nTrace linked incidents.")
	if normalized != "Trace linked incidents." {
		t.Fatalf("unexpected normalized content %q", normalized)
	}
	if plan == nil {
		t.Fatalf("expected cw plan")
	}
	if plan.Mode != "analyze" {
		t.Fatalf("expected mode analyze, got %q", plan.Mode)
	}
	if plan.Project != "engram-vault" {
		t.Fatalf("expected project engram-vault, got %q", plan.Project)
	}
	if !plan.CitationsRequired {
		t.Fatalf("expected citations_required=true")
	}
}

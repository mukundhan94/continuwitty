package evalops

import (
	"testing"
	"time"
)

func TestRunDefaultSuitePasses(t *testing.T) {
	definition := DefaultSuiteDefinition()
	summary := RunSuite(definition, time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), "run-1")
	if !summary.Passed {
		t.Fatalf("expected suite to pass")
	}
	if summary.TotalCases != 8 {
		t.Fatalf("expected 8 cases, got %d", summary.TotalCases)
	}
	if summary.Score != 1 {
		t.Fatalf("expected perfect score, got %.4f", summary.Score)
	}
	if len(summary.Dimensions) != 4 {
		t.Fatalf("expected 4 dimensions, got %d", len(summary.Dimensions))
	}
}

func TestRunSuiteFailsWhenRequiredTermMissing(t *testing.T) {
	definition := DefaultSuiteDefinition()
	definition.Cases[0].OutputText = "LangGraph remains selected."
	summary := RunSuite(definition, time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), "run-2")
	if summary.Passed {
		t.Fatalf("expected suite to fail when required term is missing")
	}
	if summary.Score >= 1 {
		t.Fatalf("expected reduced score, got %.4f", summary.Score)
	}
	if len(summary.Cases) == 0 {
		t.Fatalf("expected case results")
	}
	first := summary.Cases[0]
	if first.Passed {
		t.Fatalf("expected first case to fail")
	}
	if len(first.Checks) == 0 {
		t.Fatalf("expected check details on failure")
	}
}

package evalops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendHistoryTrimsToMaxEntries(t *testing.T) {
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "history.jsonl")

	runs := []SuiteRun{
		{RunID: "run-1", SuiteVersion: "eval-suite-v1", GeneratedAt: time.Now().UTC(), Score: 1},
		{RunID: "run-2", SuiteVersion: "eval-suite-v1", GeneratedAt: time.Now().UTC(), Score: 1},
		{RunID: "run-3", SuiteVersion: "eval-suite-v1", GeneratedAt: time.Now().UTC(), Score: 1},
	}
	for _, run := range runs {
		_, err := AppendHistory(historyPath, run, 2)
		if err != nil {
			t.Fatalf("append history: %v", err)
		}
	}

	history, err := LoadHistory(historyPath)
	if err != nil {
		t.Fatalf("load history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected trimmed history length 2, got %d", len(history))
	}
	if history[0].RunID != "run-2" || history[1].RunID != "run-3" {
		t.Fatalf("unexpected history order: %#v", history)
	}
}

func TestWriteTrendReportIncludesGateStatus(t *testing.T) {
	tempDir := t.TempDir()
	reportPath := filepath.Join(tempDir, "trend-report.md")
	history := []SuiteRun{
		{
			RunID:        "run-1",
			SuiteVersion: "eval-suite-v1",
			GeneratedAt:  time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
			Score:        1,
			Passed:       true,
			PassedCases:  6,
			TotalCases:   6,
			Dimensions: []DimensionSummary{
				{Dimension: DimensionContinuity, Passed: 2, Total: 2, Score: 1},
				{Dimension: DimensionCitationTrust, Passed: 2, Total: 2, Score: 1},
				{Dimension: DimensionMemoryDrift, Passed: 2, Total: 2, Score: 1},
			},
		},
	}
	gate := DeltaGateResult{
		Passed: false,
		Violations: []GateViolation{
			{Message: "overall score delta vs baseline -0.1000 is below threshold -0.0300"},
		},
	}
	if err := WriteTrendReport(TrendReportWriteInput{
		Path:        reportPath,
		History:     history,
		Baseline:    &history[0],
		Gate:        gate,
		GeneratedAt: time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("write report: %v", err)
	}
	body, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "Delta gate: `FAILED`") {
		t.Fatalf("expected failed gate status in report, got %q", text)
	}
	if !strings.Contains(text, "overall score delta vs baseline") {
		t.Fatalf("expected violation in report, got %q", text)
	}
}

package evalops

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// BuildTrendReport builds a markdown summary for recent runs and regression deltas.
func BuildTrendReport(
	history []SuiteRun,
	baseline *SuiteRun,
	gate DeltaGateResult,
	generatedAt time.Time,
) string {
	latest := LastRun(history)
	var previous *SuiteRun
	if len(history) >= 2 {
		copyRun := history[len(history)-2]
		previous = &copyRun
	}

	lines := make([]string, 0, 64)
	lines = append(lines, "# EvalOps Trend Report")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Generated at: %s", generatedAt.UTC().Format(time.RFC3339)))
	lines = append(lines, "")

	if latest == nil {
		lines = append(lines, "No run history available.")
		return strings.Join(lines, "\n") + "\n"
	}

	lines = append(lines, "## Latest Run")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("- Run ID: `%s`", latest.RunID))
	lines = append(lines, fmt.Sprintf("- Suite version: `%s`", latest.SuiteVersion))
	lines = append(lines, fmt.Sprintf("- Score: `%.4f`", latest.Score))
	lines = append(lines, fmt.Sprintf("- Passed: `%t` (%d/%d cases)", latest.Passed, latest.PassedCases, latest.TotalCases))
	lines = append(lines, fmt.Sprintf("- Delta gate: `%s`", gateStatusLabel(gate.Passed)))
	lines = append(lines, fmt.Sprintf("- Overall delta vs baseline: `%s`", formatDelta(gate.OverallDeltaBaseline)))
	lines = append(lines, fmt.Sprintf("- Overall delta vs previous: `%s`", formatDelta(gate.OverallDeltaPrevious)))
	lines = append(lines, "")

	lines = append(lines, "## Recent Runs")
	lines = append(lines, "")
	lines = append(lines, "| Run ID | Generated At | Score | Passed Cases | Pass |")
	lines = append(lines, "|---|---|---:|---:|---|")
	for _, run := range recentRuns(history, 10) {
		lines = append(
			lines,
			fmt.Sprintf(
				"| `%s` | %s | %.4f | %d/%d | %t |",
				run.RunID,
				run.GeneratedAt.UTC().Format(time.RFC3339),
				run.Score,
				run.PassedCases,
				run.TotalCases,
				run.Passed,
			),
		)
	}
	lines = append(lines, "")

	lines = append(lines, "## Dimension Deltas")
	lines = append(lines, "")
	lines = append(lines, "| Dimension | Latest | Baseline | Delta vs baseline | Previous | Delta vs previous |")
	lines = append(lines, "|---|---:|---:|---:|---:|---:|")
	for _, summary := range latest.Dimensions {
		baselineScore := formatScorePointer(findDimensionScore(baseline, summary.Dimension))
		previousScore := formatScorePointer(findDimensionScore(previous, summary.Dimension))
		baselineDelta := formatDimensionDelta(gate.DimensionDeltaBaseline, summary.Dimension)
		previousDelta := formatDimensionDelta(gate.DimensionDeltaPrevious, summary.Dimension)
		lines = append(
			lines,
			fmt.Sprintf(
				"| `%s` | %.4f | %s | %s | %s | %s |",
				summary.Dimension,
				summary.Score,
				baselineScore,
				baselineDelta,
				previousScore,
				previousDelta,
			),
		)
	}
	lines = append(lines, "")

	if baseline != nil {
		lines = append(lines, "## Baseline")
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("- Baseline run id: `%s`", baseline.RunID))
		lines = append(lines, fmt.Sprintf("- Baseline score: `%.4f`", baseline.Score))
		lines = append(lines, "")
	}

	lines = append(lines, "## Gate Violations")
	lines = append(lines, "")
	if len(gate.Violations) == 0 {
		lines = append(lines, "- None")
		return strings.Join(lines, "\n") + "\n"
	}
	for _, violation := range gate.Violations {
		lines = append(lines, fmt.Sprintf("- %s", violation.Message))
	}
	return strings.Join(lines, "\n") + "\n"
}

// WriteTrendReport writes markdown report output.
func WriteTrendReport(path string, history []SuiteRun, baseline *SuiteRun, gate DeltaGateResult, generatedAt time.Time) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil
	}
	if err := ensureParentDir(trimmed); err != nil {
		return err
	}
	report := BuildTrendReport(history, baseline, gate, generatedAt)
	return osWriteFile(trimmed, report)
}

func gateStatusLabel(passed bool) string {
	if passed {
		return "PASSED"
	}
	return "FAILED"
}

func formatDelta(value *float64) string {
	if value == nil {
		return "n/a"
	}
	return fmt.Sprintf("%.4f", *value)
}

func formatScorePointer(value *float64) string {
	if value == nil {
		return "n/a"
	}
	return fmt.Sprintf("%.4f", *value)
}

func formatDimensionDelta(values map[Dimension]float64, dimension Dimension) string {
	if len(values) == 0 {
		return "n/a"
	}
	delta, exists := values[dimension]
	if !exists {
		return "n/a"
	}
	return fmt.Sprintf("%.4f", delta)
}

func findDimensionScore(run *SuiteRun, dimension Dimension) *float64 {
	if run == nil {
		return nil
	}
	for _, summary := range run.Dimensions {
		if summary.Dimension != dimension {
			continue
		}
		return floatPtr(summary.Score)
	}
	return nil
}

func recentRuns(history []SuiteRun, maxCount int) []SuiteRun {
	if maxCount <= 0 || len(history) <= maxCount {
		return history
	}
	return history[len(history)-maxCount:]
}

func osWriteFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

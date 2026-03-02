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
	latest, previous := latestAndPreviousRun(history)
	lines := trendReportHeader(generatedAt)
	if latest == nil {
		lines = append(lines, "No run history available.")
		return markdownFromLines(lines)
	}

	appendLatestRunSection(&lines, *latest, gate)
	appendRecentRunsSection(&lines, history)
	appendDimensionDeltasSection(&lines, *latest, baseline, previous, gate)
	appendBaselineSection(&lines, baseline)
	appendGateViolationsSection(&lines, gate)
	return markdownFromLines(lines)
}

func latestAndPreviousRun(history []SuiteRun) (*SuiteRun, *SuiteRun) {
	latest := LastRun(history)
	if len(history) < 2 {
		return latest, nil
	}
	copyRun := history[len(history)-2]
	return latest, &copyRun
}

func trendReportHeader(generatedAt time.Time) []string {
	lines := make([]string, 0, 64)
	lines = append(lines, "# EvalOps Trend Report")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Generated at: %s", generatedAt.UTC().Format(time.RFC3339)))
	lines = append(lines, "")
	return lines
}

func appendLatestRunSection(lines *[]string, latest SuiteRun, gate DeltaGateResult) {
	*lines = append(*lines, "## Latest Run")
	*lines = append(*lines, "")
	*lines = append(*lines, fmt.Sprintf("- Run ID: `%s`", latest.RunID))
	*lines = append(*lines, fmt.Sprintf("- Suite version: `%s`", latest.SuiteVersion))
	*lines = append(*lines, fmt.Sprintf("- Score: `%.4f`", latest.Score))
	*lines = append(*lines, fmt.Sprintf("- Passed: `%t` (%d/%d cases)", latest.Passed, latest.PassedCases, latest.TotalCases))
	*lines = append(*lines, fmt.Sprintf("- Delta gate: `%s`", gateStatusLabel(gate.Passed)))
	*lines = append(*lines, fmt.Sprintf("- Overall delta vs baseline: `%s`", formatDelta(gate.OverallDeltaBaseline)))
	*lines = append(*lines, fmt.Sprintf("- Overall delta vs previous: `%s`", formatDelta(gate.OverallDeltaPrevious)))
	*lines = append(*lines, "")
}

func appendRecentRunsSection(lines *[]string, history []SuiteRun) {
	*lines = append(*lines, "## Recent Runs")
	*lines = append(*lines, "")
	*lines = append(*lines, "| Run ID | Generated At | Score | Passed Cases | Pass |")
	*lines = append(*lines, "|---|---|---:|---:|---|")
	for _, run := range recentRuns(history, 10) {
		*lines = append(
			*lines,
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
	*lines = append(*lines, "")
}

func appendDimensionDeltasSection(
	lines *[]string,
	latest SuiteRun,
	baseline *SuiteRun,
	previous *SuiteRun,
	gate DeltaGateResult,
) {
	*lines = append(*lines, "## Dimension Deltas")
	*lines = append(*lines, "")
	*lines = append(*lines, "| Dimension | Latest | Baseline | Delta vs baseline | Previous | Delta vs previous |")
	*lines = append(*lines, "|---|---:|---:|---:|---:|---:|")
	for _, summary := range latest.Dimensions {
		*lines = append(
			*lines,
			dimensionDeltaRow(summary, baseline, previous, gate),
		)
	}
	*lines = append(*lines, "")
}

func dimensionDeltaRow(
	summary DimensionSummary,
	baseline *SuiteRun,
	previous *SuiteRun,
	gate DeltaGateResult,
) string {
	baselineScore := formatScorePointer(findDimensionScore(baseline, summary.Dimension))
	previousScore := formatScorePointer(findDimensionScore(previous, summary.Dimension))
	baselineDelta := formatDimensionDelta(gate.DimensionDeltaBaseline, summary.Dimension)
	previousDelta := formatDimensionDelta(gate.DimensionDeltaPrevious, summary.Dimension)
	return fmt.Sprintf(
		"| `%s` | %.4f | %s | %s | %s | %s |",
		summary.Dimension,
		summary.Score,
		baselineScore,
		baselineDelta,
		previousScore,
		previousDelta,
	)
}

func appendBaselineSection(lines *[]string, baseline *SuiteRun) {
	if baseline == nil {
		return
	}
	*lines = append(*lines, "## Baseline")
	*lines = append(*lines, "")
	*lines = append(*lines, fmt.Sprintf("- Baseline run id: `%s`", baseline.RunID))
	*lines = append(*lines, fmt.Sprintf("- Baseline score: `%.4f`", baseline.Score))
	*lines = append(*lines, "")
}

func appendGateViolationsSection(lines *[]string, gate DeltaGateResult) {
	*lines = append(*lines, "## Gate Violations")
	*lines = append(*lines, "")
	if len(gate.Violations) == 0 {
		*lines = append(*lines, "- None")
		return
	}
	for _, violation := range gate.Violations {
		*lines = append(*lines, fmt.Sprintf("- %s", violation.Message))
	}
}

func markdownFromLines(lines []string) string {
	return strings.Join(lines, "\n") + "\n"
}

// WriteTrendReport writes markdown report output.
type TrendReportWriteInput struct {
	Path        string
	History     []SuiteRun
	Baseline    *SuiteRun
	Gate        DeltaGateResult
	GeneratedAt time.Time
}

func WriteTrendReport(input TrendReportWriteInput) error {
	trimmed := strings.TrimSpace(input.Path)
	if trimmed == "" {
		return nil
	}
	if err := ensureParentDir(trimmed); err != nil {
		return err
	}
	report := BuildTrendReport(input.History, input.Baseline, input.Gate, input.GeneratedAt)
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

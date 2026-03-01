package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"engram/internal/evalops"
)

const (
	defaultLatestOutPath   = "data/evals/latest.json"
	defaultHistoryPath     = "data/evals/history.jsonl"
	defaultBaselinePath    = "evals/baselines/eval-suite-v1.json"
	defaultTrendReportPath = "data/evals/trend-report.md"
)

func main() {
	os.Exit(run())
}

func run() int {
	options := parseFlags()
	startedAt := time.Now().UTC()
	runID := startedAt.Format("20060102T150405Z")

	historyBefore, err := evalops.LoadHistory(options.historyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load eval history: %v\n", err)
		return 1
	}
	previous := evalops.LastRun(historyBefore)

	suite := evalops.DefaultSuiteDefinition()
	runSummary := evalops.RunSuite(suite, startedAt, runID)

	if err := evalops.WriteSuiteRun(options.outPath, runSummary); err != nil {
		fmt.Fprintf(os.Stderr, "write latest eval run: %v\n", err)
		return 1
	}

	history, err := evalops.AppendHistory(options.historyPath, runSummary, options.maxHistoryEntries)
	if err != nil {
		fmt.Fprintf(os.Stderr, "append eval history: %v\n", err)
		return 1
	}

	baseline, err := evalops.ReadSuiteRun(options.baselinePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read eval baseline: %v\n", err)
		return 1
	}

	gate := evalops.DeltaGateResult{Passed: true}
	if options.enforceDeltaGate {
		if baseline == nil {
			fmt.Fprintf(os.Stderr, "delta gate requires baseline file: %s\n", options.baselinePath)
			return 1
		}
		gate = evalops.EvaluateDeltaGate(
			runSummary,
			baseline,
			previous,
			evalops.DeltaThresholds{
				MinOverallDelta:   options.minOverallDelta,
				MinDimensionDelta: options.minDimensionDelta,
			},
		)
	}

	if err := evalops.WriteTrendReport(options.trendReportPath, history, baseline, gate, startedAt); err != nil {
		fmt.Fprintf(os.Stderr, "write eval trend report: %v\n", err)
		return 1
	}

	printSummary(runSummary, gate, baseline != nil, options.enforceDeltaGate)
	if !runSummary.Passed {
		return 1
	}
	if options.enforceDeltaGate && !gate.Passed {
		return 1
	}
	return 0
}

type cliOptions struct {
	outPath           string
	historyPath       string
	baselinePath      string
	trendReportPath   string
	maxHistoryEntries int
	minOverallDelta   float64
	minDimensionDelta float64
	enforceDeltaGate  bool
}

func parseFlags() cliOptions {
	options := cliOptions{}
	flag.StringVar(&options.outPath, "out", defaultLatestOutPath, "path to write latest eval run JSON")
	flag.StringVar(&options.historyPath, "history", defaultHistoryPath, "path to eval history JSONL")
	flag.StringVar(&options.baselinePath, "baseline", defaultBaselinePath, "path to eval baseline JSON")
	flag.StringVar(&options.trendReportPath, "trend-report", defaultTrendReportPath, "path to write markdown trend report")
	flag.IntVar(&options.maxHistoryEntries, "max-history", 60, "max number of history entries to keep")
	flag.Float64Var(&options.minOverallDelta, "min-overall-delta", -0.03, "minimum allowed overall score delta")
	flag.Float64Var(&options.minDimensionDelta, "min-dimension-delta", -0.05, "minimum allowed per-dimension score delta")
	flag.BoolVar(&options.enforceDeltaGate, "enforce-delta-gate", true, "enforce baseline/previous score delta thresholds")
	flag.Parse()
	return options
}

func printSummary(
	runSummary evalops.SuiteRun,
	gate evalops.DeltaGateResult,
	hasBaseline bool,
	deltaGateEnabled bool,
) {
	fmt.Printf("EvalOps suite: %s\n", runSummary.SuiteVersion)
	fmt.Printf("Run ID: %s\n", runSummary.RunID)
	fmt.Printf("Score: %.4f (%d/%d cases)\n", runSummary.Score, runSummary.PassedCases, runSummary.TotalCases)
	fmt.Printf("Suite pass: %t\n", runSummary.Passed)
	if !deltaGateEnabled {
		fmt.Println("Delta gate: skipped")
		return
	}
	if !hasBaseline {
		fmt.Println("Delta gate: baseline missing")
		return
	}
	fmt.Printf("Delta gate pass: %t\n", gate.Passed)
	if gate.OverallDeltaBaseline != nil {
		fmt.Printf("Overall delta vs baseline: %.4f\n", *gate.OverallDeltaBaseline)
	}
	if gate.OverallDeltaPrevious != nil {
		fmt.Printf("Overall delta vs previous: %.4f\n", *gate.OverallDeltaPrevious)
	}
	if len(gate.Violations) == 0 {
		return
	}
	fmt.Println("Gate violations:")
	for _, violation := range gate.Violations {
		fmt.Printf("- %s\n", violation.Message)
	}
}

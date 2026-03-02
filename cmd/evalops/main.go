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

type evalRunResult struct {
	runSummary        evalops.SuiteRun
	gate              evalops.DeltaGateResult
	hasBaseline       bool
	deltaGateEnforced bool
}

type trendReportInput struct {
	options   cliOptions
	history   []evalops.SuiteRun
	baseline  *evalops.SuiteRun
	gate      evalops.DeltaGateResult
	startedAt time.Time
}

func run() int {
	options := parseFlags()
	result, err := executeEvalRun(options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 1
	}
	printSummary(result.runSummary, result.gate, result.hasBaseline, result.deltaGateEnforced)
	if shouldFailEvalRun(result) {
		return 1
	}
	return 0
}

func executeEvalRun(options cliOptions) (evalRunResult, error) {
	startedAt := time.Now().UTC()
	runSummary, previous, err := runEvalSuite(options, startedAt)
	if err != nil {
		return evalRunResult{}, err
	}
	history, err := appendEvalHistory(options, runSummary)
	if err != nil {
		return evalRunResult{}, err
	}
	baseline, err := readEvalBaseline(options)
	if err != nil {
		return evalRunResult{}, err
	}
	gate, err := evaluateDeltaGate(options, runSummary, baseline, previous)
	if err != nil {
		return evalRunResult{}, err
	}
	if err := writeEvalTrendReport(
		trendReportInput{
			options:   options,
			history:   history,
			baseline:  baseline,
			gate:      gate,
			startedAt: startedAt,
		},
	); err != nil {
		return evalRunResult{}, err
	}
	return evalRunResult{
		runSummary:        runSummary,
		gate:              gate,
		hasBaseline:       baseline != nil,
		deltaGateEnforced: options.enforceDeltaGate,
	}, nil
}

func runEvalSuite(
	options cliOptions,
	startedAt time.Time,
) (evalops.SuiteRun, *evalops.SuiteRun, error) {
	historyBefore, err := evalops.LoadHistory(options.historyPath)
	if err != nil {
		return evalops.SuiteRun{}, nil, fmt.Errorf("load eval history: %w", err)
	}
	runID := startedAt.Format("20060102T150405Z")
	suite := evalops.DefaultSuiteDefinition()
	runSummary := evalops.RunSuite(suite, startedAt, runID)
	if err := evalops.WriteSuiteRun(options.outPath, runSummary); err != nil {
		return evalops.SuiteRun{}, nil, fmt.Errorf("write latest eval run: %w", err)
	}
	return runSummary, evalops.LastRun(historyBefore), nil
}

func appendEvalHistory(options cliOptions, runSummary evalops.SuiteRun) ([]evalops.SuiteRun, error) {
	history, err := evalops.AppendHistory(options.historyPath, runSummary, options.maxHistoryEntries)
	if err != nil {
		return nil, fmt.Errorf("append eval history: %w", err)
	}
	return history, nil
}

func readEvalBaseline(options cliOptions) (*evalops.SuiteRun, error) {
	baseline, err := evalops.ReadSuiteRun(options.baselinePath)
	if err != nil {
		return nil, fmt.Errorf("read eval baseline: %w", err)
	}
	return baseline, nil
}

func evaluateDeltaGate(
	options cliOptions,
	runSummary evalops.SuiteRun,
	baseline *evalops.SuiteRun,
	previous *evalops.SuiteRun,
) (evalops.DeltaGateResult, error) {
	if !options.enforceDeltaGate {
		return evalops.DeltaGateResult{Passed: true}, nil
	}
	if baseline == nil {
		return evalops.DeltaGateResult{}, fmt.Errorf(
			"delta gate requires baseline file: %s",
			options.baselinePath,
		)
	}
	return evalops.EvaluateDeltaGate(
		runSummary,
		baseline,
		previous,
		evalops.DeltaThresholds{
			MinOverallDelta:   options.minOverallDelta,
			MinDimensionDelta: options.minDimensionDelta,
		},
	), nil
}

func writeEvalTrendReport(
	input trendReportInput,
) error {
	if err := evalops.WriteTrendReport(
		input.options.trendReportPath,
		input.history,
		input.baseline,
		input.gate,
		input.startedAt,
	); err != nil {
		return fmt.Errorf("write eval trend report: %w", err)
	}
	return nil
}

func shouldFailEvalRun(result evalRunResult) bool {
	if !result.runSummary.Passed {
		return true
	}
	return result.deltaGateEnforced && !result.gate.Passed
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

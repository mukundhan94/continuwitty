package evalops

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type dimensionAccumulator struct {
	passed int
	total  int
	score  float64
}

// RunSuite executes the deterministic suite and returns a scored summary.
func RunSuite(definition SuiteDefinition, generatedAt time.Time, runID string) SuiteRun {
	results := make([]EvalCaseResult, 0, len(definition.Cases))
	dimensions := make(map[Dimension]*dimensionAccumulator, len(defaultDimensionOrder))
	passedCases := 0
	totalScore := 0.0

	for _, evalCase := range definition.Cases {
		result := evaluateCase(evalCase)
		results = append(results, result)
		totalScore += result.Score
		if result.Passed {
			passedCases++
		}
		accumulator := ensureDimensionAccumulator(dimensions, evalCase.Dimension)
		accumulator.total++
		accumulator.score += result.Score
		if result.Passed {
			accumulator.passed++
		}
	}

	totalCases := len(results)
	averageScore := averageOrZero(totalScore, totalCases)
	return SuiteRun{
		SuiteVersion: strings.TrimSpace(definition.Version),
		RunID:        strings.TrimSpace(runID),
		GeneratedAt:  generatedAt.UTC(),
		Passed:       passedCases == totalCases,
		Score:        roundScore(averageScore),
		PassedCases:  passedCases,
		TotalCases:   totalCases,
		Cases:        results,
		Dimensions:   buildDimensionSummaries(dimensions),
	}
}

func evaluateCase(evalCase EvalCase) EvalCaseResult {
	checks := make([]CheckResult, 0, 5)
	checks = appendCheckIfDefined(checks, evaluateRequiredTerms(evalCase))
	checks = appendCheckIfDefined(checks, evaluateForbiddenTerms(evalCase))
	checks = appendCheckIfDefined(checks, evaluateUsedEngramCount(evalCase))
	checks = appendCheckIfDefined(checks, evaluateRequiredSourceURLs(evalCase))
	checks = appendCheckIfDefined(checks, evaluateAnchorRecall(evalCase))
	if len(checks) == 0 {
		return EvalCaseResult{
			ID:        evalCase.ID,
			Dimension: evalCase.Dimension,
			Passed:    true,
			Score:     1,
			Checks:    []CheckResult{},
		}
	}
	return buildCaseResult(evalCase, checks)
}

func appendCheckIfDefined(checks []CheckResult, check *CheckResult) []CheckResult {
	if check == nil {
		return checks
	}
	return append(checks, *check)
}

func buildCaseResult(evalCase EvalCase, checks []CheckResult) EvalCaseResult {
	passed := true
	totalScore := 0.0
	for _, check := range checks {
		totalScore += check.Score
		if !check.Passed {
			passed = false
		}
	}
	return EvalCaseResult{
		ID:        evalCase.ID,
		Dimension: evalCase.Dimension,
		Passed:    passed,
		Score:     roundScore(averageOrZero(totalScore, len(checks))),
		Checks:    checks,
	}
}

func evaluateRequiredTerms(evalCase EvalCase) *CheckResult {
	if len(evalCase.RequiredTerms) == 0 {
		return nil
	}
	normalizedOutput := normalizeText(evalCase.OutputText)
	missing := make([]string, 0)
	matched := 0
	for _, term := range evalCase.RequiredTerms {
		if containsNormalized(normalizedOutput, term) {
			matched++
			continue
		}
		missing = append(missing, term)
	}
	return &CheckResult{
		Name:    "required_terms",
		Passed:  len(missing) == 0,
		Score:   roundScore(averageOrZero(float64(matched), len(evalCase.RequiredTerms))),
		Details: joinedListDetail("missing", missing),
	}
}

func evaluateForbiddenTerms(evalCase EvalCase) *CheckResult {
	if len(evalCase.ForbiddenTerms) == 0 {
		return nil
	}
	normalizedOutput := normalizeText(evalCase.OutputText)
	violations := make([]string, 0)
	for _, term := range evalCase.ForbiddenTerms {
		if containsNormalized(normalizedOutput, term) {
			violations = append(violations, term)
		}
	}
	score := 1.0
	if len(violations) > 0 {
		score = 0
	}
	return &CheckResult{
		Name:    "forbidden_terms",
		Passed:  len(violations) == 0,
		Score:   score,
		Details: joinedListDetail("present", violations),
	}
}

func evaluateUsedEngramCount(evalCase EvalCase) *CheckResult {
	if evalCase.MinUsedEngramCount <= 0 {
		return nil
	}
	count := len(evalCase.UsedEngramIDs)
	passed := count >= evalCase.MinUsedEngramCount
	score := 0.0
	if passed {
		score = 1.0
	}
	return &CheckResult{
		Name:    "used_engram_count",
		Passed:  passed,
		Score:   score,
		Details: fmt.Sprintf("required=%d actual=%d", evalCase.MinUsedEngramCount, count),
	}
}

func evaluateRequiredSourceURLs(evalCase EvalCase) *CheckResult {
	if len(evalCase.RequiredSourceURLs) == 0 {
		return nil
	}
	available := make(map[string]struct{}, len(evalCase.SourceReferences))
	for _, source := range evalCase.SourceReferences {
		normalized := normalizeURL(source.URL)
		if normalized == "" {
			continue
		}
		available[normalized] = struct{}{}
	}
	missing := make([]string, 0)
	matched := 0
	for _, url := range evalCase.RequiredSourceURLs {
		normalized := normalizeURL(url)
		if normalized == "" {
			continue
		}
		if _, exists := available[normalized]; exists {
			matched++
			continue
		}
		missing = append(missing, url)
	}
	return &CheckResult{
		Name:    "required_source_urls",
		Passed:  len(missing) == 0,
		Score:   roundScore(averageOrZero(float64(matched), len(evalCase.RequiredSourceURLs))),
		Details: joinedListDetail("missing", missing),
	}
}

func evaluateAnchorRecall(evalCase EvalCase) *CheckResult {
	if len(evalCase.BaselineAnchorTerms) == 0 {
		return nil
	}
	threshold := evalCase.MinAnchorRecall
	if threshold <= 0 {
		threshold = 1
	}
	normalizedOutput := normalizeText(evalCase.OutputText)
	missing := make([]string, 0)
	matched := 0
	for _, anchor := range evalCase.BaselineAnchorTerms {
		if containsNormalized(normalizedOutput, anchor) {
			matched++
			continue
		}
		missing = append(missing, anchor)
	}
	recall := roundScore(averageOrZero(float64(matched), len(evalCase.BaselineAnchorTerms)))
	return &CheckResult{
		Name:    "anchor_recall",
		Passed:  recall >= threshold,
		Score:   recall,
		Details: fmt.Sprintf("threshold=%.2f %s", threshold, joinedListDetail("missing", missing)),
	}
}

func buildDimensionSummaries(accumulators map[Dimension]*dimensionAccumulator) []DimensionSummary {
	ordered := make([]DimensionSummary, 0, len(accumulators))
	for _, dimension := range defaultDimensionOrder {
		accumulator, exists := accumulators[dimension]
		if !exists {
			continue
		}
		ordered = append(
			ordered,
			DimensionSummary{
				Dimension: dimension,
				Passed:    accumulator.passed,
				Total:     accumulator.total,
				Score:     roundScore(averageOrZero(accumulator.score, accumulator.total)),
			},
		)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return dimensionIndex(ordered[i].Dimension) < dimensionIndex(ordered[j].Dimension)
	})
	return ordered
}

func ensureDimensionAccumulator(
	accumulators map[Dimension]*dimensionAccumulator,
	dimension Dimension,
) *dimensionAccumulator {
	accumulator, exists := accumulators[dimension]
	if !exists {
		accumulator = &dimensionAccumulator{}
		accumulators[dimension] = accumulator
	}
	return accumulator
}

func dimensionIndex(dimension Dimension) int {
	for index, ordered := range defaultDimensionOrder {
		if ordered == dimension {
			return index
		}
	}
	return len(defaultDimensionOrder)
}

func joinedListDetail(prefix string, values []string) string {
	if len(values) == 0 {
		return ""
	}
	return fmt.Sprintf("%s=%s", prefix, strings.Join(values, ", "))
}

func containsNormalized(normalizedOutput, rawQuery string) bool {
	normalizedQuery := normalizeText(rawQuery)
	if normalizedQuery == "" {
		return false
	}
	return strings.Contains(normalizedOutput, normalizedQuery)
}

func normalizeURL(rawURL string) string {
	return strings.ToLower(strings.TrimSpace(rawURL))
}

func normalizeText(value string) string {
	lower := strings.ToLower(value)
	replacer := strings.NewReplacer(
		"\n", " ",
		"\t", " ",
		",", " ",
		".", " ",
		";", " ",
		":", " ",
		"!", " ",
		"?", " ",
		"[", " ",
		"]", " ",
		"(", " ",
		")", " ",
		"-", " ",
	)
	normalized := replacer.Replace(lower)
	return strings.Join(strings.Fields(normalized), " ")
}

func averageOrZero(total float64, count int) float64 {
	if count <= 0 {
		return 0
	}
	return total / float64(count)
}

func roundScore(value float64) float64 {
	return math.Round(value*10000) / 10000
}

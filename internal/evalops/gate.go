package evalops

import "fmt"

// EvaluateDeltaGate applies baseline/previous delta thresholds to the current run.
func EvaluateDeltaGate(
	current SuiteRun,
	baseline *SuiteRun,
	previous *SuiteRun,
	thresholds DeltaThresholds,
) DeltaGateResult {
	result := DeltaGateResult{
		Passed:                 true,
		DimensionDeltaBaseline: map[Dimension]float64{},
		DimensionDeltaPrevious: map[Dimension]float64{},
		Violations:             []GateViolation{},
	}
	if baseline != nil {
		delta := roundScore(current.Score - baseline.Score)
		result.OverallDeltaBaseline = floatPtr(delta)
		appendOverallViolationIfNeeded(&result, "baseline", delta, thresholds.MinOverallDelta)
		collectDimensionDelta(dimensionDeltaInput{
			currentRun:     current,
			referenceRun:   *baseline,
			threshold:      thresholds.MinDimensionDelta,
			referenceLabel: "baseline",
			store:          result.DimensionDeltaBaseline,
			result:         &result,
		})
	}
	if previous != nil {
		delta := roundScore(current.Score - previous.Score)
		result.OverallDeltaPrevious = floatPtr(delta)
		appendOverallViolationIfNeeded(&result, "previous", delta, thresholds.MinOverallDelta)
		collectDimensionDelta(dimensionDeltaInput{
			currentRun:     current,
			referenceRun:   *previous,
			threshold:      thresholds.MinDimensionDelta,
			referenceLabel: "previous",
			store:          result.DimensionDeltaPrevious,
			result:         &result,
		})
	}
	return result
}

func appendOverallViolationIfNeeded(
	result *DeltaGateResult,
	reference string,
	delta float64,
	threshold float64,
) {
	if delta >= threshold {
		return
	}
	result.Passed = false
	result.Violations = append(
		result.Violations,
		GateViolation{
			Scope:     "overall",
			Reference: reference,
			Delta:     delta,
			Threshold: threshold,
			Message: fmt.Sprintf(
				"overall score delta vs %s %.4f is below threshold %.4f",
				reference,
				delta,
				threshold,
			),
		},
	)
}

type dimensionDeltaInput struct {
	currentRun     SuiteRun
	referenceRun   SuiteRun
	threshold      float64
	referenceLabel string
	store          map[Dimension]float64
	result         *DeltaGateResult
}

func collectDimensionDelta(input dimensionDeltaInput) {
	for _, currentDimension := range input.currentRun.Dimensions {
		referenceScore, exists := dimensionScore(input.referenceRun, currentDimension.Dimension)
		if !exists {
			continue
		}
		delta := roundScore(currentDimension.Score - referenceScore)
		input.store[currentDimension.Dimension] = delta
		if delta >= input.threshold {
			continue
		}
		input.result.Passed = false
		input.result.Violations = append(
			input.result.Violations,
			GateViolation{
				Scope:     "dimension",
				Dimension: string(currentDimension.Dimension),
				Reference: input.referenceLabel,
				Delta:     delta,
				Threshold: input.threshold,
				Message: fmt.Sprintf(
					"dimension %s delta vs %s %.4f is below threshold %.4f",
					currentDimension.Dimension,
					input.referenceLabel,
					delta,
					input.threshold,
				),
			},
		)
	}
}

func dimensionScore(run SuiteRun, target Dimension) (float64, bool) {
	for _, summary := range run.Dimensions {
		if summary.Dimension != target {
			continue
		}
		return summary.Score, true
	}
	return 0, false
}

func floatPtr(value float64) *float64 {
	copyValue := value
	return &copyValue
}

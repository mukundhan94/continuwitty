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
		collectDimensionDelta(
			current,
			*baseline,
			thresholds.MinDimensionDelta,
			"baseline",
			result.DimensionDeltaBaseline,
			&result,
		)
	}
	if previous != nil {
		delta := roundScore(current.Score - previous.Score)
		result.OverallDeltaPrevious = floatPtr(delta)
		appendOverallViolationIfNeeded(&result, "previous", delta, thresholds.MinOverallDelta)
		collectDimensionDelta(
			current,
			*previous,
			thresholds.MinDimensionDelta,
			"previous",
			result.DimensionDeltaPrevious,
			&result,
		)
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

func collectDimensionDelta(
	current SuiteRun,
	referenceRun SuiteRun,
	threshold float64,
	referenceLabel string,
	store map[Dimension]float64,
	result *DeltaGateResult,
) {
	for _, currentDimension := range current.Dimensions {
		referenceScore, exists := dimensionScore(referenceRun, currentDimension.Dimension)
		if !exists {
			continue
		}
		delta := roundScore(currentDimension.Score - referenceScore)
		store[currentDimension.Dimension] = delta
		if delta >= threshold {
			continue
		}
		result.Passed = false
		result.Violations = append(
			result.Violations,
			GateViolation{
				Scope:     "dimension",
				Dimension: string(currentDimension.Dimension),
				Reference: referenceLabel,
				Delta:     delta,
				Threshold: threshold,
				Message: fmt.Sprintf(
					"dimension %s delta vs %s %.4f is below threshold %.4f",
					currentDimension.Dimension,
					referenceLabel,
					delta,
					threshold,
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

package evalops

import "testing"

func TestEvaluateDeltaGatePassesWithinThresholds(t *testing.T) {
	baseline := testRun(0.90, map[Dimension]float64{
		DimensionContinuity:    0.90,
		DimensionCitationTrust: 0.90,
		DimensionMemoryDrift:   0.90,
	})
	previous := testRun(0.91, map[Dimension]float64{
		DimensionContinuity:    0.91,
		DimensionCitationTrust: 0.91,
		DimensionMemoryDrift:   0.91,
	})
	current := testRun(0.89, map[Dimension]float64{
		DimensionContinuity:    0.89,
		DimensionCitationTrust: 0.89,
		DimensionMemoryDrift:   0.89,
	})

	result := EvaluateDeltaGate(
		current,
		&baseline,
		&previous,
		DeltaThresholds{MinOverallDelta: -0.03, MinDimensionDelta: -0.05},
	)
	if !result.Passed {
		t.Fatalf("expected gate to pass: %#v", result.Violations)
	}
}

func TestEvaluateDeltaGateFailsForDimensionRegression(t *testing.T) {
	baseline := testRun(1.0, map[Dimension]float64{
		DimensionContinuity:    1.0,
		DimensionCitationTrust: 1.0,
		DimensionMemoryDrift:   1.0,
	})
	current := testRun(0.94, map[Dimension]float64{
		DimensionContinuity:    0.98,
		DimensionCitationTrust: 0.97,
		DimensionMemoryDrift:   0.80,
	})

	result := EvaluateDeltaGate(
		current,
		&baseline,
		nil,
		DeltaThresholds{MinOverallDelta: -0.10, MinDimensionDelta: -0.05},
	)
	if result.Passed {
		t.Fatalf("expected dimension gate to fail")
	}
	if len(result.Violations) == 0 {
		t.Fatalf("expected at least one violation")
	}
	foundMemoryDrift := false
	for _, violation := range result.Violations {
		if violation.Dimension == string(DimensionMemoryDrift) {
			foundMemoryDrift = true
		}
	}
	if !foundMemoryDrift {
		t.Fatalf("expected memory drift violation, got %#v", result.Violations)
	}
}

func testRun(score float64, dimensionScores map[Dimension]float64) SuiteRun {
	dimensions := make([]DimensionSummary, 0, len(dimensionScores))
	for _, dimension := range defaultDimensionOrder {
		dimScore, exists := dimensionScores[dimension]
		if !exists {
			continue
		}
		dimensions = append(dimensions, DimensionSummary{Dimension: dimension, Passed: 1, Total: 1, Score: dimScore})
	}
	return SuiteRun{Score: score, Dimensions: dimensions}
}

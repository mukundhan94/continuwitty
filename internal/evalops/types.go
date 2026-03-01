package evalops

import "time"

// Dimension identifies an EvalOps quality dimension.
type Dimension string

const (
	DimensionContinuity    Dimension = "continuity"
	DimensionCitationTrust Dimension = "citation_trust"
	DimensionMemoryDrift   Dimension = "memory_drift"
)

var defaultDimensionOrder = []Dimension{
	DimensionContinuity,
	DimensionCitationTrust,
	DimensionMemoryDrift,
}

// DefaultDimensionOrder returns the stable ordering used in reports.
func DefaultDimensionOrder() []Dimension {
	ordered := make([]Dimension, len(defaultDimensionOrder))
	copy(ordered, defaultDimensionOrder)
	return ordered
}

// SourceReference captures evidence references used by an evaluated response.
type SourceReference struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

// EvalCase captures the response sample and quality rules for one deterministic evaluation case.
type EvalCase struct {
	ID                  string            `json:"id"`
	Dimension           Dimension         `json:"dimension"`
	Prompt              string            `json:"prompt"`
	OutputText          string            `json:"output_text"`
	UsedEngramIDs       []string          `json:"used_engram_ids,omitempty"`
	SourceReferences    []SourceReference `json:"source_references,omitempty"`
	RequiredTerms       []string          `json:"required_terms,omitempty"`
	ForbiddenTerms      []string          `json:"forbidden_terms,omitempty"`
	RequiredSourceURLs  []string          `json:"required_source_urls,omitempty"`
	MinUsedEngramCount  int               `json:"min_used_engram_count,omitempty"`
	BaselineAnchorTerms []string          `json:"baseline_anchor_terms,omitempty"`
	MinAnchorRecall     float64           `json:"min_anchor_recall,omitempty"`
}

// SuiteDefinition captures versioned case fixtures for one run.
type SuiteDefinition struct {
	Version string     `json:"version"`
	Cases   []EvalCase `json:"cases"`
}

// CheckResult captures an individual check evaluation.
type CheckResult struct {
	Name    string  `json:"name"`
	Passed  bool    `json:"passed"`
	Score   float64 `json:"score"`
	Details string  `json:"details,omitempty"`
}

// EvalCaseResult captures the scored outcome of a single case.
type EvalCaseResult struct {
	ID        string        `json:"id"`
	Dimension Dimension     `json:"dimension"`
	Passed    bool          `json:"passed"`
	Score     float64       `json:"score"`
	Checks    []CheckResult `json:"checks"`
}

// DimensionSummary captures score and pass ratios for one dimension.
type DimensionSummary struct {
	Dimension Dimension `json:"dimension"`
	Passed    int       `json:"passed"`
	Total     int       `json:"total"`
	Score     float64   `json:"score"`
}

// SuiteRun captures one eval-suite execution result.
type SuiteRun struct {
	SuiteVersion string             `json:"suite_version"`
	RunID        string             `json:"run_id"`
	GeneratedAt  time.Time          `json:"generated_at"`
	Passed       bool               `json:"passed"`
	Score        float64            `json:"score"`
	PassedCases  int                `json:"passed_cases"`
	TotalCases   int                `json:"total_cases"`
	Cases        []EvalCaseResult   `json:"cases"`
	Dimensions   []DimensionSummary `json:"dimensions"`
}

// DeltaThresholds controls allowed score regressions for release gates.
type DeltaThresholds struct {
	MinOverallDelta   float64 `json:"min_overall_delta"`
	MinDimensionDelta float64 `json:"min_dimension_delta"`
}

// GateViolation captures one threshold violation.
type GateViolation struct {
	Scope     string  `json:"scope"`
	Dimension string  `json:"dimension,omitempty"`
	Reference string  `json:"reference"`
	Delta     float64 `json:"delta"`
	Threshold float64 `json:"threshold"`
	Message   string  `json:"message"`
}

// DeltaGateResult captures regression deltas and gate decision.
type DeltaGateResult struct {
	Passed                 bool                  `json:"passed"`
	OverallDeltaBaseline   *float64              `json:"overall_delta_baseline,omitempty"`
	OverallDeltaPrevious   *float64              `json:"overall_delta_previous,omitempty"`
	DimensionDeltaBaseline map[Dimension]float64 `json:"dimension_delta_baseline,omitempty"`
	DimensionDeltaPrevious map[Dimension]float64 `json:"dimension_delta_previous,omitempty"`
	Violations             []GateViolation       `json:"violations,omitempty"`
}

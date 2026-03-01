package models

import "github.com/google/uuid"

// EngramLinkHygieneCategory identifies graph-quality recommendation classes.
type EngramLinkHygieneCategory string

const (
	EngramLinkHygieneCategoryDuplicateTarget  EngramLinkHygieneCategory = "duplicate_target"
	EngramLinkHygieneCategoryConflictRelation EngramLinkHygieneCategory = "conflict_relation"
	EngramLinkHygieneCategoryStaleLowValue    EngramLinkHygieneCategory = "stale_low_value"
)

// EngramLinkHygieneRecommendation captures one graph hygiene recommendation.
type EngramLinkHygieneRecommendation struct {
	Category        EngramLinkHygieneCategory `json:"category"`
	Severity        string                    `json:"severity"`
	SourceEngramID  uuid.UUID                 `json:"source_engram_id"`
	TargetEngramID  uuid.UUID                 `json:"target_engram_id"`
	LinkIDs         []uuid.UUID               `json:"link_ids"`
	Detail          string                    `json:"detail"`
	SuggestedAction string                    `json:"suggested_action"`
	Score           float64                   `json:"score"`
}

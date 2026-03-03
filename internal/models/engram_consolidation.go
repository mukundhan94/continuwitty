package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ConsolidationSuggestionType identifies the grouping strategy used for a consolidation suggestion.
type ConsolidationSuggestionType string

const (
	ConsolidationSuggestionTypeExactDuplicate ConsolidationSuggestionType = "exact_duplicate"
	ConsolidationSuggestionTypeThemeDuplicate ConsolidationSuggestionType = "theme_duplicate"
	ConsolidationSuggestionTypeSuperseded     ConsolidationSuggestionType = "superseded"
	ConsolidationSuggestionTypeComplementary  ConsolidationSuggestionType = "complementary"
)

// ConsolidationSuggestionStatus captures lifecycle status for consolidation suggestions.
type ConsolidationSuggestionStatus string

const (
	ConsolidationSuggestionStatusSuggested ConsolidationSuggestionStatus = "suggested"
	ConsolidationSuggestionStatusMerged    ConsolidationSuggestionStatus = "merged"
	ConsolidationSuggestionStatusRejected  ConsolidationSuggestionStatus = "rejected"
)

// ParseConsolidationSuggestionStatus validates and normalizes a suggestion status.
func ParseConsolidationSuggestionStatus(value string) (ConsolidationSuggestionStatus, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch ConsolidationSuggestionStatus(normalized) {
	case ConsolidationSuggestionStatusSuggested:
		return ConsolidationSuggestionStatusSuggested, nil
	case ConsolidationSuggestionStatusMerged:
		return ConsolidationSuggestionStatusMerged, nil
	case ConsolidationSuggestionStatusRejected:
		return ConsolidationSuggestionStatusRejected, nil
	default:
		return "", fmt.Errorf("unsupported consolidation suggestion status %q", value)
	}
}

// EngramConsolidationSuggestion describes a persisted consolidation recommendation.
type EngramConsolidationSuggestion struct {
	SuggestionID      uuid.UUID                     `json:"suggestion_id"`
	ProjectID         string                        `json:"project_id"`
	SourceEngramIDs   []uuid.UUID                   `json:"source_engram_ids"`
	ConsolidationType ConsolidationSuggestionType   `json:"consolidation_type"`
	Reason            string                        `json:"reason"`
	ConsolidationHash string                        `json:"consolidation_hash"`
	ConfidenceScore   float64                       `json:"confidence_score"`
	Status            ConsolidationSuggestionStatus `json:"status"`
	SuggestedAt       time.Time                     `json:"suggested_at"`
	UpdatedAt         time.Time                     `json:"updated_at"`
	ActionedAt        *time.Time                    `json:"actioned_at,omitempty"`
	ActionTakenBy     *uuid.UUID                    `json:"action_taken_by,omitempty"`
}

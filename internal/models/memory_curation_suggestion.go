package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MemoryCurationSuggestionType identifies the suggested memory action category.
type MemoryCurationSuggestionType string

const (
	MemoryCurationSuggestionTypeAutoSave      MemoryCurationSuggestionType = "auto_save"
	MemoryCurationSuggestionTypeConsolidate   MemoryCurationSuggestionType = "consolidate"
	MemoryCurationSuggestionTypeContradiction MemoryCurationSuggestionType = "contradiction"
	MemoryCurationSuggestionTypeLink          MemoryCurationSuggestionType = "link"
)

// ParseMemoryCurationSuggestionType validates and normalizes suggestion type input.
func ParseMemoryCurationSuggestionType(value string) (MemoryCurationSuggestionType, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch MemoryCurationSuggestionType(normalized) {
	case MemoryCurationSuggestionTypeAutoSave:
		return MemoryCurationSuggestionTypeAutoSave, nil
	case MemoryCurationSuggestionTypeConsolidate:
		return MemoryCurationSuggestionTypeConsolidate, nil
	case MemoryCurationSuggestionTypeContradiction:
		return MemoryCurationSuggestionTypeContradiction, nil
	case MemoryCurationSuggestionTypeLink:
		return MemoryCurationSuggestionTypeLink, nil
	default:
		return "", fmt.Errorf("unsupported memory curation suggestion type %q", value)
	}
}

// MemoryCurationSuggestionStatus captures lifecycle status for suggestion actioning.
type MemoryCurationSuggestionStatus string

const (
	MemoryCurationSuggestionStatusSuggested MemoryCurationSuggestionStatus = "suggested"
	MemoryCurationSuggestionStatusAccepted  MemoryCurationSuggestionStatus = "accepted"
	MemoryCurationSuggestionStatusRejected  MemoryCurationSuggestionStatus = "rejected"
	MemoryCurationSuggestionStatusApplied   MemoryCurationSuggestionStatus = "applied"
)

// ParseMemoryCurationSuggestionStatus validates and normalizes suggestion status input.
func ParseMemoryCurationSuggestionStatus(value string) (MemoryCurationSuggestionStatus, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch MemoryCurationSuggestionStatus(normalized) {
	case MemoryCurationSuggestionStatusSuggested:
		return MemoryCurationSuggestionStatusSuggested, nil
	case MemoryCurationSuggestionStatusAccepted:
		return MemoryCurationSuggestionStatusAccepted, nil
	case MemoryCurationSuggestionStatusRejected:
		return MemoryCurationSuggestionStatusRejected, nil
	case MemoryCurationSuggestionStatusApplied:
		return MemoryCurationSuggestionStatusApplied, nil
	default:
		return "", fmt.Errorf("unsupported memory curation suggestion status %q", value)
	}
}

// MemoryCurationSuggestion describes one persisted autonomous curation recommendation.
type MemoryCurationSuggestion struct {
	SuggestionID    uuid.UUID                      `json:"suggestion_id"`
	ProjectID       string                         `json:"project_id"`
	SessionID       *uuid.UUID                     `json:"session_id,omitempty"`
	SuggestionType  MemoryCurationSuggestionType   `json:"suggestion_type"`
	Reason          string                         `json:"reason"`
	Recommendation  string                         `json:"recommendation"`
	PayloadJSON     map[string]any                 `json:"payload_json"`
	ConfidenceScore float64                        `json:"confidence_score"`
	Status          MemoryCurationSuggestionStatus `json:"status"`
	SuggestedAt     time.Time                      `json:"suggested_at"`
	UpdatedAt       time.Time                      `json:"updated_at"`
	ActionedAt      *time.Time                     `json:"actioned_at,omitempty"`
	ActionTakenBy   *uuid.UUID                     `json:"action_taken_by,omitempty"`
}

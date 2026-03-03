package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ContradictionAlertStatus captures lifecycle state for contradiction alerts.
type ContradictionAlertStatus string

const (
	ContradictionAlertStatusOpen      ContradictionAlertStatus = "open"
	ContradictionAlertStatusResolved  ContradictionAlertStatus = "resolved"
	ContradictionAlertStatusDismissed ContradictionAlertStatus = "dismissed"
)

// ParseContradictionAlertStatus validates and normalizes contradiction status values.
func ParseContradictionAlertStatus(value string) (ContradictionAlertStatus, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch ContradictionAlertStatus(normalized) {
	case ContradictionAlertStatusOpen:
		return ContradictionAlertStatusOpen, nil
	case ContradictionAlertStatusResolved:
		return ContradictionAlertStatusResolved, nil
	case ContradictionAlertStatusDismissed:
		return ContradictionAlertStatusDismissed, nil
	default:
		return "", fmt.Errorf("unsupported contradiction alert status %q", value)
	}
}

// EngramContradictionAlert describes a persisted contradiction warning between linked engrams.
type EngramContradictionAlert struct {
	AlertID              uuid.UUID                `json:"alert_id"`
	ProjectID            string                   `json:"project_id"`
	SourceEngramID       uuid.UUID                `json:"source_engram_id"`
	TargetEngramID       uuid.UUID                `json:"target_engram_id"`
	ContradictionLinkIDs []uuid.UUID              `json:"contradiction_link_ids"`
	Reason               string                   `json:"reason"`
	AlertHash            string                   `json:"alert_hash"`
	ConfidenceScore      float64                  `json:"confidence_score"`
	Status               ContradictionAlertStatus `json:"status"`
	DetectedAt           time.Time                `json:"detected_at"`
	UpdatedAt            time.Time                `json:"updated_at"`
	ResolvedAt           *time.Time               `json:"resolved_at,omitempty"`
	ResolvedBy           *uuid.UUID               `json:"resolved_by,omitempty"`
}

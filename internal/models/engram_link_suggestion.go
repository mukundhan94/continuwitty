package models

import (
	"time"

	"github.com/google/uuid"
)

// EngramLinkSuggestion captures a proposed source->target edge candidate.
type EngramLinkSuggestion struct {
	SourceEngramID  uuid.UUID              `json:"source_engram_id"`
	TargetEngramID  uuid.UUID              `json:"target_engram_id"`
	ProjectID       string                 `json:"project_id"`
	TargetTitle     string                 `json:"target_title"`
	TargetAbstract  string                 `json:"target_abstract"`
	TargetCreatedAt time.Time              `json:"target_created_at"`
	RelationType    EngramLinkRelationType `json:"relation_type"`
	Weight          float64                `json:"weight"`
	TemporalWeight  float64                `json:"temporal_weight"`
	Confidence      float64                `json:"confidence"`
	Score           float64                `json:"score"`
	Origin          EngramLinkOrigin       `json:"origin"`
	Status          EngramLinkStatus       `json:"status"`
	Reasons         []string               `json:"reasons"`
	EvidenceJSON    map[string]any         `json:"evidence_json"`
}

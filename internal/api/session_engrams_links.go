package api

import (
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

const (
	defaultEngramLinkListLimit          = 50
	defaultEngramLinkSuggestLimit       = 5
	defaultEngramLinkSuggestMaxCandiate = 20
	defaultEngramTraceMaxDepth          = 2
	defaultEngramTraceMaxNeighbors      = 20
	defaultEngramLinkHygieneLimit       = 500
	defaultEngramLinkHygieneStaleDays   = 120
	defaultEngramLinkHygieneLowValue    = 0.25
)

// SessionEngramLinkCreateInput captures authenticated create-link route inputs.
type SessionEngramLinkCreateInput struct {
	SourceEngramID   uuid.UUID
	TargetEngramID   uuid.UUID
	RelationType     models.EngramLinkRelationType
	Weight           float64
	TemporalWeight   float64
	Confidence       float64
	Origin           models.EngramLinkOrigin
	Status           models.EngramLinkStatus
	EvidenceJSON     map[string]any
	CreatedByUserID  uuid.UUID
	ActorUserID      uuid.UUID
	LastReinforcedAt *time.Time
}

// SessionEngramLinkListInput captures authenticated list-link route inputs.
type SessionEngramLinkListInput struct {
	SourceEngramID  uuid.UUID
	ActorUserID     uuid.UUID
	RelationType    *models.EngramLinkRelationType
	IncludeArchived bool
	Limit           int
	Offset          int
}

// SessionEngramLinkUpdateInput captures authenticated update-link route inputs.
type SessionEngramLinkUpdateInput struct {
	LinkID           uuid.UUID
	ActorUserID      uuid.UUID
	Weight           *float64
	TemporalWeight   *float64
	Confidence       *float64
	Status           *models.EngramLinkStatus
	EvidenceJSON     *map[string]any
	LastReinforcedAt *time.Time
}

// SessionEngramLinkArchiveInput captures authenticated archive-link route inputs.
type SessionEngramLinkArchiveInput struct {
	LinkID      uuid.UUID
	ActorUserID uuid.UUID
}

// SessionEngramLinkSuggestInput captures authenticated suggestion route inputs.
type SessionEngramLinkSuggestInput struct {
	SourceEngramID  uuid.UUID
	ActorUserID     uuid.UUID
	Limit           int
	MaxCandidates   int
	MinimumScore    float64
	IncludeArchived bool
}

// SessionEngramTraceInput captures authenticated trace route inputs.
type SessionEngramTraceInput struct {
	RootEngramID    uuid.UUID
	ActorUserID     uuid.UUID
	MaxDepth        int
	MaxNeighbors    int
	IncludeArchived bool
}

// SessionEngramLinkHygieneInput captures authenticated hygiene recommendation inputs.
type SessionEngramLinkHygieneInput struct {
	SourceEngramID    uuid.UUID
	ActorUserID       uuid.UUID
	IncludeArchived   bool
	Limit             int
	StaleAfterDays    int
	LowValueThreshold float64
}

type createEngramLinkPayload struct {
	TargetEngramID   uuid.UUID      `json:"target_engram_id"`
	RelationType     string         `json:"relation_type"`
	Weight           *float64       `json:"weight,omitempty"`
	TemporalWeight   *float64       `json:"temporal_weight,omitempty"`
	Confidence       *float64       `json:"confidence,omitempty"`
	Origin           string         `json:"origin"`
	Status           string         `json:"status"`
	EvidenceJSON     map[string]any `json:"evidence_json,omitempty"`
	LastReinforcedAt *time.Time     `json:"last_reinforced_at,omitempty"`
}

type updateEngramLinkPayload struct {
	Weight           *float64        `json:"weight,omitempty"`
	TemporalWeight   *float64        `json:"temporal_weight,omitempty"`
	Confidence       *float64        `json:"confidence,omitempty"`
	Status           *string         `json:"status,omitempty"`
	EvidenceJSON     *map[string]any `json:"evidence_json,omitempty"`
	LastReinforcedAt *time.Time      `json:"last_reinforced_at,omitempty"`
}

type suggestEngramLinkPayload struct {
	Limit           int      `json:"limit"`
	MaxCandidates   int      `json:"max_candidates"`
	MinimumScore    *float64 `json:"minimum_score,omitempty"`
	IncludeArchived *bool    `json:"include_archived,omitempty"`
}

type traceEngramPayload struct {
	MaxDepth        int   `json:"max_depth"`
	MaxNeighbors    int   `json:"max_neighbors"`
	IncludeArchived *bool `json:"include_archived,omitempty"`
}

type hygieneEngramLinkPayload struct {
	IncludeArchived   *bool    `json:"include_archived,omitempty"`
	Limit             int      `json:"limit"`
	StaleAfterDays    int      `json:"stale_after_days"`
	LowValueThreshold *float64 `json:"low_value_threshold,omitempty"`
}

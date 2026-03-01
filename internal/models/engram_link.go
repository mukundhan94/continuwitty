package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// EngramLinkRelationType captures directed relationship semantics between two engrams.
type EngramLinkRelationType string

const (
	EngramLinkRelationSupports    EngramLinkRelationType = "supports"
	EngramLinkRelationDependsOn   EngramLinkRelationType = "depends_on"
	EngramLinkRelationContradicts EngramLinkRelationType = "contradicts"
	EngramLinkRelationRelatedTo   EngramLinkRelationType = "related_to"
	EngramLinkRelationDerivedFrom EngramLinkRelationType = "derived_from"
)

// ParseEngramLinkRelationType validates relationship type values.
func ParseEngramLinkRelationType(value string) (EngramLinkRelationType, error) {
	switch EngramLinkRelationType(value) {
	case EngramLinkRelationSupports,
		EngramLinkRelationDependsOn,
		EngramLinkRelationContradicts,
		EngramLinkRelationRelatedTo,
		EngramLinkRelationDerivedFrom:
		return EngramLinkRelationType(value), nil
	default:
		return "", fmt.Errorf("unsupported engram link relation type %q", value)
	}
}

// EngramLinkOrigin captures where a link originated.
type EngramLinkOrigin string

const (
	EngramLinkOriginManual    EngramLinkOrigin = "manual"
	EngramLinkOriginSuggested EngramLinkOrigin = "suggested"
	EngramLinkOriginInferred  EngramLinkOrigin = "inferred"
	EngramLinkOriginSystem    EngramLinkOrigin = "system"
)

// ParseEngramLinkOrigin validates link origin values.
func ParseEngramLinkOrigin(value string) (EngramLinkOrigin, error) {
	switch EngramLinkOrigin(value) {
	case EngramLinkOriginManual,
		EngramLinkOriginSuggested,
		EngramLinkOriginInferred,
		EngramLinkOriginSystem:
		return EngramLinkOrigin(value), nil
	default:
		return "", fmt.Errorf("unsupported engram link origin %q", value)
	}
}

// EngramLinkStatus captures lifecycle state for an engram link.
type EngramLinkStatus string

const (
	EngramLinkStatusActive    EngramLinkStatus = "active"
	EngramLinkStatusSuggested EngramLinkStatus = "suggested"
	EngramLinkStatusArchived  EngramLinkStatus = "archived"
	EngramLinkStatusRejected  EngramLinkStatus = "rejected"
)

// ParseEngramLinkStatus validates link status values.
func ParseEngramLinkStatus(value string) (EngramLinkStatus, error) {
	switch EngramLinkStatus(value) {
	case EngramLinkStatusActive,
		EngramLinkStatusSuggested,
		EngramLinkStatusArchived,
		EngramLinkStatusRejected:
		return EngramLinkStatus(value), nil
	default:
		return "", fmt.Errorf("unsupported engram link status %q", value)
	}
}

// EngramLinkRecord models persisted engram-to-engram directed edge rows.
type EngramLinkRecord struct {
	LinkID           uuid.UUID              `json:"link_id"`
	ProjectID        string                 `json:"project_id"`
	SourceEngramID   uuid.UUID              `json:"source_engram_id"`
	TargetEngramID   uuid.UUID              `json:"target_engram_id"`
	RelationType     EngramLinkRelationType `json:"relation_type"`
	Weight           float64                `json:"weight"`
	TemporalWeight   float64                `json:"temporal_weight"`
	Confidence       float64                `json:"confidence"`
	Origin           EngramLinkOrigin       `json:"origin"`
	Status           EngramLinkStatus       `json:"status"`
	EvidenceJSON     map[string]any         `json:"evidence_json"`
	CreatedByUserID  uuid.UUID              `json:"created_by_user_id"`
	LastReinforcedAt *time.Time             `json:"last_reinforced_at,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// EngramLinkTraversalStep captures one traversal edge plus depth context.
type EngramLinkTraversalStep struct {
	Depth int              `json:"depth"`
	Link  EngramLinkRecord `json:"link"`
}

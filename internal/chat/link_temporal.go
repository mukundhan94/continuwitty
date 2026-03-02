package chat

import (
	"math"
	"time"

	"engram/internal/models"
)

const (
	linkTemporalDecayHalfLifeHours = 45.0 * 24.0
	linkRecencyHalfLifeHours       = 30.0 * 24.0
	linkReinforcementBoost         = 0.18
)

func linkTemporalReference(link models.EngramLinkRecord) time.Time {
	if link.LastReinforcedAt != nil && !link.LastReinforcedAt.IsZero() {
		return *link.LastReinforcedAt
	}
	return link.CreatedAt
}

func decayScoreByHalfLife(ageHours float64, halfLifeHours float64) float64 {
	if ageHours <= 0 {
		return 1.0
	}
	return clampFloat(math.Exp(-math.Ln2*(ageHours/halfLifeHours)), 0.0, 1.0)
}

// DecayedLinkTemporalWeight applies a half-life decay to persisted temporal weight.
func DecayedLinkTemporalWeight(now time.Time, link models.EngramLinkRecord) float64 {
	reference := linkTemporalReference(link)
	if reference.IsZero() {
		return clampFloat(link.TemporalWeight, 0.0, 1.0)
	}
	ageHours := now.Sub(reference).Hours()
	decay := decayScoreByHalfLife(ageHours, linkTemporalDecayHalfLifeHours)
	return clampFloat(link.TemporalWeight*decay, 0.0, 1.0)
}

// ReinforcedLinkTemporalWeight decays stale temporal weight and applies reinforcement boost.
func ReinforcedLinkTemporalWeight(now time.Time, link models.EngramLinkRecord) float64 {
	return clampFloat(DecayedLinkTemporalWeight(now, link)+linkReinforcementBoost, 0.0, 1.0)
}

func scoreLinkRecencyAt(now time.Time, link models.EngramLinkRecord) float64 {
	reference := linkTemporalReference(link)
	if reference.IsZero() {
		return 0.5
	}
	return decayScoreByHalfLife(now.Sub(reference).Hours(), linkRecencyHalfLifeHours)
}

package chat

import (
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestDecayedLinkTemporalWeightDecreasesWithAge(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	freshLink := models.EngramLinkRecord{
		LinkID:         uuid.MustParse("00000000-0000-0000-0000-000000009001"),
		TemporalWeight: 0.8,
		CreatedAt:      now,
	}
	staleLink := models.EngramLinkRecord{
		LinkID:         uuid.MustParse("00000000-0000-0000-0000-000000009002"),
		TemporalWeight: 0.8,
		CreatedAt:      now.Add(-120 * 24 * time.Hour),
	}

	freshWeight := DecayedLinkTemporalWeight(now, freshLink)
	staleWeight := DecayedLinkTemporalWeight(now, staleLink)
	if staleWeight >= freshWeight {
		t.Fatalf("expected stale weight < fresh weight, got stale=%f fresh=%f", staleWeight, freshWeight)
	}
}

func TestReinforcedLinkTemporalWeightBoostsAfterDecay(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	link := models.EngramLinkRecord{
		LinkID:           uuid.MustParse("00000000-0000-0000-0000-000000009010"),
		TemporalWeight:   0.55,
		CreatedAt:        now.Add(-60 * 24 * time.Hour),
		LastReinforcedAt: nil,
	}

	decayed := DecayedLinkTemporalWeight(now, link)
	reinforced := ReinforcedLinkTemporalWeight(now, link)
	if reinforced <= decayed {
		t.Fatalf("expected reinforced weight > decayed weight, got reinforced=%f decayed=%f", reinforced, decayed)
	}
	if reinforced > 1.0 {
		t.Fatalf("expected reinforced weight <= 1, got %f", reinforced)
	}
}

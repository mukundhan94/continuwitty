package main

import (
	"reflect"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestDedupeUUIDsSkipsNilAndPreservesOrder(t *testing.T) {
	first := uuid.MustParse("00000000-0000-0000-0000-000000008001")
	second := uuid.MustParse("00000000-0000-0000-0000-000000008002")
	deduped := dedupeUUIDs([]uuid.UUID{first, uuid.Nil, first, second, second})
	expected := []uuid.UUID{first, second}
	if !reflect.DeepEqual(expected, deduped) {
		t.Fatalf("expected %v, got %v", expected, deduped)
	}
}

func TestStatusForReinforcedLinkPromotesSuggestedToActive(t *testing.T) {
	suggested := models.EngramLinkRecord{Status: models.EngramLinkStatusSuggested}
	promoted := statusForReinforcedLink(suggested)
	if promoted == nil {
		t.Fatalf("expected suggested link status to be promoted")
	}
	if *promoted != models.EngramLinkStatusActive {
		t.Fatalf("expected active status, got %s", *promoted)
	}

	active := models.EngramLinkRecord{Status: models.EngramLinkStatusActive}
	if statusForReinforcedLink(active) != nil {
		t.Fatalf("expected active link to keep nil status override")
	}
}

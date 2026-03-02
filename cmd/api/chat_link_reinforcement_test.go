package main

import (
	"context"
	"reflect"
	"testing"
	"time"

	"engram/internal/graph"
	"engram/internal/models"
	"engram/internal/repository"

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

func TestLinkHygieneRunTrackerHonorsInterval(t *testing.T) {
	tracker := newLinkHygieneRunTracker()
	source := uuid.MustParse("00000000-0000-0000-0000-000000008100")
	now := time.Date(2026, time.March, 1, 10, 0, 0, 0, time.UTC)
	interval := 24 * time.Hour

	if !tracker.shouldRun(source, now, interval) {
		t.Fatalf("expected first run to be due")
	}
	tracker.markRun(source, now)
	if tracker.shouldRun(source, now.Add(2*time.Hour), interval) {
		t.Fatalf("expected run to be deferred inside interval")
	}
	if !tracker.shouldRun(source, now.Add(interval), interval) {
		t.Fatalf("expected run to become due at interval boundary")
	}
}

func TestSelectAutoArchiveLinkIDsFiltersActionsAndDedupes(t *testing.T) {
	first := uuid.MustParse("00000000-0000-0000-0000-000000008201")
	second := uuid.MustParse("00000000-0000-0000-0000-000000008202")
	ignored := uuid.MustParse("00000000-0000-0000-0000-000000008203")
	recommendations := []models.EngramLinkHygieneRecommendation{
		{
			SuggestedAction: "archive_stale_low_value",
			LinkIDs:         []uuid.UUID{first},
		},
		{
			SuggestedAction: "archive_weaker_duplicate",
			LinkIDs:         []uuid.UUID{ignored},
		},
		{
			SuggestedAction: "archive_stale_low_value",
			LinkIDs:         []uuid.UUID{first, second},
		},
		{
			SuggestedAction: "archive_stale_low_value",
			LinkIDs:         []uuid.UUID{second},
		},
	}

	selected := selectAutoArchiveLinkIDs(recommendations, 2)
	expected := []uuid.UUID{first, second}
	if !reflect.DeepEqual(expected, selected) {
		t.Fatalf("expected %v, got %v", expected, selected)
	}
}

func TestScheduledLinkHygieneExecutorRunsByDueInterval(t *testing.T) {
	source := uuid.MustParse("00000000-0000-0000-0000-000000008301")
	actor := uuid.MustParse("00000000-0000-0000-0000-000000008302")
	archivedLink := uuid.MustParse("00000000-0000-0000-0000-000000008303")
	now := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)

	recommendCalls := 0
	archiveCalls := 0
	executor := &scheduledLinkHygieneExecutor{
		interval:       24 * time.Hour,
		maxAutoArchive: 5,
		tracker:        newLinkHygieneRunTracker(),
		recommend: func(ctx context.Context, graphInput graph.LinkHygieneInput) ([]models.EngramLinkHygieneRecommendation, error) {
			recommendCalls++
			if graphInput.SourceEngramID != source {
				t.Fatalf("unexpected source engram id: %s", graphInput.SourceEngramID)
			}
			if graphInput.ActorUserID != actor {
				t.Fatalf("unexpected actor user id: %s", graphInput.ActorUserID)
			}
			return []models.EngramLinkHygieneRecommendation{
				{
					SuggestedAction: "archive_stale_low_value",
					LinkIDs:         []uuid.UUID{archivedLink},
				},
			}, nil
		},
		archive: func(ctx context.Context, input repository.EngramLinkArchiveInput) (*models.EngramLinkRecord, error) {
			archiveCalls++
			return &models.EngramLinkRecord{}, nil
		},
	}

	if err := executor.execute(context.Background(), actor, []uuid.UUID{source}, now); err != nil {
		t.Fatalf("execute first run: %v", err)
	}
	if err := executor.execute(context.Background(), actor, []uuid.UUID{source}, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("execute second run: %v", err)
	}
	if err := executor.execute(context.Background(), actor, []uuid.UUID{source}, now.Add(25*time.Hour)); err != nil {
		t.Fatalf("execute third run: %v", err)
	}

	if recommendCalls != 2 {
		t.Fatalf("expected 2 recommend calls, got %d", recommendCalls)
	}
	if archiveCalls != 2 {
		t.Fatalf("expected 2 archive calls, got %d", archiveCalls)
	}
}

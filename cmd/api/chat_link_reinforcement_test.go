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

func TestScheduledLinkHygieneExecutorPersistsLinkCurationSuggestions(t *testing.T) {
	source := uuid.MustParse("00000000-0000-0000-0000-000000008401")
	target := uuid.MustParse("00000000-0000-0000-0000-000000008402")
	actor := uuid.MustParse("00000000-0000-0000-0000-000000008403")
	autoArchivedLink := uuid.MustParse("00000000-0000-0000-0000-000000008404")
	manualReviewLink := uuid.MustParse("00000000-0000-0000-0000-000000008405")
	now := time.Date(2026, time.March, 2, 9, 0, 0, 0, time.UTC)
	projectID := "engram-vault"

	archiveCalls := 0
	createCalls := 0
	created := []repository.MemoryCurationSuggestionCreateInput{}
	executor := &scheduledLinkHygieneExecutor{
		interval:       24 * time.Hour,
		maxAutoArchive: 5,
		tracker:        newLinkHygieneRunTracker(),
		recommend: func(
			context.Context,
			graph.LinkHygieneInput,
		) ([]models.EngramLinkHygieneRecommendation, error) {
			return []models.EngramLinkHygieneRecommendation{
				{
					SourceEngramID:  source,
					TargetEngramID:  target,
					SuggestedAction: "archive_stale_low_value",
					LinkIDs:         []uuid.UUID{autoArchivedLink},
					Detail:          "stale low value",
					Severity:        "low",
					Category:        models.EngramLinkHygieneCategoryStaleLowValue,
					Score:           0.1,
				},
				{
					SourceEngramID:  source,
					TargetEngramID:  target,
					SuggestedAction: "review_relation_conflict",
					LinkIDs:         []uuid.UUID{manualReviewLink},
					Detail:          "conflict relation",
					Severity:        "high",
					Category:        models.EngramLinkHygieneCategoryConflictRelation,
					Score:           0.4,
				},
			}, nil
		},
		archive: func(
			context.Context,
			repository.EngramLinkArchiveInput,
		) (*models.EngramLinkRecord, error) {
			archiveCalls++
			return &models.EngramLinkRecord{}, nil
		},
		resolveSourceProjectID: func(context.Context, uuid.UUID) (*string, error) {
			return &projectID, nil
		},
		listLinkCurationSuggestions: func(
			context.Context,
			string,
		) ([]models.MemoryCurationSuggestion, error) {
			return []models.MemoryCurationSuggestion{}, nil
		},
		createMemoryCurationSuggestion: func(
			_ context.Context,
			input repository.MemoryCurationSuggestionCreateInput,
		) (*models.MemoryCurationSuggestion, error) {
			createCalls++
			created = append(created, input)
			return &models.MemoryCurationSuggestion{}, nil
		},
	}

	if err := executor.executeForSource(context.Background(), actor, source, now); err != nil {
		t.Fatalf("executeForSource: %v", err)
	}

	assertLinkCurationSuggestionCounts(t, archiveCalls, createCalls)
	assertLinkCurationSuggestionPayload(
		t,
		created,
		projectID,
		manualReviewLink,
	)
}

func TestPersistLinkCurationSuggestionsSkipsExistingDedupedEntry(t *testing.T) {
	source := uuid.MustParse("00000000-0000-0000-0000-000000008501")
	target := uuid.MustParse("00000000-0000-0000-0000-000000008502")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000008503")
	projectID := "engram-vault"
	createCalls := 0
	executor := &scheduledLinkHygieneExecutor{
		resolveSourceProjectID: func(context.Context, uuid.UUID) (*string, error) {
			return &projectID, nil
		},
		listLinkCurationSuggestions: func(
			context.Context,
			string,
		) ([]models.MemoryCurationSuggestion, error) {
			return []models.MemoryCurationSuggestion{
				{
					PayloadJSON: map[string]any{
						"link_id":          linkID.String(),
						"target_engram_id": target.String(),
						"suggested_action": "review_relation_conflict",
					},
				},
			}, nil
		},
		createMemoryCurationSuggestion: func(
			context.Context,
			repository.MemoryCurationSuggestionCreateInput,
		) (*models.MemoryCurationSuggestion, error) {
			createCalls++
			return &models.MemoryCurationSuggestion{}, nil
		},
	}

	err := executor.persistLinkCurationSuggestions(
		context.Background(),
		source,
		[]models.EngramLinkHygieneRecommendation{
			{
				SourceEngramID:  source,
				TargetEngramID:  target,
				SuggestedAction: "review_relation_conflict",
				LinkIDs:         []uuid.UUID{linkID},
				Detail:          "conflict relation",
				Severity:        "high",
				Category:        models.EngramLinkHygieneCategoryConflictRelation,
				Score:           0.4,
			},
		},
		time.Date(2026, time.March, 2, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("persistLinkCurationSuggestions: %v", err)
	}
	if createCalls != 0 {
		t.Fatalf("expected no created link curation suggestion for deduped recommendation")
	}
}

func assertLinkCurationSuggestionCounts(t *testing.T, archiveCalls int, createCalls int) {
	t.Helper()
	if archiveCalls != 1 {
		t.Fatalf("expected 1 auto archive call, got %d", archiveCalls)
	}
	if createCalls != 1 {
		t.Fatalf("expected 1 created link curation suggestion, got %d", createCalls)
	}
}

func assertLinkCurationSuggestionPayload(
	t *testing.T,
	created []repository.MemoryCurationSuggestionCreateInput,
	projectID string,
	linkID uuid.UUID,
) {
	t.Helper()
	if len(created) != 1 {
		t.Fatalf("expected exactly one created suggestion, got %d", len(created))
	}
	input := created[0]
	if input.SuggestionType != models.MemoryCurationSuggestionTypeLink {
		t.Fatalf("expected link suggestion type, got %s", input.SuggestionType)
	}
	if input.ProjectID != projectID {
		t.Fatalf("expected project id %q, got %q", projectID, input.ProjectID)
	}
	payload := input.PayloadJSON
	if payload == nil {
		t.Fatalf("expected link payload")
	}
	if payload["link_id"] != linkID.String() {
		t.Fatalf("expected payload link_id %s, got %v", linkID, payload["link_id"])
	}
	if payload["suggested_action"] != "review_relation_conflict" {
		t.Fatalf(
			"expected payload suggested_action review_relation_conflict, got %v",
			payload["suggested_action"],
		)
	}
}

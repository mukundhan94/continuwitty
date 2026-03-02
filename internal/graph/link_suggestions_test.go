package graph

import (
	"context"
	"strings"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestLinkSuggestionServiceSuggestLinksRanksHybridSignals(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000008001")
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-000000008002")
	targetOne := uuid.MustParse("00000000-0000-0000-0000-000000008003")
	targetTwo := uuid.MustParse("00000000-0000-0000-0000-000000008004")
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	service := newHybridLinkSuggestionService(
		t,
		hybridLinkSuggestionFixture{
			actorUserID:    actorUserID,
			sourceEngramID: sourceEngramID,
			targetOne:      targetOne,
			targetTwo:      targetTwo,
			now:            now,
		},
	)

	suggestions, err := service.SuggestLinks(
		context.Background(),
		LinkSuggestionInput{
			SourceEngramID: sourceEngramID,
			ActorUserID:    actorUserID,
			Limit:          2,
			MaxCandidates:  4,
			MinimumScore:   0.1,
		},
	)
	if err != nil {
		t.Fatalf("expected suggestions without error: %v", err)
	}
	assertHybridSuggestions(t, suggestions, targetOne)
}

func TestLinkSuggestionServiceSuggestLinksReturnsEmptyWhenSourceMissing(t *testing.T) {
	service := &LinkSuggestionService{
		getRehydrationBundle: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.RehydrationBundle, error) {
			return nil, nil
		},
		queryEngrams: func(_ context.Context, _ models.EngramQueryRequest, _ uuid.UUID) ([]models.EngramQueryResult, error) {
			t.Fatalf("query should not execute when source is missing")
			return nil, nil
		},
		getEngramSources: func(_ context.Context, _ uuid.UUID, _ int, _ uuid.UUID) ([]models.EngramSourceRecord, error) {
			t.Fatalf("source lookup should not execute when source is missing")
			return nil, nil
		},
		listEngramLinks: func(_ context.Context, _ repository.EngramLinkListInput) ([]models.EngramLinkRecord, error) {
			t.Fatalf("link lookup should not execute when source is missing")
			return nil, nil
		},
		now: nowUTC,
	}

	suggestions, err := service.SuggestLinks(
		context.Background(),
		LinkSuggestionInput{
			SourceEngramID: uuid.MustParse("00000000-0000-0000-0000-000000008010"),
			ActorUserID:    uuid.MustParse("00000000-0000-0000-0000-000000008011"),
		},
	)
	if err != nil {
		t.Fatalf("expected no error for missing source, got %v", err)
	}
	requireEqual(t, 0, len(suggestions))
}

func TestLinkSuggestionServiceSuggestLinksValidatesInput(t *testing.T) {
	service := &LinkSuggestionService{}
	_, err := service.SuggestLinks(
		context.Background(),
		LinkSuggestionInput{
			SourceEngramID: uuid.Nil,
			ActorUserID:    uuid.MustParse("00000000-0000-0000-0000-000000008012"),
		},
	)
	if err == nil {
		t.Fatalf("expected validation error for missing source engram id")
	}
	_, err = service.SuggestLinks(
		context.Background(),
		LinkSuggestionInput{
			SourceEngramID: uuid.MustParse("00000000-0000-0000-0000-000000008013"),
			ActorUserID:    uuid.MustParse("00000000-0000-0000-0000-000000008014"),
			MinimumScore:   1.2,
		},
	)
	if err == nil {
		t.Fatalf("expected validation error for minimum_score")
	}
}

func containsReasonPrefix(reasons []string, prefix string) bool {
	for _, reason := range reasons {
		if strings.HasPrefix(reason, prefix) {
			return true
		}
	}
	return false
}

func requireEqual[T comparable](t *testing.T, expected T, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

type hybridLinkSuggestionFixture struct {
	actorUserID    uuid.UUID
	sourceEngramID uuid.UUID
	targetOne      uuid.UUID
	targetTwo      uuid.UUID
	now            time.Time
}

func newHybridLinkSuggestionService(
	t *testing.T,
	fixture hybridLinkSuggestionFixture,
) *LinkSuggestionService {
	t.Helper()
	return &LinkSuggestionService{
		getRehydrationBundle: buildHybridRehydrationLookup(t, fixture),
		queryEngrams:         buildHybridQueryEngramsLookup(t, fixture),
		getEngramSources:     buildHybridSourceLookup(t, fixture),
		listEngramLinks:      buildHybridLinkListLookup(t, fixture),
		now: func() time.Time { return fixture.now },
	}
}

func buildHybridRehydrationLookup(
	t *testing.T,
	fixture hybridLinkSuggestionFixture,
) func(context.Context, uuid.UUID, uuid.UUID) (*models.RehydrationBundle, error) {
	t.Helper()
	return func(
		_ context.Context,
		engramID uuid.UUID,
		requestActorUserID uuid.UUID,
	) (*models.RehydrationBundle, error) {
		requireEqual(t, fixture.sourceEngramID, engramID)
		requireEqual(t, fixture.actorUserID, requestActorUserID)
		return &models.RehydrationBundle{
			EngramID:                fixture.sourceEngramID,
			ProjectID:               "proj-1",
			Title:                   "Source",
			CompactSummary:          "Checkpoint strategy",
			DetailedSummaryMarkdown: "Uses reliable checkpointing and citations",
		}, nil
	}
}

func buildHybridQueryEngramsLookup(
	t *testing.T,
	fixture hybridLinkSuggestionFixture,
) func(context.Context, models.EngramQueryRequest, uuid.UUID) ([]models.EngramQueryResult, error) {
	t.Helper()
	return func(
		_ context.Context,
		request models.EngramQueryRequest,
		requestActorUserID uuid.UUID,
	) ([]models.EngramQueryResult, error) {
		requireEqual(t, fixture.actorUserID, requestActorUserID)
		requireEqual(t, 4, request.TopK)
		if request.ProjectID == nil || *request.ProjectID != "proj-1" {
			t.Fatalf("expected project filter for source project")
		}
		return []models.EngramQueryResult{
			{
				EngramID:  fixture.sourceEngramID,
				ProjectID: "proj-1",
				Title:     "Source",
				Abstract:  "source",
				CreatedAt: fixture.now.Add(-1 * time.Hour),
				Distance:  0.01,
			},
			{
				EngramID:  fixture.targetOne,
				ProjectID: "proj-1",
				Title:     "Checkpoint references",
				Abstract:  "Shared citations and memory continuity",
				CreatedAt: fixture.now.Add(-24 * time.Hour),
				Distance:  0.1,
			},
			{
				EngramID:  fixture.targetTwo,
				ProjectID: "proj-1",
				Title:     "Already linked",
				Abstract:  "Should be filtered by existing links",
				CreatedAt: fixture.now.Add(-6 * time.Hour),
				Distance:  0.05,
			},
		}, nil
	}
}

func buildHybridSourceLookup(
	t *testing.T,
	fixture hybridLinkSuggestionFixture,
) func(context.Context, uuid.UUID, int, uuid.UUID) ([]models.EngramSourceRecord, error) {
	t.Helper()
	return func(
		_ context.Context,
		engramID uuid.UUID,
		limit int,
		requestActorUserID uuid.UUID,
	) ([]models.EngramSourceRecord, error) {
		requireEqual(t, sourceSuggestionLimit, limit)
		requireEqual(t, fixture.actorUserID, requestActorUserID)
		switch engramID {
		case fixture.sourceEngramID:
			return []models.EngramSourceRecord{
				{URL: "https://example.com/a"},
				{URL: "https://example.com/b"},
			}, nil
		case fixture.targetOne:
			return []models.EngramSourceRecord{
				{URL: "https://example.com/a"},
			}, nil
		default:
			return []models.EngramSourceRecord{}, nil
		}
	}
}

func buildHybridLinkListLookup(
	t *testing.T,
	fixture hybridLinkSuggestionFixture,
) func(context.Context, repository.EngramLinkListInput) ([]models.EngramLinkRecord, error) {
	t.Helper()
	return func(
		_ context.Context,
		input repository.EngramLinkListInput,
	) ([]models.EngramLinkRecord, error) {
		requireEqual(t, fixture.sourceEngramID, input.SourceEngramID)
		return []models.EngramLinkRecord{
			{
				SourceEngramID: fixture.sourceEngramID,
				TargetEngramID: fixture.targetTwo,
			},
		}, nil
	}
}

func assertHybridSuggestions(
	t *testing.T,
	suggestions []models.EngramLinkSuggestion,
	expectedTarget uuid.UUID,
) {
	t.Helper()
	requireEqual(t, 1, len(suggestions))
	suggestion := suggestions[0]
	requireEqual(t, expectedTarget, suggestion.TargetEngramID)
	requireEqual(t, models.EngramLinkOriginSuggested, suggestion.Origin)
	requireEqual(t, models.EngramLinkStatusSuggested, suggestion.Status)
	if suggestion.Score <= 0 {
		t.Fatalf("expected positive suggestion score")
	}
	if !containsReasonPrefix(suggestion.Reasons, "shared_sources=") {
		t.Fatalf("expected shared source reason in suggestion")
	}
}

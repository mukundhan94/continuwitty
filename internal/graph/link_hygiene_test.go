package graph

import (
	"context"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestLinkHygieneServiceRecommendDetectsDuplicateConflictAndStale(t *testing.T) {
	now := time.Date(2026, 3, 1, 13, 0, 0, 0, time.UTC)
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000a001")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-00000000a002")
	targetDuplicate := uuid.MustParse("00000000-0000-0000-0000-00000000a003")
	targetConflict := uuid.MustParse("00000000-0000-0000-0000-00000000a004")
	targetStale := uuid.MustParse("00000000-0000-0000-0000-00000000a005")
	links := buildHygieneLinksFixture(hygieneLinkFixtureInput{
		sourceEngramID: sourceEngramID,
		targetDuplicate: targetDuplicate,
		targetConflict:  targetConflict,
		targetStale:     targetStale,
		now:             now,
	})

	service := &LinkHygieneService{
		listEngramLinks: func(
			_ context.Context,
			input repository.EngramLinkListInput,
		) ([]models.EngramLinkRecord, error) {
			if input.SourceEngramID != sourceEngramID {
				t.Fatalf("expected source engram id %s", sourceEngramID)
			}
			if input.ActorUserID != actorUserID {
				t.Fatalf("expected actor user id %s", actorUserID)
			}
			return links, nil
		},
		nowUTC: func() time.Time { return now },
	}

	recommendations, err := service.Recommend(
		context.Background(),
		LinkHygieneInput{
			SourceEngramID:    sourceEngramID,
			ActorUserID:       actorUserID,
			IncludeArchived:   false,
			StaleAfterDays:    90,
			LowValueThreshold: 0.3,
		},
	)
	if err != nil {
		t.Fatalf("recommend link hygiene: %v", err)
	}
	assertHygieneCategoriesPresent(t, recommendations)
}

func TestLinkHygieneServiceRejectsMissingIDs(t *testing.T) {
	service := &LinkHygieneService{}
	_, err := service.Recommend(context.Background(), LinkHygieneInput{})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

type hygieneLinkFixtureInput struct {
	sourceEngramID  uuid.UUID
	targetDuplicate uuid.UUID
	targetConflict  uuid.UUID
	targetStale     uuid.UUID
	now             time.Time
}

func buildHygieneLinksFixture(input hygieneLinkFixtureInput) []models.EngramLinkRecord {
	return []models.EngramLinkRecord{
		{
			LinkID:         uuid.MustParse("00000000-0000-0000-0000-00000000a010"),
			SourceEngramID: input.sourceEngramID,
			TargetEngramID: input.targetDuplicate,
			RelationType:   models.EngramLinkRelationSupports,
			Weight:         0.9,
			TemporalWeight: 0.8,
			Confidence:     0.9,
			Status:         models.EngramLinkStatusActive,
			CreatedAt:      input.now.Add(-2 * 24 * time.Hour),
		},
		{
			LinkID:         uuid.MustParse("00000000-0000-0000-0000-00000000a011"),
			SourceEngramID: input.sourceEngramID,
			TargetEngramID: input.targetDuplicate,
			RelationType:   models.EngramLinkRelationRelatedTo,
			Weight:         0.25,
			TemporalWeight: 0.2,
			Confidence:     0.3,
			Status:         models.EngramLinkStatusActive,
			CreatedAt:      input.now.Add(-10 * 24 * time.Hour),
		},
		{
			LinkID:         uuid.MustParse("00000000-0000-0000-0000-00000000a012"),
			SourceEngramID: input.sourceEngramID,
			TargetEngramID: input.targetConflict,
			RelationType:   models.EngramLinkRelationSupports,
			Weight:         0.6,
			TemporalWeight: 0.5,
			Confidence:     0.7,
			Status:         models.EngramLinkStatusActive,
			CreatedAt:      input.now.Add(-5 * 24 * time.Hour),
		},
		{
			LinkID:         uuid.MustParse("00000000-0000-0000-0000-00000000a013"),
			SourceEngramID: input.sourceEngramID,
			TargetEngramID: input.targetConflict,
			RelationType:   models.EngramLinkRelationContradicts,
			Weight:         0.45,
			TemporalWeight: 0.4,
			Confidence:     0.5,
			Status:         models.EngramLinkStatusActive,
			CreatedAt:      input.now.Add(-7 * 24 * time.Hour),
		},
		{
			LinkID:         uuid.MustParse("00000000-0000-0000-0000-00000000a014"),
			SourceEngramID: input.sourceEngramID,
			TargetEngramID: input.targetStale,
			RelationType:   models.EngramLinkRelationRelatedTo,
			Weight:         0.2,
			TemporalWeight: 0.15,
			Confidence:     0.2,
			Status:         models.EngramLinkStatusActive,
			CreatedAt:      input.now.Add(-220 * 24 * time.Hour),
		},
	}
}

func assertHygieneCategoriesPresent(
	t *testing.T,
	recommendations []models.EngramLinkHygieneRecommendation,
) {
	t.Helper()
	categories := map[models.EngramLinkHygieneCategory]bool{}
	for _, recommendation := range recommendations {
		categories[recommendation.Category] = true
	}
	required := []models.EngramLinkHygieneCategory{
		models.EngramLinkHygieneCategoryDuplicateTarget,
		models.EngramLinkHygieneCategoryConflictRelation,
		models.EngramLinkHygieneCategoryStaleLowValue,
	}
	for _, category := range required {
		if !categories[category] {
			t.Fatalf("expected category %s in recommendations: %+v", category, recommendations)
		}
	}
}

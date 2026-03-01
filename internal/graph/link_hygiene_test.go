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

	links := []models.EngramLinkRecord{
		{
			LinkID:         uuid.MustParse("00000000-0000-0000-0000-00000000a010"),
			SourceEngramID: sourceEngramID,
			TargetEngramID: targetDuplicate,
			RelationType:   models.EngramLinkRelationSupports,
			Weight:         0.9,
			TemporalWeight: 0.8,
			Confidence:     0.9,
			Status:         models.EngramLinkStatusActive,
			CreatedAt:      now.Add(-2 * 24 * time.Hour),
		},
		{
			LinkID:         uuid.MustParse("00000000-0000-0000-0000-00000000a011"),
			SourceEngramID: sourceEngramID,
			TargetEngramID: targetDuplicate,
			RelationType:   models.EngramLinkRelationRelatedTo,
			Weight:         0.25,
			TemporalWeight: 0.2,
			Confidence:     0.3,
			Status:         models.EngramLinkStatusActive,
			CreatedAt:      now.Add(-10 * 24 * time.Hour),
		},
		{
			LinkID:         uuid.MustParse("00000000-0000-0000-0000-00000000a012"),
			SourceEngramID: sourceEngramID,
			TargetEngramID: targetConflict,
			RelationType:   models.EngramLinkRelationSupports,
			Weight:         0.6,
			TemporalWeight: 0.5,
			Confidence:     0.7,
			Status:         models.EngramLinkStatusActive,
			CreatedAt:      now.Add(-5 * 24 * time.Hour),
		},
		{
			LinkID:         uuid.MustParse("00000000-0000-0000-0000-00000000a013"),
			SourceEngramID: sourceEngramID,
			TargetEngramID: targetConflict,
			RelationType:   models.EngramLinkRelationContradicts,
			Weight:         0.45,
			TemporalWeight: 0.4,
			Confidence:     0.5,
			Status:         models.EngramLinkStatusActive,
			CreatedAt:      now.Add(-7 * 24 * time.Hour),
		},
		{
			LinkID:         uuid.MustParse("00000000-0000-0000-0000-00000000a014"),
			SourceEngramID: sourceEngramID,
			TargetEngramID: targetStale,
			RelationType:   models.EngramLinkRelationRelatedTo,
			Weight:         0.2,
			TemporalWeight: 0.15,
			Confidence:     0.2,
			Status:         models.EngramLinkStatusActive,
			CreatedAt:      now.Add(-220 * 24 * time.Hour),
		},
	}

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

	hasDuplicate := false
	hasConflict := false
	hasStale := false
	for _, recommendation := range recommendations {
		switch recommendation.Category {
		case models.EngramLinkHygieneCategoryDuplicateTarget:
			hasDuplicate = true
		case models.EngramLinkHygieneCategoryConflictRelation:
			hasConflict = true
		case models.EngramLinkHygieneCategoryStaleLowValue:
			hasStale = true
		}
	}
	if !hasDuplicate || !hasConflict || !hasStale {
		t.Fatalf("expected duplicate/conflict/stale recommendations, got %+v", recommendations)
	}
}

func TestLinkHygieneServiceRejectsMissingIDs(t *testing.T) {
	service := &LinkHygieneService{}
	_, err := service.Recommend(context.Background(), LinkHygieneInput{})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

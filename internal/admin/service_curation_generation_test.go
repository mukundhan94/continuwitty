package admin

import (
	"context"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestSyncConsolidationCurationSuggestionsResetsAndCreates(t *testing.T) {
	service := NewService(nil, 256, nil)
	projectID := "engram-vault"
	suggestedAt := time.Date(2026, 3, 4, 9, 0, 0, 0, time.UTC)
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000c101")
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000c102")
	capturedReset := repository.MemoryCurationSuggestionResetInput{}
	capturedCreate := []repository.MemoryCurationSuggestionCreateInput{}
	service.deps.resetMemoryCurationSuggestions = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.MemoryCurationSuggestionResetInput,
	) (int, error) {
		capturedReset = input
		return 1, nil
	}
	service.deps.listConsolidationSuggestions = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ConsolidationSuggestionListInput,
	) ([]models.EngramConsolidationSuggestion, error) {
		if input.ProjectID == nil || *input.ProjectID != projectID {
			t.Fatalf("expected project filter to be forwarded")
		}
		if input.Status == nil || *input.Status != models.ConsolidationSuggestionStatusSuggested {
			t.Fatalf("expected suggested status filter")
		}
		return []models.EngramConsolidationSuggestion{
			{
				SuggestionID:      suggestionID,
				ProjectID:         projectID,
				SourceEngramIDs:   []uuid.UUID{sourceEngramID},
				ConsolidationType: models.ConsolidationSuggestionTypeExactDuplicate,
				Reason:            "duplicate title",
				ConsolidationHash: "hash",
				ConfidenceScore:   0.88,
				Status:            models.ConsolidationSuggestionStatusSuggested,
			},
		}, nil
	}
	service.deps.createMemoryCurationSuggestion = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.MemoryCurationSuggestionCreateInput,
	) (*models.MemoryCurationSuggestion, error) {
		capturedCreate = append(capturedCreate, input)
		return &models.MemoryCurationSuggestion{}, nil
	}

	err := service.syncConsolidationCurationSuggestions(context.Background(), &projectID, suggestedAt)
	requireNoError(t, err)
	requireEqual(t, projectID, derefString(capturedReset.ProjectID))
	requireEqual(t, models.MemoryCurationSuggestionTypeConsolidate, capturedReset.SuggestionType)
	requireEqual(t, 1, len(capturedCreate))
	requireEqual(t, projectID, capturedCreate[0].ProjectID)
	requireEqual(t, models.MemoryCurationSuggestionTypeConsolidate, capturedCreate[0].SuggestionType)
	requireEqual(t, suggestedAt, capturedCreate[0].SuggestedAt)
	requireEqual(t, 0.88, capturedCreate[0].ConfidenceScore)
}

func TestSyncContradictionCurationSuggestionsResetsAndCreates(t *testing.T) {
	service := NewService(nil, 256, nil)
	projectID := "engram-vault"
	suggestedAt := time.Date(2026, 3, 4, 9, 10, 0, 0, time.UTC)
	alertID := uuid.MustParse("00000000-0000-0000-0000-00000000c201")
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000c202")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000c203")
	capturedReset := repository.MemoryCurationSuggestionResetInput{}
	capturedCreate := []repository.MemoryCurationSuggestionCreateInput{}
	service.deps.resetMemoryCurationSuggestions = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.MemoryCurationSuggestionResetInput,
	) (int, error) {
		capturedReset = input
		return 1, nil
	}
	service.deps.listContradictionAlerts = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ContradictionAlertListInput,
	) ([]models.EngramContradictionAlert, error) {
		if input.ProjectID == nil || *input.ProjectID != projectID {
			t.Fatalf("expected project filter to be forwarded")
		}
		if input.Status == nil || *input.Status != models.ContradictionAlertStatusOpen {
			t.Fatalf("expected open status filter")
		}
		return []models.EngramContradictionAlert{
			{
				AlertID:              alertID,
				ProjectID:            projectID,
				SourceEngramID:       sourceEngramID,
				TargetEngramID:       targetEngramID,
				ContradictionLinkIDs: []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-00000000c204")},
				Reason:               "conflicting evidence",
				ConfidenceScore:      0.91,
				Status:               models.ContradictionAlertStatusOpen,
			},
		}, nil
	}
	service.deps.createMemoryCurationSuggestion = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.MemoryCurationSuggestionCreateInput,
	) (*models.MemoryCurationSuggestion, error) {
		capturedCreate = append(capturedCreate, input)
		return &models.MemoryCurationSuggestion{}, nil
	}

	err := service.syncContradictionCurationSuggestions(context.Background(), &projectID, suggestedAt)
	requireNoError(t, err)
	requireEqual(t, projectID, derefString(capturedReset.ProjectID))
	requireEqual(t, models.MemoryCurationSuggestionTypeContradiction, capturedReset.SuggestionType)
	requireEqual(t, 1, len(capturedCreate))
	requireEqual(t, projectID, capturedCreate[0].ProjectID)
	requireEqual(t, models.MemoryCurationSuggestionTypeContradiction, capturedCreate[0].SuggestionType)
	requireEqual(t, suggestedAt, capturedCreate[0].SuggestedAt)
	requireEqual(t, 0.91, capturedCreate[0].ConfidenceScore)
}

func TestBoundedCurationConfidenceClampsValues(t *testing.T) {
	requireEqual(t, 0.0, boundedCurationConfidence(-0.5))
	requireEqual(t, 1.0, boundedCurationConfidence(1.5))
	requireEqual(t, 0.42, boundedCurationConfidence(0.42))
}

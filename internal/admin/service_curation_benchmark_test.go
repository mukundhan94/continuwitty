package admin

import (
	"context"
	"fmt"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func BenchmarkActionMemoryCurationSuggestion(b *testing.B) {
	service := NewService(nil, 256, nil)
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000cb01")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-00000000cb02")
	projectID := "engram-vault"
	service.deps.applyMemoryCurationSuggestionAction = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.MemoryCurationSuggestionActionInput,
	) (*models.MemoryCurationSuggestion, error) {
		return &models.MemoryCurationSuggestion{
			SuggestionID:  input.SuggestionID,
			ProjectID:     projectID,
			Status:        input.Status,
			ActionedAt:    &input.ActionedAt,
			ActionTakenBy: &input.ActorUserID,
		}, nil
	}

	request := MemoryCurationSuggestionActionRequest{
		ProjectID: &projectID,
		Status:    models.MemoryCurationSuggestionStatusApplied,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := service.ActionMemoryCurationSuggestion(
			context.Background(),
			suggestionID,
			actorUserID,
			request,
		); err != nil {
			b.Fatalf("action failed: %v", err)
		}
	}
}

func BenchmarkSyncConsolidationCurationSuggestions50Candidates(b *testing.B) {
	benchmarkSyncConsolidationCurationSuggestions(b, 50)
}

func BenchmarkSyncConsolidationCurationSuggestions200Candidates(b *testing.B) {
	benchmarkSyncConsolidationCurationSuggestions(b, 200)
}

func benchmarkSyncConsolidationCurationSuggestions(b *testing.B, count int) {
	service := NewService(nil, 256, nil)
	projectID := "engram-vault"
	suggestedAt := time.Date(2026, 3, 4, 11, 0, 0, 0, time.UTC)
	candidates := make([]models.EngramConsolidationSuggestion, 0, count)
	for i := 0; i < count; i++ {
		candidates = append(candidates, models.EngramConsolidationSuggestion{
			SuggestionID:      uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012x", i+1)),
			ProjectID:         projectID,
			SourceEngramIDs:   []uuid.UUID{uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0001-%012x", i+1))},
			ConsolidationType: models.ConsolidationSuggestionTypeExactDuplicate,
			Reason:            "duplicate title",
			ConsolidationHash: fmt.Sprintf("hash-%d", i+1),
			ConfidenceScore:   0.8,
			Status:            models.ConsolidationSuggestionStatusSuggested,
		})
	}

	service.deps.resetMemoryCurationSuggestions = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.MemoryCurationSuggestionResetInput,
	) (int, error) {
		return count, nil
	}
	service.deps.listConsolidationSuggestions = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.ConsolidationSuggestionListInput,
	) ([]models.EngramConsolidationSuggestion, error) {
		return candidates, nil
	}
	service.deps.createMemoryCurationSuggestion = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.MemoryCurationSuggestionCreateInput,
	) (*models.MemoryCurationSuggestion, error) {
		return &models.MemoryCurationSuggestion{}, nil
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := service.syncConsolidationCurationSuggestions(context.Background(), &projectID, suggestedAt); err != nil {
			b.Fatalf("sync failed: %v", err)
		}
	}
}

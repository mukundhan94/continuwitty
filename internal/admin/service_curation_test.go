package admin

import (
	"context"
	"errors"
	"testing"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestListMemoryCurationSuggestionsUsesRepositoryRequestObject(t *testing.T) {
	service := NewService(nil, 256, nil)
	projectID := "engram-vault"
	sessionID := uuid.MustParse("00000000-0000-0000-0000-00000000c011")
	suggestionType := models.MemoryCurationSuggestionTypeLink
	status := models.MemoryCurationSuggestionStatusSuggested
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000c012")
	captured := repository.MemoryCurationSuggestionListInput{}
	service.deps.listMemoryCurationSuggestions = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.MemoryCurationSuggestionListInput,
	) ([]models.MemoryCurationSuggestion, error) {
		captured = input
		return []models.MemoryCurationSuggestion{
			{
				SuggestionID:   suggestionID,
				ProjectID:      projectID,
				SessionID:      &sessionID,
				SuggestionType: suggestionType,
				Status:         status,
			},
		}, nil
	}

	listed, err := service.ListMemoryCurationSuggestions(
		context.Background(),
		MemoryCurationSuggestionListRequest{
			ProjectID:      &projectID,
			SessionID:      &sessionID,
			SuggestionType: &suggestionType,
			Status:         &status,
			Limit:          25,
			Offset:         4,
		},
	)
	requireNoError(t, err)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	requireEqual(t, sessionID, *captured.SessionID)
	requireEqual(t, suggestionType, *captured.SuggestionType)
	requireEqual(t, status, *captured.Status)
	requireEqual(t, 25, captured.Limit)
	requireEqual(t, 4, captured.Offset)
	requireEqual(t, 1, len(listed))
	requireEqual(t, suggestionID, listed[0].SuggestionID)
}

func TestActionMemoryCurationSuggestionUsesRepositoryRequestObject(t *testing.T) {
	service := NewService(nil, 256, nil)
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000c021")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-00000000c022")
	projectID := "engram-vault"
	captured := repository.MemoryCurationSuggestionActionInput{}
	service.deps.applyMemoryCurationSuggestionAction = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.MemoryCurationSuggestionActionInput,
	) (*models.MemoryCurationSuggestion, error) {
		captured = input
		return &models.MemoryCurationSuggestion{
			SuggestionID: suggestionID,
			ProjectID:    projectID,
			Status:       models.MemoryCurationSuggestionStatusAccepted,
		}, nil
	}

	updated, err := service.ActionMemoryCurationSuggestion(
		context.Background(),
		suggestionID,
		actorUserID,
		MemoryCurationSuggestionActionRequest{
			ProjectID: &projectID,
			Status:    models.MemoryCurationSuggestionStatusAccepted,
		},
	)
	requireNoError(t, err)
	requireEqual(t, suggestionID, captured.SuggestionID)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	requireEqual(t, actorUserID, captured.ActorUserID)
	requireEqual(t, models.MemoryCurationSuggestionStatusAccepted, captured.Status)
	if updated == nil {
		t.Fatalf("expected updated memory curation suggestion")
	}
	requireEqual(t, suggestionID, updated.SuggestionID)
}

func TestActionMemoryCurationSuggestionRejectsInvalidStatus(t *testing.T) {
	service := NewService(nil, 256, nil)
	_, err := service.ActionMemoryCurationSuggestion(
		context.Background(),
		uuid.MustParse("00000000-0000-0000-0000-00000000c031"),
		uuid.MustParse("00000000-0000-0000-0000-00000000c032"),
		MemoryCurationSuggestionActionRequest{
			Status: models.MemoryCurationSuggestionStatusSuggested,
		},
	)
	if !errors.Is(err, ErrMemoryCurationSuggestionActionInvalid) {
		t.Fatalf("expected ErrMemoryCurationSuggestionActionInvalid, got %v", err)
	}
}

func TestActionMemoryCurationSuggestionReturnsNotFound(t *testing.T) {
	service := NewService(nil, 256, nil)
	service.deps.applyMemoryCurationSuggestionAction = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.MemoryCurationSuggestionActionInput,
	) (*models.MemoryCurationSuggestion, error) {
		return nil, nil
	}

	_, err := service.ActionMemoryCurationSuggestion(
		context.Background(),
		uuid.MustParse("00000000-0000-0000-0000-00000000c041"),
		uuid.MustParse("00000000-0000-0000-0000-00000000c042"),
		MemoryCurationSuggestionActionRequest{Status: models.MemoryCurationSuggestionStatusRejected},
	)
	if !errors.Is(err, ErrMemoryCurationSuggestionNotFound) {
		t.Fatalf("expected ErrMemoryCurationSuggestionNotFound, got %v", err)
	}
}

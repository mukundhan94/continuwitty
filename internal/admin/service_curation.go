package admin

import (
	"context"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

// ListMemoryCurationSuggestions lists persisted memory curation suggestions.
func (s *Service) ListMemoryCurationSuggestions(
	ctx context.Context,
	request MemoryCurationSuggestionListRequest,
) ([]models.MemoryCurationSuggestion, error) {
	return s.deps.listMemoryCurationSuggestions(
		ctx,
		s.db,
		repository.MemoryCurationSuggestionListInput{
			ProjectID:      request.ProjectID,
			SessionID:      request.SessionID,
			SuggestionType: request.SuggestionType,
			Status:         request.Status,
			Limit:          request.Limit,
			Offset:         request.Offset,
		},
	)
}

// ActionMemoryCurationSuggestion marks a memory curation suggestion as accepted/rejected/applied.
func (s *Service) ActionMemoryCurationSuggestion(
	ctx context.Context,
	suggestionID uuid.UUID,
	actorUserID uuid.UUID,
	request MemoryCurationSuggestionActionRequest,
) (*models.MemoryCurationSuggestion, error) {
	if !isMemoryCurationActionStatus(request.Status) {
		return nil, ErrMemoryCurationSuggestionActionInvalid
	}
	updated, err := s.deps.applyMemoryCurationSuggestionAction(
		ctx,
		s.db,
		repository.MemoryCurationSuggestionActionInput{
			SuggestionID: suggestionID,
			ProjectID:    request.ProjectID,
			Status:       request.Status,
			ActorUserID:  actorUserID,
		},
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrMemoryCurationSuggestionNotFound
	}
	return updated, nil
}

func isMemoryCurationActionStatus(status models.MemoryCurationSuggestionStatus) bool {
	return status == models.MemoryCurationSuggestionStatusAccepted ||
		status == models.MemoryCurationSuggestionStatusRejected ||
		status == models.MemoryCurationSuggestionStatusApplied
}

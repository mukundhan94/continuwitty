package admin

import (
	"context"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

const minimumConsolidationSuggestionGroupSize = 2

// RefreshEngramConsolidationSuggestions refreshes deterministic consolidation suggestions.
func (s *Service) RefreshEngramConsolidationSuggestions(
	ctx context.Context,
	request EngramConsolidationSuggestionRefreshRequest,
) (EngramConsolidationSuggestionRefreshResponse, error) {
	if request.MinGroupSize != nil && *request.MinGroupSize < minimumConsolidationSuggestionGroupSize {
		return EngramConsolidationSuggestionRefreshResponse{}, ErrConsolidationMinGroupSizeInvalid
	}
	refreshInput := repository.ConsolidationSuggestionRefreshInput{
		ProjectID: request.ProjectID,
	}
	if request.MinGroupSize != nil {
		refreshInput.MinGroupSize = *request.MinGroupSize
	}
	refreshed, err := s.deps.refreshConsolidationSuggestions(ctx, s.db, refreshInput)
	if err != nil {
		return EngramConsolidationSuggestionRefreshResponse{}, err
	}
	return EngramConsolidationSuggestionRefreshResponse{
		ProjectID:    refreshed.ProjectID,
		MinGroupSize: refreshed.MinGroupSize,
		SuggestedAt:  refreshed.SuggestedAt,
		UpdatedCount: refreshed.UpdatedCount,
	}, nil
}

// ListEngramConsolidationSuggestions lists persisted consolidation suggestions.
func (s *Service) ListEngramConsolidationSuggestions(
	ctx context.Context,
	request EngramConsolidationSuggestionListRequest,
) ([]models.EngramConsolidationSuggestion, error) {
	return s.deps.listConsolidationSuggestions(
		ctx,
		s.db,
		repository.ConsolidationSuggestionListInput{
			ProjectID: request.ProjectID,
			Status:    request.Status,
			Limit:     request.Limit,
			Offset:    request.Offset,
		},
	)
}

// ActionEngramConsolidationSuggestion marks a suggestion as merged/rejected.
func (s *Service) ActionEngramConsolidationSuggestion(
	ctx context.Context,
	suggestionID uuid.UUID,
	actorUserID uuid.UUID,
	request EngramConsolidationSuggestionActionRequest,
) (*models.EngramConsolidationSuggestion, error) {
	if !isConsolidationActionStatus(request.Status) {
		return nil, ErrConsolidationSuggestionActionInvalid
	}
	updated, err := s.deps.applyConsolidationSuggestionAction(
		ctx,
		s.db,
		repository.ConsolidationSuggestionActionInput{
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
		return nil, ErrConsolidationSuggestionNotFound
	}
	return updated, nil
}

func isConsolidationActionStatus(status models.ConsolidationSuggestionStatus) bool {
	return status == models.ConsolidationSuggestionStatusMerged ||
		status == models.ConsolidationSuggestionStatusRejected
}

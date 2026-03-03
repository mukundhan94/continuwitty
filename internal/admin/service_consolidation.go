package admin

import (
	"context"

	"engram/internal/models"
	"engram/internal/repository"
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

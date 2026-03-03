package admin

import (
	"context"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

// RefreshEngramContradictionAlerts refreshes deterministic contradiction alerts from active contradiction links.
func (s *Service) RefreshEngramContradictionAlerts(
	ctx context.Context,
	request EngramContradictionAlertRefreshRequest,
) (EngramContradictionAlertRefreshResponse, error) {
	refreshed, err := s.deps.refreshContradictionAlerts(
		ctx,
		s.db,
		repository.ContradictionAlertRefreshInput{
			ProjectID: request.ProjectID,
		},
	)
	if err != nil {
		return EngramContradictionAlertRefreshResponse{}, err
	}
	return EngramContradictionAlertRefreshResponse{
		ProjectID:    refreshed.ProjectID,
		DetectedAt:   refreshed.DetectedAt,
		UpdatedCount: refreshed.UpdatedCount,
	}, nil
}

// ListEngramContradictionAlerts lists persisted contradiction alerts.
func (s *Service) ListEngramContradictionAlerts(
	ctx context.Context,
	request EngramContradictionAlertListRequest,
) ([]models.EngramContradictionAlert, error) {
	return s.deps.listContradictionAlerts(
		ctx,
		s.db,
		repository.ContradictionAlertListInput{
			ProjectID: request.ProjectID,
			Status:    request.Status,
			Limit:     request.Limit,
			Offset:    request.Offset,
		},
	)
}

// ResolveEngramContradictionAlert marks one contradiction alert as resolved or dismissed.
func (s *Service) ResolveEngramContradictionAlert(
	ctx context.Context,
	alertID uuid.UUID,
	actorUserID uuid.UUID,
	request EngramContradictionAlertResolveRequest,
) (*models.EngramContradictionAlert, error) {
	if !isContradictionAlertResolveStatus(request.Status) {
		return nil, ErrContradictionAlertResolveStatusInvalid
	}
	resolved, err := s.deps.resolveContradictionAlert(
		ctx,
		s.db,
		repository.ContradictionAlertResolveInput{
			AlertID:    alertID,
			ProjectID:  request.ProjectID,
			Status:     request.Status,
			ResolvedBy: actorUserID,
		},
	)
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		return nil, ErrContradictionAlertNotFound
	}
	return resolved, nil
}

func isContradictionAlertResolveStatus(status models.ContradictionAlertStatus) bool {
	return status == models.ContradictionAlertStatusResolved ||
		status == models.ContradictionAlertStatusDismissed
}

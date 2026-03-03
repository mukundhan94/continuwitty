package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestRefreshEngramContradictionAlertsUsesRepositoryRequestObject(t *testing.T) {
	service := NewService(nil, 256, nil)
	projectID := "engram-vault"
	detectedAt := time.Date(2026, 3, 3, 20, 0, 0, 0, time.UTC)
	captured := repository.ContradictionAlertRefreshInput{}
	service.deps.resetMemoryCurationSuggestions = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.MemoryCurationSuggestionResetInput,
	) (int, error) {
		return 0, nil
	}
	service.deps.listContradictionAlerts = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.ContradictionAlertListInput,
	) ([]models.EngramContradictionAlert, error) {
		return nil, nil
	}
	service.deps.createMemoryCurationSuggestion = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.MemoryCurationSuggestionCreateInput,
	) (*models.MemoryCurationSuggestion, error) {
		return &models.MemoryCurationSuggestion{}, nil
	}
	service.deps.refreshContradictionAlerts = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ContradictionAlertRefreshInput,
	) (repository.ContradictionAlertRefreshInput, error) {
		captured = input
		return repository.ContradictionAlertRefreshInput{
			ProjectID:    input.ProjectID,
			DetectedAt:   detectedAt,
			UpdatedCount: 4,
		}, nil
	}

	response, err := service.RefreshEngramContradictionAlerts(
		context.Background(),
		EngramContradictionAlertRefreshRequest{ProjectID: &projectID},
	)
	requireNoError(t, err)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	requireEqual(t, "engram-vault", derefString(response.ProjectID))
	requireEqual(t, detectedAt, response.DetectedAt)
	requireEqual(t, 4, response.UpdatedCount)
}

func TestListEngramContradictionAlertsUsesRepositoryRequestObject(t *testing.T) {
	service := NewService(nil, 256, nil)
	projectID := "engram-vault"
	status := models.ContradictionAlertStatusOpen
	alertID := uuid.MustParse("00000000-0000-0000-0000-00000000f011")
	captured := repository.ContradictionAlertListInput{}
	service.deps.listContradictionAlerts = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ContradictionAlertListInput,
	) ([]models.EngramContradictionAlert, error) {
		captured = input
		return []models.EngramContradictionAlert{
			{
				AlertID:   alertID,
				ProjectID: "engram-vault",
				Status:    models.ContradictionAlertStatusOpen,
			},
		}, nil
	}

	listed, err := service.ListEngramContradictionAlerts(
		context.Background(),
		EngramContradictionAlertListRequest{
			ProjectID: &projectID,
			Status:    &status,
			Limit:     20,
			Offset:    4,
		},
	)
	requireNoError(t, err)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	requireEqual(t, status, *captured.Status)
	requireEqual(t, 20, captured.Limit)
	requireEqual(t, 4, captured.Offset)
	requireEqual(t, 1, len(listed))
	requireEqual(t, alertID, listed[0].AlertID)
}

func TestResolveEngramContradictionAlertUsesRepositoryRequestObject(t *testing.T) {
	service := NewService(nil, 256, nil)
	alertID := uuid.MustParse("00000000-0000-0000-0000-00000000f021")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-00000000f022")
	projectID := "engram-vault"
	captured := repository.ContradictionAlertResolveInput{}
	service.deps.resolveContradictionAlert = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ContradictionAlertResolveInput,
	) (*models.EngramContradictionAlert, error) {
		captured = input
		return &models.EngramContradictionAlert{
			AlertID:   alertID,
			ProjectID: projectID,
			Status:    models.ContradictionAlertStatusResolved,
		}, nil
	}

	resolved, err := service.ResolveEngramContradictionAlert(
		context.Background(),
		alertID,
		actorUserID,
		EngramContradictionAlertResolveRequest{
			ProjectID: &projectID,
			Status:    models.ContradictionAlertStatusResolved,
		},
	)
	requireNoError(t, err)
	requireEqual(t, alertID, captured.AlertID)
	requireEqual(t, "engram-vault", derefString(captured.ProjectID))
	requireEqual(t, actorUserID, captured.ResolvedBy)
	requireEqual(t, models.ContradictionAlertStatusResolved, captured.Status)
	if resolved == nil {
		t.Fatalf("expected resolved contradiction alert")
	}
	requireEqual(t, alertID, resolved.AlertID)
}

func TestResolveEngramContradictionAlertRejectsInvalidStatus(t *testing.T) {
	service := NewService(nil, 256, nil)
	_, err := service.ResolveEngramContradictionAlert(
		context.Background(),
		uuid.MustParse("00000000-0000-0000-0000-00000000f031"),
		uuid.MustParse("00000000-0000-0000-0000-00000000f032"),
		EngramContradictionAlertResolveRequest{
			Status: models.ContradictionAlertStatusOpen,
		},
	)
	if !errors.Is(err, ErrContradictionAlertResolveStatusInvalid) {
		t.Fatalf("expected ErrContradictionAlertResolveStatusInvalid, got %v", err)
	}
}

func TestResolveEngramContradictionAlertReturnsNotFound(t *testing.T) {
	service := NewService(nil, 256, nil)
	service.deps.resolveContradictionAlert = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.ContradictionAlertResolveInput,
	) (*models.EngramContradictionAlert, error) {
		return nil, nil
	}

	_, err := service.ResolveEngramContradictionAlert(
		context.Background(),
		uuid.MustParse("00000000-0000-0000-0000-00000000f041"),
		uuid.MustParse("00000000-0000-0000-0000-00000000f042"),
		EngramContradictionAlertResolveRequest{
			Status: models.ContradictionAlertStatusDismissed,
		},
	)
	if !errors.Is(err, ErrContradictionAlertNotFound) {
		t.Fatalf("expected ErrContradictionAlertNotFound, got %v", err)
	}
}

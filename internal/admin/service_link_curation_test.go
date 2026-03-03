package admin

import (
	"context"
	"errors"
	"testing"

	"engram/internal/graph"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestRefreshEngramLinkCurationSuggestionsCreatesManualSuggestions(t *testing.T) {
	service := NewService(nil, 256, nil)
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000d101")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000d102")
	autoArchivedLinkID := uuid.MustParse("00000000-0000-0000-0000-00000000d103")
	manualLinkID := uuid.MustParse("00000000-0000-0000-0000-00000000d104")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-00000000d105")
	projectID := "engram-vault"
	capturedCreates := []repository.MemoryCurationSuggestionCreateInput{}
	service.deps.getAdminEngram = func(
		_ context.Context,
		_ repository.Queryer,
		engramID uuid.UUID,
		includeDeleted bool,
	) (*models.AdminEngramRecord, error) {
		requireEqual(t, sourceEngramID, engramID)
		requireEqual(t, false, includeDeleted)
		return &models.AdminEngramRecord{EngramID: sourceEngramID, ProjectID: projectID}, nil
	}
	service.deps.recommendLinkHygiene = func(
		_ context.Context,
		_ repository.Queryer,
		input graph.LinkHygieneInput,
	) ([]models.EngramLinkHygieneRecommendation, error) {
		requireEqual(t, sourceEngramID, input.SourceEngramID)
		requireEqual(t, actorUserID, input.ActorUserID)
		return []models.EngramLinkHygieneRecommendation{
			{
				SourceEngramID:  sourceEngramID,
				TargetEngramID:  targetEngramID,
				SuggestedAction: "archive_stale_low_value",
				LinkIDs:         []uuid.UUID{autoArchivedLinkID},
				Severity:        "low",
			},
			{
				SourceEngramID:  sourceEngramID,
				TargetEngramID:  targetEngramID,
				SuggestedAction: "review_relation_conflict",
				LinkIDs:         []uuid.UUID{manualLinkID},
				Detail:          "conflicting relation types",
				Severity:        "high",
				Category:        models.EngramLinkHygieneCategoryConflictRelation,
				Score:           0.42,
			},
		}, nil
	}
	service.deps.listMemoryCurationSuggestions = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.MemoryCurationSuggestionListInput,
	) ([]models.MemoryCurationSuggestion, error) {
		requireEqual(t, projectID, derefString(input.ProjectID))
		requireEqual(t, models.MemoryCurationSuggestionTypeLink, derefCurationType(input.SuggestionType))
		requireEqual(t, models.MemoryCurationSuggestionStatusSuggested, derefCurationStatus(input.Status))
		return []models.MemoryCurationSuggestion{}, nil
	}
	service.deps.createMemoryCurationSuggestion = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.MemoryCurationSuggestionCreateInput,
	) (*models.MemoryCurationSuggestion, error) {
		capturedCreates = append(capturedCreates, input)
		return &models.MemoryCurationSuggestion{}, nil
	}

	response, err := service.RefreshEngramLinkCurationSuggestions(
		context.Background(),
		actorUserID,
		EngramLinkCurationSuggestionRefreshRequest{SourceEngramID: sourceEngramID},
	)
	requireNoError(t, err)
	requireEqual(t, projectID, derefString(response.ProjectID))
	requireEqual(t, sourceEngramID, response.SourceEngramID)
	requireEqual(t, 1, response.UpdatedCount)
	requireEqual(t, 1, len(capturedCreates))
	requireEqual(t, projectID, capturedCreates[0].ProjectID)
	requireEqual(t, models.MemoryCurationSuggestionTypeLink, capturedCreates[0].SuggestionType)
	requireEqual(t, 0.9, capturedCreates[0].ConfidenceScore)
	requireEqual(
		t,
		"review_relation_conflict",
		capturedCreates[0].PayloadJSON["suggested_action"],
	)
	payloadLinkID, ok := payloadStringValue(capturedCreates[0].PayloadJSON, "link_id")
	requireEqual(t, true, ok)
	requireEqual(t, manualLinkID.String(), payloadLinkID)
}

func TestRefreshEngramLinkCurationSuggestionsSkipsExistingDedupedPayload(t *testing.T) {
	service := NewService(nil, 256, nil)
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000d201")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000d202")
	linkID := uuid.MustParse("00000000-0000-0000-0000-00000000d203")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-00000000d204")
	projectID := "engram-vault"
	createCalls := 0
	service.deps.getAdminEngram = func(
		context.Context,
		repository.Queryer,
		uuid.UUID,
		bool,
	) (*models.AdminEngramRecord, error) {
		return &models.AdminEngramRecord{EngramID: sourceEngramID, ProjectID: projectID}, nil
	}
	service.deps.recommendLinkHygiene = func(
		context.Context,
		repository.Queryer,
		graph.LinkHygieneInput,
	) ([]models.EngramLinkHygieneRecommendation, error) {
		return []models.EngramLinkHygieneRecommendation{
			{
				SourceEngramID:  sourceEngramID,
				TargetEngramID:  targetEngramID,
				SuggestedAction: "review_relation_conflict",
				LinkIDs:         []uuid.UUID{linkID},
				Severity:        "high",
			},
		}, nil
	}
	service.deps.listMemoryCurationSuggestions = func(
		context.Context,
		repository.Queryer,
		repository.MemoryCurationSuggestionListInput,
	) ([]models.MemoryCurationSuggestion, error) {
		return []models.MemoryCurationSuggestion{
			{
				PayloadJSON: map[string]any{
					"link_id":          linkID.String(),
					"target_engram_id": targetEngramID.String(),
					"suggested_action": "review_relation_conflict",
				},
			},
		}, nil
	}
	service.deps.createMemoryCurationSuggestion = func(
		context.Context,
		repository.Queryer,
		repository.MemoryCurationSuggestionCreateInput,
	) (*models.MemoryCurationSuggestion, error) {
		createCalls++
		return &models.MemoryCurationSuggestion{}, nil
	}

	response, err := service.RefreshEngramLinkCurationSuggestions(
		context.Background(),
		actorUserID,
		EngramLinkCurationSuggestionRefreshRequest{SourceEngramID: sourceEngramID},
	)
	requireNoError(t, err)
	requireEqual(t, 0, response.UpdatedCount)
	requireEqual(t, 0, createCalls)
}

func TestRefreshEngramLinkCurationSuggestionsReturnsNotFoundWhenSourceMissing(t *testing.T) {
	service := NewService(nil, 256, nil)
	service.deps.getAdminEngram = func(
		context.Context,
		repository.Queryer,
		uuid.UUID,
		bool,
	) (*models.AdminEngramRecord, error) {
		return nil, nil
	}

	_, err := service.RefreshEngramLinkCurationSuggestions(
		context.Background(),
		uuid.MustParse("00000000-0000-0000-0000-00000000d301"),
		EngramLinkCurationSuggestionRefreshRequest{
			SourceEngramID: uuid.MustParse("00000000-0000-0000-0000-00000000d302"),
		},
	)
	if !errors.Is(err, ErrEngramNotFound) {
		t.Fatalf("expected ErrEngramNotFound, got %v", err)
	}
}

func derefCurationType(value *models.MemoryCurationSuggestionType) models.MemoryCurationSuggestionType {
	if value == nil {
		return ""
	}
	return *value
}

func derefCurationStatus(
	value *models.MemoryCurationSuggestionStatus,
) models.MemoryCurationSuggestionStatus {
	if value == nil {
		return ""
	}
	return *value
}

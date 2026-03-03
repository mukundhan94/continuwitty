package admin

import (
	"context"
	"fmt"
	"time"

	"engram/internal/models"
	"engram/internal/repository"
)

const memoryCurationSyncListLimit = 500

func (s *Service) syncConsolidationCurationSuggestions(
	ctx context.Context,
	projectID *string,
	suggestedAt time.Time,
) error {
	_, err := s.deps.resetMemoryCurationSuggestions(
		ctx,
		s.db,
		repository.MemoryCurationSuggestionResetInput{
			ProjectID:      projectID,
			SuggestionType: models.MemoryCurationSuggestionTypeConsolidate,
		},
	)
	if err != nil {
		return err
	}
	status := models.ConsolidationSuggestionStatusSuggested
	candidates, err := s.deps.listConsolidationSuggestions(
		ctx,
		s.db,
		repository.ConsolidationSuggestionListInput{
			ProjectID: projectID,
			Status:    &status,
			Limit:     memoryCurationSyncListLimit,
			Offset:    0,
		},
	)
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		reason := fmt.Sprintf(
			"Consolidation candidate groups %d potentially duplicate engrams",
			len(candidate.SourceEngramIDs),
		)
		_, err := s.deps.createMemoryCurationSuggestion(
			ctx,
			s.db,
			repository.MemoryCurationSuggestionCreateInput{
				ProjectID:      candidate.ProjectID,
				SuggestionType: models.MemoryCurationSuggestionTypeConsolidate,
				Reason:         reason,
				Recommendation: "Review consolidation suggestion and merge if duplicates are confirmed.",
				PayloadJSON: map[string]any{
					"consolidation_suggestion_id": candidate.SuggestionID,
					"source_engram_ids":           candidate.SourceEngramIDs,
					"consolidation_type":          candidate.ConsolidationType,
					"consolidation_reason":        candidate.Reason,
					"consolidation_hash":          candidate.ConsolidationHash,
					"confidence_score":            candidate.ConfidenceScore,
				},
				ConfidenceScore: boundedCurationConfidence(candidate.ConfidenceScore),
				SuggestedAt:     suggestedAt,
			},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) syncContradictionCurationSuggestions(
	ctx context.Context,
	projectID *string,
	suggestedAt time.Time,
) error {
	_, err := s.deps.resetMemoryCurationSuggestions(
		ctx,
		s.db,
		repository.MemoryCurationSuggestionResetInput{
			ProjectID:      projectID,
			SuggestionType: models.MemoryCurationSuggestionTypeContradiction,
		},
	)
	if err != nil {
		return err
	}
	status := models.ContradictionAlertStatusOpen
	alerts, err := s.deps.listContradictionAlerts(
		ctx,
		s.db,
		repository.ContradictionAlertListInput{
			ProjectID: projectID,
			Status:    &status,
			Limit:     memoryCurationSyncListLimit,
			Offset:    0,
		},
	)
	if err != nil {
		return err
	}
	for _, alert := range alerts {
		reason := fmt.Sprintf(
			"Contradiction detected between engrams %s and %s",
			alert.SourceEngramID,
			alert.TargetEngramID,
		)
		_, err := s.deps.createMemoryCurationSuggestion(
			ctx,
			s.db,
			repository.MemoryCurationSuggestionCreateInput{
				ProjectID:      alert.ProjectID,
				SuggestionType: models.MemoryCurationSuggestionTypeContradiction,
				Reason:         reason,
				Recommendation: "Review contradiction evidence and resolve or dismiss the alert.",
				PayloadJSON: map[string]any{
					"contradiction_alert_id": alert.AlertID,
					"source_engram_id":       alert.SourceEngramID,
					"target_engram_id":       alert.TargetEngramID,
					"reason":                 alert.Reason,
					"confidence_score":       alert.ConfidenceScore,
					"link_ids":               alert.ContradictionLinkIDs,
				},
				ConfidenceScore: boundedCurationConfidence(alert.ConfidenceScore),
				SuggestedAt:     suggestedAt,
			},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func boundedCurationConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

package admin

import (
	"context"
	"strings"
	"time"

	"engram/internal/graph"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

const curationLinkAutoArchiveAction = "archive_stale_low_value"

// RefreshEngramLinkCurationSuggestions rebuilds link curation suggestions for one source engram.
func (s *Service) RefreshEngramLinkCurationSuggestions(
	ctx context.Context,
	actorUserID uuid.UUID,
	request EngramLinkCurationSuggestionRefreshRequest,
) (EngramLinkCurationSuggestionRefreshResponse, error) {
	sourceEngram, err := s.deps.getAdminEngram(ctx, s.db, request.SourceEngramID, false)
	if err != nil {
		return EngramLinkCurationSuggestionRefreshResponse{}, err
	}
	if sourceEngram == nil {
		return EngramLinkCurationSuggestionRefreshResponse{}, ErrEngramNotFound
	}
	projectID := strings.TrimSpace(sourceEngram.ProjectID)
	if projectID == "" {
		return EngramLinkCurationSuggestionRefreshResponse{}, ErrProjectIDRequired
	}
	recommendations, err := s.deps.recommendLinkHygiene(
		ctx,
		s.db,
		graph.LinkHygieneInput{
			SourceEngramID:    request.SourceEngramID,
			ActorUserID:       actorUserID,
			IncludeArchived:   request.IncludeArchived,
			Limit:             request.Limit,
			StaleAfterDays:    request.StaleAfterDays,
			LowValueThreshold: request.LowValueThreshold,
		},
	)
	if err != nil {
		return EngramLinkCurationSuggestionRefreshResponse{}, err
	}
	suggestedAt := time.Now().UTC()
	createdCount, err := s.persistRecommendedLinkCurationSuggestions(
		ctx,
		projectID,
		request.SourceEngramID,
		recommendations,
		suggestedAt,
	)
	if err != nil {
		return EngramLinkCurationSuggestionRefreshResponse{}, err
	}
	return EngramLinkCurationSuggestionRefreshResponse{
		ProjectID:      &projectID,
		SourceEngramID: request.SourceEngramID,
		SuggestedAt:    suggestedAt,
		UpdatedCount:   createdCount,
	}, nil
}

func (s *Service) persistRecommendedLinkCurationSuggestions(
	ctx context.Context,
	projectID string,
	sourceEngramID uuid.UUID,
	recommendations []models.EngramLinkHygieneRecommendation,
	suggestedAt time.Time,
) (int, error) {
	dedupedKeys, err := s.listSuggestedLinkCurationDedupKeys(ctx, projectID)
	if err != nil {
		return 0, err
	}
	createdCount := 0
	for _, recommendation := range recommendations {
		createInput, dedupKey, shouldCreate := linkCurationSuggestionCreateInput(
			projectID,
			sourceEngramID,
			recommendation,
			suggestedAt,
		)
		if !shouldCreate {
			continue
		}
		if _, exists := dedupedKeys[dedupKey]; exists {
			continue
		}
		if _, err := s.deps.createMemoryCurationSuggestion(ctx, s.db, createInput); err != nil {
			return createdCount, err
		}
		dedupedKeys[dedupKey] = struct{}{}
		createdCount++
	}
	return createdCount, nil
}

func (s *Service) listSuggestedLinkCurationDedupKeys(
	ctx context.Context,
	projectID string,
) (map[string]struct{}, error) {
	suggestionType := models.MemoryCurationSuggestionTypeLink
	status := models.MemoryCurationSuggestionStatusSuggested
	existing, err := s.deps.listMemoryCurationSuggestions(
		ctx,
		s.db,
		repository.MemoryCurationSuggestionListInput{
			ProjectID:      &projectID,
			SuggestionType: &suggestionType,
			Status:         &status,
			Limit:          memoryCurationSyncListLimit,
			Offset:         0,
		},
	)
	if err != nil {
		return nil, err
	}
	return existingLinkCurationDedupKeys(existing), nil
}

func linkCurationSuggestionCreateInput(
	projectID string,
	sourceEngramID uuid.UUID,
	recommendation models.EngramLinkHygieneRecommendation,
	suggestedAt time.Time,
) (repository.MemoryCurationSuggestionCreateInput, string, bool) {
	if shouldSkipLinkCurationSuggestion(recommendation) {
		return repository.MemoryCurationSuggestionCreateInput{}, "", false
	}
	linkID := firstRecommendedLinkID(recommendation.LinkIDs)
	if linkID == uuid.Nil {
		return repository.MemoryCurationSuggestionCreateInput{}, "", false
	}
	dedupKey := linkCurationSuggestionDedupKey(
		linkID.String(),
		recommendation.TargetEngramID.String(),
		recommendation.SuggestedAction,
	)
	return repository.MemoryCurationSuggestionCreateInput{
		ProjectID:      projectID,
		SuggestionType: models.MemoryCurationSuggestionTypeLink,
		Reason:         linkCurationReasonFromRecommendation(recommendation),
		Recommendation: linkCurationRecommendationFromAction(recommendation),
		PayloadJSON: map[string]any{
			"source_engram_id": sourceEngramID.String(),
			"target_engram_id": recommendation.TargetEngramID.String(),
			"link_id":          linkID.String(),
			"suggested_action": recommendation.SuggestedAction,
			"category":         recommendation.Category,
			"severity":         recommendation.Severity,
			"score":            recommendation.Score,
			"detail":           recommendation.Detail,
		},
		ConfidenceScore: linkCurationConfidenceFromSeverity(recommendation.Severity),
		SuggestedAt:     suggestedAt,
	}, dedupKey, true
}

func shouldSkipLinkCurationSuggestion(recommendation models.EngramLinkHygieneRecommendation) bool {
	if len(recommendation.LinkIDs) == 0 {
		return true
	}
	action := strings.TrimSpace(strings.ToLower(recommendation.SuggestedAction))
	return action == curationLinkAutoArchiveAction
}

func firstRecommendedLinkID(linkIDs []uuid.UUID) uuid.UUID {
	for _, linkID := range linkIDs {
		if linkID != uuid.Nil {
			return linkID
		}
	}
	return uuid.Nil
}

func existingLinkCurationDedupKeys(
	suggestions []models.MemoryCurationSuggestion,
) map[string]struct{} {
	deduped := make(map[string]struct{}, len(suggestions))
	for _, suggestion := range suggestions {
		linkID, hasLinkID := payloadStringValue(suggestion.PayloadJSON, "link_id")
		action, hasAction := payloadStringValue(suggestion.PayloadJSON, "suggested_action")
		targetEngramID, hasTarget := payloadStringValue(suggestion.PayloadJSON, "target_engram_id")
		if !hasLinkID || !hasAction || !hasTarget {
			continue
		}
		deduped[linkCurationSuggestionDedupKey(linkID, targetEngramID, action)] = struct{}{}
	}
	return deduped
}

func linkCurationSuggestionDedupKey(linkID string, targetEngramID string, action string) string {
	return strings.Join(
		[]string{
			strings.TrimSpace(strings.ToLower(linkID)),
			strings.TrimSpace(strings.ToLower(targetEngramID)),
			strings.TrimSpace(strings.ToLower(action)),
		},
		"|",
	)
}

func payloadStringValue(payload map[string]any, key string) (string, bool) {
	if payload == nil {
		return "", false
	}
	rawValue, exists := payload[key]
	if !exists {
		return "", false
	}
	value, ok := rawValue.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func linkCurationReasonFromRecommendation(
	recommendation models.EngramLinkHygieneRecommendation,
) string {
	detail := strings.TrimSpace(recommendation.Detail)
	if detail != "" {
		return detail
	}
	return "Graph hygiene recommendation identified a link quality issue."
}

func linkCurationRecommendationFromAction(
	recommendation models.EngramLinkHygieneRecommendation,
) string {
	action := strings.TrimSpace(recommendation.SuggestedAction)
	if action == "" {
		return "Review link hygiene recommendation."
	}
	return "Review and apply link hygiene action: " + action + "."
}

func linkCurationConfidenceFromSeverity(severity string) float64 {
	switch strings.TrimSpace(strings.ToLower(severity)) {
	case "high":
		return 0.9
	case "medium":
		return 0.75
	default:
		return 0.6
	}
}

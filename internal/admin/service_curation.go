package admin

import (
	"context"
	"fmt"
	"strings"

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
	if request.Status == models.MemoryCurationSuggestionStatusApplied {
		if err := s.applyCurationSuggestionSideEffects(ctx, suggestionID, actorUserID, request.ProjectID); err != nil {
			return nil, err
		}
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

func (s *Service) applyCurationSuggestionSideEffects(
	ctx context.Context,
	suggestionID uuid.UUID,
	actorUserID uuid.UUID,
	requestProjectID *string,
) error {
	suggestion, err := s.loadCurationSuggestionForApply(ctx, suggestionID, requestProjectID)
	if err != nil {
		return err
	}
	projectID := curationProjectIDScope(suggestion.ProjectID, requestProjectID)
	return s.applyTypedCurationSideEffect(ctx, *suggestion, actorUserID, projectID)
}

func (s *Service) loadCurationSuggestionForApply(
	ctx context.Context,
	suggestionID uuid.UUID,
	requestProjectID *string,
) (*models.MemoryCurationSuggestion, error) {
	suggestion, err := s.deps.getMemoryCurationSuggestion(ctx, s.db, suggestionID, requestProjectID)
	if err != nil {
		return nil, err
	}
	if suggestion == nil {
		return nil, ErrMemoryCurationSuggestionNotFound
	}
	return suggestion, nil
}

func (s *Service) applyTypedCurationSideEffect(
	ctx context.Context,
	suggestion models.MemoryCurationSuggestion,
	actorUserID uuid.UUID,
	projectID *string,
) error {
	switch suggestion.SuggestionType {
	case models.MemoryCurationSuggestionTypeConsolidate:
		return applyCurationPayloadAction(
			suggestion.PayloadJSON,
			"consolidation_suggestion_id",
			func(relatedID uuid.UUID) error {
				updated, err := s.deps.applyConsolidationSuggestionAction(
					ctx,
					s.db,
					repository.ConsolidationSuggestionActionInput{
						SuggestionID: relatedID,
						ProjectID:    projectID,
						Status:       models.ConsolidationSuggestionStatusMerged,
						ActorUserID:  actorUserID,
					},
				)
				if err != nil {
					return err
				}
				if updated == nil {
					return ErrConsolidationSuggestionNotFound
				}
				return nil
			},
		)
	case models.MemoryCurationSuggestionTypeContradiction:
		return applyCurationPayloadAction(
			suggestion.PayloadJSON,
			"contradiction_alert_id",
			func(relatedID uuid.UUID) error {
				resolved, err := s.deps.resolveContradictionAlert(
					ctx,
					s.db,
					repository.ContradictionAlertResolveInput{
						AlertID:    relatedID,
						ProjectID:  projectID,
						Status:     models.ContradictionAlertStatusResolved,
						ResolvedBy: actorUserID,
					},
				)
				if err != nil {
					return err
				}
				if resolved == nil {
					return ErrContradictionAlertNotFound
				}
				return nil
			},
		)
	default:
		return nil
	}
}

func curationProjectIDScope(suggestionProjectID string, requestProjectID *string) *string {
	if requestProjectID != nil && strings.TrimSpace(*requestProjectID) != "" {
		trimmed := strings.TrimSpace(*requestProjectID)
		return &trimmed
	}
	if strings.TrimSpace(suggestionProjectID) == "" {
		return nil
	}
	projectID := suggestionProjectID
	return &projectID
}

func applyCurationPayloadAction(
	payload map[string]any,
	key string,
	action func(relatedID uuid.UUID) error,
) error {
	relatedID, err := curationSuggestionPayloadUUID(payload, key)
	if err != nil {
		return err
	}
	return action(relatedID)
}

func curationSuggestionPayloadUUID(payload map[string]any, key string) (uuid.UUID, error) {
	if payload == nil {
		return uuid.Nil, ErrMemoryCurationSuggestionPayloadInvalid
	}
	rawValue, exists := payload[key]
	if !exists {
		return uuid.Nil, ErrMemoryCurationSuggestionPayloadInvalid
	}
	value, ok := rawValue.(string)
	if !ok {
		return uuid.Nil, ErrMemoryCurationSuggestionPayloadInvalid
	}
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %s", ErrMemoryCurationSuggestionPayloadInvalid, key)
	}
	return parsed, nil
}

func isMemoryCurationActionStatus(status models.MemoryCurationSuggestionStatus) bool {
	return status == models.MemoryCurationSuggestionStatusAccepted ||
		status == models.MemoryCurationSuggestionStatusRejected ||
		status == models.MemoryCurationSuggestionStatusApplied
}

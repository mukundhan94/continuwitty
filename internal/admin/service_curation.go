package admin

import (
	"context"
	"fmt"
	"strings"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

var curationLinkArchiveActions = map[string]struct{}{
	"archive_weaker_duplicate": {},
	"archive_stale_low_value":  {},
	"review_relation_conflict": {},
}

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
	case models.MemoryCurationSuggestionTypeLink:
		return s.applyLinkCurationSuggestion(ctx, suggestion.PayloadJSON, actorUserID)
	default:
		return nil
	}
}

func (s *Service) applyLinkCurationSuggestion(
	ctx context.Context,
	payload map[string]any,
	actorUserID uuid.UUID,
) error {
	action, err := curationSuggestionPayloadString(payload, "suggested_action")
	if err != nil {
		return err
	}
	if !isLinkCurationArchiveAction(action) {
		return ErrMemoryCurationSuggestionApplyUnsupported
	}
	linkID, err := curationSuggestionPayloadUUID(payload, "link_id")
	if err != nil {
		return err
	}
	archived, err := s.deps.archiveEngramLink(
		ctx,
		s.db,
		repository.EngramLinkArchiveInput{
			LinkID:      linkID,
			ActorUserID: actorUserID,
		},
	)
	if err != nil {
		return err
	}
	if archived == nil {
		return ErrMemoryCurationSuggestionApplyUnsupported
	}
	return nil
}

func isLinkCurationArchiveAction(action string) bool {
	_, ok := curationLinkArchiveActions[strings.TrimSpace(strings.ToLower(action))]
	return ok
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
	value, err := curationSuggestionPayloadString(payload, key)
	if err != nil {
		return uuid.Nil, err
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %s", ErrMemoryCurationSuggestionPayloadInvalid, key)
	}
	return parsed, nil
}

func curationSuggestionPayloadString(payload map[string]any, key string) (string, error) {
	if payload == nil {
		return "", ErrMemoryCurationSuggestionPayloadInvalid
	}
	rawValue, exists := payload[key]
	if !exists {
		return "", ErrMemoryCurationSuggestionPayloadInvalid
	}
	value, ok := rawValue.(string)
	if !ok {
		return "", ErrMemoryCurationSuggestionPayloadInvalid
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", ErrMemoryCurationSuggestionPayloadInvalid
	}
	return trimmed, nil
}

func isMemoryCurationActionStatus(status models.MemoryCurationSuggestionStatus) bool {
	return status == models.MemoryCurationSuggestionStatusAccepted ||
		status == models.MemoryCurationSuggestionStatusRejected ||
		status == models.MemoryCurationSuggestionStatusApplied
}

package chat

import (
	"context"
	"strings"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

// ContinueSession creates a new session from an existing one and carries pinned resources.
func (service *SessionOperationsService) ContinueSession(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	payload models.ContinueSessionRequest,
) (models.ContinueSessionResponse, error) {
	session, err := service.GetSession(ctx, actorUserID, sessionID)
	if err != nil {
		return models.ContinueSessionResponse{}, err
	}
	continued, err := service.deps.createChatSession(
		ctx,
		service.db,
		repository.ChatSessionCreateInput{
			OwnerUserID: actorUserID,
			Payload:     buildContinueSessionPayload(*session, payload),
		},
	)
	if err != nil {
		return models.ContinueSessionResponse{}, err
	}
	if continued == nil {
		return models.ContinueSessionResponse{}, NewChatValidationError("Unable to continue chat session")
	}
	carriedEngramIDs, err := service.copyPinnedEngramsToSession(ctx, actorUserID, session.SessionID, continued.SessionID)
	if err != nil {
		return models.ContinueSessionResponse{}, err
	}
	if err := service.copyPinnedDocumentsToSession(ctx, actorUserID, session.SessionID, continued.SessionID); err != nil {
		return models.ContinueSessionResponse{}, err
	}
	return models.ContinueSessionResponse{
		Session:          *continued,
		CarriedEngramIDs: carriedEngramIDs,
	}, nil
}

func buildContinueSessionPayload(
	session models.ChatSessionRecord,
	request models.ContinueSessionRequest,
) models.ChatSessionCreateRequest {
	return models.ChatSessionCreateRequest{
		ProjectID:               session.ProjectID,
		Title:                   resolveContinueSessionTitle(session.Title, request.Title),
		Provider:                session.Provider,
		ModelID:                 session.ModelID,
		SystemPrompt:            session.SystemPrompt,
		VisibilityScope:         session.VisibilityScope,
		AutosaveEnabled:         session.AutosaveEnabled,
		AutosaveStrategy:        session.AutosaveStrategy,
		AutosaveIntervalMinutes: session.AutosaveIntervalMinutes,
		AutosaveMinMessages:     session.AutosaveMinMessages,
		RetentionDays:           session.RetentionDays,
		RetentionMaxSnapshots:   session.RetentionMaxSnapshots,
	}
}

func resolveContinueSessionTitle(currentTitle string, requestedTitle *string) string {
	if requestedTitle != nil {
		trimmed := strings.TrimSpace(*requestedTitle)
		if trimmed != "" {
			return trimmed
		}
	}
	return currentTitle + " (continued)"
}

func (service *SessionOperationsService) copyPinnedEngramsToSession(
	ctx context.Context,
	actorUserID uuid.UUID,
	sourceSessionID uuid.UUID,
	targetSessionID uuid.UUID,
) ([]uuid.UUID, error) {
	pinnedEngrams, err := service.deps.listPinnedEngrams(
		ctx,
		service.db,
		repository.ChatPinnedListInput{SessionID: sourceSessionID, ActorUserID: actorUserID},
	)
	if err != nil {
		return nil, err
	}
	carried := make([]uuid.UUID, 0, len(pinnedEngrams))
	for _, pinned := range pinnedEngrams {
		copied, err := service.deps.pinEngram(
			ctx,
			service.db,
			repository.ChatPinEngramInput{
				SessionID:   targetSessionID,
				EngramID:    pinned.EngramID,
				ActorUserID: actorUserID,
			},
		)
		if err != nil {
			return nil, err
		}
		if copied != nil {
			carried = append(carried, copied.EngramID)
		}
	}
	return carried, nil
}

func (service *SessionOperationsService) copyPinnedDocumentsToSession(
	ctx context.Context,
	actorUserID uuid.UUID,
	sourceSessionID uuid.UUID,
	targetSessionID uuid.UUID,
) error {
	pinnedDocuments, err := service.deps.listPinnedDocuments(
		ctx,
		service.db,
		repository.ChatPinnedListInput{SessionID: sourceSessionID, ActorUserID: actorUserID},
	)
	if err != nil {
		return err
	}
	for _, pinnedDocument := range pinnedDocuments {
		_, err := service.deps.pinDocument(
			ctx,
			service.db,
			repository.ChatPinDocumentInput{
				SessionID:   targetSessionID,
				DocumentID:  pinnedDocument.DocumentID,
				ActorUserID: actorUserID,
			},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

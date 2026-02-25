package chat

import (
	"context"
	"fmt"
	"strings"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

const defaultSessionOperationsEmbeddingDim = 256

var genericSnapshotAbstracts = map[string]struct{}{
	"":                                   {},
	"snapshot from active chat session.": {},
	"snapshot from active chat session":  {},
	"chat snapshot":                      {},
	"session snapshot":                   {},
}

type saveSessionAsEngramBuildInput struct {
	Session      models.ChatSessionRecord
	ActorUserID  uuid.UUID
	Messages     []models.ChatMessageRecord
	Payload      models.SaveSessionAsEngramRequest
	EmbeddingDim int
}

// SaveSessionAsEngram persists a chat-session snapshot as an engram.
func (service *SessionOperationsService) SaveSessionAsEngram(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	payload models.SaveSessionAsEngramRequest,
) (models.SaveSessionAsEngramResponse, error) {
	session, err := service.GetSession(ctx, actorUserID, sessionID)
	if err != nil {
		return models.SaveSessionAsEngramResponse{}, err
	}
	messages, err := service.deps.listChatMessages(
		ctx,
		service.db,
		repository.ChatMessageListInput{
			SessionID:   session.SessionID,
			ActorUserID: actorUserID,
			Limit:       500,
			Offset:      0,
		},
	)
	if err != nil {
		return models.SaveSessionAsEngramResponse{}, err
	}
	if len(messages) == 0 {
		return models.SaveSessionAsEngramResponse{}, NewChatValidationError("Cannot save an empty chat session as engram")
	}
	created, err := service.deps.createEngram(
		ctx,
		service.db,
		buildSaveSessionAsEngramInput(saveSessionAsEngramBuildInput{
			Session:      *session,
			ActorUserID:  actorUserID,
			Messages:     messages,
			Payload:      payload,
			EmbeddingDim: service.embeddingDim,
		}),
	)
	if err != nil {
		return models.SaveSessionAsEngramResponse{}, err
	}
	if created == nil {
		return models.SaveSessionAsEngramResponse{}, NewChatValidationError("Unable to save chat session as engram")
	}
	return models.SaveSessionAsEngramResponse{
		EngramID:  created.EngramID,
		SessionID: session.SessionID,
		CreatedAt: created.CreatedAt,
	}, nil
}

func buildSaveSessionAsEngramInput(input saveSessionAsEngramBuildInput) repository.CreateEngramInput {
	threadID := fmt.Sprintf("chat-session:%s", input.Session.SessionID)
	retrievalText := retrievalTextFromMessages(input.Messages, 8)
	sourceSessionID := input.Session.SessionID
	return repository.CreateEngramInput{
		Payload: models.MemoryEngramCreate{
			ProjectID:               input.Session.ProjectID,
			ThreadID:                &threadID,
			Title:                   input.Payload.Title,
			Abstract:                resolveSaveSessionAbstract(input.Payload.Abstract, input.Messages),
			DetailedSummaryMarkdown: transcriptMarkdown(input.Session, input.Messages),
			Tags:                    append([]string(nil), input.Payload.Tags...),
			Keywords:                append([]string(nil), input.Payload.Keywords...),
			VisibilityScope:         resolveSaveSessionVisibilityScope(input.Payload.VisibilityScope),
			SourceSessionID:         &sourceSessionID,
			RetrievalText:           &retrievalText,
		},
		EmbeddingDim:     input.EmbeddingDim,
		OwnerUserID:      &input.ActorUserID,
		EnrichmentOrigin: "chat.save_as_engram",
	}
}

func resolveSaveSessionAbstract(
	abstract string,
	messages []models.ChatMessageRecord,
) string {
	normalized := strings.TrimSpace(abstract)
	if !isGenericSnapshotAbstract(normalized) {
		return normalized
	}
	derived := DeriveChatSnapshotAbstract(messages, 320)
	if derived != "" {
		return derived
	}
	return normalized
}

func isGenericSnapshotAbstract(abstract string) bool {
	_, isGeneric := genericSnapshotAbstracts[strings.ToLower(strings.TrimSpace(abstract))]
	return isGeneric
}

func resolveSaveSessionVisibilityScope(scope models.VisibilityScope) string {
	if scope == "" {
		return string(models.VisibilityScopePrivate)
	}
	return string(scope)
}

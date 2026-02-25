package chat

import (
	"context"
	"errors"

	"engram/internal/models"
	"engram/internal/providers"

	"github.com/google/uuid"
)

var errChatServiceDependenciesIncomplete = errors.New("chat service dependencies incomplete")

// ChatSendResponse captures send-message outputs.
type ChatSendResponse struct {
	SessionID            uuid.UUID             `json:"session_id"`
	MessageID            uuid.UUID             `json:"message_id"`
	ReplyMessageID       uuid.UUID             `json:"reply_message_id"`
	AssistantText        string                `json:"assistant_text"`
	UsedEngramIDs        []uuid.UUID           `json:"used_engram_ids"`
	UsedDocumentChunkIDs []uuid.UUID           `json:"used_document_chunk_ids"`
	SourceReferences     []ChatSourceReference `json:"source_references"`
	DebugTrace           map[string]any        `json:"debug_trace,omitempty"`
}

// ChatServiceDependencies captures dependencies used by the chat service.
type ChatServiceDependencies struct {
	Runtime                        *ChatMessageRuntime
	ResolveProvider                func(provider models.ChatProvider) (providers.ChatProviderAdapter, error)
	RunSessionLifecycleMaintenance func(ctx context.Context, actorUserID uuid.UUID, session models.ChatSessionRecord) error
}

// ChatService coordinates runtime preparation, provider execution, and response shaping.
type ChatService struct {
	runtime                        *ChatMessageRuntime
	resolveProvider                func(provider models.ChatProvider) (providers.ChatProviderAdapter, error)
	runSessionLifecycleMaintenance func(ctx context.Context, actorUserID uuid.UUID, session models.ChatSessionRecord) error
}

// NewChatService builds a chat service with injected dependencies.
func NewChatService(dependencies ChatServiceDependencies) *ChatService {
	service := &ChatService{
		runtime:                        dependencies.Runtime,
		resolveProvider:                dependencies.ResolveProvider,
		runSessionLifecycleMaintenance: dependencies.RunSessionLifecycleMaintenance,
	}
	if service.runSessionLifecycleMaintenance == nil {
		service.runSessionLifecycleMaintenance = func(context.Context, uuid.UUID, models.ChatSessionRecord) error {
			return nil
		}
	}
	return service
}

// SendMessage executes non-streaming generation and persists assistant output.
func (service *ChatService) SendMessage(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	payload ChatMessageCreateRequest,
) (ChatSendResponse, error) {
	if err := service.validateDependencies(); err != nil {
		return ChatSendResponse{}, err
	}
	prepared, err := service.runtime.PrepareGeneration(ctx, actorUserID, sessionID, payload)
	if err != nil {
		return ChatSendResponse{}, err
	}
	adapter, err := service.resolveProvider(prepared.Session.Provider)
	if err != nil {
		return ChatSendResponse{}, err
	}
	result, err := adapter.Generate(ctx, prepared.ProviderRequest)
	if err != nil {
		return ChatSendResponse{}, MapProviderError(err)
	}
	assistantMessage, err := service.runtime.PersistAssistantReply(ctx, actorUserID, prepared, result)
	if err != nil {
		return ChatSendResponse{}, err
	}
	if err := service.runSessionLifecycleMaintenance(ctx, actorUserID, prepared.Session); err != nil {
		return ChatSendResponse{}, err
	}
	return ChatSendResponse{
		SessionID:            prepared.Session.SessionID,
		MessageID:            prepared.UserMessage.MessageID,
		ReplyMessageID:       assistantMessage.MessageID,
		AssistantText:        result.Text,
		UsedEngramIDs:        prepared.Context.UsedEngramIDs,
		UsedDocumentChunkIDs: prepared.Context.UsedDocumentChunkIDs,
		SourceReferences:     prepared.Context.SourceReferences,
		DebugTrace:           nil,
	}, nil
}

// StreamMessageEvents executes streaming generation and returns ordered stream events.
func (service *ChatService) StreamMessageEvents(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	payload ChatMessageCreateRequest,
) ([]StreamEvent, error) {
	if err := service.validateDependencies(); err != nil {
		return nil, err
	}
	prepared, err := service.runtime.PrepareGeneration(ctx, actorUserID, sessionID, payload)
	if err != nil {
		return nil, err
	}
	adapter, err := service.resolveProvider(prepared.Session.Provider)
	if err != nil {
		return nil, err
	}
	events := []StreamEvent{{Type: "meta", Payload: BuildStreamMetaPayload(prepared)}}
	streamResult, err := service.runtime.YieldStreamChunks(ctx, adapter.StreamGenerate, prepared)
	if err != nil {
		return nil, err
	}
	if streamResult.ProviderError != nil {
		events = append(events, streamProviderErrorEvent(streamResult.ProviderError))
		return events, nil
	}
	events = append(events, streamResult.Events...)

	assistantMessage, err := service.runtime.PersistAssistantReply(
		ctx,
		actorUserID,
		prepared,
		providers.ProviderGenerateResult{
			Provider:   prepared.Session.Provider,
			ModelID:    prepared.Session.ModelID,
			Text:       streamResult.FullText,
			TokenUsage: map[string]int{},
		},
	)
	if err != nil {
		if serviceError, ok := err.(*ChatServiceError); ok {
			events = append(events, streamPersistenceErrorEvent(serviceError))
			return events, nil
		}
		return nil, err
	}
	if err := service.runSessionLifecycleMaintenance(ctx, actorUserID, prepared.Session); err != nil {
		return nil, err
	}
	events = append(
		events,
		StreamEvent{
			Type:    "done",
			Payload: BuildStreamDonePayload(prepared, *assistantMessage, streamResult.FullText, nil),
		},
	)
	return events, nil
}

func (service *ChatService) validateDependencies() error {
	switch {
	case service.runtime == nil:
		return errChatServiceDependenciesIncomplete
	case service.resolveProvider == nil:
		return errChatServiceDependenciesIncomplete
	default:
		return nil
	}
}

func streamProviderErrorEvent(providerError *ChatProviderExecutionError) StreamEvent {
	return StreamEvent{
		Type: "error",
		Payload: map[string]any{
			"detail":      providerError.Detail(),
			"status_code": providerError.StatusCode(),
			"error_code":  providerError.ErrorCode(),
		},
	}
}

func streamPersistenceErrorEvent(serviceError *ChatServiceError) StreamEvent {
	return StreamEvent{
		Type: "error",
		Payload: map[string]any{
			"detail":      serviceError.Detail(),
			"status_code": serviceError.StatusCode(),
			"error_code":  "persistence_error",
		},
	}
}

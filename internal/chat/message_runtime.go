package chat

import (
	"context"
	"engram/internal/models"
	"engram/internal/providers"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	chatHistoryLoadLimit = 200
)

var errMessageRuntimeDependenciesIncomplete = errors.New("chat message runtime dependencies incomplete")

// ChatMessageCreateRequest captures inbound chat message payloads for runtime preparation.
type ChatMessageCreateRequest struct {
	ContentText string `json:"content_text"`
}

// RuntimeMessageMetadata captures optional metadata persisted with a chat message.
type RuntimeMessageMetadata struct {
	Provider      *string
	ModelID       *string
	TokenUsage    map[string]int
	UsedEngramIDs []uuid.UUID
}

// RuntimeMessageCreateInput captures runtime message creation parameters.
type RuntimeMessageCreateInput struct {
	SessionID   uuid.UUID
	ActorUserID uuid.UUID
	Role        string
	ContentText string
	Metadata    *RuntimeMessageMetadata
}

// StreamGenerateFunc captures provider stream generation callback shape.
type StreamGenerateFunc func(ctx context.Context, request providers.ProviderGenerateRequest) (<-chan string, error)

// StreamEvent captures stream event type and payload details.
type StreamEvent struct {
	Type    string
	Payload map[string]any
}

// StreamChunkResult captures emitted chunk events and final full text output.
type StreamChunkResult struct {
	Events        []StreamEvent
	FullText      string
	ProviderError *ChatProviderExecutionError
}

// PreparedGeneration captures runtime preparation artifacts before provider execution.
type PreparedGeneration struct {
	Session               models.ChatSessionRecord
	UserMessage           models.ChatMessageRecord
	Context               AssembledChatContext
	ProviderRequest       providers.ProviderGenerateRequest
	PrepareDurationMS     float64
	ContextDurationMS     float64
	HistoryLoadDurationMS float64
}

// ChatMessageRuntimeDependencies captures runtime configuration for chat generation.
type ChatMessageRuntimeDependencies struct {
	EmbeddingDim              int
	ChatDebugEnabled          bool
	ChatDebugIncludeRawOutput bool
	GetSession                func(ctx context.Context, actorUserID uuid.UUID, sessionID uuid.UUID) (*models.ChatSessionRecord, error)
	CreateChatMessage         func(ctx context.Context, input RuntimeMessageCreateInput) (*models.ChatMessageRecord, error)
	ListChatMessages          func(ctx context.Context, sessionID uuid.UUID, actorUserID uuid.UUID, limit int, offset int) ([]models.ChatMessageRecord, error)
	AssembleChatContext       func(ctx context.Context, request ChatContextRequest) (AssembledChatContext, error)
}

// ChatMessageRuntime coordinates chat generation and stream payload shaping.
type ChatMessageRuntime struct {
	embeddingDim              int
	chatDebugEnabled          bool
	chatDebugIncludeRawOutput bool
	getSession                func(ctx context.Context, actorUserID uuid.UUID, sessionID uuid.UUID) (*models.ChatSessionRecord, error)
	createChatMessage         func(ctx context.Context, input RuntimeMessageCreateInput) (*models.ChatMessageRecord, error)
	listChatMessages          func(ctx context.Context, sessionID uuid.UUID, actorUserID uuid.UUID, limit int, offset int) ([]models.ChatMessageRecord, error)
	assembleChatContext       func(ctx context.Context, request ChatContextRequest) (AssembledChatContext, error)
}

// NewChatMessageRuntime builds a message runtime from dependency configuration.
func NewChatMessageRuntime(dependencies ChatMessageRuntimeDependencies) *ChatMessageRuntime {
	return &ChatMessageRuntime{
		embeddingDim:              dependencies.EmbeddingDim,
		chatDebugEnabled:          dependencies.ChatDebugEnabled,
		chatDebugIncludeRawOutput: dependencies.ChatDebugIncludeRawOutput,
		getSession:                dependencies.GetSession,
		createChatMessage:         dependencies.CreateChatMessage,
		listChatMessages:          dependencies.ListChatMessages,
		assembleChatContext:       dependencies.AssembleChatContext,
	}
}

// PrepareGeneration creates user message and provider request inputs for generation.
func (runtime *ChatMessageRuntime) PrepareGeneration(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	payload ChatMessageCreateRequest,
) (PreparedGeneration, error) {
	if strings.TrimSpace(payload.ContentText) == "" {
		return PreparedGeneration{}, NewChatValidationError("Message content cannot be empty")
	}
	if err := runtime.validatePrepareDependencies(); err != nil {
		return PreparedGeneration{}, err
	}
	prepareStartedAt := time.Now()
	session, userMessage, err := runtime.prepareSessionAndUserMessage(
		ctx,
		actorUserID,
		sessionID,
		payload.ContentText,
	)
	if err != nil {
		return PreparedGeneration{}, err
	}

	assembledContext, contextDurationMS, err := runtime.prepareChatContext(
		ctx,
		actorUserID,
		session,
		payload.ContentText,
	)
	if err != nil {
		return PreparedGeneration{}, err
	}

	history, historyLoadDurationMS, err := runtime.loadMessageHistory(ctx, session.SessionID, actorUserID)
	if err != nil {
		return PreparedGeneration{}, err
	}

	return PreparedGeneration{
		Session:               session,
		UserMessage:           userMessage,
		Context:               assembledContext,
		ProviderRequest:       buildPreparedProviderRequest(session, history, assembledContext.ContextMarkdown),
		PrepareDurationMS:     durationMS(prepareStartedAt),
		ContextDurationMS:     contextDurationMS,
		HistoryLoadDurationMS: historyLoadDurationMS,
	}, nil
}

// PersistAssistantReply writes the assistant message with provider and token metadata.
func (runtime *ChatMessageRuntime) PersistAssistantReply(
	ctx context.Context,
	actorUserID uuid.UUID,
	prepared PreparedGeneration,
	result providers.ProviderGenerateResult,
) (*models.ChatMessageRecord, error) {
	if runtime.createChatMessage == nil {
		return nil, errMessageRuntimeDependenciesIncomplete
	}
	provider := string(prepared.Session.Provider)
	if result.Provider != "" {
		provider = string(result.Provider)
	}
	modelID := prepared.Session.ModelID
	if strings.TrimSpace(result.ModelID) != "" {
		modelID = strings.TrimSpace(result.ModelID)
	}
	record, err := runtime.createChatMessage(
		ctx,
		RuntimeMessageCreateInput{
			SessionID:   prepared.Session.SessionID,
			ActorUserID: actorUserID,
			Role:        "assistant",
			ContentText: result.Text,
			Metadata: &RuntimeMessageMetadata{
				Provider:      &provider,
				ModelID:       &modelID,
				TokenUsage:    cloneTokenUsageIntMap(result.TokenUsage),
				UsedEngramIDs: append([]uuid.UUID{}, prepared.Context.UsedEngramIDs...),
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, NewChatSessionNotFoundError("")
	}
	return record, nil
}

// YieldStreamChunks streams provider tokens into chunk events and aggregates full text.
func (runtime *ChatMessageRuntime) YieldStreamChunks(
	ctx context.Context,
	streamGenerate StreamGenerateFunc,
	prepared PreparedGeneration,
) (StreamChunkResult, error) {
	if streamGenerate == nil {
		return StreamChunkResult{}, errMessageRuntimeDependenciesIncomplete
	}
	stream, err := streamGenerate(ctx, prepared.ProviderRequest)
	if err != nil {
		mapped := MapProviderError(err)
		if providerErr, ok := mapped.(*ChatProviderExecutionError); ok {
			return StreamChunkResult{ProviderError: providerErr}, nil
		}
		return StreamChunkResult{}, mapped
	}
	return collectStreamChunks(stream), nil
}

// BuildStreamMetaPayload returns stream metadata sent ahead of streamed chunks.
func BuildStreamMetaPayload(prepared PreparedGeneration) map[string]any {
	return map[string]any{
		"session_id":              prepared.Session.SessionID,
		"message_id":              prepared.UserMessage.MessageID,
		"used_engram_ids":         prepared.Context.UsedEngramIDs,
		"used_document_chunk_ids": prepared.Context.UsedDocumentChunkIDs,
		"source_references":       prepared.Context.SourceReferences,
	}
}

// BuildStreamMetaPayload returns stream metadata sent ahead of streamed chunks.
func (runtime *ChatMessageRuntime) BuildStreamMetaPayload(prepared PreparedGeneration) map[string]any {
	_ = runtime
	return BuildStreamMetaPayload(prepared)
}

// BuildStreamDonePayload returns final stream payload after completion persistence.
func BuildStreamDonePayload(
	prepared PreparedGeneration,
	assistantMessage models.ChatMessageRecord,
	fullText string,
	debugTrace map[string]any,
) map[string]any {
	payload := map[string]any{
		"session_id":              prepared.Session.SessionID,
		"message_id":              prepared.UserMessage.MessageID,
		"reply_message_id":        assistantMessage.MessageID,
		"assistant_text":          fullText,
		"used_engram_ids":         prepared.Context.UsedEngramIDs,
		"used_document_chunk_ids": prepared.Context.UsedDocumentChunkIDs,
		"source_references":       prepared.Context.SourceReferences,
		"debug_trace":             nil,
	}
	if debugTrace != nil {
		payload["debug_trace"] = debugTrace
	}
	return payload
}

// BuildStreamDonePayload returns final stream payload after completion persistence.
func (runtime *ChatMessageRuntime) BuildStreamDonePayload(
	prepared PreparedGeneration,
	assistantMessage models.ChatMessageRecord,
	fullText string,
	debugTrace map[string]any,
) map[string]any {
	_ = runtime
	return BuildStreamDonePayload(prepared, assistantMessage, fullText, debugTrace)
}

func (runtime *ChatMessageRuntime) validatePrepareDependencies() error {
	switch {
	case runtime.getSession == nil:
		return errMessageRuntimeDependenciesIncomplete
	case runtime.createChatMessage == nil:
		return errMessageRuntimeDependenciesIncomplete
	case runtime.listChatMessages == nil:
		return errMessageRuntimeDependenciesIncomplete
	case runtime.assembleChatContext == nil:
		return errMessageRuntimeDependenciesIncomplete
	default:
		return nil
	}
}

func (runtime *ChatMessageRuntime) prepareSessionAndUserMessage(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	contentText string,
) (models.ChatSessionRecord, models.ChatMessageRecord, error) {
	session, err := runtime.getSession(ctx, actorUserID, sessionID)
	if err != nil {
		return models.ChatSessionRecord{}, models.ChatMessageRecord{}, err
	}
	if session == nil {
		return models.ChatSessionRecord{}, models.ChatMessageRecord{}, NewChatSessionNotFoundError("")
	}
	userMessage, err := runtime.createChatMessage(
		ctx,
		RuntimeMessageCreateInput{
			SessionID:   session.SessionID,
			ActorUserID: actorUserID,
			Role:        "user",
			ContentText: contentText,
		},
	)
	if err != nil {
		return models.ChatSessionRecord{}, models.ChatMessageRecord{}, err
	}
	if userMessage == nil {
		return models.ChatSessionRecord{}, models.ChatMessageRecord{}, NewChatSessionNotFoundError("")
	}
	return *session, *userMessage, nil
}

func (runtime *ChatMessageRuntime) prepareChatContext(
	ctx context.Context,
	actorUserID uuid.UUID,
	session models.ChatSessionRecord,
	contentText string,
) (AssembledChatContext, float64, error) {
	contextStartedAt := time.Now()
	assembledContext, err := runtime.assembleChatContext(
		ctx,
		ChatContextRequest{
			Session:      session,
			ActorUserID:  actorUserID,
			UserQuery:    contentText,
			EmbeddingDim: runtime.embeddingDim,
		},
	)
	if err != nil {
		return AssembledChatContext{}, 0, err
	}
	return assembledContext, durationMS(contextStartedAt), nil
}

func (runtime *ChatMessageRuntime) loadMessageHistory(
	ctx context.Context,
	sessionID uuid.UUID,
	actorUserID uuid.UUID,
) ([]models.ChatMessageRecord, float64, error) {
	historyStartedAt := time.Now()
	history, err := runtime.listChatMessages(ctx, sessionID, actorUserID, chatHistoryLoadLimit, 0)
	if err != nil {
		return nil, 0, err
	}
	return history, durationMS(historyStartedAt), nil
}

func buildPreparedProviderRequest(
	session models.ChatSessionRecord,
	history []models.ChatMessageRecord,
	contextMarkdown string,
) providers.ProviderGenerateRequest {
	return providers.ProviderGenerateRequest{
		ModelID:      session.ModelID,
		Messages:     HistoryAsProviderMessages(history, defaultChatHistoryLimit),
		SystemPrompt: BuildSystemPrompt(session.SystemPrompt, contextMarkdown),
	}
}

func durationMS(startedAt time.Time) float64 {
	return float64(time.Since(startedAt)) / float64(time.Millisecond)
}

func cloneTokenUsageIntMap(tokenUsage map[string]int) map[string]int {
	if tokenUsage == nil {
		return map[string]int{}
	}
	cloned := make(map[string]int, len(tokenUsage))
	for key, value := range tokenUsage {
		cloned[key] = value
	}
	return cloned
}

func collectStreamChunks(stream <-chan string) StreamChunkResult {
	events := make([]StreamEvent, 0)
	builder := strings.Builder{}
	for part := range stream {
		if part == "" {
			continue
		}
		builder.WriteString(part)
		events = append(events, StreamEvent{Type: "chunk", Payload: map[string]any{"text": part}})
	}
	return StreamChunkResult{
		Events:        events,
		FullText:      builder.String(),
		ProviderError: nil,
	}
}

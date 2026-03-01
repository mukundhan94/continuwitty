package chat

import (
	"context"
	"errors"
	"time"

	"engram/internal/models"
	"engram/internal/providers"

	"github.com/google/uuid"
)

const (
	chatOperationSend   = "send"
	chatOperationStream = "stream"

	chatStreamOutcomeCompleted       = "completed"
	chatStreamOutcomeProviderError   = "provider_error"
	chatStreamOutcomePersistenceFail = "persistence_error"

	chatTracePrepareStart  = "prepare_start"
	chatTracePrepareDone   = "prepare_done"
	chatTracePrepareFailed = "prepare_failed"
	chatTraceProviderTry   = "provider_attempt"
	chatTraceProviderOK    = "provider_success"
	chatTraceProviderFail  = "provider_failure"
	chatTraceProviderOpen  = "provider_circuit_open"
	chatTracePersistOK     = "persist_success"
	chatTracePersistFail   = "persist_failed"
	chatTraceLifecycleOK   = "lifecycle_success"
	chatTraceLifecycleFail = "lifecycle_failed"
)

var errChatServiceDependenciesIncomplete = errors.New("chat service dependencies incomplete")

// ChatSendResponse captures send-message outputs.
type ChatSendResponse struct {
	SessionID            uuid.UUID             `json:"session_id"`
	MessageID            uuid.UUID             `json:"message_id"`
	ReplyMessageID       uuid.UUID             `json:"reply_message_id"`
	AssistantText        string                `json:"assistant_text"`
	PromptPolicyVersion  string                `json:"prompt_policy_version,omitempty"`
	UsedEngramIDs        []uuid.UUID           `json:"used_engram_ids"`
	UsedEngramLinkIDs    []uuid.UUID           `json:"used_engram_link_ids"`
	EngramTracePaths     []EngramTracePath     `json:"engram_trace_paths"`
	UsedDocumentChunkIDs []uuid.UUID           `json:"used_document_chunk_ids"`
	SourceReferences     []ChatSourceReference `json:"source_references"`
	DebugTrace           map[string]any        `json:"debug_trace,omitempty"`
}

// ChatServiceDependencies captures dependencies used by the chat service.
type ChatServiceDependencies struct {
	Runtime                        *ChatMessageRuntime
	ResolveProvider                func(provider models.ChatProvider) (providers.ChatProviderAdapter, error)
	RunSessionLifecycleMaintenance func(ctx context.Context, actorUserID uuid.UUID, session models.ChatSessionRecord) error
	ProviderFallback               ProviderFallbackStrategy
	CircuitPolicy                  ProviderCircuitPolicy
	Observability                  ObservabilityRecorder
	ResolveTraceID                 func(ctx context.Context) string
	NowUTC                         func() time.Time
}

// ChatService coordinates runtime preparation, provider execution, and response shaping.
type ChatService struct {
	runtime                        *ChatMessageRuntime
	resolveProvider                func(provider models.ChatProvider) (providers.ChatProviderAdapter, error)
	runSessionLifecycleMaintenance func(ctx context.Context, actorUserID uuid.UUID, session models.ChatSessionRecord) error
	providerFallback               ProviderFallbackStrategy
	circuitPolicy                  ProviderCircuitPolicy
	observability                  ObservabilityRecorder
	resolveTraceID                 func(ctx context.Context) string
	nowUTC                         func() time.Time
}

// NewChatService builds a chat service with injected dependencies.
func NewChatService(dependencies ChatServiceDependencies) *ChatService {
	service := &ChatService{
		runtime:                        dependencies.Runtime,
		resolveProvider:                dependencies.ResolveProvider,
		runSessionLifecycleMaintenance: dependencies.RunSessionLifecycleMaintenance,
		providerFallback:               dependencies.ProviderFallback,
		circuitPolicy:                  dependencies.CircuitPolicy,
		observability:                  dependencies.Observability,
		resolveTraceID:                 dependencies.ResolveTraceID,
		nowUTC:                         dependencies.NowUTC,
	}
	if service.runSessionLifecycleMaintenance == nil {
		service.runSessionLifecycleMaintenance = func(context.Context, uuid.UUID, models.ChatSessionRecord) error {
			return nil
		}
	}
	if service.providerFallback == nil {
		service.providerFallback = noopProviderFallbackStrategy{}
	}
	if service.circuitPolicy == nil {
		service.circuitPolicy = noopProviderCircuitPolicy{}
	}
	if service.observability == nil {
		service.observability = noopObservabilityRecorder{}
	}
	if service.resolveTraceID == nil {
		service.resolveTraceID = func(context.Context) string { return "" }
	}
	if service.nowUTC == nil {
		service.nowUTC = func() time.Time { return time.Now().UTC() }
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
	traceID := service.resolveTraceID(ctx)
	prepared, err := service.prepareGenerationWithTrace(
		ctx,
		actorUserID,
		sessionID,
		payload,
		traceID,
		chatOperationSend,
	)
	if err != nil {
		return ChatSendResponse{}, err
	}
	result, providerCandidate, err := service.generateWithFallback(ctx, traceID, prepared)
	if err != nil {
		return ChatSendResponse{}, err
	}
	assistantMessage, err := service.persistAssistantReplyWithTrace(
		ctx,
		actorUserID,
		prepared,
		result,
		traceID,
		chatOperationSend,
		providerCandidate.Provider,
	)
	if err != nil {
		return ChatSendResponse{}, err
	}
	if err := service.runLifecycleWithTrace(
		ctx,
		actorUserID,
		prepared,
		traceID,
		chatOperationSend,
		providerCandidate.Provider,
	); err != nil {
		return ChatSendResponse{}, err
	}
	return buildChatSendResponse(prepared, *assistantMessage, result.Text), nil
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
	traceID := service.resolveTraceID(ctx)
	prepared, err := service.prepareGenerationWithTrace(
		ctx,
		actorUserID,
		sessionID,
		payload,
		traceID,
		chatOperationStream,
	)
	if err != nil {
		return nil, err
	}
	events := []StreamEvent{{Type: "meta", Payload: BuildStreamMetaPayload(prepared)}}

	streamStartedAt := service.nowUTC()
	streamResult, providerCandidate, err := service.streamWithFallback(ctx, traceID, prepared)
	if err != nil {
		return nil, err
	}
	if streamResult.ProviderError != nil {
		events = append(events, streamProviderErrorEvent(streamResult.ProviderError))
		service.recordStreamHealth(
			StreamHealthSample{
				Provider:   providerCandidate.Provider,
				Operation:  chatOperationStream,
				Outcome:    chatStreamOutcomeProviderError,
				ErrorCode:  streamResult.ProviderError.ErrorCode(),
				ChunkCount: 0,
				Duration:   service.nowUTC().Sub(streamStartedAt),
			},
		)
		return events, nil
	}
	events = append(events, streamResult.Events...)

	assistantResult := providers.ProviderGenerateResult{
		Provider:   providerCandidate.Provider,
		ModelID:    providerCandidate.ModelID,
		Text:       streamResult.FullText,
		TokenUsage: map[string]int{},
	}
	assistantMessage, err := service.persistAssistantReplyWithTrace(
		ctx,
		actorUserID,
		prepared,
		assistantResult,
		traceID,
		chatOperationStream,
		providerCandidate.Provider,
	)
	if err != nil {
		if serviceError, ok := err.(*ChatServiceError); ok {
			events = append(events, streamPersistenceErrorEvent(serviceError))
			service.recordStreamHealth(
				StreamHealthSample{
					Provider:   providerCandidate.Provider,
					Operation:  chatOperationStream,
					Outcome:    chatStreamOutcomePersistenceFail,
					ChunkCount: streamChunkCount(streamResult.Events),
					Duration:   service.nowUTC().Sub(streamStartedAt),
				},
			)
			return events, nil
		}
		return nil, err
	}
	if err := service.runLifecycleWithTrace(
		ctx,
		actorUserID,
		prepared,
		traceID,
		chatOperationStream,
		providerCandidate.Provider,
	); err != nil {
		return nil, err
	}
	events = append(
		events,
		StreamEvent{
			Type:    "done",
			Payload: BuildStreamDonePayload(prepared, *assistantMessage, streamResult.FullText, nil),
		},
	)
	service.recordStreamHealth(
		StreamHealthSample{
			Provider:   providerCandidate.Provider,
			Operation:  chatOperationStream,
			Outcome:    chatStreamOutcomeCompleted,
			ChunkCount: streamChunkCount(streamResult.Events),
			Duration:   service.nowUTC().Sub(streamStartedAt),
		},
	)
	return events, nil
}

func (service *ChatService) prepareGenerationWithTrace(
	ctx context.Context,
	actorUserID uuid.UUID,
	sessionID uuid.UUID,
	payload ChatMessageCreateRequest,
	traceID string,
	operation string,
) (PreparedGeneration, error) {
	startedAt := service.nowUTC()
	service.recordLifecycleTrace(traceID, operation, chatTracePrepareStart, nil, "", "", 0)
	prepared, err := service.runtime.PrepareGeneration(ctx, actorUserID, sessionID, payload)
	if err != nil {
		service.recordLifecycleTrace(
			traceID,
			operation,
			chatTracePrepareFailed,
			nil,
			"",
			"",
			service.nowUTC().Sub(startedAt),
		)
		return PreparedGeneration{}, err
	}
	service.recordLifecycleTrace(
		traceID,
		operation,
		chatTracePrepareDone,
		&prepared,
		prepared.Session.Provider,
		"",
		service.nowUTC().Sub(startedAt),
	)
	return prepared, nil
}

func (service *ChatService) persistAssistantReplyWithTrace(
	ctx context.Context,
	actorUserID uuid.UUID,
	prepared PreparedGeneration,
	result providers.ProviderGenerateResult,
	traceID string,
	operation string,
	provider models.ChatProvider,
) (*models.ChatMessageRecord, error) {
	assistantMessage, err := service.runtime.PersistAssistantReply(ctx, actorUserID, prepared, result)
	if err != nil {
		service.recordLifecycleTrace(traceID, operation, chatTracePersistFail, &prepared, provider, "", 0)
		return nil, err
	}
	service.recordLifecycleTrace(traceID, operation, chatTracePersistOK, &prepared, provider, "", 0)
	return assistantMessage, nil
}

func (service *ChatService) runLifecycleWithTrace(
	ctx context.Context,
	actorUserID uuid.UUID,
	prepared PreparedGeneration,
	traceID string,
	operation string,
	provider models.ChatProvider,
) error {
	if err := service.runSessionLifecycleMaintenance(ctx, actorUserID, prepared.Session); err != nil {
		service.recordLifecycleTrace(traceID, operation, chatTraceLifecycleFail, &prepared, provider, "", 0)
		return err
	}
	service.recordLifecycleTrace(traceID, operation, chatTraceLifecycleOK, &prepared, provider, "", 0)
	return nil
}

func buildChatSendResponse(
	prepared PreparedGeneration,
	assistantMessage models.ChatMessageRecord,
	assistantText string,
) ChatSendResponse {
	return ChatSendResponse{
		SessionID:            prepared.Session.SessionID,
		MessageID:            prepared.UserMessage.MessageID,
		ReplyMessageID:       assistantMessage.MessageID,
		AssistantText:        assistantText,
		PromptPolicyVersion:  prepared.PromptPolicyVersion,
		UsedEngramIDs:        prepared.Context.UsedEngramIDs,
		UsedEngramLinkIDs:    prepared.Context.UsedEngramLinkIDs,
		EngramTracePaths:     prepared.Context.EngramTracePaths,
		UsedDocumentChunkIDs: prepared.Context.UsedDocumentChunkIDs,
		SourceReferences:     prepared.Context.SourceReferences,
		DebugTrace:           nil,
	}
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

func (service *ChatService) generateWithFallback(
	ctx context.Context,
	traceID string,
	prepared PreparedGeneration,
) (providers.ProviderGenerateResult, ProviderFallbackCandidate, error) {
	candidates := service.providerFallback.Candidates(prepared.Session)
	for index, candidate := range candidates {
		if !service.circuitPolicy.Allow(candidate.Provider) {
			providerErr := NewChatProviderExecutionError(
				"provider circuit is open",
				503,
				"provider_circuit_open",
			)
			service.recordProviderFailure(candidate.Provider, chatOperationSend, providerErr.ErrorCode())
			service.recordLifecycleTrace(
				traceID,
				chatOperationSend,
				chatTraceProviderOpen,
				&prepared,
				candidate.Provider,
				providerErr.ErrorCode(),
				0,
			)
			if index == len(candidates)-1 {
				return providers.ProviderGenerateResult{}, candidate, providerErr
			}
			continue
		}
		adapter, err := service.resolveProvider(candidate.Provider)
		if err != nil {
			return providers.ProviderGenerateResult{}, candidate, err
		}
		service.recordLifecycleTrace(
			traceID,
			chatOperationSend,
			chatTraceProviderTry,
			&prepared,
			candidate.Provider,
			"",
			0,
		)
		result, err := adapter.Generate(ctx, service.providerRequestWithCandidate(prepared, candidate))
		if err == nil {
			service.circuitPolicy.RecordResult(candidate.Provider, "")
			result.Provider = candidate.Provider
			result.ModelID = candidate.ModelID
			service.recordLifecycleTrace(
				traceID,
				chatOperationSend,
				chatTraceProviderOK,
				&prepared,
				candidate.Provider,
				"",
				0,
			)
			return result, candidate, nil
		}
		mapped := MapProviderError(err)
		providerErr, ok := mapped.(*ChatProviderExecutionError)
		if !ok {
			return providers.ProviderGenerateResult{}, candidate, mapped
		}
		service.circuitPolicy.RecordResult(candidate.Provider, providerErr.ErrorCode())
		service.recordProviderFailure(candidate.Provider, chatOperationSend, providerErr.ErrorCode())
		service.recordLifecycleTrace(
			traceID,
			chatOperationSend,
			chatTraceProviderFail,
			&prepared,
			candidate.Provider,
			providerErr.ErrorCode(),
			0,
		)
		if !isTransientProviderErrorCode(providerErr.ErrorCode()) || index == len(candidates)-1 {
			return providers.ProviderGenerateResult{}, candidate, providerErr
		}
	}
	return providers.ProviderGenerateResult{}, ProviderFallbackCandidate{}, NewChatProviderExecutionError(
		"provider execution failed",
		502,
		"provider_error",
	)
}

func (service *ChatService) streamWithFallback(
	ctx context.Context,
	traceID string,
	prepared PreparedGeneration,
) (StreamChunkResult, ProviderFallbackCandidate, error) {
	candidates := service.providerFallback.Candidates(prepared.Session)
	for index, candidate := range candidates {
		if !service.circuitPolicy.Allow(candidate.Provider) {
			providerErr := NewChatProviderExecutionError(
				"provider circuit is open",
				503,
				"provider_circuit_open",
			)
			service.recordProviderFailure(candidate.Provider, chatOperationStream, providerErr.ErrorCode())
			service.recordLifecycleTrace(
				traceID,
				chatOperationStream,
				chatTraceProviderOpen,
				&prepared,
				candidate.Provider,
				providerErr.ErrorCode(),
				0,
			)
			if index == len(candidates)-1 {
				return StreamChunkResult{ProviderError: providerErr}, candidate, nil
			}
			continue
		}
		adapter, err := service.resolveProvider(candidate.Provider)
		if err != nil {
			return StreamChunkResult{}, candidate, err
		}
		service.recordLifecycleTrace(
			traceID,
			chatOperationStream,
			chatTraceProviderTry,
			&prepared,
			candidate.Provider,
			"",
			0,
		)
		candidatePrepared := prepared
		candidatePrepared.ProviderRequest = service.providerRequestWithCandidate(prepared, candidate)
		streamResult, err := service.runtime.YieldStreamChunks(
			ctx,
			adapter.StreamGenerate,
			candidatePrepared,
		)
		if err != nil {
			return StreamChunkResult{}, candidate, err
		}
		if streamResult.ProviderError == nil {
			service.circuitPolicy.RecordResult(candidate.Provider, "")
			service.recordLifecycleTrace(
				traceID,
				chatOperationStream,
				chatTraceProviderOK,
				&prepared,
				candidate.Provider,
				"",
				0,
			)
			return streamResult, candidate, nil
		}
		service.circuitPolicy.RecordResult(candidate.Provider, streamResult.ProviderError.ErrorCode())
		service.recordProviderFailure(candidate.Provider, chatOperationStream, streamResult.ProviderError.ErrorCode())
		service.recordLifecycleTrace(
			traceID,
			chatOperationStream,
			chatTraceProviderFail,
			&prepared,
			candidate.Provider,
			streamResult.ProviderError.ErrorCode(),
			0,
		)
		if !isTransientProviderErrorCode(streamResult.ProviderError.ErrorCode()) || index == len(candidates)-1 {
			return streamResult, candidate, nil
		}
	}
	return StreamChunkResult{}, ProviderFallbackCandidate{}, NewChatProviderExecutionError(
		"provider execution failed",
		502,
		"provider_error",
	)
}

func (service *ChatService) providerRequestWithCandidate(
	prepared PreparedGeneration,
	candidate ProviderFallbackCandidate,
) providers.ProviderGenerateRequest {
	request := prepared.ProviderRequest
	request.ModelID = candidate.ModelID
	return request
}

func (service *ChatService) recordProviderFailure(
	provider models.ChatProvider,
	operation string,
	errorCode string,
) {
	service.observability.RecordProviderFailure(
		ProviderFailureSample{
			Provider:  provider,
			Operation: operation,
			ErrorCode: errorCode,
		},
	)
}

func (service *ChatService) recordStreamHealth(sample StreamHealthSample) {
	service.observability.RecordStreamHealth(sample)
}

func (service *ChatService) recordLifecycleTrace(
	traceID string,
	operation string,
	stage string,
	prepared *PreparedGeneration,
	provider models.ChatProvider,
	errorCode string,
	duration time.Duration,
) {
	sessionID := uuid.Nil
	messageID := uuid.Nil
	if prepared != nil {
		sessionID = prepared.Session.SessionID
		messageID = prepared.UserMessage.MessageID
	}
	service.observability.RecordLifecycleTrace(
		LifecycleTraceSample{
			TraceID:   traceID,
			Operation: operation,
			Stage:     stage,
			Provider:  provider,
			SessionID: sessionID,
			MessageID: messageID,
			ErrorCode: errorCode,
			Duration:  duration,
		},
	)
}

func streamChunkCount(events []StreamEvent) int {
	count := 0
	for _, event := range events {
		if event.Type == "chunk" {
			count++
		}
	}
	return count
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

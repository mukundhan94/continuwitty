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
	chatTraceReinforceOK   = "link_reinforce_success"
	chatTraceReinforceFail = "link_reinforce_failed"
	chatTraceAccessOK      = "engram_access_record_success"
	chatTraceAccessFail    = "engram_access_record_failed"
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
	CWPlanApplied        *CWQueryPlan          `json:"cw_plan_applied,omitempty"`
	UsedEngramIDs        []uuid.UUID           `json:"used_engram_ids"`
	UsedEngramLinkIDs    []uuid.UUID           `json:"used_engram_link_ids"`
	EngramTracePaths     []EngramTracePath     `json:"engram_trace_paths"`
	UsedDocumentChunkIDs []uuid.UUID           `json:"used_document_chunk_ids"`
	SourceReferences     []ChatSourceReference `json:"source_references"`
	RetrievalAudit       *ChatRetrievalAudit   `json:"retrieval_audit,omitempty"`
	DebugTrace           map[string]any        `json:"debug_trace,omitempty"`
}

// ChatServiceDependencies captures dependencies used by the chat service.
type ChatServiceDependencies struct {
	Runtime                        *ChatMessageRuntime
	ResolveProvider                func(provider models.ChatProvider) (providers.ChatProviderAdapter, error)
	ReinforceEngramLinks           func(ctx context.Context, actorUserID uuid.UUID, linkIDs []uuid.UUID) error
	RecordEngramAccess             func(ctx context.Context, sessionID uuid.UUID, accessSource string, engramIDs []uuid.UUID) error
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
	reinforceEngramLinks           func(ctx context.Context, actorUserID uuid.UUID, linkIDs []uuid.UUID) error
	recordEngramAccess             func(ctx context.Context, sessionID uuid.UUID, accessSource string, engramIDs []uuid.UUID) error
	runSessionLifecycleMaintenance func(ctx context.Context, actorUserID uuid.UUID, session models.ChatSessionRecord) error
	providerFallback               ProviderFallbackStrategy
	circuitPolicy                  ProviderCircuitPolicy
	observability                  ObservabilityRecorder
	resolveTraceID                 func(ctx context.Context) string
	nowUTC                         func() time.Time
}

type reinforceTraceInput struct {
	ActorUserID uuid.UUID
	Prepared    PreparedGeneration
	TraceID     string
	Operation   string
	Provider    models.ChatProvider
}

type accessTraceInput struct {
	Prepared  PreparedGeneration
	TraceID   string
	Operation string
	Provider  models.ChatProvider
}

// NewChatService builds a chat service with injected dependencies.
func NewChatService(dependencies ChatServiceDependencies) *ChatService {
	service := &ChatService{
		runtime:                        dependencies.Runtime,
		resolveProvider:                dependencies.ResolveProvider,
		reinforceEngramLinks:           dependencies.ReinforceEngramLinks,
		recordEngramAccess:             dependencies.RecordEngramAccess,
		runSessionLifecycleMaintenance: dependencies.RunSessionLifecycleMaintenance,
		providerFallback:               dependencies.ProviderFallback,
		circuitPolicy:                  dependencies.CircuitPolicy,
		observability:                  dependencies.Observability,
		resolveTraceID:                 dependencies.ResolveTraceID,
		nowUTC:                         dependencies.NowUTC,
	}
	applyChatServiceExecutionDefaults(service)
	applyChatServiceObservationDefaults(service)
	return service
}

func applyChatServiceExecutionDefaults(service *ChatService) {
	if service.runSessionLifecycleMaintenance == nil {
		service.runSessionLifecycleMaintenance = func(context.Context, uuid.UUID, models.ChatSessionRecord) error {
			return nil
		}
	}
	if service.reinforceEngramLinks == nil {
		service.reinforceEngramLinks = func(context.Context, uuid.UUID, []uuid.UUID) error { return nil }
	}
	if service.recordEngramAccess == nil {
		service.recordEngramAccess = func(context.Context, uuid.UUID, string, []uuid.UUID) error { return nil }
	}
	if service.providerFallback == nil {
		service.providerFallback = noopProviderFallbackStrategy{}
	}
	if service.circuitPolicy == nil {
		service.circuitPolicy = noopProviderCircuitPolicy{}
	}
}

func applyChatServiceObservationDefaults(service *ChatService) {
	if service.observability == nil {
		service.observability = noopObservabilityRecorder{}
	}
	if service.resolveTraceID == nil {
		service.resolveTraceID = func(context.Context) string { return "" }
	}
	if service.nowUTC == nil {
		service.nowUTC = func() time.Time { return time.Now().UTC() }
	}
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
	service.reinforceLinksWithTrace(ctx, reinforceTraceInput{
		ActorUserID: actorUserID,
		Prepared:    prepared,
		TraceID:     traceID,
		Operation:   chatOperationSend,
		Provider:    providerCandidate.Provider,
	})
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
	service.recordEngramAccessWithTrace(ctx, accessTraceInput{
		Prepared:  prepared,
		TraceID:   traceID,
		Operation: chatOperationSend,
		Provider:  providerCandidate.Provider,
	})
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
		return service.handleStreamProviderFailure(
			events,
			streamResult.ProviderError,
			providerCandidate.Provider,
			streamStartedAt,
		), nil
	}
	events = append(events, streamResult.Events...)
	return service.completeStreamSuccess(
		ctx,
		actorUserID,
		traceID,
		prepared,
		streamResult,
		providerCandidate,
		events,
		streamStartedAt,
	)
}

func (service *ChatService) completeStreamSuccess(
	ctx context.Context,
	actorUserID uuid.UUID,
	traceID string,
	prepared PreparedGeneration,
	streamResult StreamChunkResult,
	providerCandidate ProviderFallbackCandidate,
	events []StreamEvent,
	streamStartedAt time.Time,
) ([]StreamEvent, error) {
	assistantMessage, err := service.persistAssistantReplyWithTrace(
		ctx,
		actorUserID,
		prepared,
		streamAssistantResult(providerCandidate, streamResult),
		traceID,
		chatOperationStream,
		providerCandidate.Provider,
	)
	if err != nil {
		return service.mapStreamPersistenceError(
			err,
			events,
			providerCandidate.Provider,
			streamResult.Events,
			streamStartedAt,
		)
	}
	service.reinforceLinksWithTrace(ctx, reinforceTraceInput{
		ActorUserID: actorUserID,
		Prepared:    prepared,
		TraceID:     traceID,
		Operation:   chatOperationStream,
		Provider:    providerCandidate.Provider,
	})
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
	service.recordEngramAccessWithTrace(ctx, accessTraceInput{
		Prepared:  prepared,
		TraceID:   traceID,
		Operation: chatOperationStream,
		Provider:  providerCandidate.Provider,
	})
	events = append(events, streamDoneEvent(prepared, *assistantMessage, streamResult.FullText))
	service.recordStreamCompletion(providerCandidate.Provider, streamResult.Events, streamStartedAt)
	return events, nil
}

func streamAssistantResult(
	providerCandidate ProviderFallbackCandidate,
	streamResult StreamChunkResult,
) providers.ProviderGenerateResult {
	return providers.ProviderGenerateResult{
		Provider:   providerCandidate.Provider,
		ModelID:    providerCandidate.ModelID,
		Text:       streamResult.FullText,
		TokenUsage: map[string]int{},
	}
}

func (service *ChatService) mapStreamPersistenceError(
	err error,
	events []StreamEvent,
	provider models.ChatProvider,
	streamEvents []StreamEvent,
	streamStartedAt time.Time,
) ([]StreamEvent, error) {
	serviceError, ok := err.(*ChatServiceError)
	if ok {
		return service.handleStreamPersistenceFailure(
			events,
			serviceError,
			provider,
			streamEvents,
			streamStartedAt,
		), nil
	}
	return nil, err
}

func streamDoneEvent(
	prepared PreparedGeneration,
	assistantMessage models.ChatMessageRecord,
	fullText string,
) StreamEvent {
	return StreamEvent{
		Type:    "done",
		Payload: BuildStreamDonePayload(prepared, assistantMessage, fullText, nil),
	}
}

func (service *ChatService) handleStreamProviderFailure(
	events []StreamEvent,
	providerError *ChatProviderExecutionError,
	provider models.ChatProvider,
	streamStartedAt time.Time,
) []StreamEvent {
	events = append(events, streamProviderErrorEvent(providerError))
	service.recordStreamHealth(
		StreamHealthSample{
			Provider:   provider,
			Operation:  chatOperationStream,
			Outcome:    chatStreamOutcomeProviderError,
			ErrorCode:  providerError.ErrorCode(),
			ChunkCount: 0,
			Duration:   service.nowUTC().Sub(streamStartedAt),
		},
	)
	return events
}

func (service *ChatService) handleStreamPersistenceFailure(
	events []StreamEvent,
	serviceError *ChatServiceError,
	provider models.ChatProvider,
	streamEvents []StreamEvent,
	streamStartedAt time.Time,
) []StreamEvent {
	events = append(events, streamPersistenceErrorEvent(serviceError))
	service.recordStreamHealth(
		StreamHealthSample{
			Provider:   provider,
			Operation:  chatOperationStream,
			Outcome:    chatStreamOutcomePersistenceFail,
			ChunkCount: streamChunkCount(streamEvents),
			Duration:   service.nowUTC().Sub(streamStartedAt),
		},
	)
	return events
}

func (service *ChatService) recordStreamCompletion(
	provider models.ChatProvider,
	streamEvents []StreamEvent,
	streamStartedAt time.Time,
) {
	service.recordStreamHealth(
		StreamHealthSample{
			Provider:   provider,
			Operation:  chatOperationStream,
			Outcome:    chatStreamOutcomeCompleted,
			ChunkCount: streamChunkCount(streamEvents),
			Duration:   service.nowUTC().Sub(streamStartedAt),
		},
	)
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

func (service *ChatService) reinforceLinksWithTrace(ctx context.Context, input reinforceTraceInput) {
	if len(input.Prepared.Context.UsedEngramLinkIDs) == 0 {
		return
	}
	if err := service.reinforceEngramLinks(
		ctx,
		input.ActorUserID,
		input.Prepared.Context.UsedEngramLinkIDs,
	); err != nil {
		service.recordLifecycleTrace(
			input.TraceID,
			input.Operation,
			chatTraceReinforceFail,
			&input.Prepared,
			input.Provider,
			"link_reinforce_error",
			0,
		)
		return
	}
	service.recordLifecycleTrace(
		input.TraceID,
		input.Operation,
		chatTraceReinforceOK,
		&input.Prepared,
		input.Provider,
		"",
		0,
	)
}

func (service *ChatService) recordEngramAccessWithTrace(ctx context.Context, input accessTraceInput) {
	engramIDs := dedupeUUIDs(input.Prepared.Context.UsedEngramIDs)
	if len(engramIDs) == 0 {
		return
	}
	accessSource := resolveEngramAccessSource(input.Operation)
	if err := service.recordEngramAccess(
		ctx,
		input.Prepared.Session.SessionID,
		accessSource,
		engramIDs,
	); err != nil {
		service.recordLifecycleTrace(
			input.TraceID,
			input.Operation,
			chatTraceAccessFail,
			&input.Prepared,
			input.Provider,
			"engram_access_record_error",
			0,
		)
		return
	}
	service.recordLifecycleTrace(
		input.TraceID,
		input.Operation,
		chatTraceAccessOK,
		&input.Prepared,
		input.Provider,
		"",
		0,
	)
}

func resolveEngramAccessSource(operation string) string {
	switch operation {
	case chatOperationStream:
		return "chat_stream"
	default:
		return "chat_send"
	}
}

func dedupeUUIDs(values []uuid.UUID) []uuid.UUID {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(values))
	deduped := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	return deduped
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
		CWPlanApplied:        prepared.CWPlanApplied,
		UsedEngramIDs:        prepared.Context.UsedEngramIDs,
		UsedEngramLinkIDs:    prepared.Context.UsedEngramLinkIDs,
		EngramTracePaths:     prepared.Context.EngramTracePaths,
		UsedDocumentChunkIDs: prepared.Context.UsedDocumentChunkIDs,
		SourceReferences:     prepared.Context.SourceReferences,
		RetrievalAudit:       prepared.Context.RetrievalAudit,
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

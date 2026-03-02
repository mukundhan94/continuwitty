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
		prepareGenerationInput{
			ActorUserID: actorUserID,
			SessionID:   sessionID,
			Payload:     payload,
			TraceID:     traceID,
			Operation:   chatOperationSend,
		},
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
		persistAssistantInput{
			ActorUserID: actorUserID,
			Prepared:    prepared,
			Result:      result,
			TraceID:     traceID,
			Operation:   chatOperationSend,
			Provider:    providerCandidate.Provider,
		},
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
		lifecycleRunInput{
			ActorUserID: actorUserID,
			Prepared:    prepared,
			TraceID:     traceID,
			Operation:   chatOperationSend,
			Provider:    providerCandidate.Provider,
		},
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
		prepareGenerationInput{
			ActorUserID: actorUserID,
			SessionID:   sessionID,
			Payload:     payload,
			TraceID:     traceID,
			Operation:   chatOperationStream,
		},
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
		return service.appendStreamFailure(
			streamFailureInput{
				Events:          events,
				Provider:        providerCandidate.Provider,
				Outcome:         chatStreamOutcomeProviderError,
				ErrorCode:       streamResult.ProviderError.ErrorCode(),
				Detail:          streamResult.ProviderError.Detail(),
				StatusCode:      streamResult.ProviderError.StatusCode(),
				ChunkCount:      0,
				StreamStartedAt: streamStartedAt,
			},
		), nil
	}
	events = append(events, streamResult.Events...)
	return service.completeStreamSuccess(
		ctx,
		streamSuccessInput{
			ActorUserID:       actorUserID,
			TraceID:           traceID,
			Prepared:          prepared,
			StreamResult:      streamResult,
			ProviderCandidate: providerCandidate,
			Events:            events,
			StreamStartedAt:   streamStartedAt,
		},
	)
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

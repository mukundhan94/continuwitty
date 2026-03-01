package chat

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/governance"
	"engram/internal/models"
	"engram/internal/providers"

	"github.com/google/uuid"
)

type serviceRuntimeState struct {
	session            models.ChatSessionRecord
	userMessage        models.ChatMessageRecord
	assistantMessage   models.ChatMessageRecord
	context            AssembledChatContext
	createInputs       []RuntimeMessageCreateInput
	returnNilAssistant bool
}

type serviceAdapterStub struct {
	provider     models.ChatProvider
	generateText string
	generateErr  error
	streamParts  []string
	streamErr    error
}

type serviceObservabilityRecorderStub struct {
	providerFailures []ProviderFailureSample
	streamHealth     []StreamHealthSample
	lifecycleTraces  []LifecycleTraceSample
}

func (recorder *serviceObservabilityRecorderStub) RecordProviderFailure(sample ProviderFailureSample) {
	recorder.providerFailures = append(recorder.providerFailures, sample)
}

func (recorder *serviceObservabilityRecorderStub) RecordStreamHealth(sample StreamHealthSample) {
	recorder.streamHealth = append(recorder.streamHealth, sample)
}

func (recorder *serviceObservabilityRecorderStub) RecordLifecycleTrace(sample LifecycleTraceSample) {
	recorder.lifecycleTraces = append(recorder.lifecycleTraces, sample)
}

func (adapter *serviceAdapterStub) Provider() models.ChatProvider {
	return adapter.provider
}

func (adapter *serviceAdapterStub) Generate(_ context.Context, request providers.ProviderGenerateRequest) (providers.ProviderGenerateResult, error) {
	if adapter.generateErr != nil {
		return providers.ProviderGenerateResult{}, adapter.generateErr
	}
	return providers.ProviderGenerateResult{
		Provider:   adapter.provider,
		ModelID:    request.ModelID,
		Text:       adapter.generateText,
		TokenUsage: map[string]int{"input_tokens": 9, "output_tokens": 4, "total_tokens": 13},
	}, nil
}

func (adapter *serviceAdapterStub) StreamGenerate(_ context.Context, _ providers.ProviderGenerateRequest) (<-chan string, error) {
	if adapter.streamErr != nil {
		return nil, adapter.streamErr
	}
	ch := make(chan string, len(adapter.streamParts))
	for _, part := range adapter.streamParts {
		ch <- part
	}
	close(ch)
	return ch, nil
}

func (adapter *serviceAdapterStub) Healthcheck() (map[string]string, error) {
	return map[string]string{"status": "ok"}, nil
}

func TestSendMessageReturnsUsedEngramIDsAndSources(t *testing.T) {
	state := newServiceRuntimeState()
	runtime := runtimeFromServiceState(&state)
	lifecycleCalls := 0
	service := NewChatService(
		ChatServiceDependencies{
			Runtime: runtime,
			ResolveProvider: func(provider models.ChatProvider) (providers.ChatProviderAdapter, error) {
				return &serviceAdapterStub{provider: provider, generateText: "Proceed with option A."}, nil
			},
			RunSessionLifecycleMaintenance: func(context.Context, uuid.UUID, models.ChatSessionRecord) error {
				lifecycleCalls++
				return nil
			},
		},
	)

	response, err := service.SendMessage(
		context.Background(),
		state.session.OwnerUserID,
		state.session.SessionID,
		ChatMessageCreateRequest{ContentText: "How should we proceed?"},
	)
	if err != nil {
		t.Fatalf("send message: %v", err)
	}

	requireEqualAnyRuntime(t, state.session.SessionID, response.SessionID)
	requireEqualAnyRuntime(t, state.userMessage.MessageID, response.MessageID)
	requireEqualAnyRuntime(t, state.assistantMessage.MessageID, response.ReplyMessageID)
	requireEqualAnyRuntime(t, "Proceed with option A.", response.AssistantText)
	requireEqualAnyRuntime(t, governance.DefaultChatPromptPolicyVersion, response.PromptPolicyVersion)
	requireEqualAnyRuntime(t, state.context.UsedEngramIDs, response.UsedEngramIDs)
	requireEqualAnyRuntime(t, state.context.UsedEngramLinkIDs, response.UsedEngramLinkIDs)
	requireEqualAnyRuntime(t, state.context.EngramTracePaths, response.EngramTracePaths)
	requireEqualAnyRuntime(t, state.context.UsedDocumentChunkIDs, response.UsedDocumentChunkIDs)
	requireEqualAnyRuntime(t, state.context.SourceReferences, response.SourceReferences)
	requireEqualIntRuntime(t, 1, lifecycleCalls)
	requireEqualIntRuntime(t, 2, len(state.createInputs))
	requireEqualAnyRuntime(t, state.context.UsedEngramIDs, state.createInputs[1].Metadata.UsedEngramIDs)
}

func TestStreamMessageEventsEmitsMetaChunksAndDone(t *testing.T) {
	state := newServiceRuntimeState()
	runtime := runtimeFromServiceState(&state)
	lifecycleCalls := 0
	service := NewChatService(
		ChatServiceDependencies{
			Runtime: runtime,
			ResolveProvider: func(provider models.ChatProvider) (providers.ChatProviderAdapter, error) {
				return &serviceAdapterStub{provider: provider, streamParts: []string{"part-1 ", "", "part-2"}}, nil
			},
			RunSessionLifecycleMaintenance: func(context.Context, uuid.UUID, models.ChatSessionRecord) error {
				lifecycleCalls++
				return nil
			},
		},
	)

	events, err := service.StreamMessageEvents(
		context.Background(),
		state.session.OwnerUserID,
		state.session.SessionID,
		ChatMessageCreateRequest{ContentText: "Stream this response"},
	)
	if err != nil {
		t.Fatalf("stream message events: %v", err)
	}

	requireEqualAnyRuntime(t, []string{"meta", "chunk", "chunk", "done"}, eventKinds(events))
	donePayload := events[len(events)-1].Payload
	requireEqualAnyRuntime(t, "part-1 part-2", donePayload["assistant_text"])
	requireEqualAnyRuntime(t, state.assistantMessage.MessageID, donePayload["reply_message_id"])
	requireEqualAnyRuntime(t, governance.DefaultChatPromptPolicyVersion, donePayload["prompt_policy_version"])
	requireEqualAnyRuntime(t, nil, donePayload["debug_trace"])
	requireEqualIntRuntime(t, 1, lifecycleCalls)
}

func TestStreamMessageEventsEmitsProviderErrorEvent(t *testing.T) {
	state := newServiceRuntimeState()
	runtime := runtimeFromServiceState(&state)
	service := NewChatService(
		ChatServiceDependencies{
			Runtime: runtime,
			ResolveProvider: func(provider models.ChatProvider) (providers.ChatProviderAdapter, error) {
				return &serviceAdapterStub{provider: provider, streamErr: providers.NewProviderRateLimitError("too many requests")}, nil
			},
		},
	)

	events, err := service.StreamMessageEvents(
		context.Background(),
		state.session.OwnerUserID,
		state.session.SessionID,
		ChatMessageCreateRequest{ContentText: "Stream this response"},
	)
	if err != nil {
		t.Fatalf("stream message events: %v", err)
	}

	requireEqualAnyRuntime(t, []string{"meta", "error"}, eventKinds(events))
	errorPayload := events[1].Payload
	requireEqualAnyRuntime(t, 429, errorPayload["status_code"])
	requireEqualAnyRuntime(t, "provider_rate_limit", errorPayload["error_code"])
	requireEqualIntRuntime(t, 1, len(state.createInputs))
}

func TestSendMessageFallsBackToSecondaryProviderOnTransientFailure(t *testing.T) {
	state := newServiceRuntimeState()
	state.session.ModelID = "primary-model"
	state.session.Provider = models.ChatProviderOpenAI
	runtime := runtimeFromServiceState(&state)
	service := NewChatService(
		ChatServiceDependencies{
			Runtime: runtime,
			ResolveProvider: func(provider models.ChatProvider) (providers.ChatProviderAdapter, error) {
				switch provider {
				case models.ChatProviderOpenAI:
					return &serviceAdapterStub{
						provider:    provider,
						generateErr: providers.NewProviderRateLimitError("too many requests"),
					}, nil
				case models.ChatProviderAnthropic:
					return &serviceAdapterStub{
						provider:     provider,
						generateText: "fallback response",
					}, nil
				default:
					return nil, nil
				}
			},
			ProviderFallback: NewStaticProviderFallbackStrategy(
				ProviderFallbackStrategyOptions{
					Enabled:                true,
					FallbackOrder:          []models.ChatProvider{models.ChatProviderAnthropic},
					DefaultFallbackModelID: "fallback-model",
				},
			),
		},
	)

	response, err := service.SendMessage(
		context.Background(),
		state.session.OwnerUserID,
		state.session.SessionID,
		ChatMessageCreateRequest{ContentText: "hello"},
	)
	if err != nil {
		t.Fatalf("send message with fallback: %v", err)
	}
	requireEqualAnyRuntime(t, "fallback response", response.AssistantText)
	assistantInput := state.createInputs[len(state.createInputs)-1]
	if assistantInput.Metadata == nil || assistantInput.Metadata.Provider == nil {
		t.Fatalf("expected assistant provider metadata")
	}
	requireEqualAnyRuntime(t, string(models.ChatProviderAnthropic), *assistantInput.Metadata.Provider)
	requireEqualAnyRuntime(t, "fallback-model", *assistantInput.Metadata.ModelID)
}

func TestSendMessageReturnsCircuitOpenWhenNoFallbackCandidateAvailable(t *testing.T) {
	state := newServiceRuntimeState()
	runtime := runtimeFromServiceState(&state)
	clock := time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC)
	circuitPolicy := NewSimpleProviderCircuitPolicy(
		ProviderCircuitPolicyOptions{
			Enabled:          true,
			FailureThreshold: 1,
			Cooldown:         time.Minute,
			NowUTC:           func() time.Time { return clock },
		},
	)
	circuitPolicy.RecordResult(state.session.Provider, "provider_rate_limit")
	service := NewChatService(
		ChatServiceDependencies{
			Runtime: runtime,
			ResolveProvider: func(provider models.ChatProvider) (providers.ChatProviderAdapter, error) {
				return &serviceAdapterStub{provider: provider, generateText: "unused"}, nil
			},
			CircuitPolicy: circuitPolicy,
		},
	)

	_, err := service.SendMessage(
		context.Background(),
		state.session.OwnerUserID,
		state.session.SessionID,
		ChatMessageCreateRequest{ContentText: "hello"},
	)
	if err == nil {
		t.Fatalf("expected provider circuit open error")
	}
	providerErr, ok := err.(*ChatProviderExecutionError)
	if !ok {
		t.Fatalf("expected provider execution error, got %T", err)
	}
	requireEqualAnyRuntime(t, "provider_circuit_open", providerErr.ErrorCode())
}

func TestStreamMessageEventsFallsBackAfterTransientProviderFailure(t *testing.T) {
	state := newServiceRuntimeState()
	runtime := runtimeFromServiceState(&state)
	service := NewChatService(
		ChatServiceDependencies{
			Runtime: runtime,
			ResolveProvider: func(provider models.ChatProvider) (providers.ChatProviderAdapter, error) {
				if provider == models.ChatProviderOpenAI {
					return &serviceAdapterStub{provider: provider, streamErr: providers.NewProviderRateLimitError("throttled")}, nil
				}
				return &serviceAdapterStub{provider: provider, streamParts: []string{"fallback ", "stream"}}, nil
			},
			ProviderFallback: NewStaticProviderFallbackStrategy(
				ProviderFallbackStrategyOptions{
					Enabled:                true,
					FallbackOrder:          []models.ChatProvider{models.ChatProviderAnthropic},
					DefaultFallbackModelID: "fallback-model",
				},
			),
		},
	)

	events, err := service.StreamMessageEvents(
		context.Background(),
		state.session.OwnerUserID,
		state.session.SessionID,
		ChatMessageCreateRequest{ContentText: "stream with fallback"},
	)
	if err != nil {
		t.Fatalf("stream message with fallback: %v", err)
	}
	requireEqualAnyRuntime(t, []string{"meta", "chunk", "chunk", "done"}, eventKinds(events))
	donePayload := events[len(events)-1].Payload
	requireEqualAnyRuntime(t, "fallback stream", donePayload["assistant_text"])
}

func TestSendMessageEmitsLifecycleTraceAndProviderFailureSamples(t *testing.T) {
	state := newServiceRuntimeState()
	runtime := runtimeFromServiceState(&state)
	observability := &serviceObservabilityRecorderStub{}
	service := NewChatService(
		ChatServiceDependencies{
			Runtime: runtime,
			ResolveProvider: func(provider models.ChatProvider) (providers.ChatProviderAdapter, error) {
				if provider == models.ChatProviderOpenAI {
					return &serviceAdapterStub{provider: provider, generateErr: providers.NewProviderRateLimitError("throttled")}, nil
				}
				return &serviceAdapterStub{provider: provider, generateText: "ok"}, nil
			},
			ProviderFallback: NewStaticProviderFallbackStrategy(
				ProviderFallbackStrategyOptions{
					Enabled:                true,
					FallbackOrder:          []models.ChatProvider{models.ChatProviderAnthropic},
					DefaultFallbackModelID: "fallback-model",
				},
			),
			Observability:  observability,
			ResolveTraceID: func(context.Context) string { return "trace-123" },
		},
	)

	_, err := service.SendMessage(
		context.Background(),
		state.session.OwnerUserID,
		state.session.SessionID,
		ChatMessageCreateRequest{ContentText: "record telemetry"},
	)
	if err != nil {
		t.Fatalf("send message telemetry: %v", err)
	}
	if len(observability.providerFailures) == 0 {
		t.Fatalf("expected provider failure sample")
	}
	requireEqualAnyRuntime(t, models.ChatProviderOpenAI, observability.providerFailures[0].Provider)
	requireEqualAnyRuntime(t, "provider_rate_limit", observability.providerFailures[0].ErrorCode)
	if len(observability.lifecycleTraces) == 0 {
		t.Fatalf("expected lifecycle traces")
	}
	hasPrepare := false
	hasProviderFail := false
	hasLifecycleSuccess := false
	for _, sample := range observability.lifecycleTraces {
		if sample.Stage == chatTracePrepareDone {
			hasPrepare = true
		}
		if sample.Stage == chatTraceProviderFail && sample.ErrorCode == "provider_rate_limit" {
			hasProviderFail = true
		}
		if sample.Stage == chatTraceLifecycleOK {
			hasLifecycleSuccess = true
		}
	}
	if !hasPrepare || !hasProviderFail || !hasLifecycleSuccess {
		t.Fatalf("missing expected lifecycle trace samples: %+v", observability.lifecycleTraces)
	}
}

func TestSendMessageReinforcesUsedEngramLinks(t *testing.T) {
	state := newServiceRuntimeState()
	runtime := runtimeFromServiceState(&state)
	reinforcementCalls := 0
	var reinforcedActor uuid.UUID
	var reinforcedIDs []uuid.UUID
	service := NewChatService(
		ChatServiceDependencies{
			Runtime: runtime,
			ResolveProvider: func(provider models.ChatProvider) (providers.ChatProviderAdapter, error) {
				return &serviceAdapterStub{provider: provider, generateText: "ok"}, nil
			},
			ReinforceEngramLinks: func(
				_ context.Context,
				actorUserID uuid.UUID,
				linkIDs []uuid.UUID,
			) error {
				reinforcementCalls++
				reinforcedActor = actorUserID
				reinforcedIDs = append([]uuid.UUID(nil), linkIDs...)
				return nil
			},
		},
	)

	_, err := service.SendMessage(
		context.Background(),
		state.session.OwnerUserID,
		state.session.SessionID,
		ChatMessageCreateRequest{ContentText: "reinforce links"},
	)
	if err != nil {
		t.Fatalf("send message with reinforcement: %v", err)
	}

	requireEqualIntRuntime(t, 1, reinforcementCalls)
	requireEqualAnyRuntime(t, state.session.OwnerUserID, reinforcedActor)
	requireEqualAnyRuntime(t, state.context.UsedEngramLinkIDs, reinforcedIDs)
}

func TestSendMessageContinuesWhenLinkReinforcementFails(t *testing.T) {
	state := newServiceRuntimeState()
	runtime := runtimeFromServiceState(&state)
	observability := &serviceObservabilityRecorderStub{}
	service := NewChatService(
		ChatServiceDependencies{
			Runtime: runtime,
			ResolveProvider: func(provider models.ChatProvider) (providers.ChatProviderAdapter, error) {
				return &serviceAdapterStub{provider: provider, generateText: "ok"}, nil
			},
			ReinforceEngramLinks: func(context.Context, uuid.UUID, []uuid.UUID) error {
				return errors.New("reinforce failed")
			},
			Observability:  observability,
			ResolveTraceID: func(context.Context) string { return "trace-777" },
		},
	)

	response, err := service.SendMessage(
		context.Background(),
		state.session.OwnerUserID,
		state.session.SessionID,
		ChatMessageCreateRequest{ContentText: "ignore reinforce failure"},
	)
	if err != nil {
		t.Fatalf("send message should succeed despite reinforce failure: %v", err)
	}
	requireEqualAnyRuntime(t, "ok", response.AssistantText)

	hasReinforceFailTrace := false
	for _, sample := range observability.lifecycleTraces {
		if sample.Stage == chatTraceReinforceFail && sample.ErrorCode == "link_reinforce_error" {
			hasReinforceFailTrace = true
			break
		}
	}
	if !hasReinforceFailTrace {
		t.Fatalf("expected reinforce failure lifecycle trace, got: %+v", observability.lifecycleTraces)
	}
}

func runtimeFromServiceState(state *serviceRuntimeState) *ChatMessageRuntime {
	return NewChatMessageRuntime(
		ChatMessageRuntimeDependencies{
			EmbeddingDim: 256,
			GetSession: func(context.Context, uuid.UUID, uuid.UUID) (*models.ChatSessionRecord, error) {
				session := state.session
				return &session, nil
			},
			CreateChatMessage: func(_ context.Context, input RuntimeMessageCreateInput) (*models.ChatMessageRecord, error) {
				state.createInputs = append(state.createInputs, input)
				if input.Role == "user" {
					record := state.userMessage
					record.ContentText = input.ContentText
					return &record, nil
				}
				if state.returnNilAssistant {
					return nil, nil
				}
				record := state.assistantMessage
				record.ContentText = input.ContentText
				return &record, nil
			},
			ListChatMessages: func(context.Context, uuid.UUID, uuid.UUID, int, int) ([]models.ChatMessageRecord, error) {
				return []models.ChatMessageRecord{state.userMessage}, nil
			},
			AssembleChatContext: func(_ context.Context, _ ChatContextRequest) (AssembledChatContext, error) {
				return state.context, nil
			},
		},
	)
}

func newServiceRuntimeState() serviceRuntimeState {
	now := time.Now().UTC()
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007201")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000007202")
	userMessageID := uuid.MustParse("00000000-0000-0000-0000-000000007203")
	assistantMessageID := uuid.MustParse("00000000-0000-0000-0000-000000007204")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000007205")
	linkedEngramID := uuid.MustParse("00000000-0000-0000-0000-000000007206")
	engramLinkID := uuid.MustParse("00000000-0000-0000-0000-000000007207")
	sourceTitle := "Source"
	source := ChatSourceReference{
		SourceType:  "engram_source",
		EngramID:    engramID,
		EngramTitle: "Referenced",
		URL:         "https://example.com/source",
		Title:       &sourceTitle,
		Snippet:     "Snippet",
		CapturedAt:  now,
	}
	return serviceRuntimeState{
		session: models.ChatSessionRecord{
			SessionID:       sessionID,
			OwnerUserID:     actorUserID,
			ProjectID:       "project-chat",
			Title:           "Session",
			Provider:        models.ChatProviderOpenAI,
			ModelID:         "gpt-4o-mini",
			SystemPrompt:    "be helpful",
			VisibilityScope: models.VisibilityScopePrivate,
			AutosaveEnabled: false,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		userMessage: models.ChatMessageRecord{
			MessageID:      userMessageID,
			SessionID:      sessionID,
			Role:           "user",
			ContentText:    "How should we proceed?",
			TokenUsageJSON: map[string]any{},
			UsedEngramIDs:  []uuid.UUID{},
			CreatedAt:      now,
		},
		assistantMessage: models.ChatMessageRecord{
			MessageID:      assistantMessageID,
			SessionID:      sessionID,
			Role:           "assistant",
			ContentText:    "Proceed with option A.",
			TokenUsageJSON: map[string]any{},
			UsedEngramIDs:  []uuid.UUID{},
			CreatedAt:      now,
		},
		context: AssembledChatContext{
			ContextMarkdown:   "ctx",
			UsedEngramIDs:     []uuid.UUID{engramID},
			UsedEngramLinkIDs: []uuid.UUID{engramLinkID},
			EngramTracePaths: []EngramTracePath{
				{
					RootEngramID:   engramID,
					TargetEngramID: linkedEngramID,
					Depth:          1,
					LinkIDs:        []uuid.UUID{engramLinkID},
					EngramIDs:      []uuid.UUID{engramID, linkedEngramID},
					Score:          0.82,
				},
			},
			UsedDocumentChunkIDs: []uuid.UUID{},
			SourceReferences:     []ChatSourceReference{source},
		},
		createInputs: []RuntimeMessageCreateInput{},
	}
}

func eventKinds(events []StreamEvent) []string {
	kinds := make([]string, 0, len(events))
	for _, event := range events {
		kinds = append(kinds, event.Type)
	}
	return kinds
}

func requireEqualAnyService(t *testing.T, expected any, actual any) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

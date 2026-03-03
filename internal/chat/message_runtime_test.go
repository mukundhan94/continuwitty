package chat

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/providers"

	"github.com/google/uuid"
)

func TestBuildStreamMetaPayloadIncludesContextReferences(t *testing.T) {
	prepared := preparedGenerationFixture()
	runtime := NewChatMessageRuntime(ChatMessageRuntimeDependencies{EmbeddingDim: 256})

	payload := runtime.BuildStreamMetaPayload(prepared)

	requireEqualAnyRuntime(t, prepared.Session.SessionID, payload["session_id"])
	requireEqualAnyRuntime(t, prepared.UserMessage.MessageID, payload["message_id"])
	requireEqualAnyRuntime(t, prepared.PromptPolicyVersion, payload["prompt_policy_version"])
	requireEqualAnyRuntime(t, prepared.Context.UsedEngramIDs, payload["used_engram_ids"])
	requireEqualAnyRuntime(t, prepared.Context.UsedEngramLinkIDs, payload["used_engram_link_ids"])
	requireEqualAnyRuntime(t, prepared.Context.EngramTracePaths, payload["engram_trace_paths"])
	requireEqualAnyRuntime(t, prepared.Context.ContradictionWarnings, payload["contradiction_warnings"])
	requireEqualAnyRuntime(t, prepared.Context.UsedDocumentChunkIDs, payload["used_document_chunk_ids"])
	requireEqualAnyRuntime(t, prepared.Context.SourceReferences, payload["source_references"])
	requireEqualAnyRuntime(t, prepared.Context.RetrievalAudit, payload["retrieval_audit"])
}

func TestBuildStreamDonePayloadIncludesReplyAndContextFields(t *testing.T) {
	prepared := preparedGenerationFixture()
	assistantMessageID := uuid.MustParse("00000000-0000-0000-0000-000000007004")
	assistant := models.ChatMessageRecord{
		MessageID:   assistantMessageID,
		SessionID:   prepared.Session.SessionID,
		Role:        "assistant",
		ContentText: "answer",
		CreatedAt:   time.Now().UTC(),
	}
	runtime := NewChatMessageRuntime(ChatMessageRuntimeDependencies{EmbeddingDim: 256})
	debugTrace := map[string]any{"llm_call_duration_ms": 12.5}

	payload := runtime.BuildStreamDonePayload(prepared, assistant, "answer", debugTrace)

	requireEqualAnyRuntime(t, prepared.Session.SessionID, payload["session_id"])
	requireEqualAnyRuntime(t, prepared.UserMessage.MessageID, payload["message_id"])
	requireEqualAnyRuntime(t, assistantMessageID, payload["reply_message_id"])
	requireEqualAnyRuntime(t, "answer", payload["assistant_text"])
	requireEqualAnyRuntime(t, prepared.PromptPolicyVersion, payload["prompt_policy_version"])
	requireEqualAnyRuntime(t, prepared.Context.UsedEngramIDs, payload["used_engram_ids"])
	requireEqualAnyRuntime(t, prepared.Context.UsedEngramLinkIDs, payload["used_engram_link_ids"])
	requireEqualAnyRuntime(t, prepared.Context.EngramTracePaths, payload["engram_trace_paths"])
	requireEqualAnyRuntime(t, prepared.Context.ContradictionWarnings, payload["contradiction_warnings"])
	requireEqualAnyRuntime(t, prepared.Context.UsedDocumentChunkIDs, payload["used_document_chunk_ids"])
	requireEqualAnyRuntime(t, prepared.Context.SourceReferences, payload["source_references"])
	requireEqualAnyRuntime(t, prepared.Context.RetrievalAudit, payload["retrieval_audit"])
	requireEqualAnyRuntime(t, debugTrace, payload["debug_trace"])
}

func TestPrepareGenerationBuildsProviderRequestFromHistoryAndContext(t *testing.T) {
	base := preparedGenerationFixture()
	createInputs := make([]RuntimeMessageCreateInput, 0)
	runtime := runtimeForPrepareGeneration(t, base, &createInputs)

	prepared, err := runtime.PrepareGeneration(
		context.Background(),
		base.Session.OwnerUserID,
		base.Session.SessionID,
		ChatMessageCreateRequest{ContentText: "How should we proceed?"},
	)
	if err != nil {
		t.Fatalf("prepare generation: %v", err)
	}
	assertPreparedGenerationRequest(t, prepared, createInputs)
}

func TestPrepareGenerationForwardsLinkRecallOverrides(t *testing.T) {
	base := preparedGenerationFixture()
	requests := make([]ChatContextRequest, 0, 1)
	runtime := NewChatMessageRuntime(
		ChatMessageRuntimeDependencies{
			EmbeddingDim: 256,
			GetSession: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.ChatSessionRecord, error) {
				session := base.Session
				return &session, nil
			},
			CreateChatMessage: func(_ context.Context, input RuntimeMessageCreateInput) (*models.ChatMessageRecord, error) {
				record := base.UserMessage
				record.ContentText = input.ContentText
				return &record, nil
			},
			ListChatMessages: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int, _ int) ([]models.ChatMessageRecord, error) {
				return []models.ChatMessageRecord{}, nil
			},
			AssembleChatContext: func(_ context.Context, request ChatContextRequest) (AssembledChatContext, error) {
				requests = append(requests, request)
				return base.Context, nil
			},
		},
	)
	enabled := false
	depth := 2
	maxNeighbors := 9
	noiseSuppressionEnabled := false
	noiseThreshold := 0.72
	contextTokenBudget := 640

	_, err := runtime.PrepareGeneration(
		context.Background(),
		base.Session.OwnerUserID,
		base.Session.SessionID,
		ChatMessageCreateRequest{
			ContentText:                 "How should we proceed?",
			ContextTokenBudget:          &contextTokenBudget,
			LinkRecallEnabled:           &enabled,
			LinkRecallDepth:             &depth,
			LinkRecallMaxNeighbors:      &maxNeighbors,
			LinkNoiseSuppressionEnabled: &noiseSuppressionEnabled,
			LinkNoiseScoreThreshold:     &noiseThreshold,
		},
	)
	if err != nil {
		t.Fatalf("prepare generation: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected one context assembly request, got %d", len(requests))
	}
	requireOptionalIntRuntime(t, requests[0].ContextTokenBudget, 640, "context_token_budget")
	requireOptionalBoolRuntime(t, requests[0].LinkRecallEnabled, false, "link_recall_enabled")
	requireOptionalIntRuntime(t, requests[0].LinkRecallDepth, 2, "link_recall_depth")
	requireOptionalIntRuntime(t, requests[0].LinkRecallMaxNeighbors, 9, "link_recall_max_neighbors")
	requireOptionalBoolRuntime(t, requests[0].LinkNoiseSuppressionEnabled, false, "link_noise_suppression_enabled")
	requireOptionalFloatRuntime(t, requests[0].LinkNoiseScoreThreshold, 0.72, "link_noise_score_threshold")
}

func TestPrepareGenerationParsesCWQueryProtocolAndSanitizesUserContent(t *testing.T) {
	base := preparedGenerationFixture()
	contextRequests := make([]ChatContextRequest, 0, 1)
	createdMessages := make([]RuntimeMessageCreateInput, 0, 1)
	runtime := NewChatMessageRuntime(
		ChatMessageRuntimeDependencies{
			EmbeddingDim: 256,
			GetSession: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.ChatSessionRecord, error) {
				session := base.Session
				return &session, nil
			},
			CreateChatMessage: func(_ context.Context, input RuntimeMessageCreateInput) (*models.ChatMessageRecord, error) {
				createdMessages = append(createdMessages, input)
				record := base.UserMessage
				record.ContentText = input.ContentText
				return &record, nil
			},
			ListChatMessages: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int, _ int) ([]models.ChatMessageRecord, error) {
				return []models.ChatMessageRecord{}, nil
			},
			AssembleChatContext: func(_ context.Context, request ChatContextRequest) (AssembledChatContext, error) {
				contextRequests = append(contextRequests, request)
				return base.Context, nil
			},
		},
	)

	prepared, err := runtime.PrepareGeneration(
		context.Background(),
		base.Session.OwnerUserID,
		base.Session.SessionID,
		ChatMessageCreateRequest{
			ContentText: "cw> mode=retrieve project=ops citations=required\nFind linked incidents from this quarter.",
		},
	)
	if err != nil {
		t.Fatalf("prepare generation: %v", err)
	}
	if len(createdMessages) != 1 {
		t.Fatalf("expected one user-message create call, got %d", len(createdMessages))
	}
	if createdMessages[0].ContentText != "Find linked incidents from this quarter." {
		t.Fatalf("expected sanitized user content, got %q", createdMessages[0].ContentText)
	}
	if len(contextRequests) != 1 {
		t.Fatalf("expected one context assembly request, got %d", len(contextRequests))
	}
	if contextRequests[0].UserQuery != "Find linked incidents from this quarter." {
		t.Fatalf("expected sanitized query for context assembly, got %q", contextRequests[0].UserQuery)
	}
	if prepared.CWPlanApplied == nil {
		t.Fatalf("expected cw plan metadata")
	}
	requireEqualAnyRuntime(t, "retrieve", prepared.CWPlanApplied.Mode)
	requireEqualAnyRuntime(t, "ops", prepared.CWPlanApplied.Project)
	requireEqualAnyRuntime(t, true, prepared.CWPlanApplied.CitationsRequired)
}

func TestPrepareGenerationRejectsEmptyPayload(t *testing.T) {
	runtime := NewChatMessageRuntime(ChatMessageRuntimeDependencies{})
	_, err := runtime.PrepareGeneration(
		context.Background(),
		uuid.MustParse("00000000-0000-0000-0000-000000007101"),
		uuid.MustParse("00000000-0000-0000-0000-000000007102"),
		ChatMessageCreateRequest{ContentText: "   "},
	)
	validationErr := requireChatServiceError(t, err)
	requireEqualIntRuntime(t, 400, validationErr.StatusCode())
	requireEqualAnyRuntime(t, "Message content cannot be empty", validationErr.Detail())
}

func TestBuildStreamPayloadIncludesCWPlanMetadata(t *testing.T) {
	prepared := preparedGenerationFixture()
	prepared.CWPlanApplied = &CWQueryPlan{
		Mode:              "analyze",
		Project:           "project-ops",
		CitationsRequired: true,
	}
	runtime := NewChatMessageRuntime(ChatMessageRuntimeDependencies{EmbeddingDim: 256})
	assistant := models.ChatMessageRecord{
		MessageID: uuid.MustParse("00000000-0000-0000-0000-000000007011"),
		SessionID: prepared.Session.SessionID,
		Role:      "assistant",
	}

	metaPayload := runtime.BuildStreamMetaPayload(prepared)
	donePayload := runtime.BuildStreamDonePayload(prepared, assistant, "answer", nil)
	requireEqualAnyRuntime(t, prepared.CWPlanApplied, metaPayload["cw_plan_applied"])
	requireEqualAnyRuntime(t, prepared.CWPlanApplied, donePayload["cw_plan_applied"])
}

func TestPersistAssistantReplyWritesProviderMetadata(t *testing.T) {
	prepared := preparedGenerationFixture()
	actorUserID := prepared.Session.OwnerUserID
	capturedInputs := make([]RuntimeMessageCreateInput, 0)
	runtime := NewChatMessageRuntime(
		ChatMessageRuntimeDependencies{
			CreateChatMessage: func(_ context.Context, input RuntimeMessageCreateInput) (*models.ChatMessageRecord, error) {
				capturedInputs = append(capturedInputs, input)
				record := chatHistoryRecord(prepared.Session.SessionID, "assistant", input.ContentText)
				return &record, nil
			},
		},
	)
	result := providers.ProviderGenerateResult{
		Provider:   prepared.Session.Provider,
		ModelID:    prepared.Session.ModelID,
		Text:       "Proceed with option A.",
		TokenUsage: map[string]int{"input_tokens": 9, "output_tokens": 4, "total_tokens": 13},
	}

	record, err := runtime.PersistAssistantReply(context.Background(), actorUserID, prepared, result)
	if err != nil {
		t.Fatalf("persist assistant reply: %v", err)
	}
	requireEqualAnyRuntime(t, "Proceed with option A.", record.ContentText)
	if len(capturedInputs) != 1 {
		t.Fatalf("expected one assistant message create call, got %d", len(capturedInputs))
	}
	metadata := capturedInputs[0].Metadata
	if metadata == nil {
		t.Fatalf("expected assistant metadata to be set")
	}
	requireEqualAnyRuntime(t, []uuid.UUID{prepared.Context.UsedEngramIDs[0]}, metadata.UsedEngramIDs)
	requireEqualAnyRuntime(t, 13, metadata.TokenUsage["total_tokens"])
	if metadata.Provider == nil || *metadata.Provider != string(prepared.Session.Provider) {
		t.Fatalf("expected provider metadata to match session provider")
	}
	if metadata.ModelID == nil || *metadata.ModelID != prepared.Session.ModelID {
		t.Fatalf("expected model metadata to match session model")
	}
}

func TestYieldStreamChunksEmitsChunkEventsAndAggregatesText(t *testing.T) {
	prepared := preparedGenerationFixture()
	runtime := NewChatMessageRuntime(ChatMessageRuntimeDependencies{EmbeddingDim: 256})

	result, err := runtime.YieldStreamChunks(
		context.Background(),
		streamGenerateFromParts("part-1 ", "", "part-2"),
		prepared,
	)
	if err != nil {
		t.Fatalf("yield stream chunks: %v", err)
	}
	if result.ProviderError != nil {
		t.Fatalf("expected no provider error, got %v", result.ProviderError)
	}
	requireEqualAnyRuntime(t, "part-1 part-2", result.FullText)
	requireEqualAnyRuntime(
		t,
		[]StreamEvent{
			{Type: "chunk", Payload: map[string]any{"text": "part-1 "}},
			{Type: "chunk", Payload: map[string]any{"text": "part-2"}},
		},
		result.Events,
	)
}

func TestYieldStreamChunksMapsProviderErrors(t *testing.T) {
	prepared := preparedGenerationFixture()
	runtime := NewChatMessageRuntime(ChatMessageRuntimeDependencies{EmbeddingDim: 256})

	result, err := runtime.YieldStreamChunks(
		context.Background(),
		func(context.Context, providers.ProviderGenerateRequest) (<-chan string, error) {
			return nil, providers.NewProviderRateLimitError("too many requests")
		},
		prepared,
	)
	if err != nil {
		t.Fatalf("expected mapped provider error result, got %v", err)
	}
	if result.ProviderError == nil {
		t.Fatalf("expected provider error result")
	}
	requireEqualIntRuntime(t, 429, result.ProviderError.StatusCode())
	requireEqualAnyRuntime(t, "provider_rate_limit", result.ProviderError.ErrorCode())
}

func preparedGenerationFixture() PreparedGeneration {
	now := time.Now().UTC()
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007001")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000007002")
	userMessageID := uuid.MustParse("00000000-0000-0000-0000-000000007003")
	ids := preparedFixtureIDs{
		engramID:        uuid.MustParse("00000000-0000-0000-0000-000000007005"),
		documentChunkID: uuid.MustParse("00000000-0000-0000-0000-000000007006"),
		documentID:      uuid.MustParse("00000000-0000-0000-0000-000000007007"),
		linkID:          uuid.MustParse("00000000-0000-0000-0000-000000007008"),
		linkedEngramID:  uuid.MustParse("00000000-0000-0000-0000-000000007009"),
	}
	return PreparedGeneration{
		Session: models.ChatSessionRecord{
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
		UserMessage: models.ChatMessageRecord{
			MessageID:      userMessageID,
			SessionID:      sessionID,
			Role:           "user",
			ContentText:    "Q",
			TokenUsageJSON: map[string]any{},
			UsedEngramIDs:  []uuid.UUID{},
			CreatedAt:      now,
		},
		Context: preparedContextFixture(now, ids),
		ProviderRequest: providers.ProviderGenerateRequest{
			ModelID:      "gpt-4o-mini",
			Messages:     []providers.ProviderMessage{{Role: "user", Content: "Q"}},
			SystemPrompt: "sys",
		},
		PromptPolicyVersion:   "chat-prompt-policy-test-v1",
		PrepareDurationMS:     1.0,
		ContextDurationMS:     1.0,
		HistoryLoadDurationMS: 1.0,
	}
}

type preparedFixtureIDs struct {
	engramID        uuid.UUID
	documentChunkID uuid.UUID
	documentID      uuid.UUID
	linkID          uuid.UUID
	linkedEngramID  uuid.UUID
}

func preparedContextFixture(now time.Time, ids preparedFixtureIDs) AssembledChatContext {
	chunkIndex := 1
	sourceTitle := "runbook.md"
	return AssembledChatContext{
		ContextMarkdown:   "ctx",
		UsedEngramIDs:     []uuid.UUID{ids.engramID},
		UsedEngramLinkIDs: []uuid.UUID{ids.linkID},
		EngramTracePaths: []EngramTracePath{
			{
				RootEngramID:   ids.engramID,
				TargetEngramID: ids.linkedEngramID,
				Depth:          1,
				LinkIDs:        []uuid.UUID{ids.linkID},
				EngramIDs:      []uuid.UUID{ids.engramID, ids.linkedEngramID},
				Score:          0.91,
			},
		},
		UsedDocumentChunkIDs: []uuid.UUID{ids.documentChunkID},
		SourceReferences: []ChatSourceReference{
			{
				SourceType:  "document_chunk",
				EngramID:    ids.documentID,
				EngramTitle: "Runbook",
				URL:         "document://" + ids.documentID.String(),
				Title:       &sourceTitle,
				Snippet:     "snippet",
				CapturedAt:  now,
				DocumentID:  &ids.documentID,
				ChunkID:     &ids.documentChunkID,
				ChunkIndex:  &chunkIndex,
			},
		},
		RetrievalAudit: preparedRetrievalAuditFixture(),
	}
}

func preparedRetrievalAuditFixture() *ChatRetrievalAudit {
	return &ChatRetrievalAudit{
		CandidateEngramCount:        3,
		PackedEngramCount:           2,
		BlockedEngramCandidateCount: 1,
		LinkedTraceCandidateCount:   1,
		SuppressedTracePathCount:    0,
		FilteredTracePathCount:      0,
		TruncatedTracePathCount:     0,
		CrossProjectEngramCount:     1,
		CrossProjectTracePathCount:  1,
		CrossProjectProjectIDs:      []string{"project-federated"},
	}
}

func streamGenerateFromParts(parts ...string) StreamGenerateFunc {
	return func(context.Context, providers.ProviderGenerateRequest) (<-chan string, error) {
		chunks := make(chan string, len(parts))
		for _, part := range parts {
			chunks <- part
		}
		close(chunks)
		return chunks, nil
	}
}

func requireEqualAnyRuntime(t *testing.T, expected any, actual any) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func requireEqualIntRuntime(t *testing.T, expected int, actual int) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %d, got %d", expected, actual)
	}
}

func requireOptionalBoolRuntime(t *testing.T, value *bool, expected bool, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%v, got %v", field, expected, value)
	}
}

func requireOptionalIntRuntime(t *testing.T, value *int, expected int, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%d, got %v", field, expected, value)
	}
}

func requireOptionalFloatRuntime(t *testing.T, value *float64, expected float64, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%v, got %v", field, expected, value)
	}
}

func requireChatServiceError(t *testing.T, err error) *ChatServiceError {
	t.Helper()
	if err == nil {
		t.Fatalf("expected chat service error, got nil")
	}
	serviceErr, ok := err.(*ChatServiceError)
	if !ok {
		t.Fatalf("expected ChatServiceError, got %T", err)
	}
	return serviceErr
}

func chatHistoryRecord(sessionID uuid.UUID, role string, content string) models.ChatMessageRecord {
	return models.ChatMessageRecord{
		MessageID:      uuid.New(),
		SessionID:      sessionID,
		Role:           role,
		ContentText:    content,
		TokenUsageJSON: map[string]any{},
		UsedEngramIDs:  []uuid.UUID{},
		CreatedAt:      time.Now().UTC(),
	}
}

func runtimeForPrepareGeneration(
	t *testing.T,
	base PreparedGeneration,
	createInputs *[]RuntimeMessageCreateInput,
) *ChatMessageRuntime {
	t.Helper()
	return NewChatMessageRuntime(
		ChatMessageRuntimeDependencies{
			EmbeddingDim: 256,
			GetSession: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.ChatSessionRecord, error) {
				session := base.Session
				return &session, nil
			},
			CreateChatMessage: func(_ context.Context, input RuntimeMessageCreateInput) (*models.ChatMessageRecord, error) {
				*createInputs = append(*createInputs, input)
				if input.Role != "user" {
					t.Fatalf("expected user role on prepare, got %q", input.Role)
				}
				record := base.UserMessage
				record.ContentText = input.ContentText
				return &record, nil
			},
			ListChatMessages: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int, _ int) ([]models.ChatMessageRecord, error) {
				return []models.ChatMessageRecord{
					chatHistoryRecord(base.Session.SessionID, "system", "ignore"),
					chatHistoryRecord(base.Session.SessionID, "user", "first"),
					chatHistoryRecord(base.Session.SessionID, "assistant", "second"),
					chatHistoryRecord(base.Session.SessionID, "tool", "ignore"),
				}, nil
			},
			AssembleChatContext: func(_ context.Context, request ChatContextRequest) (AssembledChatContext, error) {
				if request.UserQuery != "How should we proceed?" {
					t.Fatalf("unexpected context query %q", request.UserQuery)
				}
				contextValue := base.Context
				contextValue.ContextMarkdown = "# Engram Retrieval Context\n\nsummary"
				return contextValue, nil
			},
		},
	)
}

func assertPreparedGenerationRequest(
	t *testing.T,
	prepared PreparedGeneration,
	createInputs []RuntimeMessageCreateInput,
) {
	t.Helper()
	if len(createInputs) != 1 {
		t.Fatalf("expected one user message create call, got %d", len(createInputs))
	}
	requireEqualAnyRuntime(t, "How should we proceed?", createInputs[0].ContentText)
	requireEqualAnyRuntime(
		t,
		[]providers.ProviderMessage{
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "second"},
		},
		prepared.ProviderRequest.Messages,
	)
	if !strings.Contains(prepared.ProviderRequest.SystemPrompt, "Use the retrieved engram context below when relevant.") {
		t.Fatalf("expected system prompt to include retrieval guidance")
	}
	if hasNegativeRuntimeDuration(prepared) {
		t.Fatalf("expected non-negative prepare/context/history durations")
	}
}

func hasNegativeRuntimeDuration(prepared PreparedGeneration) bool {
	return prepared.PrepareDurationMS < 0 || prepared.ContextDurationMS < 0 || prepared.HistoryLoadDurationMS < 0
}

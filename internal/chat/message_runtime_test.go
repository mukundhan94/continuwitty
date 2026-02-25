package chat

import (
	"reflect"
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
	requireEqualAnyRuntime(t, prepared.Context.UsedEngramIDs, payload["used_engram_ids"])
	requireEqualAnyRuntime(t, prepared.Context.UsedDocumentChunkIDs, payload["used_document_chunk_ids"])
	requireEqualAnyRuntime(t, prepared.Context.SourceReferences, payload["source_references"])
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
	requireEqualAnyRuntime(t, prepared.Context.UsedEngramIDs, payload["used_engram_ids"])
	requireEqualAnyRuntime(t, prepared.Context.UsedDocumentChunkIDs, payload["used_document_chunk_ids"])
	requireEqualAnyRuntime(t, prepared.Context.SourceReferences, payload["source_references"])
	requireEqualAnyRuntime(t, debugTrace, payload["debug_trace"])
}

func preparedGenerationFixture() PreparedGeneration {
	now := time.Now().UTC()
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000007001")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000007002")
	userMessageID := uuid.MustParse("00000000-0000-0000-0000-000000007003")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000007005")
	documentChunkID := uuid.MustParse("00000000-0000-0000-0000-000000007006")
	documentID := uuid.MustParse("00000000-0000-0000-0000-000000007007")
	chunkIndex := 1
	sourceTitle := "runbook.md"
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
		Context: AssembledChatContext{
			ContextMarkdown:      "ctx",
			UsedEngramIDs:        []uuid.UUID{engramID},
			UsedDocumentChunkIDs: []uuid.UUID{documentChunkID},
			SourceReferences: []ChatSourceReference{
				{
					SourceType:  "document_chunk",
					EngramID:    documentID,
					EngramTitle: "Runbook",
					URL:         "document://" + documentID.String(),
					Title:       &sourceTitle,
					Snippet:     "snippet",
					CapturedAt:  now,
					DocumentID:  &documentID,
					ChunkID:     &documentChunkID,
					ChunkIndex:  &chunkIndex,
				},
			},
		},
		ProviderRequest: providers.ProviderGenerateRequest{
			ModelID:      "gpt-4o-mini",
			Messages:     []providers.ProviderMessage{{Role: "user", Content: "Q"}},
			SystemPrompt: "sys",
		},
		PrepareDurationMS:     1.0,
		ContextDurationMS:     1.0,
		HistoryLoadDurationMS: 1.0,
	}
}

func requireEqualAnyRuntime(t *testing.T, expected any, actual any) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

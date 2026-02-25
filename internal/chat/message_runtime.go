package chat

import (
	"engram/internal/models"
	"engram/internal/providers"
)

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
}

// ChatMessageRuntime coordinates chat generation and stream payload shaping.
type ChatMessageRuntime struct {
	embeddingDim              int
	chatDebugEnabled          bool
	chatDebugIncludeRawOutput bool
}

// NewChatMessageRuntime builds a message runtime from dependency configuration.
func NewChatMessageRuntime(dependencies ChatMessageRuntimeDependencies) *ChatMessageRuntime {
	return &ChatMessageRuntime{
		embeddingDim:              dependencies.EmbeddingDim,
		chatDebugEnabled:          dependencies.ChatDebugEnabled,
		chatDebugIncludeRawOutput: dependencies.ChatDebugIncludeRawOutput,
	}
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

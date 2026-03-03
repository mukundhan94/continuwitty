package chat

import (
	"context"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestAssembleChatContextPopulatesTokenBudgetAuditForNonEmptyContext(t *testing.T) {
	session := contextSessionRecord()
	mergeCase := contextMergeCase{
		pinnedID:         uuid.MustParse("00000000-0000-0000-0000-000000006801"),
		retrievedID:      uuid.MustParse("00000000-0000-0000-0000-000000006802"),
		pinnedDocumentID: uuid.MustParse("00000000-0000-0000-0000-000000006803"),
		pinnedChunkID:    uuid.MustParse("00000000-0000-0000-0000-000000006804"),
		retrievedChunkID: uuid.MustParse("00000000-0000-0000-0000-000000006805"),
	}
	deps := installContextMergeDependencies(t, mergeCase)

	assembled, err := AssembleChatContext(
		context.Background(),
		contextRequest(session, "what should we do", 4),
		deps,
	)
	if err != nil {
		t.Fatalf("assemble chat context: %v", err)
	}
	if assembled.RetrievalAudit == nil {
		t.Fatalf("expected retrieval audit")
	}
	if assembled.RetrievalAudit.ContextTokenBudget != defaultChatContextTokenBudget {
		t.Fatalf("expected default context token budget %d", defaultChatContextTokenBudget)
	}
	if assembled.RetrievalAudit.ContextTokenEstimate <= 0 {
		t.Fatalf("expected positive token estimate")
	}
}

func TestAssembleChatContextPopulatesTokenBudgetAuditForEmptyContext(t *testing.T) {
	session := contextSessionRecord()
	deps := ChatContextDependencies{
		ListPinnedEngramSummaries: func(
			_ context.Context,
			_ uuid.UUID,
			_ uuid.UUID,
		) ([]models.EngramSummary, error) {
			return []models.EngramSummary{}, nil
		},
		ListPinnedDocuments: func(
			_ context.Context,
			_ uuid.UUID,
			_ uuid.UUID,
		) ([]models.PinnedDocumentRecord, error) {
			return []models.PinnedDocumentRecord{}, nil
		},
		QueryEngrams: func(
			_ context.Context,
			_ uuid.UUID,
			_ models.EngramQueryRequest,
			_ int,
		) ([]models.EngramQueryResult, error) {
			return []models.EngramQueryResult{}, nil
		},
		GetRehydrationBundle: func(
			_ context.Context,
			_ uuid.UUID,
			_ uuid.UUID,
		) (*models.RehydrationBundle, error) {
			return nil, nil
		},
		QueryDocumentChunks: func(
			_ context.Context,
			_ uuid.UUID,
			_ models.DocumentChunkQueryRequest,
			_ int,
		) ([]models.DocumentChunkQueryResult, error) {
			return []models.DocumentChunkQueryResult{}, nil
		},
	}

	assembled, err := AssembleChatContext(
		context.Background(),
		contextRequest(session, "nothing", 4),
		deps,
	)
	if err != nil {
		t.Fatalf("assemble chat context: %v", err)
	}
	if assembled.RetrievalAudit == nil {
		t.Fatalf("expected retrieval audit")
	}
	if assembled.RetrievalAudit.ContextTokenBudget != defaultChatContextTokenBudget {
		t.Fatalf("expected default context token budget %d", defaultChatContextTokenBudget)
	}
	if assembled.RetrievalAudit.ContextTokenEstimate != 0 {
		t.Fatalf("expected zero token estimate for empty context")
	}
	if assembled.RetrievalAudit.ContextTokenTruncated {
		t.Fatalf("expected empty context not to be marked truncated")
	}
}

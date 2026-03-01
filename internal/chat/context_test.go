package chat

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

type contextMergeCase struct {
	pinnedID         uuid.UUID
	retrievedID      uuid.UUID
	pinnedDocumentID uuid.UUID
	pinnedChunkID    uuid.UUID
	retrievedChunkID uuid.UUID
}

func TestAssembleChatContextMergesPinnedAndRetrieved(t *testing.T) {
	session := contextSessionRecord()
	mergeCase := contextMergeCase{
		pinnedID:         uuid.MustParse("00000000-0000-0000-0000-000000006601"),
		retrievedID:      uuid.MustParse("00000000-0000-0000-0000-000000006602"),
		pinnedDocumentID: uuid.MustParse("00000000-0000-0000-0000-000000006603"),
		pinnedChunkID:    uuid.MustParse("00000000-0000-0000-0000-000000006604"),
		retrievedChunkID: uuid.MustParse("00000000-0000-0000-0000-000000006605"),
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

	requireUUIDSliceEqual(
		t,
		[]uuid.UUID{mergeCase.pinnedID, mergeCase.retrievedID},
		assembled.UsedEngramIDs,
	)
	requireUUIDSliceEqual(
		t,
		[]uuid.UUID{mergeCase.pinnedChunkID, mergeCase.retrievedChunkID},
		assembled.UsedDocumentChunkIDs,
	)
	if len(assembled.SourceReferences) != 4 {
		t.Fatalf("expected 4 source references, got %d", len(assembled.SourceReferences))
	}
	requireContains(t, assembled.ContextMarkdown, "Engram Retrieval Context")
	requireContains(t, assembled.ContextMarkdown, "Pinned summary")
	requireContains(t, assembled.ContextMarkdown, "Pinned detailed notes")
	requireContains(t, assembled.ContextMarkdown, "Retrieved summary")
	requireContains(t, assembled.ContextMarkdown, "Pinned Document Context")
	requireContains(t, assembled.ContextMarkdown, "Document Retrieval Context")
}

func TestAssembleChatContextDedupesDuplicateSourceURLs(t *testing.T) {
	session := contextSessionRecord()
	pinnedID := uuid.MustParse("00000000-0000-0000-0000-000000006611")
	retrievedID := uuid.MustParse("00000000-0000-0000-0000-000000006612")
	sharedURL := "https://example.com/shared-runbook"
	deps := ChatContextDependencies{
		ListPinnedEngramSummaries: func(
			_ context.Context,
			_ uuid.UUID,
			actorUserID uuid.UUID,
		) ([]models.EngramSummary, error) {
			return []models.EngramSummary{contextEngramSummary(pinnedID, "Pinned", actorUserID)}, nil
		},
		ListPinnedDocuments: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]models.PinnedDocumentRecord, error) {
			return []models.PinnedDocumentRecord{}, nil
		},
		QueryEngrams: func(
			_ context.Context,
			actorUserID uuid.UUID,
			_ models.EngramQueryRequest,
			_ int,
		) ([]models.EngramQueryResult, error) {
			return []models.EngramQueryResult{contextEngramQueryResult(retrievedID, "Retrieved", actorUserID, 0.1)}, nil
		},
		GetRehydrationBundle: func(_ context.Context, engramID uuid.UUID, _ uuid.UUID) (*models.RehydrationBundle, error) {
			title := "Retrieved"
			if engramID == pinnedID {
				title = "Pinned"
			}
			bundle := contextBundle(engramID, title, &sharedURL)
			return &bundle, nil
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
		contextRequest(session, "what should we do", 4),
		deps,
	)
	if err != nil {
		t.Fatalf("assemble chat context: %v", err)
	}

	requireUUIDSliceEqual(t, []uuid.UUID{pinnedID, retrievedID}, assembled.UsedEngramIDs)
	if len(assembled.SourceReferences) != 1 {
		t.Fatalf("expected one deduped source reference, got %d", len(assembled.SourceReferences))
	}
	if assembled.SourceReferences[0].URL != sharedURL {
		t.Fatalf("expected source reference url %q, got %q", sharedURL, assembled.SourceReferences[0].URL)
	}
}

func TestAssembleChatContextDedupesMultipleChunksFromSameDocument(t *testing.T) {
	session := contextSessionRecord()
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000006621")
	sharedDocumentID := uuid.MustParse("00000000-0000-0000-0000-000000006622")
	deps := dependenciesForDuplicateChunks(engramID, sharedDocumentID)

	assembled, err := AssembleChatContext(
		context.Background(),
		contextRequest(session, "agent workflow", 4),
		deps,
	)
	if err != nil {
		t.Fatalf("assemble chat context: %v", err)
	}

	documentReferences := filterSourceReferencesByType(assembled.SourceReferences, documentSourceType)
	if len(documentReferences) != 1 {
		t.Fatalf("expected one document source reference, got %d", len(documentReferences))
	}
	if documentReferences[0].DocumentID == nil || *documentReferences[0].DocumentID != sharedDocumentID {
		t.Fatalf("expected shared document id %s in source references", sharedDocumentID)
	}
	if documentReferences[0].Title == nil || *documentReferences[0].Title != "AGENT.md" {
		t.Fatalf("expected source title AGENT.md, got %v", documentReferences[0].Title)
	}
}

func TestAssembleChatContextUsesAllPinnedDocuments(t *testing.T) {
	session := contextSessionRecord()
	pinnedDocumentA := uuid.MustParse("00000000-0000-0000-0000-000000006631")
	pinnedDocumentB := uuid.MustParse("00000000-0000-0000-0000-000000006632")
	chunkA := uuid.MustParse("00000000-0000-0000-0000-000000006633")
	chunkB := uuid.MustParse("00000000-0000-0000-0000-000000006634")
	deps := dependenciesForAllPinnedDocuments(
		session.SessionID,
		pinnedDocumentA,
		pinnedDocumentB,
		chunkA,
		chunkB,
	)

	assembled, err := AssembleChatContext(
		context.Background(),
		contextRequest(session, "run checks", 1),
		deps,
	)
	if err != nil {
		t.Fatalf("assemble chat context: %v", err)
	}

	requireUUIDSliceEqual(t, []uuid.UUID{chunkA, chunkB}, assembled.UsedDocumentChunkIDs)
	requireContains(t, assembled.ContextMarkdown, "Pinned Document Context")
	documentReferences := filterSourceReferencesByType(assembled.SourceReferences, documentSourceType)
	referenceDocumentIDs := sourceReferenceDocumentIDs(documentReferences)
	requireUUIDSetEqual(t, []uuid.UUID{pinnedDocumentA, pinnedDocumentB}, referenceDocumentIDs)
}

func TestBundleSectionTruncatesDetailedExcerpt(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000006641")
	bundle := contextBundle(engramID, "Long", nil)
	bundle.DetailedSummaryMarkdown = strings.Repeat("x", 1300)

	section := bundleSection(bundle)

	requireContains(t, section, "Detailed notes excerpt:\n")
	if strings.Contains(section, strings.Repeat("x", 1200)) {
		t.Fatalf("expected detailed notes excerpt to be truncated")
	}
	requireContains(t, section, "...")
}

func TestAssembleChatContextReturnsEmptyWhenNoSources(t *testing.T) {
	session := contextSessionRecord()
	deps := ChatContextDependencies{
		ListPinnedEngramSummaries: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]models.EngramSummary, error) {
			return []models.EngramSummary{}, nil
		},
		ListPinnedDocuments: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]models.PinnedDocumentRecord, error) {
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
		GetRehydrationBundle: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.RehydrationBundle, error) {
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
		contextRequest(session, "no data", 4),
		deps,
	)
	if err != nil {
		t.Fatalf("assemble chat context: %v", err)
	}

	if assembled.ContextMarkdown != "" {
		t.Fatalf("expected empty context markdown, got %q", assembled.ContextMarkdown)
	}
	if len(assembled.UsedEngramIDs) != 0 {
		t.Fatalf("expected no used engram ids, got %d", len(assembled.UsedEngramIDs))
	}
	if len(assembled.UsedDocumentChunkIDs) != 0 {
		t.Fatalf("expected no used document chunk ids, got %d", len(assembled.UsedDocumentChunkIDs))
	}
	if len(assembled.SourceReferences) != 0 {
		t.Fatalf("expected no source references, got %d", len(assembled.SourceReferences))
	}
}

func TestAssembleChatContextUsesDocumentTopKForRetrieval(t *testing.T) {
	session := contextSessionRecord()
	observedTopK := make([]int, 0)
	deps := ChatContextDependencies{
		ListPinnedEngramSummaries: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]models.EngramSummary, error) {
			return []models.EngramSummary{}, nil
		},
		ListPinnedDocuments: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]models.PinnedDocumentRecord, error) {
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
		GetRehydrationBundle: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.RehydrationBundle, error) {
			return nil, nil
		},
		QueryDocumentChunks: func(
			_ context.Context,
			_ uuid.UUID,
			request models.DocumentChunkQueryRequest,
			_ int,
		) ([]models.DocumentChunkQueryResult, error) {
			if len(request.DocumentIDs) == 0 {
				observedTopK = append(observedTopK, request.TopK)
			}
			return []models.DocumentChunkQueryResult{}, nil
		},
	}

	_, err := AssembleChatContext(
		context.Background(),
		contextRequest(session, "top-k check", 2),
		deps,
	)
	if err != nil {
		t.Fatalf("assemble chat context: %v", err)
	}

	if !reflect.DeepEqual(observedTopK, []int{2}) {
		t.Fatalf("expected observed top-k [2], got %v", observedTopK)
	}
}

func TestAssembleChatContextIncludesLinkedEngramsAndTraceMetadataByDefault(t *testing.T) {
	session := contextSessionRecord()
	rootEngramID := uuid.MustParse("00000000-0000-0000-0000-000000006671")
	linkedEngramID := uuid.MustParse("00000000-0000-0000-0000-000000006672")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000006673")
	deps := dependenciesForLinkedRecallDefault(rootEngramID, linkedEngramID, linkID)
	assembled, err := AssembleChatContext(
		context.Background(),
		contextRequest(session, "linked recall", 4),
		deps,
	)
	if err != nil {
		t.Fatalf("assemble chat context: %v", err)
	}
	assertLinkedTraceContext(t, assembled, rootEngramID, linkedEngramID, linkID)
}

func TestAssembleChatContextLinkRecallDisabledSkipsTraversal(t *testing.T) {
	session := contextSessionRecord()
	rootEngramID := uuid.MustParse("00000000-0000-0000-0000-000000006681")
	traversalCalls := 0
	deps := ChatContextDependencies{
		ListPinnedEngramSummaries: func(
			_ context.Context,
			_ uuid.UUID,
			actorUserID uuid.UUID,
		) ([]models.EngramSummary, error) {
			return []models.EngramSummary{contextEngramSummary(rootEngramID, "Root", actorUserID)}, nil
		},
		ListPinnedDocuments: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]models.PinnedDocumentRecord, error) {
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
		GetRehydrationBundle: func(_ context.Context, engramID uuid.UUID, _ uuid.UUID) (*models.RehydrationBundle, error) {
			bundle := contextBundle(engramID, "Root", nil)
			return &bundle, nil
		},
		TraverseEngramLinks: func(
			_ context.Context,
			_ uuid.UUID,
			_ uuid.UUID,
			_ int,
			_ int,
			_ bool,
		) ([]models.EngramLinkTraversalStep, error) {
			traversalCalls++
			return []models.EngramLinkTraversalStep{}, nil
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
	request := contextRequest(session, "disable linked recall", 4)
	request.LinkRecallEnabled = boolPtr(false)

	assembled, err := AssembleChatContext(context.Background(), request, deps)
	if err != nil {
		t.Fatalf("assemble chat context: %v", err)
	}

	if traversalCalls != 0 {
		t.Fatalf("expected traversal not to run when link recall is disabled")
	}
	if len(assembled.UsedEngramLinkIDs) != 0 {
		t.Fatalf("expected no used engram link ids, got %d", len(assembled.UsedEngramLinkIDs))
	}
	if len(assembled.EngramTracePaths) != 0 {
		t.Fatalf("expected no trace paths, got %d", len(assembled.EngramTracePaths))
	}
}

func TestAssembleChatContextLinkRecallBoundsTraversalInputs(t *testing.T) {
	session := contextSessionRecord()
	rootEngramID := uuid.MustParse("00000000-0000-0000-0000-000000006691")
	observedDepth := 0
	observedNeighbors := 0
	deps := ChatContextDependencies{
		ListPinnedEngramSummaries: func(
			_ context.Context,
			_ uuid.UUID,
			actorUserID uuid.UUID,
		) ([]models.EngramSummary, error) {
			return []models.EngramSummary{contextEngramSummary(rootEngramID, "Root", actorUserID)}, nil
		},
		ListPinnedDocuments: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]models.PinnedDocumentRecord, error) {
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
		GetRehydrationBundle: func(_ context.Context, engramID uuid.UUID, _ uuid.UUID) (*models.RehydrationBundle, error) {
			bundle := contextBundle(engramID, "Root", nil)
			return &bundle, nil
		},
		TraverseEngramLinks: func(
			_ context.Context,
			_ uuid.UUID,
			_ uuid.UUID,
			maxDepth int,
			maxNeighbors int,
			_ bool,
		) ([]models.EngramLinkTraversalStep, error) {
			observedDepth = maxDepth
			observedNeighbors = maxNeighbors
			return []models.EngramLinkTraversalStep{}, nil
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
	request := contextRequest(session, "bounded traversal", 4)
	request.LinkRecallDepth = contextIntPtr(99)
	request.LinkRecallMaxNeighbors = contextIntPtr(999)

	_, err := AssembleChatContext(context.Background(), request, deps)
	if err != nil {
		t.Fatalf("assemble chat context: %v", err)
	}

	if observedDepth != maxLinkRecallDepth {
		t.Fatalf("expected bounded depth %d, got %d", maxLinkRecallDepth, observedDepth)
	}
	if observedNeighbors != maxLinkRecallMaxNeighbors {
		t.Fatalf("expected bounded max neighbors %d, got %d", maxLinkRecallMaxNeighbors, observedNeighbors)
	}
}

func installContextMergeDependencies(t *testing.T, mergeCase contextMergeCase) ChatContextDependencies {
	t.Helper()
	return ChatContextDependencies{
		ListPinnedEngramSummaries: func(
			_ context.Context,
			_ uuid.UUID,
			actorUserID uuid.UUID,
		) ([]models.EngramSummary, error) {
			return []models.EngramSummary{contextEngramSummary(mergeCase.pinnedID, "Pinned", actorUserID)}, nil
		},
		ListPinnedDocuments: func(
			_ context.Context,
			_ uuid.UUID,
			actorUserID uuid.UUID,
		) ([]models.PinnedDocumentRecord, error) {
			return mergeCasePinnedDocuments(actorUserID, mergeCase), nil
		},
		QueryEngrams: func(
			_ context.Context,
			actorUserID uuid.UUID,
			_ models.EngramQueryRequest,
			_ int,
		) ([]models.EngramQueryResult, error) {
			return []models.EngramQueryResult{
				contextEngramQueryResult(mergeCase.retrievedID, "Retrieved", actorUserID, 0.1),
				contextEngramQueryResult(mergeCase.pinnedID, "Pinned", actorUserID, 0.2),
			}, nil
		},
		GetRehydrationBundle: func(_ context.Context, engramID uuid.UUID, _ uuid.UUID) (*models.RehydrationBundle, error) {
			title := mergeCaseBundleTitle(engramID, mergeCase.pinnedID)
			bundle := contextBundle(engramID, title, nil)
			return &bundle, nil
		},
		QueryDocumentChunks: mergeCaseQueryDocumentChunks(t, mergeCase),
	}
}

func dependenciesForDuplicateChunks(
	engramID uuid.UUID,
	sharedDocumentID uuid.UUID,
) ChatContextDependencies {
	return ChatContextDependencies{
		ListPinnedEngramSummaries: func(
			_ context.Context,
			_ uuid.UUID,
			actorUserID uuid.UUID,
		) ([]models.EngramSummary, error) {
			return []models.EngramSummary{contextEngramSummary(engramID, "Pinned", actorUserID)}, nil
		},
		ListPinnedDocuments: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]models.PinnedDocumentRecord, error) {
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
		GetRehydrationBundle: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.RehydrationBundle, error) {
			bundle := contextBundle(engramID, "Pinned", nil)
			return &bundle, nil
		},
		QueryDocumentChunks: func(
			_ context.Context,
			_ uuid.UUID,
			_ models.DocumentChunkQueryRequest,
			_ int,
		) ([]models.DocumentChunkQueryResult, error) {
			return duplicateDocumentChunks(sharedDocumentID), nil
		},
	}
}

func duplicateDocumentChunks(sharedDocumentID uuid.UUID) []models.DocumentChunkQueryResult {
	now := time.Now().UTC()
	sourceName := "AGENT.md"
	return []models.DocumentChunkQueryResult{
		{
			ChunkID:         uuid.MustParse("00000000-0000-0000-0000-000000006623"),
			DocumentID:      sharedDocumentID,
			ProjectID:       "project-chat",
			Title:           "AGENT",
			SourceName:      &sourceName,
			ChunkIndex:      0,
			Snippet:         "chunk 0",
			CreatedAt:       now,
			VisibilityScope: models.VisibilityScopeProject,
			Distance:        0.1,
		},
		{
			ChunkID:         uuid.MustParse("00000000-0000-0000-0000-000000006624"),
			DocumentID:      sharedDocumentID,
			ProjectID:       "project-chat",
			Title:           "AGENT",
			SourceName:      &sourceName,
			ChunkIndex:      1,
			Snippet:         "chunk 1",
			CreatedAt:       now,
			VisibilityScope: models.VisibilityScopeProject,
			Distance:        0.2,
		},
	}
}

func dependenciesForAllPinnedDocuments(
	sessionID uuid.UUID,
	pinnedDocumentA uuid.UUID,
	pinnedDocumentB uuid.UUID,
	chunkA uuid.UUID,
	chunkB uuid.UUID,
) ChatContextDependencies {
	return ChatContextDependencies{
		ListPinnedEngramSummaries: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]models.EngramSummary, error) {
			return []models.EngramSummary{}, nil
		},
		ListPinnedDocuments: func(
			_ context.Context,
			_ uuid.UUID,
			actorUserID uuid.UUID,
		) ([]models.PinnedDocumentRecord, error) {
			return pinnedDocumentsForSession(sessionID, actorUserID, pinnedDocumentA, pinnedDocumentB), nil
		},
		QueryEngrams: func(
			_ context.Context,
			_ uuid.UUID,
			_ models.EngramQueryRequest,
			_ int,
		) ([]models.EngramQueryResult, error) {
			return []models.EngramQueryResult{}, nil
		},
		GetRehydrationBundle: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.RehydrationBundle, error) {
			return nil, nil
		},
		QueryDocumentChunks: func(
			_ context.Context,
			_ uuid.UUID,
			request models.DocumentChunkQueryRequest,
			_ int,
		) ([]models.DocumentChunkQueryResult, error) {
			return pinnedDocumentChunkResults(request, pinnedDocumentA, pinnedDocumentB, chunkA, chunkB), nil
		},
	}
}

func pinnedDocumentsForSession(
	sessionID uuid.UUID,
	actorUserID uuid.UUID,
	pinnedDocumentA uuid.UUID,
	pinnedDocumentB uuid.UUID,
) []models.PinnedDocumentRecord {
	now := time.Now().UTC()
	return []models.PinnedDocumentRecord{
		{SessionID: sessionID, DocumentID: pinnedDocumentA, PinnedByUserID: actorUserID, CreatedAt: now},
		{SessionID: sessionID, DocumentID: pinnedDocumentB, PinnedByUserID: actorUserID, CreatedAt: now},
	}
}

func pinnedDocumentChunkResults(
	request models.DocumentChunkQueryRequest,
	pinnedDocumentA uuid.UUID,
	pinnedDocumentB uuid.UUID,
	chunkA uuid.UUID,
	chunkB uuid.UUID,
) []models.DocumentChunkQueryResult {
	now := time.Now().UTC()
	sourceA := "doc-a.md"
	sourceB := "doc-b.md"
	switch {
	case reflect.DeepEqual(request.DocumentIDs, []uuid.UUID{pinnedDocumentA}):
		return []models.DocumentChunkQueryResult{
			{
				ChunkID:         chunkA,
				DocumentID:      pinnedDocumentA,
				ProjectID:       "project-chat",
				Title:           "Doc A",
				SourceName:      &sourceA,
				ChunkIndex:      0,
				Snippet:         "doc a",
				CreatedAt:       now,
				VisibilityScope: models.VisibilityScopeProject,
				Distance:        0.3,
			},
		}
	case reflect.DeepEqual(request.DocumentIDs, []uuid.UUID{pinnedDocumentB}):
		return []models.DocumentChunkQueryResult{
			{
				ChunkID:         chunkB,
				DocumentID:      pinnedDocumentB,
				ProjectID:       "project-chat",
				Title:           "Doc B",
				SourceName:      &sourceB,
				ChunkIndex:      1,
				Snippet:         "doc b",
				CreatedAt:       now,
				VisibilityScope: models.VisibilityScopeProject,
				Distance:        0.4,
			},
		}
	default:
		return []models.DocumentChunkQueryResult{}
	}
}

func mergeCasePinnedDocuments(actorUserID uuid.UUID, mergeCase contextMergeCase) []models.PinnedDocumentRecord {
	now := time.Now().UTC()
	return []models.PinnedDocumentRecord{
		{
			SessionID:      uuid.MustParse("00000000-0000-0000-0000-000000006600"),
			DocumentID:     mergeCase.pinnedDocumentID,
			PinnedByUserID: actorUserID,
			CreatedAt:      now,
		},
	}
}

func mergeCaseBundleTitle(engramID uuid.UUID, pinnedID uuid.UUID) string {
	if engramID == pinnedID {
		return "Pinned"
	}
	return "Retrieved"
}

func mergeCaseQueryDocumentChunks(
	t *testing.T,
	mergeCase contextMergeCase,
) func(context.Context, uuid.UUID, models.DocumentChunkQueryRequest, int) ([]models.DocumentChunkQueryResult, error) {
	t.Helper()
	return func(
		_ context.Context,
		_ uuid.UUID,
		request models.DocumentChunkQueryRequest,
		_ int,
	) ([]models.DocumentChunkQueryResult, error) {
		if len(request.DocumentIDs) > 0 {
			return mergeCasePinnedChunkResult(t, request.DocumentIDs, mergeCase), nil
		}
		return mergeCaseRetrievedChunkResult(mergeCase), nil
	}
}

func mergeCasePinnedChunkResult(
	t *testing.T,
	documentIDs []uuid.UUID,
	mergeCase contextMergeCase,
) []models.DocumentChunkQueryResult {
	t.Helper()
	expected := []uuid.UUID{mergeCase.pinnedDocumentID}
	if !reflect.DeepEqual(documentIDs, expected) {
		t.Fatalf("expected scoped request document ids %v, got %v", expected, documentIDs)
	}
	now := time.Now().UTC()
	sourceName := "runbook.md"
	return []models.DocumentChunkQueryResult{
		{
			ChunkID:         mergeCase.pinnedChunkID,
			DocumentID:      mergeCase.pinnedDocumentID,
			ProjectID:       "project-chat",
			Title:           "Pinned Runbook",
			SourceName:      &sourceName,
			ChunkIndex:      0,
			Snippet:         "Pinned runbook excerpt for immediate response context.",
			CreatedAt:       now,
			VisibilityScope: models.VisibilityScopeProject,
			Distance:        0.1,
		},
	}
}

func mergeCaseRetrievedChunkResult(mergeCase contextMergeCase) []models.DocumentChunkQueryResult {
	now := time.Now().UTC()
	sourceName := "incident.md"
	return []models.DocumentChunkQueryResult{
		{
			ChunkID:         mergeCase.retrievedChunkID,
			DocumentID:      uuid.MustParse("00000000-0000-0000-0000-000000006606"),
			ProjectID:       "project-chat",
			Title:           "Retrieved Incident Notes",
			SourceName:      &sourceName,
			ChunkIndex:      2,
			Snippet:         "Escalate when queue depth remains high for 20 minutes.",
			CreatedAt:       now,
			VisibilityScope: models.VisibilityScopeProject,
			Distance:        0.2,
		},
	}
}

func contextSessionRecord() models.ChatSessionRecord {
	now := time.Now().UTC()
	return models.ChatSessionRecord{
		SessionID:       uuid.MustParse("00000000-0000-0000-0000-000000006650"),
		OwnerUserID:     uuid.MustParse("00000000-0000-0000-0000-000000006651"),
		ProjectID:       "project-chat",
		Title:           "Session",
		Provider:        models.ChatProviderOpenAI,
		ModelID:         "gpt-4o-mini",
		SystemPrompt:    "",
		VisibilityScope: models.VisibilityScopePrivate,
		AutosaveEnabled: false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func contextRequest(session models.ChatSessionRecord, query string, documentTopK int) ChatContextRequest {
	return ChatContextRequest{
		Session:      session,
		ActorUserID:  session.OwnerUserID,
		UserQuery:    query,
		EmbeddingDim: 256,
		DocumentTopK: documentTopK,
	}
}

func dependenciesForLinkedRecallDefault(
	rootEngramID uuid.UUID,
	linkedEngramID uuid.UUID,
	linkID uuid.UUID,
) ChatContextDependencies {
	return ChatContextDependencies{
		ListPinnedEngramSummaries: func(
			_ context.Context,
			_ uuid.UUID,
			actorUserID uuid.UUID,
		) ([]models.EngramSummary, error) {
			return []models.EngramSummary{contextEngramSummary(rootEngramID, "Root", actorUserID)}, nil
		},
		ListPinnedDocuments: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]models.PinnedDocumentRecord, error) {
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
		GetRehydrationBundle: func(_ context.Context, engramID uuid.UUID, _ uuid.UUID) (*models.RehydrationBundle, error) {
			title := "Root"
			if engramID == linkedEngramID {
				title = "Linked"
			}
			bundle := contextBundle(engramID, title, nil)
			return &bundle, nil
		},
		TraverseEngramLinks: func(
			_ context.Context,
			rootID uuid.UUID,
			_ uuid.UUID,
			_ int,
			_ int,
			_ bool,
		) ([]models.EngramLinkTraversalStep, error) {
			if rootID != rootEngramID {
				return []models.EngramLinkTraversalStep{}, nil
			}
			return []models.EngramLinkTraversalStep{
				{
					Depth: 1,
					Link:  contextLinkRecord(linkID, rootEngramID, linkedEngramID),
				},
			}, nil
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
}

func assertLinkedTraceContext(
	t *testing.T,
	assembled AssembledChatContext,
	rootEngramID uuid.UUID,
	linkedEngramID uuid.UUID,
	linkID uuid.UUID,
) {
	t.Helper()
	requireUUIDSliceEqual(t, []uuid.UUID{rootEngramID, linkedEngramID}, assembled.UsedEngramIDs)
	requireUUIDSliceEqual(t, []uuid.UUID{linkID}, assembled.UsedEngramLinkIDs)
	if len(assembled.EngramTracePaths) != 1 {
		t.Fatalf("expected one trace path, got %d", len(assembled.EngramTracePaths))
	}
	trace := assembled.EngramTracePaths[0]
	requireUUIDSliceEqual(t, []uuid.UUID{rootEngramID, linkedEngramID}, trace.EngramIDs)
	requireUUIDSliceEqual(t, []uuid.UUID{linkID}, trace.LinkIDs)
	if trace.Depth != 1 {
		t.Fatalf("expected trace depth 1, got %d", trace.Depth)
	}
	requireContains(t, assembled.ContextMarkdown, "Linked summary")
}

func contextLinkRecord(
	linkID uuid.UUID,
	sourceEngramID uuid.UUID,
	targetEngramID uuid.UUID,
) models.EngramLinkRecord {
	now := time.Now().UTC()
	return models.EngramLinkRecord{
		LinkID:         linkID,
		ProjectID:      "project-chat",
		SourceEngramID: sourceEngramID,
		TargetEngramID: targetEngramID,
		RelationType:   models.EngramLinkRelationSupports,
		Weight:         0.91,
		TemporalWeight: 0.86,
		Confidence:     0.88,
		Origin:         models.EngramLinkOriginManual,
		Status:         models.EngramLinkStatusActive,
		EvidenceJSON:   map[string]any{"reason": "linked for continuity"},
		CreatedByUserID: uuid.MustParse(
			"00000000-0000-0000-0000-000000006699",
		),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func contextBundle(engramID uuid.UUID, title string, citationURL *string) models.RehydrationBundle {
	now := time.Now().UTC()
	url := fmtOrFallbackURL(citationURL, title)
	sourceTitle := title + " Source"
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000006652")
	return models.RehydrationBundle{
		EngramID:                engramID,
		ProjectID:               "project-chat",
		Title:                   title,
		CompactSummary:          title + " summary",
		DetailedSummaryMarkdown: title + " detailed notes",
		KeyDecisions: []map[string]any{
			{"decision": title + " decision", "rationale": "because"},
		},
		OpenQuestions:   []string{title + " question"},
		TopCitations:    []models.RehydrationCitation{{URL: url, Title: &sourceTitle, Snippet: title + " snippet", CapturedAt: now}},
		ContextMarkdown: "ctx",
		OwnerUserID:     &ownerUserID,
		VisibilityScope: "private",
	}
}

func fmtOrFallbackURL(url *string, title string) string {
	if url != nil {
		return *url
	}
	return "https://example.com/" + strings.ToLower(title)
}

func contextEngramSummary(engramID uuid.UUID, title string, actorUserID uuid.UUID) models.EngramSummary {
	actor := actorUserID
	threadID := "thread-1"
	return models.EngramSummary{
		EngramID:        engramID,
		ProjectID:       "project-chat",
		ThreadID:        &threadID,
		Title:           title,
		Abstract:        title + " abstract",
		CreatedAt:       time.Now().UTC(),
		Tags:            []string{},
		Keywords:        []string{},
		OwnerUserID:     &actor,
		VisibilityScope: "private",
	}
}

func contextEngramQueryResult(
	engramID uuid.UUID,
	title string,
	actorUserID uuid.UUID,
	distance float64,
) models.EngramQueryResult {
	actor := actorUserID
	return models.EngramQueryResult{
		EngramID:        engramID,
		ProjectID:       "project-chat",
		Title:           title,
		Abstract:        title + " abstract",
		CreatedAt:       time.Now().UTC(),
		Tags:            []string{},
		Keywords:        []string{},
		OwnerUserID:     &actor,
		VisibilityScope: "private",
		Distance:        distance,
	}
}

func filterSourceReferencesByType(
	references []ChatSourceReference,
	sourceType string,
) []ChatSourceReference {
	filtered := make([]ChatSourceReference, 0)
	for _, reference := range references {
		if reference.SourceType != sourceType {
			continue
		}
		filtered = append(filtered, reference)
	}
	return filtered
}

func sourceReferenceDocumentIDs(references []ChatSourceReference) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(references))
	for _, reference := range references {
		if reference.DocumentID == nil {
			continue
		}
		ids = append(ids, *reference.DocumentID)
	}
	return ids
}

func requireContains(t *testing.T, value string, expectedSubstring string) {
	t.Helper()
	if !strings.Contains(value, expectedSubstring) {
		t.Fatalf("expected %q to contain %q", value, expectedSubstring)
	}
}

func requireUUIDSliceEqual(t *testing.T, expected []uuid.UUID, actual []uuid.UUID) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected uuid slice %v, got %v", expected, actual)
	}
}

func requireUUIDSetEqual(t *testing.T, expected []uuid.UUID, actual []uuid.UUID) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Fatalf("expected %d uuids, got %d", len(expected), len(actual))
	}
	expectedSet := make(map[uuid.UUID]struct{}, len(expected))
	for _, item := range expected {
		expectedSet[item] = struct{}{}
	}
	for _, item := range actual {
		if _, exists := expectedSet[item]; !exists {
			t.Fatalf("unexpected uuid in set: %s", item)
		}
	}
}

func boolPtr(value bool) *bool {
	copy := value
	return &copy
}

func contextIntPtr(value int) *int {
	copy := value
	return &copy
}

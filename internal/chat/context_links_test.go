package chat

import (
	"context"
	"testing"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestRankEngramContextCandidatesEnsuresLinkedCoverageInTopWindow(t *testing.T) {
	seedA := uuid.MustParse("00000000-0000-0000-0000-000000006701")
	seedB := uuid.MustParse("00000000-0000-0000-0000-000000006702")
	seedC := uuid.MustParse("00000000-0000-0000-0000-000000006703")
	linked := uuid.MustParse("00000000-0000-0000-0000-000000006704")

	ranked := rankEngramContextCandidates(
		[]uuid.UUID{seedA, seedB, seedC},
		[]uuid.UUID{linked},
		map[uuid.UUID]float64{
			seedA: 0.99,
			seedB: 0.95,
			seedC: 0.92,
		},
		[]EngramTracePath{
			{
				RootEngramID:   seedA,
				TargetEngramID: linked,
				Depth:          1,
				LinkIDs:        []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000006705")},
				EngramIDs:      []uuid.UUID{seedA, linked},
				Score:          0.93,
			},
		},
		2,
	)
	if len(ranked) < 2 {
		t.Fatalf("expected at least two ranked candidates, got %d", len(ranked))
	}
	if !containsUUID(ranked[:2], linked) {
		t.Fatalf("expected linked candidate in top selection window, got %v", ranked[:2])
	}
}

func TestAssembleChatContextBackfillsWhenLinkedBundleUnavailable(t *testing.T) {
	session := contextSessionRecord()
	rootEngramID := uuid.MustParse("00000000-0000-0000-0000-000000006711")
	fallbackSeedID := uuid.MustParse("00000000-0000-0000-0000-000000006712")
	linkedEngramID := uuid.MustParse("00000000-0000-0000-0000-000000006713")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000006714")

	deps := dependenciesForUnavailableLinkedBundle(rootEngramID, fallbackSeedID, linkedEngramID, linkID)
	request := contextRequest(session, "federated fallback", 2)
	request.MaxEngrams = 2

	assembled, err := AssembleChatContext(context.Background(), request, deps)
	if err != nil {
		t.Fatalf("assemble chat context: %v", err)
	}

	requireUUIDSliceEqual(t, []uuid.UUID{rootEngramID, fallbackSeedID}, assembled.UsedEngramIDs)
	if containsUUID(assembled.UsedEngramIDs, linkedEngramID) {
		t.Fatalf("expected linked engram to be skipped when bundle lookup is unavailable")
	}
	if len(assembled.UsedEngramLinkIDs) != 0 {
		t.Fatalf("expected no linked ids when linked candidate is not packed, got %v", assembled.UsedEngramLinkIDs)
	}
	if len(assembled.EngramTracePaths) != 0 {
		t.Fatalf("expected no trace paths when linked candidate is not packed, got %d", len(assembled.EngramTracePaths))
	}
	if assembled.RetrievalAudit == nil {
		t.Fatalf("expected retrieval audit metadata")
	}
	requireEqualAnyRuntime(t, 1, assembled.RetrievalAudit.BlockedEngramCandidateCount)
	requireEqualAnyRuntime(t, 0, assembled.RetrievalAudit.CrossProjectEngramCount)
}

func TestAssembleChatContextRetrievalAuditCapturesCrossProjectUsage(t *testing.T) {
	session := contextSessionRecord()
	rootEngramID := uuid.MustParse("00000000-0000-0000-0000-000000006721")
	linkedEngramID := uuid.MustParse("00000000-0000-0000-0000-000000006722")
	linkID := uuid.MustParse("00000000-0000-0000-0000-000000006723")

	deps := dependenciesForLinkedRecallDefault(rootEngramID, linkedEngramID, linkID)
	deps.GetRehydrationBundle = func(
		_ context.Context,
		engramID uuid.UUID,
		_ uuid.UUID,
	) (*models.RehydrationBundle, error) {
		switch engramID {
		case rootEngramID:
			bundle := contextBundle(rootEngramID, "Root", nil)
			bundle.ProjectID = session.ProjectID
			return &bundle, nil
		case linkedEngramID:
			bundle := contextBundle(linkedEngramID, "Linked", nil)
			bundle.ProjectID = "project-federated"
			return &bundle, nil
		default:
			return nil, nil
		}
	}
	request := contextRequest(session, "cross project trace", 2)
	request.MaxEngrams = 2

	assembled, err := AssembleChatContext(context.Background(), request, deps)
	if err != nil {
		t.Fatalf("assemble chat context: %v", err)
	}

	if assembled.RetrievalAudit == nil {
		t.Fatalf("expected retrieval audit metadata")
	}
	requireEqualAnyRuntime(t, 1, assembled.RetrievalAudit.CrossProjectEngramCount)
	requireEqualAnyRuntime(t, 1, assembled.RetrievalAudit.CrossProjectTracePathCount)
	requireEqualAnyRuntime(t, []string{"project-federated"}, assembled.RetrievalAudit.CrossProjectProjectIDs)
}

func dependenciesForUnavailableLinkedBundle(
	rootEngramID uuid.UUID,
	fallbackSeedID uuid.UUID,
	linkedEngramID uuid.UUID,
	linkID uuid.UUID,
) ChatContextDependencies {
	return ChatContextDependencies{
		ListPinnedEngramSummaries: unavailableLinkedPinnedSummaries(rootEngramID),
		ListPinnedDocuments: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]models.PinnedDocumentRecord, error) {
			return []models.PinnedDocumentRecord{}, nil
		},
		QueryEngrams: unavailableLinkedQuery(fallbackSeedID),
		GetRehydrationBundle: unavailableLinkedBundleLookup(
			rootEngramID,
			fallbackSeedID,
			linkedEngramID,
		),
		TraverseEngramLinks: unavailableLinkedTraversal(rootEngramID, linkedEngramID, linkID),
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

func unavailableLinkedPinnedSummaries(
	rootEngramID uuid.UUID,
) func(context.Context, uuid.UUID, uuid.UUID) ([]models.EngramSummary, error) {
	return func(
		_ context.Context,
		_ uuid.UUID,
		actorUserID uuid.UUID,
	) ([]models.EngramSummary, error) {
		return []models.EngramSummary{
			contextEngramSummary(rootEngramID, "Root", actorUserID),
		}, nil
	}
}

func unavailableLinkedQuery(
	fallbackSeedID uuid.UUID,
) func(context.Context, uuid.UUID, models.EngramQueryRequest, int) ([]models.EngramQueryResult, error) {
	return func(
		_ context.Context,
		actorUserID uuid.UUID,
		_ models.EngramQueryRequest,
		_ int,
	) ([]models.EngramQueryResult, error) {
		return []models.EngramQueryResult{
			contextEngramQueryResult(fallbackSeedID, "Fallback", actorUserID, 1.4),
		}, nil
	}
}

func unavailableLinkedBundleLookup(
	rootEngramID uuid.UUID,
	fallbackSeedID uuid.UUID,
	linkedEngramID uuid.UUID,
) func(context.Context, uuid.UUID, uuid.UUID) (*models.RehydrationBundle, error) {
	return func(
		_ context.Context,
		engramID uuid.UUID,
		_ uuid.UUID,
	) (*models.RehydrationBundle, error) {
		if engramID == linkedEngramID {
			return nil, nil
		}
		title := "Root"
		if engramID == fallbackSeedID {
			title = "Fallback"
		}
		if engramID != rootEngramID && engramID != fallbackSeedID {
			return nil, nil
		}
		bundle := contextBundle(engramID, title, nil)
		return &bundle, nil
	}
}

func unavailableLinkedTraversal(
	rootEngramID uuid.UUID,
	linkedEngramID uuid.UUID,
	linkID uuid.UUID,
) func(context.Context, uuid.UUID, uuid.UUID, int, int, bool) ([]models.EngramLinkTraversalStep, error) {
	return func(
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
				Link:  contextLinkRecord(linkID, rootEngramID, linkedEngramID, models.EngramLinkRelationSupports),
			},
		}, nil
	}
}

package chat

import (
	"context"
	"strings"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func dependenciesForLinkedRecallDefault(
	rootEngramID uuid.UUID,
	linkedEngramID uuid.UUID,
	linkID uuid.UUID,
) ChatContextDependencies {
	return dependenciesForLinkedRecallWithRelation(
		rootEngramID,
		linkedEngramID,
		linkID,
		models.EngramLinkRelationSupports,
	)
}

func dependenciesForLinkedRecallWithRelation(
	rootEngramID uuid.UUID,
	linkedEngramID uuid.UUID,
	linkID uuid.UUID,
	relationType models.EngramLinkRelationType,
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
					Link:  contextLinkRecord(linkID, rootEngramID, linkedEngramID, relationType),
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

type linkedTraceExpectation struct {
	rootEngramID   uuid.UUID
	linkedEngramID uuid.UUID
	linkID         uuid.UUID
}

func assertLinkedTraceContext(
	t *testing.T,
	assembled AssembledChatContext,
	expectation linkedTraceExpectation,
) {
	t.Helper()
	requireUUIDSliceEqual(t, []uuid.UUID{expectation.rootEngramID, expectation.linkedEngramID}, assembled.UsedEngramIDs)
	requireUUIDSliceEqual(t, []uuid.UUID{expectation.linkID}, assembled.UsedEngramLinkIDs)
	if len(assembled.EngramTracePaths) != 1 {
		t.Fatalf("expected one trace path, got %d", len(assembled.EngramTracePaths))
	}
	trace := assembled.EngramTracePaths[0]
	requireUUIDSliceEqual(t, []uuid.UUID{expectation.rootEngramID, expectation.linkedEngramID}, trace.EngramIDs)
	requireUUIDSliceEqual(t, []uuid.UUID{expectation.linkID}, trace.LinkIDs)
	if trace.Depth != 1 {
		t.Fatalf("expected trace depth 1, got %d", trace.Depth)
	}
	requireContains(t, assembled.ContextMarkdown, "Linked summary")
}

func contextLinkRecord(
	linkID uuid.UUID,
	sourceEngramID uuid.UUID,
	targetEngramID uuid.UUID,
	relationType models.EngramLinkRelationType,
) models.EngramLinkRecord {
	now := time.Now().UTC()
	return models.EngramLinkRecord{
		LinkID:         linkID,
		ProjectID:      "project-chat",
		SourceEngramID: sourceEngramID,
		TargetEngramID: targetEngramID,
		RelationType:   relationType,
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

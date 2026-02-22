package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestGetRehydrationBundleReturnsNilWhenNotFound(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000301")
	db := &fakeQueryer{
		queryRowResult: &fakeRow{err: pgx.ErrNoRows},
	}

	bundle, err := GetRehydrationBundle(
		context.Background(),
		db,
		RehydrationInput{EngramID: engramID},
	)
	requireNoError(t, err)
	if bundle != nil {
		t.Fatalf("expected nil bundle when engram not found")
	}
}

func TestGetRehydrationBundleBuildsContextWithVisibilityFilter(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000302")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000399")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000388")
	detailed := "# Chat Session Snapshot\n\n## ASSISTANT (ts)\nPrimary cause was DB CPU saturation."
	title := "Source"
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{
				engramID,
				"engram-vault",
				"Checkpoint Summary",
				"Snapshot from active chat session.",
				map[string]any{
					"detailed_summary_markdown": detailed,
					"decisions": []any{
						map[string]any{
							"decision":  "Use snapshots",
							"rationale": "Improves continuity",
						},
					},
					"open_questions": []any{"Need retention policy?"},
				},
				detailed,
				ownerUserID,
				"project",
			},
		},
		queryRowsResult: &fakeRows{
			values: [][]any{
				{
					"https://example.com/source",
					title,
					"Key evidence.",
					time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	bundle, err := GetRehydrationBundle(
		context.Background(),
		db,
		RehydrationInput{
			EngramID:    engramID,
			ActorUserID: &actorUserID,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, bundle)
	requireEqual(t, engramID, bundle.EngramID)
	requireEqual(t, "project", bundle.VisibilityScope)
	if !strings.Contains(bundle.ContextMarkdown, "## Key Decisions\n- Use snapshots: Improves continuity") {
		t.Fatalf("expected decisions section in context markdown, got %q", bundle.ContextMarkdown)
	}
	if !strings.Contains(bundle.ContextMarkdown, "## Top Citations\n- Source (https://example.com/source): Key evidence.") {
		t.Fatalf("expected citations section in context markdown, got %q", bundle.ContextMarkdown)
	}
	requireEqual(t, 1, len(db.queryRowSQL))
	if !strings.Contains(db.queryRowSQL[0], "owner_user_id = $2 OR visibility_scope = 'project' OR owner_user_id IS NULL") {
		t.Fatalf("expected visibility filter in rehydration query, got %q", db.queryRowSQL[0])
	}
}

func TestGetEngramSourcesReturnsRowsWhenVisible(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000303")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000399")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000388")
	title := "Primary source"
	snippet := "Snippet"
	sourceID := uuid.MustParse("00000000-0000-0000-0000-000000000355")
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{
				engramID,
				"engram-vault",
				"Checkpoint Summary",
				"Summary",
				map[string]any{},
				"",
				ownerUserID,
				"project",
			},
		},
		queryRowsResult: &fakeRows{
			values: [][]any{
				{
					sourceID,
					engramID,
					time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
					"https://example.com/source",
					title,
					snippet,
				},
			},
		},
	}

	sources, err := GetEngramSources(context.Background(), db, engramID, 100, &actorUserID)
	requireNoError(t, err)
	requireEqual(t, 1, len(sources))
	requireEqual(t, sourceID, sources[0].SourceID)
	requireEqual(t, "https://example.com/source", sources[0].URL)
	requireEqual(t, 1, len(db.querySQL))
	if !strings.Contains(db.querySQL[0], "SELECT source_id, engram_id, captured_at, url, title, snippet") {
		t.Fatalf("expected source query shape, got %q", db.querySQL[0])
	}
}

func TestGetEngramSourcesReturnsEmptyWhenNotVisible(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000304")
	db := &fakeQueryer{
		queryRowResult: &fakeRow{err: pgx.ErrNoRows},
	}

	sources, err := GetEngramSources(context.Background(), db, engramID, 100, nil)
	requireNoError(t, err)
	requireEqual(t, 0, len(sources))
}

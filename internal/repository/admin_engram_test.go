package repository

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestListAdminEngramsBuildsFiltersAndSearch(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000d01")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000d02")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000d03")
	threadID := "thread-admin"
	createdAt := time.Date(2026, 2, 22, 18, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{
			adminEngramRowValues(adminEngramRowFixture{
				EngramID:                engramID,
				ProjectID:               "project-docs",
				ThreadID:                &threadID,
				Title:                   "Queue pressure snapshot",
				Abstract:                "Updated abstract",
				DetailedSummaryMarkdown: "markdown",
				Tags:                    []string{"ops"},
				Keywords:                []string{"queue"},
				OwnerUserID:             &ownerUserID,
				VisibilityScope:         "private",
				SourceSessionID:         &sessionID,
				CreatedAt:               createdAt,
				UpdatedAt:               createdAt,
			}),
		}},
	}
	projectID := "project-docs"
	queryText := "queue"

	records, err := ListAdminEngrams(
		context.Background(),
		db,
		AdminEngramListInput{
			IncludeDeleted: false,
			ProjectID:      &projectID,
			SessionID:      &sessionID,
			OwnerUserID:    &ownerUserID,
			QueryText:      &queryText,
			Limit:          25,
			Offset:         5,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(records))
	requireEqual(t, models.VisibilityScopePrivate, records[0].VisibilityScope)
	requireEqual(t, 0, len(records[0].Sources))

	query := db.querySQL[0]
	if !strings.Contains(query, "e.deleted_at IS NULL") {
		t.Fatalf("expected deleted filter, got %q", query)
	}
	if !strings.Contains(query, "e.project_id = $1") {
		t.Fatalf("expected project filter, got %q", query)
	}
	if !strings.Contains(query, "e.source_session_id = $2") {
		t.Fatalf("expected session filter, got %q", query)
	}
	if !strings.Contains(query, "e.owner_user_id = $3") {
		t.Fatalf("expected owner filter, got %q", query)
	}
	if !strings.Contains(query, "e.engram_markdown ILIKE $6") {
		t.Fatalf("expected query_text filter, got %q", query)
	}
	expectedArgs := []any{projectID, sessionID, ownerUserID, "%queue%", "%queue%", "%queue%", 25, 5}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestListAdminEngramSourcesReturnsRows(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000d11")
	sourceID := uuid.MustParse("00000000-0000-0000-0000-000000000d12")
	capturedAt := time.Date(2026, 2, 22, 18, 5, 0, 0, time.UTC)
	title := "Source title"
	snippet := "snippet text"
	contentText := "full content"
	contentHash := "hash-a"
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{
			adminEngramSourceRowValues(adminEngramSourceRowFixture{
				SourceID:    sourceID,
				CapturedAt:  capturedAt,
				URL:         "https://example.com/a",
				Title:       &title,
				Snippet:     &snippet,
				ContentText: &contentText,
				ContentHash: &contentHash,
			}),
			adminEngramSourceRowValues(adminEngramSourceRowFixture{
				SourceID:   uuid.MustParse("00000000-0000-0000-0000-000000000d13"),
				CapturedAt: capturedAt,
				URL:        "https://example.com/b",
			}),
		}},
	}

	records, err := ListAdminEngramSources(context.Background(), db, engramID)
	requireNoError(t, err)
	requireEqual(t, 2, len(records))
	requireEqual(t, sourceID, records[0].SourceID)
	requireEqual(t, "https://example.com/b", records[1].URL)
	if records[1].Title != nil {
		t.Fatalf("expected nil title for second source")
	}
	if !reflect.DeepEqual([]any{engramID}, db.queryArgs[0]) {
		t.Fatalf("expected source query arg %s", engramID)
	}
}

func TestGetAdminEngramReturnsNilWhenMissing(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}

	record, err := GetAdminEngram(
		context.Background(),
		db,
		uuid.MustParse("00000000-0000-0000-0000-000000000d21"),
		false,
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil record when missing")
	}
	requireEqual(t, 0, len(db.querySQL))
}

func TestGetAdminEngramHydratesSources(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000d31")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000d32")
	createdAt := time.Date(2026, 2, 22, 18, 10, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: adminEngramRowValues(adminEngramRowFixture{
			EngramID:                engramID,
			ProjectID:               "project-docs",
			Title:                   "Admin engram",
			Abstract:                "Abstract",
			DetailedSummaryMarkdown: "Markdown",
			Tags:                    []string{"tag"},
			Keywords:                []string{"keyword"},
			OwnerUserID:             &ownerUserID,
			VisibilityScope:         "project",
			CreatedAt:               createdAt,
			UpdatedAt:               createdAt,
		})},
		queryRowsResult: &fakeRows{values: [][]any{
			adminEngramSourceRowValues(adminEngramSourceRowFixture{
				SourceID:   uuid.MustParse("00000000-0000-0000-0000-000000000d33"),
				CapturedAt: createdAt,
				URL:        "https://example.com/source",
			}),
		}},
	}

	record, err := GetAdminEngram(context.Background(), db, engramID, false)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, engramID, record.EngramID)
	requireEqual(t, models.VisibilityScopeProject, record.VisibilityScope)
	requireEqual(t, 1, len(record.Sources))
	if !strings.Contains(db.queryRowSQL[0], "AND e.deleted_at IS NULL") {
		t.Fatalf("expected non-deleted filter in get query")
	}
	requireEqual(t, 1, len(db.querySQL))
	if !strings.Contains(db.querySQL[0], "FROM sources") {
		t.Fatalf("expected source lookup query")
	}
}

func TestMoveAdminEngramProjectReturnsUpdatedRecordAndRunsDetachQuery(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000d41")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000d42")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000d43")
	createdAt := time.Date(2026, 2, 22, 18, 15, 0, 0, time.UTC)
	targetProjectID := "project-target"
	db := &fakeQueryer{
		queryRowResults: []*fakeRow{
			{values: []any{engramID}},
			{values: adminEngramRowValues(adminEngramRowFixture{
				EngramID:                engramID,
				ProjectID:               targetProjectID,
				Title:                   "Moved engram",
				Abstract:                "Abstract",
				DetailedSummaryMarkdown: "Markdown",
				Tags:                    []string{"tag"},
				Keywords:                []string{"keyword"},
				OwnerUserID:             &ownerUserID,
				VisibilityScope:         "private",
				CreatedAt:               createdAt,
				UpdatedAt:               createdAt,
			})},
		},
		queryRowsResults: []*fakeRows{
			{values: [][]any{}},
			{values: [][]any{
				adminEngramSourceRowValues(adminEngramSourceRowFixture{
					SourceID:   uuid.MustParse("00000000-0000-0000-0000-000000000d44"),
					CapturedAt: createdAt,
					URL:        "https://example.com/source",
				}),
			}},
		},
	}

	record, err := MoveAdminEngramProject(
		context.Background(),
		db,
		AdminEngramMoveProjectInput{
			EngramID:        engramID,
			TargetProjectID: targetProjectID,
			ActorUserID:     actorUserID,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, targetProjectID, record.ProjectID)
	requireEqual(t, 2, len(db.queryRowSQL))
	requireEqual(t, 2, len(db.querySQL))
	if !strings.Contains(db.querySQL[0], "DELETE FROM engram_collection_items") {
		t.Fatalf("expected collection detach query")
	}
	expectedUpdateArgs := []any{targetProjectID, actorUserID, engramID}
	if !reflect.DeepEqual(expectedUpdateArgs, db.queryRowArgs[0]) {
		t.Fatalf("expected move args %#v, got %#v", expectedUpdateArgs, db.queryRowArgs[0])
	}
	if !reflect.DeepEqual([]any{engramID}, db.queryArgs[0]) {
		t.Fatalf("expected detach args %#v, got %#v", []any{engramID}, db.queryArgs[0])
	}
}

func TestMoveAdminEngramProjectReturnsNilWhenNotFound(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}

	record, err := MoveAdminEngramProject(
		context.Background(),
		db,
		AdminEngramMoveProjectInput{
			EngramID:        uuid.MustParse("00000000-0000-0000-0000-000000000d51"),
			TargetProjectID: "project-target",
			ActorUserID:     uuid.MustParse("00000000-0000-0000-0000-000000000d52"),
		},
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil record when engram missing")
	}
	requireEqual(t, 0, len(db.querySQL))
}

func TestSoftDeleteEngramReturnsBool(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000d61")
	deletedBy := uuid.MustParse("00000000-0000-0000-0000-000000000d62")
	reason := "cleanup"
	db := &fakeQueryer{queryRowResult: &fakeRow{values: []any{engramID}}}

	deleted, err := SoftDeleteEngram(
		context.Background(),
		db,
		AdminEngramSoftDeleteInput{
			EngramID:        engramID,
			DeletedByUserID: deletedBy,
			Reason:          &reason,
		},
	)
	requireNoError(t, err)
	if !deleted {
		t.Fatalf("expected soft-delete true")
	}
	expectedArgs := []any{deletedBy, &reason, engramID}
	if !reflect.DeepEqual(expectedArgs, db.queryRowArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}

	missingDB := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
	deleted, err = SoftDeleteEngram(
		context.Background(),
		missingDB,
		AdminEngramSoftDeleteInput{EngramID: engramID, DeletedByUserID: deletedBy},
	)
	requireNoError(t, err)
	if deleted {
		t.Fatalf("expected false when engram missing")
	}
}

func TestRestoreEngramReturnsBool(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000d71")
	db := &fakeQueryer{queryRowResult: &fakeRow{values: []any{engramID}}}

	restored, err := RestoreEngram(context.Background(), db, engramID)
	requireNoError(t, err)
	if !restored {
		t.Fatalf("expected restore true")
	}

	missingDB := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
	restored, err = RestoreEngram(context.Background(), missingDB, engramID)
	requireNoError(t, err)
	if restored {
		t.Fatalf("expected false when engram missing")
	}
}

type adminEngramRowFixture struct {
	EngramID                uuid.UUID
	ProjectID               string
	ThreadID                *string
	Title                   string
	Abstract                string
	DetailedSummaryMarkdown string
	Tags                    []string
	Keywords                []string
	OwnerUserID             *uuid.UUID
	VisibilityScope         string
	SourceSessionID         *uuid.UUID
	CreatedAt               time.Time
	UpdatedAt               time.Time
	DeletedAt               *time.Time
	DeletedByUserID         *uuid.UUID
	DeleteReason            *string
}

func adminEngramRowValues(fixture adminEngramRowFixture) []any {
	var threadIDValue any
	if fixture.ThreadID != nil {
		threadIDValue = *fixture.ThreadID
	}
	var ownerUserIDValue any
	if fixture.OwnerUserID != nil {
		ownerUserIDValue = *fixture.OwnerUserID
	}
	var sourceSessionIDValue any
	if fixture.SourceSessionID != nil {
		sourceSessionIDValue = *fixture.SourceSessionID
	}
	var deletedAtValue any
	if fixture.DeletedAt != nil {
		deletedAtValue = *fixture.DeletedAt
	}
	var deletedByUserIDValue any
	if fixture.DeletedByUserID != nil {
		deletedByUserIDValue = *fixture.DeletedByUserID
	}
	var deleteReasonValue any
	if fixture.DeleteReason != nil {
		deleteReasonValue = *fixture.DeleteReason
	}
	return []any{
		fixture.EngramID,
		fixture.ProjectID,
		threadIDValue,
		fixture.Title,
		fixture.Abstract,
		fixture.DetailedSummaryMarkdown,
		fixture.Tags,
		fixture.Keywords,
		ownerUserIDValue,
		fixture.VisibilityScope,
		sourceSessionIDValue,
		fixture.CreatedAt,
		fixture.UpdatedAt,
		deletedAtValue,
		deletedByUserIDValue,
		deleteReasonValue,
	}
}

type adminEngramSourceRowFixture struct {
	SourceID    uuid.UUID
	CapturedAt  time.Time
	URL         string
	Title       *string
	Snippet     *string
	ContentText *string
	ContentHash *string
}

func adminEngramSourceRowValues(fixture adminEngramSourceRowFixture) []any {
	var titleValue any
	if fixture.Title != nil {
		titleValue = *fixture.Title
	}
	var snippetValue any
	if fixture.Snippet != nil {
		snippetValue = *fixture.Snippet
	}
	var contentTextValue any
	if fixture.ContentText != nil {
		contentTextValue = *fixture.ContentText
	}
	var contentHashValue any
	if fixture.ContentHash != nil {
		contentHashValue = *fixture.ContentHash
	}
	return []any{
		fixture.SourceID,
		fixture.CapturedAt,
		fixture.URL,
		titleValue,
		snippetValue,
		contentTextValue,
		contentHashValue,
	}
}

package repository

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestPinEngramToSessionReturnsPinnedRecord(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000701")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000702")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000703")
	createdAt := time.Date(2026, 2, 22, 11, 0, 0, 0, time.UTC)
	auditEventID := uuid.MustParse("00000000-0000-0000-0000-000000000704")
	db := &fakeQueryer{
		queryRowResults: []*fakeRow{
			{values: []any{sessionID, engramID, actorUserID, createdAt}},
			{values: []any{"proj-alpha"}},
			{values: []any{auditEventID, "proj-alpha", actorUserID, "engram.pin", "engram", nil, engramID, []byte(`{}`), createdAt}},
		},
	}

	originalNow := nowChatUTC
	nowChatUTC = func() time.Time { return createdAt }
	t.Cleanup(func() { nowChatUTC = originalNow })

	record, err := PinEngramToSession(
		context.Background(),
		db,
		ChatPinEngramInput{
			SessionID:   sessionID,
			EngramID:    engramID,
			ActorUserID: actorUserID,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, engramID, record.EngramID)
	requireEqual(t, actorUserID, record.PinnedByUserID)

	requireEqual(t, 3, len(db.queryRowSQL))
	query := db.queryRowSQL[0]
	if !strings.Contains(query, "session_pinned_engrams") {
		t.Fatalf("expected engram pin table in query, got %q", query)
	}
	if !strings.Contains(query, "ON CONFLICT (session_id, engram_id)") {
		t.Fatalf("expected conflict upsert clause in query, got %q", query)
	}
	if strings.Contains(query, "%!s(MISSING)") {
		t.Fatalf("expected all actor placeholders to be replaced, got %q", query)
	}
	if !strings.Contains(query, "r.owner_user_id = $4") {
		t.Fatalf("expected resource owner check to use actor placeholder, got %q", query)
	}
	expectedArgs := []any{sessionID, actorUserID, engramID, actorUserID, actorUserID, createdAt}
	if !reflect.DeepEqual(expectedArgs, db.queryRowArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}
}

func TestUnpinEngramFromSessionReturnsTrueWhenRemoved(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000721")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000722")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000723")
	createdAt := time.Date(2026, 2, 22, 11, 5, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResults: []*fakeRow{
			{values: []any{sessionID}},
			{values: []any{"proj-alpha"}},
			{values: []any{
				uuid.MustParse("00000000-0000-0000-0000-000000000724"),
				"proj-alpha",
				actorUserID,
				"engram.unpin",
				"engram",
				nil,
				engramID,
				[]byte(`{}`),
				createdAt,
			}},
		},
	}

	removed, err := UnpinEngramFromSession(
		context.Background(),
		db,
		ChatPinEngramInput{SessionID: sessionID, EngramID: engramID, ActorUserID: actorUserID},
	)
	requireNoError(t, err)
	if !removed {
		t.Fatalf("expected unpin to return true")
	}
	expectedArgs := []any{sessionID, engramID, actorUserID}
	if !reflect.DeepEqual(expectedArgs, db.queryRowArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}
}

func TestPinnedDocumentOperationsReturnZeroValueWhenRowMissing(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "pin",
			run: func(t *testing.T) {
				db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
				record, err := PinDocumentToSession(
					context.Background(),
					db,
					ChatPinDocumentInput{
						SessionID:   uuid.MustParse("00000000-0000-0000-0000-000000000711"),
						DocumentID:  uuid.MustParse("00000000-0000-0000-0000-000000000712"),
						ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000000713"),
					},
				)
				requireNoError(t, err)
				if record != nil {
					t.Fatalf("expected nil pinned document when resource is not visible")
				}
			},
		},
		{
			name: "unpin",
			run: func(t *testing.T) {
				db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
				removed, err := UnpinDocumentFromSession(
					context.Background(),
					db,
					ChatPinDocumentInput{
						SessionID:   uuid.MustParse("00000000-0000-0000-0000-000000000731"),
						DocumentID:  uuid.MustParse("00000000-0000-0000-0000-000000000732"),
						ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000000733"),
					},
				)
				requireNoError(t, err)
				if removed {
					t.Fatalf("expected false when no pin row is removed")
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.run(t)
		})
	}
}

func TestListPinnedEngramsBuildsExpectedQueryAndArgs(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000741")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000742")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000743")
	createdAt := time.Date(2026, 2, 22, 11, 30, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{{sessionID, engramID, actorUserID, createdAt}}},
	}

	records, err := ListPinnedEngrams(
		context.Background(),
		db,
		ChatPinnedListInput{SessionID: sessionID, ActorUserID: actorUserID},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(records))
	requireEqual(t, engramID, records[0].EngramID)

	query := db.querySQL[0]
	if !strings.Contains(query, "FROM session_pinned_engrams") {
		t.Fatalf("expected engram pinned table in query, got %q", query)
	}
	if !strings.Contains(query, "AND 1=1") {
		t.Fatalf("expected no additional resource filter clause, got %q", query)
	}
	expectedArgs := []any{sessionID, actorUserID}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestListPinnedDocumentsBuildsExpectedQueryAndArgs(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000751")
	documentID := uuid.MustParse("00000000-0000-0000-0000-000000000752")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000753")
	createdAt := time.Date(2026, 2, 22, 11, 45, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{{sessionID, documentID, actorUserID, createdAt}}},
	}

	records, err := ListPinnedDocuments(
		context.Background(),
		db,
		ChatPinnedListInput{SessionID: sessionID, ActorUserID: actorUserID},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(records))
	requireEqual(t, documentID, records[0].DocumentID)

	query := db.querySQL[0]
	if !strings.Contains(query, "JOIN documents d") {
		t.Fatalf("expected documents join in query, got %q", query)
	}
	if !strings.Contains(query, "d.owner_user_id = $3") {
		t.Fatalf("expected actor-scoped document visibility clause, got %q", query)
	}
	if strings.Contains(query, "%!s(MISSING)") {
		t.Fatalf("expected all document actor placeholders to be replaced, got %q", query)
	}
	expectedArgs := []any{sessionID, actorUserID, actorUserID}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestListPinnedEngramSummariesAppliesVisibilityFilters(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000761")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000762")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000763")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000764")
	createdAt := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	visibility := "private"
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{
			{
				engramID,
				"project-pin",
				nil,
				"Pinned",
				"summary",
				createdAt,
				[]string{"context"},
				[]string{"pin"},
				ownerUserID,
				visibility,
			},
		}},
	}

	records, err := ListPinnedEngramSummaries(
		context.Background(),
		db,
		ChatPinnedListInput{SessionID: sessionID, ActorUserID: actorUserID},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(records))
	requireEqual(t, engramID, records[0].EngramID)

	query := db.querySQL[0]
	if !strings.Contains(query, "JOIN engrams e") {
		t.Fatalf("expected engram join in query, got %q", query)
	}
	if !strings.Contains(query, "e.deleted_at IS NULL") {
		t.Fatalf("expected engram visibility clause, got %q", query)
	}
	expectedArgs := []any{sessionID, actorUserID, actorUserID}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

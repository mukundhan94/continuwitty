package repository

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRecordEngramAccessEventsSkipsEmptyBatch(t *testing.T) {
	db := &fakeQueryer{}

	err := RecordEngramAccessEvents(context.Background(), db, nil)
	requireNoError(t, err)
	requireEqual(t, 0, len(db.queryRowSQL))
}

func TestRecordEngramAccessEventsWritesEventAndUpdatesCounters(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000881")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000882")
	eventID := uuid.MustParse("00000000-0000-0000-0000-000000000883")
	accessedAt := time.Date(2026, 3, 1, 12, 10, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: []any{true}},
	}

	originalUUIDFactory := newEngramAccessEventUUID
	newEngramAccessEventUUID = func() uuid.UUID { return eventID }
	t.Cleanup(func() { newEngramAccessEventUUID = originalUUIDFactory })

	err := RecordEngramAccessEvents(
		context.Background(),
		db,
		[]EngramAccessEventRecord{
			{
				EngramID:     engramID,
				SessionID:    &sessionID,
				AccessSource: EngramAccessSourceChatSend,
				AccessedAt:   accessedAt,
			},
		},
	)
	requireNoError(t, err)

	requireEqual(t, 1, len(db.queryRowSQL))
	sql := db.queryRowSQL[0]
	if !strings.Contains(sql, "INSERT INTO engram_access_events") {
		t.Fatalf("expected access event insert statement, got %q", sql)
	}
	if !strings.Contains(sql, "UPDATE engrams") {
		t.Fatalf("expected engram aggregate update statement, got %q", sql)
	}

	requireEqual(t, 1, len(db.queryRowArgs))
	expectedArgs := []any{
		eventID,
		engramID,
		&sessionID,
		"chat_send",
		accessedAt,
	}
	if !reflect.DeepEqual(expectedArgs, db.queryRowArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}
}

func TestRecordEngramAccessEventsDefaultsSourceAndTimestamp(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000891")
	eventID := uuid.MustParse("00000000-0000-0000-0000-000000000892")
	now := time.Date(2026, 3, 1, 13, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: []any{true}},
	}

	originalUUIDFactory := newEngramAccessEventUUID
	newEngramAccessEventUUID = func() uuid.UUID { return eventID }
	t.Cleanup(func() { newEngramAccessEventUUID = originalUUIDFactory })

	originalNow := nowEngramAccessUTC
	nowEngramAccessUTC = func() time.Time { return now }
	t.Cleanup(func() { nowEngramAccessUTC = originalNow })

	err := RecordEngramAccessEvents(
		context.Background(),
		db,
		[]EngramAccessEventRecord{
			{
				EngramID: engramID,
			},
		},
	)
	requireNoError(t, err)

	expectedArgs := []any{
		eventID,
		engramID,
		(*uuid.UUID)(nil),
		"unknown",
		now,
	}
	if !reflect.DeepEqual(expectedArgs, db.queryRowArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}
}

func TestRecordEngramAccessEventsRejectsMissingEngramID(t *testing.T) {
	db := &fakeQueryer{}

	err := RecordEngramAccessEvents(
		context.Background(),
		db,
		[]EngramAccessEventRecord{
			{EngramID: uuid.Nil},
		},
	)
	if !errors.Is(err, errEngramAccessEngramIDRequired) {
		t.Fatalf("expected missing engram id error, got %v", err)
	}
	requireEqual(t, 0, len(db.queryRowSQL))
}

package repository

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestRecordEngramFeedbackPersistsFeedbackAndReturnsUpdatedCounters(t *testing.T) {
	feedbackID := uuid.MustParse("00000000-0000-0000-0000-000000000a01")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000a02")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000a03")
	createdAt := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{
				feedbackID,
				engramID,
				actorUserID,
				string(models.EngramFeedbackTypeUseful),
				"helpful in triage",
				createdAt,
				4,
				1,
			},
		},
	}

	originalIDFactory := newEngramFeedbackUUID
	newEngramFeedbackUUID = func() uuid.UUID { return feedbackID }
	t.Cleanup(func() { newEngramFeedbackUUID = originalIDFactory })

	note := "helpful in triage"
	record, err := RecordEngramFeedback(
		context.Background(),
		db,
		EngramFeedbackCreateInput{
			EngramID:     engramID,
			ActorUserID:  actorUserID,
			FeedbackType: models.EngramFeedbackTypeUseful,
			Note:         &note,
			CreatedAt:    createdAt,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, feedbackID, record.FeedbackID)
	requireEqual(t, engramID, record.EngramID)
	requireEqual(t, actorUserID, record.ActorUserID)
	requireEqual(t, models.EngramFeedbackTypeUseful, record.FeedbackType)
	requireEqual(t, "helpful in triage", record.Note)
	requireEqual(t, 4, record.UsefulCount)
	requireEqual(t, 1, record.ContradictionCount)

	requireEqual(t, 1, len(db.queryRowSQL))
	query := db.queryRowSQL[0]
	if !strings.Contains(query, "INSERT INTO engram_feedback") {
		t.Fatalf("expected feedback insert statement, got %q", query)
	}
	if !strings.Contains(query, "UPDATE engrams") {
		t.Fatalf("expected aggregate update statement, got %q", query)
	}
	if !strings.Contains(query, "visibility_scope = 'project'") {
		t.Fatalf("expected visibility enforcement in query, got %q", query)
	}
	expectedArgs := []any{
		feedbackID,
		engramID,
		actorUserID,
		"useful",
		"helpful in triage",
		createdAt,
	}
	if !reflect.DeepEqual(expectedArgs, db.queryRowArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}
}

func TestRecordEngramFeedbackDefaultsTimestampAndTrimsNote(t *testing.T) {
	feedbackID := uuid.MustParse("00000000-0000-0000-0000-000000000a11")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000a12")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000a13")
	now := time.Date(2026, 3, 2, 9, 30, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{
				feedbackID,
				engramID,
				actorUserID,
				string(models.EngramFeedbackTypeContradiction),
				"mismatch",
				now,
				2,
				3,
			},
		},
	}

	originalIDFactory := newEngramFeedbackUUID
	newEngramFeedbackUUID = func() uuid.UUID { return feedbackID }
	t.Cleanup(func() { newEngramFeedbackUUID = originalIDFactory })

	originalNow := nowEngramFeedbackUTC
	nowEngramFeedbackUTC = func() time.Time { return now }
	t.Cleanup(func() { nowEngramFeedbackUTC = originalNow })

	note := "  mismatch  "
	record, err := RecordEngramFeedback(
		context.Background(),
		db,
		EngramFeedbackCreateInput{
			EngramID:     engramID,
			ActorUserID:  actorUserID,
			FeedbackType: models.EngramFeedbackTypeContradiction,
			Note:         &note,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, "mismatch", record.Note)
	requireEqual(t, now, record.CreatedAt)
}

func TestRecordEngramFeedbackNormalizesBlankNoteToNil(t *testing.T) {
	feedbackID := uuid.MustParse("00000000-0000-0000-0000-000000000a16")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000a17")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000a18")
	now := time.Date(2026, 3, 2, 9, 40, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{
				feedbackID,
				engramID,
				actorUserID,
				string(models.EngramFeedbackTypeUseful),
				"",
				now,
				1,
				0,
			},
		},
	}

	originalIDFactory := newEngramFeedbackUUID
	newEngramFeedbackUUID = func() uuid.UUID { return feedbackID }
	t.Cleanup(func() { newEngramFeedbackUUID = originalIDFactory })

	originalNow := nowEngramFeedbackUTC
	nowEngramFeedbackUTC = func() time.Time { return now }
	t.Cleanup(func() { nowEngramFeedbackUTC = originalNow })

	note := "   "
	record, err := RecordEngramFeedback(
		context.Background(),
		db,
		EngramFeedbackCreateInput{
			EngramID:     engramID,
			ActorUserID:  actorUserID,
			FeedbackType: models.EngramFeedbackTypeUseful,
			Note:         &note,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, "", record.Note)

	requireEqual(t, 1, len(db.queryRowArgs))
	requireEqual(t, nil, db.queryRowArgs[0][4])
}

func TestRecordEngramFeedbackReturnsNilWhenEngramNotVisible(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
	record, err := RecordEngramFeedback(
		context.Background(),
		db,
		EngramFeedbackCreateInput{
			EngramID:     uuid.MustParse("00000000-0000-0000-0000-000000000a21"),
			ActorUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000a22"),
			FeedbackType: models.EngramFeedbackTypeUseful,
		},
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil feedback record when engram is not visible")
	}
}

func TestRecordEngramFeedbackValidation(t *testing.T) {
	testCases := []struct {
		name          string
		input         EngramFeedbackCreateInput
		expectErr     error
		expectMessage string
	}{
		{
			name:      "missing engram id",
			input:     EngramFeedbackCreateInput{ActorUserID: uuid.New(), FeedbackType: models.EngramFeedbackTypeUseful},
			expectErr: errEngramFeedbackEngramIDRequired,
		},
		{
			name:      "missing actor user id",
			input:     EngramFeedbackCreateInput{EngramID: uuid.New(), FeedbackType: models.EngramFeedbackTypeUseful},
			expectErr: errEngramFeedbackActorIDRequired,
		},
		{
			name:      "missing feedback type",
			input:     EngramFeedbackCreateInput{EngramID: uuid.New(), ActorUserID: uuid.New()},
			expectErr: errEngramFeedbackTypeRequired,
		},
		{
			name: "invalid feedback type",
			input: EngramFeedbackCreateInput{
				EngramID:     uuid.New(),
				ActorUserID:  uuid.New(),
				FeedbackType: models.EngramFeedbackType("invalid"),
			},
			expectMessage: "unsupported engram feedback type",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			db := &fakeQueryer{}
			record, err := RecordEngramFeedback(context.Background(), db, testCase.input)
			if err == nil {
				t.Fatalf("expected error")
			}
			if record != nil {
				t.Fatalf("expected nil record on validation error")
			}
			assertFeedbackValidationError(t, err, testCase.expectErr, testCase.expectMessage)
			requireEqual(t, 0, len(db.queryRowSQL))
		})
	}
}

func assertFeedbackValidationError(
	t *testing.T,
	err error,
	expectedErr error,
	expectedMessage string,
) {
	t.Helper()
	if expectedErr != nil {
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected error %v, got %v", expectedErr, err)
		}
		return
	}
	if expectedMessage == "" {
		return
	}
	if !strings.Contains(err.Error(), expectedMessage) {
		t.Fatalf("expected error message to contain %q, got %v", expectedMessage, err)
	}
}

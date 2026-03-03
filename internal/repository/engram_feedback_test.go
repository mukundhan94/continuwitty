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
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000a04")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000a03")
	createdAt := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{
				feedbackID,
				engramID,
				sessionID,
				actorUserID,
				string(models.EngramFeedbackTypeUseful),
				string(models.EngramFeedbackIntegrationDepthElaborated),
				"helpful in triage",
				5,
				createdAt,
				4,
				1,
				6,
				4.5,
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
			EngramID:         engramID,
			SessionID:        &sessionID,
			ActorUserID:      actorUserID,
			FeedbackType:     models.EngramFeedbackTypeUseful,
			IntegrationDepth: feedbackIntegrationDepthPtr(models.EngramFeedbackIntegrationDepthElaborated),
			Note:             &note,
			RelevanceScore:   intPtr(5),
			CreatedAt:        createdAt,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	assertPersistedFeedbackRecord(
		t,
		record,
		persistedFeedbackExpectation{
			feedbackID:         feedbackID,
			engramID:           engramID,
			sessionID:          sessionID,
			actorUserID:        actorUserID,
			feedbackType:       models.EngramFeedbackTypeUseful,
			integrationDepth:   models.EngramFeedbackIntegrationDepthElaborated,
			note:               "helpful in triage",
			relevanceScore:     5,
			usefulCount:        4,
			contradictionCount: 1,
			feedbackCount:      6,
			avgRelevance:       4.5,
		},
	)

	expectedArgs := []any{
		feedbackID,
		engramID,
		sessionID,
		actorUserID,
		"useful",
		"elaborated",
		"helpful in triage",
		5,
		createdAt,
	}
	assertRecordEngramFeedbackQuery(t, db, expectedArgs)
}

func TestRecordEngramFeedbackNormalizesNoteAndDefaultsTimestamp(t *testing.T) {
	now := time.Date(2026, 3, 2, 9, 30, 0, 0, time.UTC)
	testCases := []feedbackNormalizationCase{
		{
			name:            "trimmed note",
			feedbackID:      uuid.MustParse("00000000-0000-0000-0000-000000000a11"),
			engramID:        uuid.MustParse("00000000-0000-0000-0000-000000000a12"),
			actorUserID:     uuid.MustParse("00000000-0000-0000-0000-000000000a13"),
			feedbackType:    models.EngramFeedbackTypeContradiction,
			note:            "  mismatch  ",
			expectedNote:    "mismatch",
			expectedSQLNote: "mismatch",
		},
		{
			name:            "blank note becomes nil SQL value",
			feedbackID:      uuid.MustParse("00000000-0000-0000-0000-000000000a16"),
			engramID:        uuid.MustParse("00000000-0000-0000-0000-000000000a17"),
			actorUserID:     uuid.MustParse("00000000-0000-0000-0000-000000000a18"),
			feedbackType:    models.EngramFeedbackTypeUseful,
			note:            "   ",
			expectedNote:    "",
			expectedSQLNote: nil,
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			assertFeedbackNoteNormalizationCase(t, testCase, now)
		})
	}
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
			name: "invalid relevance score",
			input: EngramFeedbackCreateInput{
				EngramID:       uuid.New(),
				ActorUserID:    uuid.New(),
				FeedbackType:   models.EngramFeedbackTypeUseful,
				RelevanceScore: intPtr(9),
			},
			expectErr: errEngramFeedbackRelevanceInvalid,
		},
		{
			name: "invalid integration depth",
			input: EngramFeedbackCreateInput{
				EngramID:         uuid.New(),
				ActorUserID:      uuid.New(),
				FeedbackType:     models.EngramFeedbackTypeUseful,
				IntegrationDepth: feedbackIntegrationDepthPtr(models.EngramFeedbackIntegrationDepth("invalid")),
			},
			expectMessage: "unsupported engram feedback integration depth",
		},
		{
			name: "invalid session id",
			input: EngramFeedbackCreateInput{
				EngramID:     uuid.New(),
				SessionID:    uuidPointer(uuid.Nil),
				ActorUserID:  uuid.New(),
				FeedbackType: models.EngramFeedbackTypeUseful,
			},
			expectErr: errEngramFeedbackSessionIDInvalid,
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

func assertRecordEngramFeedbackQuery(
	t *testing.T,
	db *fakeQueryer,
	expectedArgs []any,
) {
	t.Helper()
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
	if !reflect.DeepEqual(expectedArgs, db.queryRowArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}
}

type feedbackNormalizationCase struct {
	name            string
	feedbackID      uuid.UUID
	engramID        uuid.UUID
	actorUserID     uuid.UUID
	feedbackType    models.EngramFeedbackType
	note            string
	expectedNote    string
	expectedSQLNote any
}

func assertFeedbackNoteNormalizationCase(
	t *testing.T,
	testCase feedbackNormalizationCase,
	now time.Time,
) {
	t.Helper()
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{
				testCase.feedbackID,
				testCase.engramID,
				nil,
				testCase.actorUserID,
				string(testCase.feedbackType),
				nil,
				testCase.expectedNote,
				nil,
				now,
				2,
				3,
				5,
				nil,
			},
		},
	}

	originalIDFactory := newEngramFeedbackUUID
	newEngramFeedbackUUID = func() uuid.UUID { return testCase.feedbackID }
	t.Cleanup(func() { newEngramFeedbackUUID = originalIDFactory })

	originalNow := nowEngramFeedbackUTC
	nowEngramFeedbackUTC = func() time.Time { return now }
	t.Cleanup(func() { nowEngramFeedbackUTC = originalNow })

	note := testCase.note
	record, err := RecordEngramFeedback(
		context.Background(),
		db,
		EngramFeedbackCreateInput{
			EngramID:     testCase.engramID,
			ActorUserID:  testCase.actorUserID,
			FeedbackType: testCase.feedbackType,
			Note:         &note,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, testCase.expectedNote, record.Note)
	requireEqual(t, now, record.CreatedAt)
	if record.RelevanceScore != nil {
		t.Fatalf("expected relevance score to be nil when omitted")
	}
	requireEqual(t, nil, db.queryRowArgs[0][5])
	requireEqual(t, testCase.expectedSQLNote, db.queryRowArgs[0][6])
	requireEqual(t, nil, db.queryRowArgs[0][7])
}

func intPtr(value int) *int {
	return &value
}

func uuidPointer(value uuid.UUID) *uuid.UUID {
	return &value
}

func feedbackIntegrationDepthPtr(
	value models.EngramFeedbackIntegrationDepth,
) *models.EngramFeedbackIntegrationDepth {
	return &value
}

type persistedFeedbackExpectation struct {
	feedbackID         uuid.UUID
	engramID           uuid.UUID
	sessionID          uuid.UUID
	actorUserID        uuid.UUID
	feedbackType       models.EngramFeedbackType
	integrationDepth   models.EngramFeedbackIntegrationDepth
	note               string
	relevanceScore     int
	usefulCount        int
	contradictionCount int
	feedbackCount      int
	avgRelevance       float64
}

func assertPersistedFeedbackRecord(
	t *testing.T,
	record *models.EngramFeedbackRecord,
	expected persistedFeedbackExpectation,
) {
	t.Helper()
	requireEqual(t, expected.feedbackID, record.FeedbackID)
	requireEqual(t, expected.engramID, record.EngramID)
	if record.SessionID == nil {
		t.Fatalf("expected session id in feedback record")
	}
	requireEqual(t, expected.sessionID, *record.SessionID)
	requireEqual(t, expected.actorUserID, record.ActorUserID)
	requireEqual(t, expected.feedbackType, record.FeedbackType)
	if record.IntegrationDepth == nil {
		t.Fatalf("expected integration depth in feedback record")
	}
	requireEqual(t, expected.integrationDepth, *record.IntegrationDepth)
	requireEqual(t, expected.note, record.Note)
	if record.RelevanceScore == nil {
		t.Fatalf("expected relevance score in feedback record")
	}
	requireEqual(t, expected.relevanceScore, *record.RelevanceScore)
	requireEqual(t, expected.usefulCount, record.UsefulCount)
	requireEqual(t, expected.contradictionCount, record.ContradictionCount)
	requireEqual(t, expected.feedbackCount, record.FeedbackCount)
	if record.AvgRelevanceFeedback == nil {
		t.Fatalf("expected avg relevance feedback in feedback record")
	}
	requireEqual(t, expected.avgRelevance, *record.AvgRelevanceFeedback)
}

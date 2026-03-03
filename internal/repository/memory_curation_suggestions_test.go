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

func TestCreateMemoryCurationSuggestionUsesDefaults(t *testing.T) {
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000d101")
	suggestedAt := time.Date(2026, 3, 3, 16, 0, 0, 0, time.UTC)
	updatedAt := suggestedAt
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{
				suggestionID,
				"engram-vault",
				nil,
				"auto_save",
				"Session ended with unresolved action items",
				"save concise session summary",
				[]byte(`{"priority":"high"}`),
				0.81,
				"suggested",
				suggestedAt,
				updatedAt,
				nil,
				nil,
			},
		},
	}
	originalUUID := newMemoryCurationSuggestionUUID
	originalNow := nowMemoryCurationSuggestionUTC
	newMemoryCurationSuggestionUUID = func() uuid.UUID { return suggestionID }
	nowMemoryCurationSuggestionUTC = func() time.Time { return suggestedAt }
	t.Cleanup(func() {
		newMemoryCurationSuggestionUUID = originalUUID
		nowMemoryCurationSuggestionUTC = originalNow
	})

	result, err := CreateMemoryCurationSuggestion(
		context.Background(),
		db,
		MemoryCurationSuggestionCreateInput{
			ProjectID:       "engram-vault",
			SuggestionType:  models.MemoryCurationSuggestionTypeAutoSave,
			Reason:          "Session ended with unresolved action items",
			Recommendation:  "save concise session summary",
			PayloadJSON:     map[string]any{"priority": "high"},
			ConfidenceScore: 0.81,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, result)
	requireEqual(t, suggestionID, result.SuggestionID)
	requireEqual(t, models.MemoryCurationSuggestionStatusSuggested, result.Status)
	requireEqual(t, models.MemoryCurationSuggestionTypeAutoSave, result.SuggestionType)

	requireEqual(t, 1, len(db.queryRowSQL))
	if !strings.Contains(db.queryRowSQL[0], "INSERT INTO memory_curation_suggestions") {
		t.Fatalf("expected insert SQL, got %q", db.queryRowSQL[0])
	}
	requireEqual(t, 1, len(db.queryRowArgs))
	requireEqual(t, "engram-vault", db.queryRowArgs[0][1])
	requireEqual(t, "auto_save", db.queryRowArgs[0][3])
	requireEqual(t, "suggested", db.queryRowArgs[0][8])
}

func TestListMemoryCurationSuggestionsUsesFilters(t *testing.T) {
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000d201")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-00000000d202")
	suggestedAt := time.Date(2026, 3, 3, 16, 10, 0, 0, time.UTC)
	updatedAt := suggestedAt.Add(2 * time.Minute)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{
			values: [][]any{
				{
					suggestionID,
					"engram-vault",
					sessionID,
					"link",
					"Related engrams discovered",
					"create link suggestion",
					[]byte(`{"target_engram_id":"00000000-0000-0000-0000-00000000d299"}`),
					0.74,
					"suggested",
					suggestedAt,
					updatedAt,
					nil,
					nil,
				},
			},
		},
	}
	projectID := "engram-vault"
	suggestionType := models.MemoryCurationSuggestionTypeLink
	status := models.MemoryCurationSuggestionStatusSuggested

	results, err := ListMemoryCurationSuggestions(
		context.Background(),
		db,
		MemoryCurationSuggestionListInput{
			ProjectID:      &projectID,
			SessionID:      &sessionID,
			SuggestionType: &suggestionType,
			Status:         &status,
			Limit:          10,
			Offset:         3,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(results))
	requireEqual(t, suggestionID, results[0].SuggestionID)
	requireEqual(t, sessionID, *results[0].SessionID)
	requireEqual(t, models.MemoryCurationSuggestionTypeLink, results[0].SuggestionType)

	requireEqual(t, 1, len(db.queryArgs))
	expectedArgs := []any{"engram-vault", sessionID, "link", "suggested", 10, 3}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestApplyMemoryCurationSuggestionActionUsesRequestObject(t *testing.T) {
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000d301")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-00000000d302")
	projectID := "engram-vault"
	suggestedAt := time.Date(2026, 3, 3, 16, 20, 0, 0, time.UTC)
	actionedAt := time.Date(2026, 3, 3, 16, 25, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{
				suggestionID,
				"engram-vault",
				nil,
				"contradiction",
				"Contradicting memory detected",
				"review contradiction before save",
				[]byte(`{"severity":"high"}`),
				0.93,
				"applied",
				suggestedAt,
				actionedAt,
				actionedAt,
				actorUserID,
			},
		},
	}

	result, err := ApplyMemoryCurationSuggestionAction(
		context.Background(),
		db,
		MemoryCurationSuggestionActionInput{
			SuggestionID: suggestionID,
			ProjectID:    &projectID,
			Status:       models.MemoryCurationSuggestionStatusApplied,
			ActorUserID:  actorUserID,
			ActionedAt:   actionedAt,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, result)
	requireEqual(t, models.MemoryCurationSuggestionStatusApplied, result.Status)
	if result.ActionTakenBy == nil || *result.ActionTakenBy != actorUserID {
		t.Fatalf("expected action_taken_by to be set")
	}
	requireEqual(t, 1, len(db.queryRowArgs))
	expectedArgs := []any{suggestionID, "applied", actionedAt, actorUserID, "engram-vault"}
	if !reflect.DeepEqual(expectedArgs, db.queryRowArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}
}

func TestApplyMemoryCurationSuggestionActionRejectsInvalidStatus(t *testing.T) {
	_, err := ApplyMemoryCurationSuggestionAction(
		context.Background(),
		&fakeQueryer{},
		MemoryCurationSuggestionActionInput{
			SuggestionID: uuid.MustParse("00000000-0000-0000-0000-00000000d311"),
			Status:       models.MemoryCurationSuggestionStatusSuggested,
			ActorUserID:  uuid.MustParse("00000000-0000-0000-0000-00000000d312"),
		},
	)
	if !errors.Is(err, errMemoryCurationActionStatusInvalid) {
		t.Fatalf("expected errMemoryCurationActionStatusInvalid, got %v", err)
	}
}

func TestApplyMemoryCurationSuggestionActionReturnsNilWhenSuggestionMissing(t *testing.T) {
	db := &fakeQueryer{
		queryRowResult: &fakeRow{err: pgx.ErrNoRows},
	}
	result, err := ApplyMemoryCurationSuggestionAction(
		context.Background(),
		db,
		MemoryCurationSuggestionActionInput{
			SuggestionID: uuid.MustParse("00000000-0000-0000-0000-00000000d321"),
			Status:       models.MemoryCurationSuggestionStatusRejected,
			ActorUserID:  uuid.MustParse("00000000-0000-0000-0000-00000000d322"),
		},
	)
	requireNoError(t, err)
	if result != nil {
		t.Fatalf("expected nil result for missing suggestion")
	}
}

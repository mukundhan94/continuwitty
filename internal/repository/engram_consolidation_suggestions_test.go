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
)

func TestRefreshExactDuplicateConsolidationSuggestionsUsesDefaults(t *testing.T) {
	now := time.Date(2026, 3, 3, 14, 0, 0, 0, time.UTC)
	groupIDOne := uuid.MustParse("00000000-0000-0000-0000-00000000c101")
	groupIDTwo := uuid.MustParse("00000000-0000-0000-0000-00000000c102")
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{
			values: [][]any{
				{
					"engram-vault",
					"incident summary",
					[]uuid.UUID{groupIDOne, groupIDTwo},
				},
			},
		},
		queryRowResult: &fakeRow{
			values: []any{true},
		},
	}
	originalNow := nowConsolidationSuggestionUTC
	nowConsolidationSuggestionUTC = func() time.Time { return now }
	t.Cleanup(func() { nowConsolidationSuggestionUTC = originalNow })

	result, err := RefreshExactDuplicateConsolidationSuggestions(
		context.Background(),
		db,
		ConsolidationSuggestionRefreshInput{},
	)
	requireNoError(t, err)
	requireEqual(t, defaultConsolidationMinGroupSize, result.MinGroupSize)
	requireEqual(t, now, result.SuggestedAt)
	requireEqual(t, 1, result.UpdatedCount)

	requireEqual(t, 1, len(db.querySQL))
	if !strings.Contains(db.querySQL[0], "GROUP BY project_id, LOWER(TRIM(title))") {
		t.Fatalf("expected duplicate-group SQL, got %q", db.querySQL[0])
	}
	requireEqual(t, 1, len(db.queryArgs))
	requireEqual(t, nil, db.queryArgs[0][0])
	requireEqual(t, defaultConsolidationMinGroupSize, db.queryArgs[0][1])

	requireEqual(t, 1, len(db.queryRowSQL))
	if !strings.Contains(db.queryRowSQL[0], "INSERT INTO engram_consolidation_suggestions") {
		t.Fatalf("expected upsert SQL, got %q", db.queryRowSQL[0])
	}
	requireEqual(t, "engram-vault", db.queryRowArgs[0][1])
	requireEqual(t, "exact_duplicate", db.queryRowArgs[0][3])
	requireEqual(t, 0.7, db.queryRowArgs[0][6].(float64))
	requireEqual(t, now, db.queryRowArgs[0][7].(time.Time))
}

func TestRefreshExactDuplicateConsolidationSuggestionsRejectsInvalidGroupSize(t *testing.T) {
	_, err := RefreshExactDuplicateConsolidationSuggestions(
		context.Background(),
		&fakeQueryer{},
		ConsolidationSuggestionRefreshInput{MinGroupSize: 1},
	)
	if !errors.Is(err, errConsolidationMinGroupSizeInvalid) {
		t.Fatalf("expected errConsolidationMinGroupSizeInvalid, got %v", err)
	}
}

func TestListEngramConsolidationSuggestionsUsesFilters(t *testing.T) {
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-00000000c201")
	firstEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000c202")
	secondEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000c203")
	suggestedAt := time.Date(2026, 3, 3, 14, 10, 0, 0, time.UTC)
	updatedAt := suggestedAt.Add(5 * time.Minute)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{
			values: [][]any{
				{
					suggestionID,
					"engram-vault",
					[]uuid.UUID{firstEngramID, secondEngramID},
					"exact_duplicate",
					"Possible duplicate title cluster",
					"hash-value",
					0.85,
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
	status := models.ConsolidationSuggestionStatusSuggested

	results, err := ListEngramConsolidationSuggestions(
		context.Background(),
		db,
		ConsolidationSuggestionListInput{
			ProjectID: &projectID,
			Status:    &status,
			Limit:     20,
			Offset:    5,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(results))
	requireEqual(t, suggestionID, results[0].SuggestionID)
	requireEqual(t, "engram-vault", results[0].ProjectID)
	if !reflect.DeepEqual(
		[]uuid.UUID{firstEngramID, secondEngramID},
		results[0].SourceEngramIDs,
	) {
		t.Fatalf("expected source engram ids to match")
	}
	requireEqual(t, models.ConsolidationSuggestionStatusSuggested, results[0].Status)

	requireEqual(t, 1, len(db.queryArgs))
	if !reflect.DeepEqual([]any{"engram-vault", "suggested", 20, 5}, db.queryArgs[0]) {
		t.Fatalf("expected list query args to match")
	}
}

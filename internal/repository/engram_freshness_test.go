package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRefreshEngramFreshnessScoresUsesDefaults(t *testing.T) {
	referenceTime := time.Date(2026, 3, 3, 11, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: []any{3}},
	}
	originalNow := nowEngramFreshnessUTC
	nowEngramFreshnessUTC = func() time.Time { return referenceTime }
	t.Cleanup(func() { nowEngramFreshnessUTC = originalNow })

	result, err := RefreshEngramFreshnessScores(
		context.Background(),
		db,
		EngramFreshnessRefreshInput{},
	)
	requireNoError(t, err)
	requireEqual(t, 3, result.UpdatedCount)
	requireEqual(t, defaultEngramFreshnessHalfLifeDays, result.HalfLifeDays)
	requireEqual(t, referenceTime, result.ReferenceTime)
	if result.ProjectID != nil {
		t.Fatalf("expected nil project id for unscoped refresh")
	}

	requireEqual(t, 1, len(db.queryRowSQL))
	if !strings.Contains(db.queryRowSQL[0], "SET") || !strings.Contains(db.queryRowSQL[0], "freshness_score") {
		t.Fatalf("expected freshness update statement, got %q", db.queryRowSQL[0])
	}
	requireEqual(t, 1, len(db.queryRowArgs))
	requireEqual(t, referenceTime, db.queryRowArgs[0][0].(time.Time))
	requireEqual(t, defaultEngramFreshnessHalfLifeDays, db.queryRowArgs[0][1].(float64))
	requireEqual[any](t, nil, db.queryRowArgs[0][2])
}

func TestRefreshEngramFreshnessScoresNormalizesProjectID(t *testing.T) {
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: []any{2}},
	}
	referenceTime := time.Date(2026, 3, 3, 11, 10, 0, 0, time.UTC)
	projectID := "  engram-vault  "
	result, err := RefreshEngramFreshnessScores(
		context.Background(),
		db,
		EngramFreshnessRefreshInput{
			ProjectID:     &projectID,
			HalfLifeDays:  30,
			ReferenceTime: referenceTime,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 2, result.UpdatedCount)
	requireEqual(t, 30.0, result.HalfLifeDays)
	if result.ProjectID == nil {
		t.Fatalf("expected normalized project id")
	}
	requireEqual(t, "engram-vault", *result.ProjectID)
	requireEqual(t, "engram-vault", db.queryRowArgs[0][2])
}

func TestRefreshEngramFreshnessScoresRejectsNegativeHalfLife(t *testing.T) {
	_, err := RefreshEngramFreshnessScores(
		context.Background(),
		&fakeQueryer{},
		EngramFreshnessRefreshInput{HalfLifeDays: -1},
	)
	if !errors.Is(err, errEngramFreshnessHalfLifeDaysInvalid) {
		t.Fatalf("expected errEngramFreshnessHalfLifeDaysInvalid, got %v", err)
	}
}

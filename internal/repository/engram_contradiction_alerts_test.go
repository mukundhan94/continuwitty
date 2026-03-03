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

func TestRefreshContradictionAlertsUsesDefaults(t *testing.T) {
	now := time.Date(2026, 3, 3, 17, 0, 0, 0, time.UTC)
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000e101")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000e102")
	linkID := uuid.MustParse("00000000-0000-0000-0000-00000000e103")
	alertID := uuid.MustParse("00000000-0000-0000-0000-00000000e104")
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{
			values: [][]any{
				{
					"engram-vault",
					sourceEngramID,
					targetEngramID,
					[]uuid.UUID{linkID},
					0.88,
				},
			},
		},
		queryRowResult: &fakeRow{
			values: []any{true},
		},
	}
	originalNow := nowContradictionAlertUTC
	nowContradictionAlertUTC = func() time.Time { return now }
	t.Cleanup(func() { nowContradictionAlertUTC = originalNow })
	originalUUID := newContradictionAlertUUID
	newContradictionAlertUUID = func() uuid.UUID { return alertID }
	t.Cleanup(func() { newContradictionAlertUUID = originalUUID })

	result, err := RefreshContradictionAlerts(
		context.Background(),
		db,
		ContradictionAlertRefreshInput{},
	)
	requireNoError(t, err)
	requireEqual(t, now, result.DetectedAt)
	requireEqual(t, 1, result.UpdatedCount)

	requireEqual(t, 1, len(db.querySQL))
	if !strings.Contains(db.querySQL[0], "relation_type = 'contradicts'") {
		t.Fatalf("expected contradiction relation filter, got %q", db.querySQL[0])
	}
	requireEqual(t, 1, len(db.queryArgs))
	requireEqual(t, nil, db.queryArgs[0][0])

	requireEqual(t, 1, len(db.queryRowSQL))
	if !strings.Contains(db.queryRowSQL[0], "INSERT INTO engram_contradiction_alerts") {
		t.Fatalf("expected contradiction alert upsert SQL, got %q", db.queryRowSQL[0])
	}
	requireEqual(t, "engram-vault", db.queryRowArgs[0][1])
	requireEqual(t, sourceEngramID, db.queryRowArgs[0][2].(uuid.UUID))
	requireEqual(t, targetEngramID, db.queryRowArgs[0][3].(uuid.UUID))
	requireEqual(t, 0.88, db.queryRowArgs[0][7].(float64))
	requireEqual(t, now, db.queryRowArgs[0][8].(time.Time))
}

func TestListContradictionAlertsUsesFilters(t *testing.T) {
	alertID := uuid.MustParse("00000000-0000-0000-0000-00000000e201")
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000e202")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000e203")
	linkID := uuid.MustParse("00000000-0000-0000-0000-00000000e204")
	detectedAt := time.Date(2026, 3, 3, 17, 10, 0, 0, time.UTC)
	updatedAt := detectedAt.Add(3 * time.Minute)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{
			values: [][]any{
				{
					alertID,
					"engram-vault",
					sourceEngramID,
					targetEngramID,
					[]uuid.UUID{linkID},
					"Active contradiction link relationship detected.",
					"hash-value",
					0.82,
					"open",
					detectedAt,
					updatedAt,
					nil,
					nil,
				},
			},
		},
	}
	projectID := "engram-vault"
	status := models.ContradictionAlertStatusOpen

	results, err := ListContradictionAlerts(
		context.Background(),
		db,
		ContradictionAlertListInput{
			ProjectID: &projectID,
			Status:    &status,
			Limit:     15,
			Offset:    2,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(results))
	requireEqual(t, alertID, results[0].AlertID)
	requireEqual(t, models.ContradictionAlertStatusOpen, results[0].Status)
	if !reflect.DeepEqual([]any{"engram-vault", "open", 15, 2}, db.queryArgs[0]) {
		t.Fatalf("expected list query args to match")
	}
}

func TestResolveContradictionAlertUsesRequestObject(t *testing.T) {
	alertID := uuid.MustParse("00000000-0000-0000-0000-00000000e301")
	sourceEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000e302")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-00000000e303")
	linkID := uuid.MustParse("00000000-0000-0000-0000-00000000e304")
	resolvedBy := uuid.MustParse("00000000-0000-0000-0000-00000000e305")
	projectID := "engram-vault"
	detectedAt := time.Date(2026, 3, 3, 17, 15, 0, 0, time.UTC)
	resolvedAt := time.Date(2026, 3, 3, 17, 20, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{
				alertID,
				"engram-vault",
				sourceEngramID,
				targetEngramID,
				[]uuid.UUID{linkID},
				"Active contradiction link relationship detected.",
				"hash-value",
				0.82,
				"resolved",
				detectedAt,
				resolvedAt,
				resolvedAt,
				resolvedBy,
			},
		},
	}

	result, err := ResolveContradictionAlert(
		context.Background(),
		db,
		ContradictionAlertResolveInput{
			AlertID:    alertID,
			ProjectID:  &projectID,
			Status:     models.ContradictionAlertStatusResolved,
			ResolvedBy: resolvedBy,
			ResolvedAt: resolvedAt,
		},
	)
	requireNoError(t, err)
	if result == nil {
		t.Fatalf("expected resolve result")
	}
	requireEqual(t, models.ContradictionAlertStatusResolved, result.Status)
	requireEqual(t, 1, len(db.queryRowArgs))
	if !reflect.DeepEqual(
		[]any{alertID, "resolved", resolvedAt, resolvedBy, "engram-vault"},
		db.queryRowArgs[0],
	) {
		t.Fatalf("expected resolve query args to match")
	}
}

func TestResolveContradictionAlertRejectsInvalidStatus(t *testing.T) {
	_, err := ResolveContradictionAlert(
		context.Background(),
		&fakeQueryer{},
		ContradictionAlertResolveInput{
			AlertID:    uuid.MustParse("00000000-0000-0000-0000-00000000e311"),
			Status:     models.ContradictionAlertStatusOpen,
			ResolvedBy: uuid.MustParse("00000000-0000-0000-0000-00000000e312"),
		},
	)
	if !errors.Is(err, errContradictionAlertResolveStatusInvalid) {
		t.Fatalf("expected errContradictionAlertResolveStatusInvalid, got %v", err)
	}
}

func TestResolveContradictionAlertReturnsNilWhenMissing(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
	result, err := ResolveContradictionAlert(
		context.Background(),
		db,
		ContradictionAlertResolveInput{
			AlertID:    uuid.MustParse("00000000-0000-0000-0000-00000000e321"),
			Status:     models.ContradictionAlertStatusResolved,
			ResolvedBy: uuid.MustParse("00000000-0000-0000-0000-00000000e322"),
		},
	)
	requireNoError(t, err)
	if result != nil {
		t.Fatalf("expected nil result for missing alert")
	}
}

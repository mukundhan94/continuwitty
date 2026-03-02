package repository

import (
	"context"
	"reflect"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestGetEngramShareRecordReturnsNilWhenMissing(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
	record, err := GetEngramShareRecord(
		context.Background(),
		db,
		uuid.MustParse("00000000-0000-0000-0000-000000000701"),
		false,
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil record when engram is missing")
	}
}

func TestGetEngramShareRecordIncludesDeletedFilterByDefault(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000702")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000703")
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{engramID, "engram-vault", ownerUserID, "private", nil},
		},
	}

	record, err := GetEngramShareRecord(context.Background(), db, engramID, false)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, models.VisibilityScopePrivate, record.VisibilityScope)
	if !reflect.DeepEqual(db.queryRowArgs[0], []any{engramID}) {
		t.Fatalf("unexpected query args: %#v", db.queryRowArgs[0])
	}
}

func TestUpdateEngramVisibilityScopeReturnsUpdatedRecord(t *testing.T) {
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000704")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000705")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000706")
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{engramID, "engram-vault", ownerUserID, "project", nil},
		},
	}

	record, err := UpdateEngramVisibilityScope(
		context.Background(),
		db,
		EngramVisibilityUpdateInput{
			EngramID:        engramID,
			VisibilityScope: models.VisibilityScopeProject,
			ActorUserID:     actorUserID,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, models.VisibilityScopeProject, record.VisibilityScope)
	requireEqual(t, "project", db.queryRowArgs[0][0].(string))
	requireEqual(t, actorUserID, db.queryRowArgs[0][2].(uuid.UUID))
	requireEqual(t, engramID, db.queryRowArgs[0][3].(uuid.UUID))
}

func TestUpdateEngramVisibilityScopeReturnsNilWhenMissing(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
	record, err := UpdateEngramVisibilityScope(
		context.Background(),
		db,
		EngramVisibilityUpdateInput{
			EngramID:        uuid.MustParse("00000000-0000-0000-0000-000000000707"),
			VisibilityScope: models.VisibilityScopePrivate,
			ActorUserID:     uuid.MustParse("00000000-0000-0000-0000-000000000708"),
		},
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil record when update target is missing")
	}
}

func TestScanEngramShareRecordRejectsInvalidVisibility(t *testing.T) {
	row := &fakeRow{
		values: []any{
			uuid.MustParse("00000000-0000-0000-0000-000000000709"),
			"engram-vault",
			uuid.MustParse("00000000-0000-0000-0000-000000000710"),
			"invalid",
			time.Date(2026, 2, 28, 18, 0, 0, 0, time.UTC),
		},
	}
	_, err := scanEngramShareRecord(row)
	if err == nil {
		t.Fatalf("expected invalid visibility parsing error")
	}
}

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
	"github.com/jackc/pgx/v5/pgconn"
)

func TestListCollectionsBuildsFilters(t *testing.T) {
	collectionID := uuid.MustParse("00000000-0000-0000-0000-000000000b01")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000b02")
	createdAt := time.Date(2026, 2, 22, 16, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{
			collectionRowValues(collectionRowFixture{
				CollectionID: collectionID,
				ProjectID:    "project-docs",
				OwnerUserID:  ownerUserID,
				Name:         "Ops",
				Description:  "Runbooks",
				CreatedAt:    createdAt,
				UpdatedAt:    createdAt,
			}),
		}},
	}
	projectID := "project-docs"

	records, err := ListCollections(
		context.Background(),
		db,
		CollectionListInput{
			ProjectID:      &projectID,
			OwnerUserID:    &ownerUserID,
			IncludeDeleted: false,
			Limit:          10,
			Offset:         2,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(records))
	requireEqual(t, collectionID, records[0].CollectionID)

	query := db.querySQL[0]
	if !strings.Contains(query, "project_id = $1") {
		t.Fatalf("expected project clause, got %q", query)
	}
	if !strings.Contains(query, "owner_user_id = $2") {
		t.Fatalf("expected owner clause, got %q", query)
	}
	if !strings.Contains(query, "deleted_at IS NULL") {
		t.Fatalf("expected active-only clause, got %q", query)
	}
	expectedArgs := []any{projectID, ownerUserID, 10, 2}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestGetAndUpdateCollectionReturnNilWhenMissing(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
	testCases := []struct {
		name    string
		execute func() (*models.EngramCollectionRecord, error)
	}{
		{
			name: "get",
			execute: func() (*models.EngramCollectionRecord, error) {
				return GetCollection(
					context.Background(),
					db,
					uuid.MustParse("00000000-0000-0000-0000-000000000b11"),
					false,
				)
			},
		},
		{
			name: "update",
			execute: func() (*models.EngramCollectionRecord, error) {
				name := "Updated"
				return UpdateCollection(
					context.Background(),
					db,
					CollectionUpdateInput{
						CollectionID: uuid.MustParse("00000000-0000-0000-0000-000000000b41"),
						Name:         &name,
					},
				)
			},
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			record, err := testCase.execute()
			requireNoError(t, err)
			if record != nil {
				t.Fatalf("expected nil collection when missing")
			}
		})
	}
}

func TestCreateCollectionUsesGeneratedID(t *testing.T) {
	collectionID := uuid.MustParse("00000000-0000-0000-0000-000000000b21")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000b22")
	createdAt := time.Date(2026, 2, 22, 16, 10, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: collectionRowValues(collectionRowFixture{
			CollectionID: collectionID,
			ProjectID:    "project-docs",
			OwnerUserID:  ownerUserID,
			Name:         "Ops",
			Description:  "Runbooks",
			CreatedAt:    createdAt,
			UpdatedAt:    createdAt,
		})},
	}

	originalCollectionUUID := newCollectionUUID
	newCollectionUUID = func() uuid.UUID { return collectionID }
	t.Cleanup(func() { newCollectionUUID = originalCollectionUUID })

	record, err := CreateCollection(
		context.Background(),
		db,
		CollectionCreateInput{
			ProjectID:   "project-docs",
			OwnerUserID: ownerUserID,
			Name:        "Ops",
			Description: "Runbooks",
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, collectionID, record.CollectionID)

	expectedArgs := []any{collectionID, "project-docs", ownerUserID, "Ops", "Runbooks"}
	if !reflect.DeepEqual(expectedArgs, db.queryRowArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}
}

func TestCreateCollectionMapsDuplicateNameError(t *testing.T) {
	db := &fakeQueryer{
		queryRowResult: &fakeRow{err: &pgconn.PgError{
			Code:           "23505",
			ConstraintName: "engram_collections_project_name_active_uidx",
			Message:        "duplicate",
		}},
	}

	record, err := CreateCollection(
		context.Background(),
		db,
		CollectionCreateInput{
			ProjectID:   "project-docs",
			OwnerUserID: uuid.MustParse("00000000-0000-0000-0000-000000000b31"),
			Name:        "Ops",
			Description: "Runbooks",
		},
	)
	if !errors.Is(err, ErrCollectionNameExists) {
		t.Fatalf("expected ErrCollectionNameExists, got %v", err)
	}
	if record != nil {
		t.Fatalf("expected nil collection on duplicate error")
	}
}

func TestSoftDeleteCollectionReturnsBool(t *testing.T) {
	collectionID := uuid.MustParse("00000000-0000-0000-0000-000000000b51")
	deletedBy := uuid.MustParse("00000000-0000-0000-0000-000000000b52")
	reason := "cleanup"
	db := &fakeQueryer{queryRowResult: &fakeRow{values: []any{collectionID}}}

	deleted, err := SoftDeleteCollection(
		context.Background(),
		db,
		CollectionSoftDeleteInput{
			CollectionID:    collectionID,
			DeletedByUserID: deletedBy,
			Reason:          &reason,
		},
	)
	requireNoError(t, err)
	if !deleted {
		t.Fatalf("expected collection to be deleted")
	}

	missingDB := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
	deleted, err = SoftDeleteCollection(
		context.Background(),
		missingDB,
		CollectionSoftDeleteInput{
			CollectionID:    collectionID,
			DeletedByUserID: deletedBy,
			Reason:          &reason,
		},
	)
	requireNoError(t, err)
	if deleted {
		t.Fatalf("expected false when collection missing")
	}
}

func TestAddCollectionItemsReturnsCountAndSkipsEmpty(t *testing.T) {
	collectionID := uuid.MustParse("00000000-0000-0000-0000-000000000b61")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000b62")
	engramA := uuid.MustParse("00000000-0000-0000-0000-000000000b63")
	engramB := uuid.MustParse("00000000-0000-0000-0000-000000000b64")
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{{engramA}, {engramB}}},
	}

	count, err := AddCollectionItems(
		context.Background(),
		db,
		CollectionAddItemsInput{
			CollectionID: collectionID,
			ActorUserID:  actorUserID,
			EngramIDs:    []uuid.UUID{engramA, engramB},
		},
	)
	requireNoError(t, err)
	requireEqual(t, 2, count)
	expectedArgs := []any{actorUserID, []uuid.UUID{engramA, engramB}, collectionID}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}

	emptyDB := &fakeQueryer{}
	count, err = AddCollectionItems(
		context.Background(),
		emptyDB,
		CollectionAddItemsInput{
			CollectionID: collectionID,
			ActorUserID:  actorUserID,
			EngramIDs:    nil,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 0, count)
	if len(emptyDB.querySQL) != 0 {
		t.Fatalf("expected no query on empty engram_ids")
	}
}

func TestRemoveCollectionItemReturnsBool(t *testing.T) {
	collectionID := uuid.MustParse("00000000-0000-0000-0000-000000000b71")
	engramID := uuid.MustParse("00000000-0000-0000-0000-000000000b72")
	db := &fakeQueryer{queryRowResult: &fakeRow{values: []any{engramID}}}

	removed, err := RemoveCollectionItem(
		context.Background(),
		db,
		CollectionRemoveItemInput{CollectionID: collectionID, EngramID: engramID},
	)
	requireNoError(t, err)
	if !removed {
		t.Fatalf("expected true when row removed")
	}

	missingDB := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
	removed, err = RemoveCollectionItem(
		context.Background(),
		missingDB,
		CollectionRemoveItemInput{CollectionID: collectionID, EngramID: engramID},
	)
	requireNoError(t, err)
	if removed {
		t.Fatalf("expected false when no row removed")
	}
}

type collectionRowFixture struct {
	CollectionID    uuid.UUID
	ProjectID       string
	OwnerUserID     uuid.UUID
	Name            string
	Description     string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
	DeletedByUserID *uuid.UUID
	DeleteReason    *string
}

func collectionRowValues(fixture collectionRowFixture) []any {
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
		fixture.CollectionID,
		fixture.ProjectID,
		fixture.OwnerUserID,
		fixture.Name,
		fixture.Description,
		fixture.CreatedAt,
		fixture.UpdatedAt,
		deletedAtValue,
		deletedByUserIDValue,
		deleteReasonValue,
	}
}

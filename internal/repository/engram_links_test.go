package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCreateEngramLinkReturnsRecord(t *testing.T) {
	record := sampleEngramLinkRecord(
		uuid.MustParse("00000000-0000-0000-0000-000000000801"),
		uuid.MustParse("00000000-0000-0000-0000-000000000811"),
		uuid.MustParse("00000000-0000-0000-0000-000000000812"),
	)
	db := &fakeQueryer{queryRowResult: &fakeRow{values: engramLinkRowValues(record)}}
	createdBy := uuid.MustParse("00000000-0000-0000-0000-000000000822")

	result, err := CreateEngramLink(
		context.Background(),
		db,
		EngramLinkCreateInput{
			SourceEngramID:  record.SourceEngramID,
			TargetEngramID:  record.TargetEngramID,
			RelationType:    record.RelationType,
			Weight:          record.Weight,
			TemporalWeight:  record.TemporalWeight,
			Confidence:      record.Confidence,
			Origin:          record.Origin,
			Status:          record.Status,
			EvidenceJSON:    record.EvidenceJSON,
			CreatedByUserID: createdBy,
			ActorUserID:     createdBy,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, result)
	requireEqual(t, record.LinkID, result.LinkID)
	requireEqual(t, models.EngramLinkRelationSupports, result.RelationType)
	requireEqual(t, models.EngramLinkStatusActive, result.Status)
	if len(db.queryRowArgs) != 1 {
		t.Fatalf("expected single query row call")
	}
	if db.queryRowArgs[0][3] != string(models.EngramLinkRelationSupports) {
		t.Fatalf("expected relation arg supports, got %#v", db.queryRowArgs[0][3])
	}
}

func TestCreateEngramLinkReturnsDuplicateError(t *testing.T) {
	db := &fakeQueryer{
		queryRowResult: &fakeRow{err: &pgconn.PgError{Code: "23505"}},
	}
	_, err := CreateEngramLink(
		context.Background(),
		db,
		EngramLinkCreateInput{
			SourceEngramID:  uuid.MustParse("00000000-0000-0000-0000-000000000831"),
			TargetEngramID:  uuid.MustParse("00000000-0000-0000-0000-000000000832"),
			RelationType:    models.EngramLinkRelationSupports,
			Weight:          0.7,
			TemporalWeight:  0.6,
			Confidence:      0.8,
			Origin:          models.EngramLinkOriginManual,
			Status:          models.EngramLinkStatusActive,
			CreatedByUserID: uuid.MustParse("00000000-0000-0000-0000-000000000833"),
			ActorUserID:     uuid.MustParse("00000000-0000-0000-0000-000000000833"),
		},
	)
	if !errors.Is(err, ErrEngramLinkExists) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestListEngramLinksReturnsRows(t *testing.T) {
	record := sampleEngramLinkRecord(
		uuid.MustParse("00000000-0000-0000-0000-000000000841"),
		uuid.MustParse("00000000-0000-0000-0000-000000000842"),
		uuid.MustParse("00000000-0000-0000-0000-000000000843"),
	)
	db := &fakeQueryer{queryRowsResult: &fakeRows{values: [][]any{engramLinkRowValues(record)}}}
	results, err := ListEngramLinks(
		context.Background(),
		db,
		EngramLinkListInput{
			SourceEngramID:  record.SourceEngramID,
			ActorUserID:     uuid.MustParse("00000000-0000-0000-0000-000000000844"),
			IncludeArchived: false,
			Limit:           20,
			Offset:          5,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(results))
	requireEqual(t, record.LinkID, results[0].LinkID)
	requireEqual(t, false, db.queryArgs[0][3])
	requireEqual(t, 20, db.queryArgs[0][4])
	requireEqual(t, 5, db.queryArgs[0][5])
}

func TestGetEngramLinkReturnsRecord(t *testing.T) {
	record := sampleEngramLinkRecord(
		uuid.MustParse("00000000-0000-0000-0000-000000000846"),
		uuid.MustParse("00000000-0000-0000-0000-000000000847"),
		uuid.MustParse("00000000-0000-0000-0000-000000000848"),
	)
	db := &fakeQueryer{queryRowResult: &fakeRow{values: engramLinkRowValues(record)}}
	found, err := GetEngramLink(
		context.Background(),
		db,
		EngramLinkGetInput{
			LinkID:          record.LinkID,
			ActorUserID:     uuid.MustParse("00000000-0000-0000-0000-000000000849"),
			IncludeArchived: true,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, found)
	requireEqual(t, record.LinkID, found.LinkID)
}

func TestUpdateEngramLinkRejectsInvalidWeight(t *testing.T) {
	weight := 1.2
	_, err := UpdateEngramLink(
		context.Background(),
		&fakeQueryer{},
		EngramLinkUpdateInput{
			LinkID:      uuid.MustParse("00000000-0000-0000-0000-000000000851"),
			ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000000852"),
			Weight:      &weight,
		},
	)
	if err == nil {
		t.Fatalf("expected invalid weight validation error")
	}
}

func TestEngramLinkOperationsReturnNilWhenMissing(t *testing.T) {
	testCases := []struct {
		name string
		run  func(context.Context, Queryer) (*models.EngramLinkRecord, error)
	}{
		{
			name: "get",
			run: func(ctx context.Context, db Queryer) (*models.EngramLinkRecord, error) {
				return GetEngramLink(
					ctx,
					db,
					EngramLinkGetInput{
						LinkID:      uuid.MustParse("00000000-0000-0000-0000-00000000084a"),
						ActorUserID: uuid.MustParse("00000000-0000-0000-0000-00000000084b"),
					},
				)
			},
		},
		{
			name: "archive",
			run: func(ctx context.Context, db Queryer) (*models.EngramLinkRecord, error) {
				return ArchiveEngramLink(
					ctx,
					db,
					EngramLinkArchiveInput{
						LinkID:      uuid.MustParse("00000000-0000-0000-0000-000000000861"),
						ActorUserID: uuid.MustParse("00000000-0000-0000-0000-000000000862"),
					},
				)
			},
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
			result, err := testCase.run(context.Background(), db)
			requireNoError(t, err)
			if result != nil {
				t.Fatalf("expected nil result when query returns no rows")
			}
		})
	}
}

func TestTraverseEngramLinksDepthLimitedAndCycleSafe(t *testing.T) {
	rootID := uuid.MustParse("00000000-0000-0000-0000-000000000871")
	aID := uuid.MustParse("00000000-0000-0000-0000-000000000872")
	bID := uuid.MustParse("00000000-0000-0000-0000-000000000873")
	cID := uuid.MustParse("00000000-0000-0000-0000-000000000874")
	db := &fakeQueryer{
		queryRowsResults: []*fakeRows{
			{values: [][]any{
				engramLinkRowValues(sampleEngramLinkRecord(uuid.MustParse("00000000-0000-0000-0000-000000000881"), rootID, aID)),
				engramLinkRowValues(sampleEngramLinkRecord(uuid.MustParse("00000000-0000-0000-0000-000000000882"), rootID, bID)),
			}},
			{values: [][]any{
				engramLinkRowValues(sampleEngramLinkRecord(uuid.MustParse("00000000-0000-0000-0000-000000000883"), aID, rootID)),
				engramLinkRowValues(sampleEngramLinkRecord(uuid.MustParse("00000000-0000-0000-0000-000000000884"), bID, cID)),
			}},
		},
	}
	steps, err := TraverseEngramLinks(
		context.Background(),
		db,
		EngramLinkTraverseInput{
			RootEngramID: rootID,
			ActorUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000875"),
			MaxDepth:     2,
			MaxNeighbors: 10,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 4, len(steps))
	requireEqual(t, 1, steps[0].Depth)
	requireEqual(t, 1, steps[1].Depth)
	requireEqual(t, 2, steps[2].Depth)
	requireEqual(t, 2, steps[3].Depth)
	if len(db.queryArgs) != 2 {
		t.Fatalf("expected two traversal queries, got %d", len(db.queryArgs))
	}
}

func sampleEngramLinkRecord(
	linkID uuid.UUID,
	sourceEngramID uuid.UUID,
	targetEngramID uuid.UUID,
) models.EngramLinkRecord {
	now := time.Date(2026, 3, 1, 11, 20, 0, 0, time.UTC)
	return models.EngramLinkRecord{
		LinkID:         linkID,
		ProjectID:      "engram-vault",
		SourceEngramID: sourceEngramID,
		TargetEngramID: targetEngramID,
		RelationType:   models.EngramLinkRelationSupports,
		Weight:         0.8,
		TemporalWeight: 0.7,
		Confidence:     0.9,
		Origin:         models.EngramLinkOriginManual,
		Status:         models.EngramLinkStatusActive,
		EvidenceJSON: map[string]any{
			"summary": "linked from continuity trace",
		},
		CreatedByUserID:  uuid.MustParse("00000000-0000-0000-0000-000000000899"),
		LastReinforcedAt: &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func engramLinkRowValues(record models.EngramLinkRecord) []any {
	var lastReinforced any
	if record.LastReinforcedAt != nil {
		lastReinforced = *record.LastReinforcedAt
	}
	return []any{
		record.LinkID,
		record.ProjectID,
		record.SourceEngramID,
		record.TargetEngramID,
		string(record.RelationType),
		record.Weight,
		record.TemporalWeight,
		record.Confidence,
		string(record.Origin),
		string(record.Status),
		record.EvidenceJSON,
		record.CreatedByUserID,
		lastReinforced,
		record.CreatedAt,
		record.UpdatedAt,
	}
}

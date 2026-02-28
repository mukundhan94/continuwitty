package repository

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestCreateProjectAuditEventUsesDefaultsAndPersistsMetadata(t *testing.T) {
	eventID := uuid.MustParse("00000000-0000-0000-0000-000000000601")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000602")
	targetEngramID := uuid.MustParse("00000000-0000-0000-0000-000000000603")
	createdAt := time.Date(2026, 2, 28, 14, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{
				eventID,
				"engram-vault",
				actorUserID,
				"engram.share",
				"engram",
				nil,
				targetEngramID,
				[]byte(`{"from_visibility_scope":"private","to_visibility_scope":"project"}`),
				createdAt,
			},
		},
	}

	record, err := CreateProjectAuditEvent(
		context.Background(),
		db,
		ProjectAuditEventCreateInput{
			EventID:        eventID,
			ProjectID:      "engram-vault",
			ActorUserID:    &actorUserID,
			EventType:      "engram.share",
			TargetType:     "engram",
			TargetEngramID: &targetEngramID,
			Metadata: map[string]any{
				"from_visibility_scope": "private",
				"to_visibility_scope":   "project",
			},
			CreatedAt: &createdAt,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, eventID, record.EventID)
	requireEqual(t, "engram.share", record.EventType)
	requireEqual(t, "private", record.Metadata["from_visibility_scope"].(string))
	requireEqual(t, "project", record.Metadata["to_visibility_scope"].(string))

	if len(db.queryRowArgs) != 1 {
		t.Fatalf("expected one query row call, got %d", len(db.queryRowArgs))
	}
	metadataArg, ok := db.queryRowArgs[0][7].(string)
	if !ok {
		t.Fatalf("expected metadata argument to be JSON string")
	}
	if !strings.Contains(metadataArg, "\"from_visibility_scope\":\"private\"") {
		t.Fatalf("unexpected metadata JSON argument: %s", metadataArg)
	}
}

func TestCreateProjectAuditEventReturnsNilOnNoRows(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
	record, err := CreateProjectAuditEvent(
		context.Background(),
		db,
		ProjectAuditEventCreateInput{
			ProjectID:  "engram-vault",
			EventType:  "project.member.add",
			TargetType: "user",
		},
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil record when insert returns no rows")
	}
}

func TestListProjectAuditEventsReturnsNewestFirstRows(t *testing.T) {
	eventOneID := uuid.MustParse("00000000-0000-0000-0000-000000000604")
	eventTwoID := uuid.MustParse("00000000-0000-0000-0000-000000000605")
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000606")
	targetUserID := uuid.MustParse("00000000-0000-0000-0000-000000000607")
	createdAt := time.Date(2026, 2, 28, 15, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{
			values: [][]any{
				{
					eventOneID,
					"engram-vault",
					actorUserID,
					"project.member.add",
					"user",
					targetUserID,
					nil,
					[]byte(`{"role":"viewer"}`),
					createdAt,
				},
				{
					eventTwoID,
					"engram-vault",
					actorUserID,
					"engram.share",
					"engram",
					nil,
					uuid.MustParse("00000000-0000-0000-0000-000000000608"),
					[]byte(`{}`),
					createdAt,
				},
			},
		},
	}

	records, err := ListProjectAuditEvents(
		context.Background(),
		db,
		ProjectAuditEventListInput{
			ProjectID: "engram-vault",
			Limit:     50,
			Offset:    4,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 2, len(records))
	requireEqual(t, eventOneID, records[0].EventID)
	requireEqual(t, "viewer", records[0].Metadata["role"].(string))
	if !reflect.DeepEqual(db.queryArgs[0], []any{"engram-vault", 50, 4}) {
		t.Fatalf("unexpected query args: %#v", db.queryArgs[0])
	}
}

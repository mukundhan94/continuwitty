package repository

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestListAdminSessionsBuildsFiltersAndParsesEnums(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000c01")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000c02")
	createdAt := time.Date(2026, 2, 22, 17, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{
			adminSessionRowValues(adminSessionRowFixture{
				sessionID:              sessionID,
				ownerUserID:            ownerUserID,
				projectID:              "project-docs",
				title:                  "Admin session",
				provider:               "openai",
				modelID:                "gpt-4o-mini",
				systemPrompt:           "",
				visibilityScope:        "private",
				autosaveEnabled:        false,
				autosaveStrategy:       "off",
				autosaveIntervalMinute: 30,
				autosaveMinMessages:    6,
				retentionDays:          30,
				retentionMaxSnapshots:  60,
				createdAt:              createdAt,
				updatedAt:              createdAt,
			}),
		}},
	}
	projectID := "project-docs"

	records, err := ListAdminSessions(
		context.Background(),
		db,
		AdminSessionListInput{
			IncludeDeleted: false,
			ProjectID:      &projectID,
			OwnerUserID:    &ownerUserID,
			Limit:          20,
			Offset:         5,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(records))
	requireEqual(t, models.ChatProviderOpenAI, records[0].Provider)
	requireEqual(t, models.VisibilityScopePrivate, records[0].VisibilityScope)

	query := db.querySQL[0]
	if !strings.Contains(query, "deleted_at IS NULL") {
		t.Fatalf("expected deleted_at filter clause, got %q", query)
	}
	if !strings.Contains(query, "project_id = $1") {
		t.Fatalf("expected project filter clause, got %q", query)
	}
	if !strings.Contains(query, "owner_user_id = $2") {
		t.Fatalf("expected owner filter clause, got %q", query)
	}
	expectedArgs := []any{projectID, ownerUserID, 20, 5}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestGetAdminSessionReturnsNilWhenMissing(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}

	record, err := GetAdminSession(
		context.Background(),
		db,
		uuid.MustParse("00000000-0000-0000-0000-000000000c11"),
		false,
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil session when missing")
	}
}

func TestSoftDeleteSessionReturnsRecord(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000c21")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000c22")
	deletedByUserID := uuid.MustParse("00000000-0000-0000-0000-000000000c23")
	createdAt := time.Date(2026, 2, 22, 17, 5, 0, 0, time.UTC)
	deletedAt := time.Date(2026, 2, 22, 17, 10, 0, 0, time.UTC)
	reason := "cleanup"
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: adminSessionRowValues(adminSessionRowFixture{
			sessionID:              sessionID,
			ownerUserID:            ownerUserID,
			projectID:              "project-docs",
			title:                  "Admin session",
			provider:               "openai",
			modelID:                "gpt-4o-mini",
			systemPrompt:           "",
			visibilityScope:        "private",
			autosaveEnabled:        false,
			autosaveStrategy:       "off",
			autosaveIntervalMinute: 30,
			autosaveMinMessages:    6,
			retentionDays:          30,
			retentionMaxSnapshots:  60,
			createdAt:              createdAt,
			updatedAt:              createdAt,
			deletedAt:              &deletedAt,
			deletedByUserID:        &deletedByUserID,
			deleteReason:           &reason,
		})},
	}

	record, err := SoftDeleteSession(
		context.Background(),
		db,
		AdminSessionSoftDeleteInput{
			SessionID:       sessionID,
			DeletedByUserID: deletedByUserID,
			Reason:          &reason,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	if record.DeletedAt == nil {
		t.Fatalf("expected deleted_at to be set")
	}
	requireEqual(t, deletedByUserID, *record.DeletedByUserID)
}

func TestRestoreSessionReturnsBool(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000c31")
	db := &fakeQueryer{queryRowResult: &fakeRow{values: []any{sessionID}}}

	restored, err := RestoreSession(context.Background(), db, sessionID)
	requireNoError(t, err)
	if !restored {
		t.Fatalf("expected true restore result")
	}

	missingDB := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
	restored, err = RestoreSession(context.Background(), missingDB, sessionID)
	requireNoError(t, err)
	if restored {
		t.Fatalf("expected false when session missing")
	}
}

func TestSoftDeleteLinkedEngramsReturnsCount(t *testing.T) {
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000c41")
	deletedBy := uuid.MustParse("00000000-0000-0000-0000-000000000c42")
	engramA := uuid.MustParse("00000000-0000-0000-0000-000000000c43")
	engramB := uuid.MustParse("00000000-0000-0000-0000-000000000c44")
	reason := "session cleanup"
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{{engramA}, {engramB}}},
	}

	count, err := SoftDeleteLinkedEngrams(
		context.Background(),
		db,
		SoftDeleteLinkedEngramsInput{
			SessionID:       sessionID,
			DeletedByUserID: deletedBy,
			Reason:          &reason,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 2, count)
	expectedArgs := []any{deletedBy, &reason, sessionID}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

type adminSessionRowFixture struct {
	sessionID              uuid.UUID
	ownerUserID            uuid.UUID
	projectID              string
	title                  string
	provider               string
	modelID                string
	systemPrompt           string
	visibilityScope        string
	autosaveEnabled        bool
	autosaveStrategy       string
	autosaveIntervalMinute int
	autosaveMinMessages    int
	retentionDays          int
	retentionMaxSnapshots  int
	createdAt              time.Time
	updatedAt              time.Time
	deletedAt              *time.Time
	deletedByUserID        *uuid.UUID
	deleteReason           *string
}

func adminSessionRowValues(fixture adminSessionRowFixture) []any {
	var deletedAtValue any
	if fixture.deletedAt != nil {
		deletedAtValue = *fixture.deletedAt
	}
	var deletedByUserIDValue any
	if fixture.deletedByUserID != nil {
		deletedByUserIDValue = *fixture.deletedByUserID
	}
	var deleteReasonValue any
	if fixture.deleteReason != nil {
		deleteReasonValue = *fixture.deleteReason
	}
	return []any{
		fixture.sessionID,
		fixture.ownerUserID,
		fixture.projectID,
		fixture.title,
		fixture.provider,
		fixture.modelID,
		fixture.systemPrompt,
		fixture.visibilityScope,
		fixture.autosaveEnabled,
		fixture.autosaveStrategy,
		fixture.autosaveIntervalMinute,
		fixture.autosaveMinMessages,
		fixture.retentionDays,
		fixture.retentionMaxSnapshots,
		fixture.createdAt,
		fixture.updatedAt,
		deletedAtValue,
		deletedByUserIDValue,
		deleteReasonValue,
	}
}

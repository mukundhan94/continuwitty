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

func TestListProjectMembersFiltersRevokedByDefault(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000501")
	addedByUserID := uuid.MustParse("00000000-0000-0000-0000-000000000502")
	createdAt := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{
			values: [][]any{
				{"engram-vault", userID, "viewer", addedByUserID, createdAt, createdAt, nil, nil},
			},
		},
	}

	records, err := ListProjectMembers(
		context.Background(),
		db,
		ProjectMemberListInput{
			ProjectID:      "engram-vault",
			IncludeRevoked: false,
			Limit:          100,
			Offset:         3,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(records))
	requireEqual(t, models.ProjectMemberRoleViewer, records[0].Role)
	if !strings.Contains(db.querySQL[0], "revoked_at IS NULL") {
		t.Fatalf("expected active-membership filter in query, got %q", db.querySQL[0])
	}
	if !reflect.DeepEqual(db.queryArgs[0], []any{"engram-vault", 100, 3}) {
		t.Fatalf("unexpected query args: %#v", db.queryArgs[0])
	}
}

func TestListProjectMembersIncludesRevokedWhenRequested(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000503")
	createdAt := time.Date(2026, 2, 28, 11, 0, 0, 0, time.UTC)
	revokedAt := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{
			values: [][]any{
				{"engram-vault", userID, "editor", nil, createdAt, createdAt, revokedAt, nil},
			},
		},
	}

	records, err := ListProjectMembers(
		context.Background(),
		db,
		ProjectMemberListInput{
			ProjectID:      "engram-vault",
			IncludeRevoked: true,
			Limit:          20,
			Offset:         0,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(records))
	requireEqual(t, models.ProjectMemberRoleEditor, records[0].Role)
	if strings.Contains(db.querySQL[0], "revoked_at IS NULL") {
		t.Fatalf("did not expect active-only filter in include_revoked query")
	}
}

func TestAddOrRestoreProjectMemberReturnsCreatedRecord(t *testing.T) {
	projectID := "engram-vault"
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000504")
	addedByUserID := uuid.MustParse("00000000-0000-0000-0000-000000000505")
	createdAt := time.Date(2026, 2, 28, 13, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{
				projectID,
				userID,
				"editor",
				addedByUserID,
				createdAt,
				createdAt,
				nil,
				nil,
			},
		},
	}

	record, err := AddOrRestoreProjectMember(
		context.Background(),
		db,
		ProjectMemberAddInput{
			ProjectID:     projectID,
			UserID:        userID,
			Role:          models.ProjectMemberRoleEditor,
			AddedByUserID: addedByUserID,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, projectID, record.ProjectID)
	requireEqual(t, models.ProjectMemberRoleEditor, record.Role)
	if !strings.Contains(db.queryRowSQL[0], "ON CONFLICT (project_id, user_id) DO UPDATE") {
		t.Fatalf("expected upsert query, got %q", db.queryRowSQL[0])
	}
	requireEqual(t, "editor", db.queryRowArgs[0][2].(string))
}

func TestUpdateProjectMemberRoleReturnsNilWhenMissing(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
	record, err := UpdateProjectMemberRole(
		context.Background(),
		db,
		ProjectMemberUpdateInput{
			ProjectID: "engram-vault",
			UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000506"),
			Role:      models.ProjectMemberRoleViewer,
		},
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil record when member is missing")
	}
}

func TestRemoveProjectMemberReturnsBooleanResult(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000507")
	revokedByUserID := uuid.MustParse("00000000-0000-0000-0000-000000000508")

	t.Run("missing row", func(t *testing.T) {
		db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
		removed, err := RemoveProjectMember(
			context.Background(),
			db,
			ProjectMemberRemoveInput{
				ProjectID:       "engram-vault",
				UserID:          userID,
				RevokedByUserID: revokedByUserID,
			},
		)
		requireNoError(t, err)
		requireEqual(t, false, removed)
	})

	t.Run("row removed", func(t *testing.T) {
		db := &fakeQueryer{queryRowResult: &fakeRow{values: []any{"engram-vault"}}}
		removed, err := RemoveProjectMember(
			context.Background(),
			db,
			ProjectMemberRemoveInput{
				ProjectID:       "engram-vault",
				UserID:          userID,
				RevokedByUserID: revokedByUserID,
			},
		)
		requireNoError(t, err)
		requireEqual(t, true, removed)
	})
}

func TestResolveActorProjectRoleHandlesMissingAndOwner(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000509")

	t.Run("missing membership", func(t *testing.T) {
		db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}
		role, err := ResolveActorProjectRole(
			context.Background(),
			db,
			ActorProjectRoleInput{
				ProjectID:   "engram-vault",
				ActorUserID: actorUserID,
			},
		)
		requireNoError(t, err)
		if role != nil {
			t.Fatalf("expected nil role for invisible project")
		}
	})

	t.Run("returns owner role", func(t *testing.T) {
		db := &fakeQueryer{queryRowResult: &fakeRow{values: []any{"owner"}}}
		role, err := ResolveActorProjectRole(
			context.Background(),
			db,
			ActorProjectRoleInput{
				ProjectID:   "engram-vault",
				ActorUserID: actorUserID,
			},
		)
		requireNoError(t, err)
		requireNotNil(t, role)
		requireEqual(t, models.ProjectMemberRoleOwner, *role)
	})
}

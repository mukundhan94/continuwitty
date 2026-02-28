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

func TestListProjectsForActorScopesNonAdminAndArchived(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000601")
	createdAt := time.Date(2026, 2, 21, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{
			values: [][]any{
				{
					"engram-vault",
					"Engram Vault",
					"Default project",
					actorUserID,
					false,
					createdAt,
					updatedAt,
					"owner",
				},
			},
		},
	}

	projects, err := ListProjectsForActor(
		context.Background(),
		db,
		ProjectListInput{
			ActorUserID: actorUserID,
			ActorRole:   "analyst",
			Limit:       10,
			Offset:      5,
		},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(projects))
	requireEqual(t, "engram-vault", projects[0].ProjectID)

	requireEqual(t, 1, len(db.querySQL))
	query := db.querySQL[0]
	if !strings.Contains(query, "owner_user_id = $1") {
		t.Fatalf("expected owner filter in query, got %q", query)
	}
	if !strings.Contains(query, "is_archived = FALSE") {
		t.Fatalf("expected archive filter in query, got %q", query)
	}
	expectedArgs := []any{actorUserID, 10, 5}
	if !reflect.DeepEqual(db.queryArgs[0], expectedArgs) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestGetProjectForActorReturnsNilWhenMissing(t *testing.T) {
	db := &fakeQueryer{
		queryRowResult: &fakeRow{err: pgx.ErrNoRows},
	}
	project, err := GetProjectForActor(
		context.Background(),
		db,
		ProjectGetInput{
			ProjectID:       "project-x",
			ActorUserID:     uuid.MustParse("00000000-0000-0000-0000-000000000611"),
			ActorRole:       "analyst",
			IncludeArchived: false,
		},
	)
	requireNoError(t, err)
	if project != nil {
		t.Fatalf("expected nil project when row missing")
	}
}

func TestCreateProjectReturnsRecord(t *testing.T) {
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000621")
	createdAt := time.Date(2026, 2, 21, 11, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{
				"project-alpha",
				"Project Alpha",
				"Alpha project",
				ownerUserID,
				false,
				createdAt,
				createdAt,
				"owner",
			},
		},
	}

	project, err := CreateProject(
		context.Background(),
		db,
		ProjectCreateInput{
			ProjectID:   "project-alpha",
			Name:        "Project Alpha",
			Description: "Alpha project",
			OwnerUserID: ownerUserID,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, project)
	requireEqual(t, "project-alpha", project.ProjectID)
	requireEqual(t, ownerUserID, project.OwnerUserID)
}

func TestEnsureProjectExistsReturnsExistingBeforeCreate(t *testing.T) {
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000631")
	createdAt := time.Date(2026, 2, 21, 12, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{
			values: []any{
				"project-existing",
				"Existing",
				"Already exists",
				ownerUserID,
				false,
				createdAt,
				createdAt,
				"owner",
			},
		},
	}

	project, err := EnsureProjectExists(
		context.Background(),
		db,
		ProjectEnsureInput{
			ProjectID:   "project-existing",
			OwnerUserID: ownerUserID,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, project)
	requireEqual(t, "project-existing", project.ProjectID)
	requireEqual(t, 1, len(db.queryRowSQL))
	if strings.Contains(db.queryRowSQL[0], "INSERT INTO projects") {
		t.Fatalf("did not expect create-project insert when project already exists")
	}
}

func TestGetUserDefaultProjectIDReturnsValueAndNil(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000641")
	defaultProjectID := "engram-vault"
	dbWithValue := &fakeQueryer{
		queryRowResult: &fakeRow{values: []any{defaultProjectID}},
	}

	value, err := GetUserDefaultProjectID(context.Background(), dbWithValue, userID)
	requireNoError(t, err)
	if value == nil || *value != defaultProjectID {
		t.Fatalf("expected default project %q, got %#v", defaultProjectID, value)
	}

	dbMissing := &fakeQueryer{
		queryRowResult: &fakeRow{err: pgx.ErrNoRows},
	}
	missing, err := GetUserDefaultProjectID(context.Background(), dbMissing, userID)
	requireNoError(t, err)
	if missing != nil {
		t.Fatalf("expected nil default project for missing user")
	}
}

func TestSetUserDefaultProjectIDReturnsBool(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000651")
	dbUpdated := &fakeQueryer{
		queryRowResult: &fakeRow{values: []any{userID}},
	}
	updated, err := SetUserDefaultProjectID(context.Background(), dbUpdated, userID, "project-1")
	requireNoError(t, err)
	if !updated {
		t.Fatalf("expected update result true")
	}

	dbMissing := &fakeQueryer{
		queryRowResult: &fakeRow{err: pgx.ErrNoRows},
	}
	notUpdated, err := SetUserDefaultProjectID(context.Background(), dbMissing, userID, "project-1")
	requireNoError(t, err)
	if notUpdated {
		t.Fatalf("expected update result false for missing user")
	}
}

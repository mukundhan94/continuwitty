package projects

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestListProjectsForwardsRepositoryInput(t *testing.T) {
	service := NewService(nil)
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000711")
	captured := repository.ProjectListInput{}
	service.deps.listProjectsForActor = func(_ context.Context, _ repository.Queryer, input repository.ProjectListInput) ([]models.ProjectRecord, error) {
		captured = input
		return []models.ProjectRecord{}, nil
	}

	_, err := service.ListProjects(context.Background(), actorUserID, models.UserRoleAnalyst, true, 50, 10)
	requireNoError(t, err)
	requireEqual(t, actorUserID, captured.ActorUserID)
	requireEqual(t, string(models.UserRoleAnalyst), captured.ActorRole)
	requireEqual(t, true, captured.IncludeArchived)
	requireEqual(t, 50, captured.Limit)
	requireEqual(t, 10, captured.Offset)
}

func TestGetProjectReturnsNilForBlankProjectID(t *testing.T) {
	service := NewService(nil)
	called := false
	service.deps.getProjectForActor = func(_ context.Context, _ repository.Queryer, _ repository.ProjectGetInput) (*models.ProjectRecord, error) {
		called = true
		return nil, nil
	}

	record, err := service.GetProject(
		context.Background(),
		uuid.MustParse("00000000-0000-0000-0000-000000000712"),
		models.UserRoleAnalyst,
		"   ",
		false,
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil record when project id is blank")
	}
	if called {
		t.Fatalf("expected repository not to be called for blank project id")
	}
}

func TestCreateProjectRejectsBlankProjectID(t *testing.T) {
	service := NewService(nil)
	_, err := service.CreateProject(
		context.Background(),
		uuid.MustParse("00000000-0000-0000-0000-000000000713"),
		models.UserRoleAdmin,
		CreateProjectRequest{ProjectID: "  "},
	)
	if !errors.Is(err, ErrProjectIDMustNotBeBlank) {
		t.Fatalf("expected ErrProjectIDMustNotBeBlank, got %v", err)
	}
}

func TestCreateProjectUsesActorOwnerWhenNonAdmin(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000714")
	requestedOwnerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000715")
	captured := runCreateProjectAndCaptureInput(
		t,
		actorUserID,
		models.UserRoleAnalyst,
		CreateProjectRequest{
			ProjectID:   "  project-alpha  ",
			Name:        "  Project Alpha  ",
			Description: "  Alpha  ",
			OwnerUserID: &requestedOwnerUserID,
		},
	)
	requireEqual(t, "project-alpha", captured.ProjectID)
	requireEqual(t, "Project Alpha", captured.Name)
	requireEqual(t, "Alpha", captured.Description)
	requireEqual(t, actorUserID, captured.OwnerUserID)
}

func TestCreateProjectAllowsAdminOwnerOverride(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000716")
	requestedOwnerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000717")
	captured := runCreateProjectAndCaptureInput(
		t,
		actorUserID,
		models.UserRoleAdmin,
		CreateProjectRequest{ProjectID: "project-admin", OwnerUserID: &requestedOwnerUserID},
	)
	requireEqual(t, requestedOwnerUserID, captured.OwnerUserID)
}

func TestSetDefaultProjectIDRequiresVisibleProject(t *testing.T) {
	service := NewService(nil)
	service.deps.getProjectForActor = func(_ context.Context, _ repository.Queryer, _ repository.ProjectGetInput) (*models.ProjectRecord, error) {
		return nil, nil
	}

	_, err := service.SetDefaultProjectID(
		context.Background(),
		uuid.MustParse("00000000-0000-0000-0000-000000000718"),
		models.UserRoleAnalyst,
		"engram-vault",
	)
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}
}

func TestSetDefaultProjectIDRequiresUserUpdate(t *testing.T) {
	service := NewService(nil)
	project := models.ProjectRecord{
		ProjectID:   "engram-vault",
		Name:        "Engram Vault",
		Description: "",
		OwnerUserID: uuid.MustParse("00000000-0000-0000-0000-000000000719"),
		IsArchived:  false,
		CreatedAt:   time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC),
	}
	service.deps.getProjectForActor = func(_ context.Context, _ repository.Queryer, _ repository.ProjectGetInput) (*models.ProjectRecord, error) {
		return &project, nil
	}
	service.deps.setUserDefault = func(_ context.Context, _ repository.Queryer, _ uuid.UUID, _ string) (bool, error) {
		return false, nil
	}

	_, err := service.SetDefaultProjectID(
		context.Background(),
		uuid.MustParse("00000000-0000-0000-0000-000000000720"),
		models.UserRoleAnalyst,
		"engram-vault",
	)
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestResolveProjectIDForWriteEnsuresExplicitProjectWhenHidden(t *testing.T) {
	service := NewService(nil)
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000721")
	getInputs := []repository.ProjectGetInput{}
	ensureInputs := []repository.ProjectEnsureInput{}
	service.deps.getProjectForActor = func(_ context.Context, _ repository.Queryer, input repository.ProjectGetInput) (*models.ProjectRecord, error) {
		getInputs = append(getInputs, input)
		return nil, nil
	}
	service.deps.ensureProjectExists = func(_ context.Context, _ repository.Queryer, input repository.ProjectEnsureInput) (*models.ProjectRecord, error) {
		ensureInputs = append(ensureInputs, input)
		return &models.ProjectRecord{ProjectID: input.ProjectID, OwnerUserID: input.OwnerUserID}, nil
	}

	resolution, err := service.ResolveProjectIDForWrite(
		context.Background(),
		actorUserID,
		models.UserRoleAnalyst,
		"  project-explicit  ",
	)
	requireNoError(t, err)
	requireEqual(t, "project-explicit", resolution.ProjectID)
	requireEqual(t, false, resolution.UsedDefaultProject)
	requireEqual(t, 1, len(getInputs))
	requireEqual(t, 1, len(ensureInputs))
	requireEqual(t, "project-explicit", ensureInputs[0].ProjectID)
	requireEqual(t, actorUserID, ensureInputs[0].OwnerUserID)
}

func TestResolveProjectIDForWriteUsesVisibleDefaultProject(t *testing.T) {
	service := NewService(nil)
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000722")
	defaultProjectID := "  project-default  "
	getInputs := []repository.ProjectGetInput{}
	service.deps.getUserDefault = func(_ context.Context, _ repository.Queryer, _ uuid.UUID) (*string, error) {
		return &defaultProjectID, nil
	}
	service.deps.getProjectForActor = func(_ context.Context, _ repository.Queryer, input repository.ProjectGetInput) (*models.ProjectRecord, error) {
		getInputs = append(getInputs, input)
		return &models.ProjectRecord{ProjectID: "project-default", OwnerUserID: actorUserID}, nil
	}

	resolution, err := service.ResolveProjectIDForWrite(
		context.Background(),
		actorUserID,
		models.UserRoleAnalyst,
		"",
	)
	requireNoError(t, err)
	requireEqual(t, "project-default", resolution.ProjectID)
	requireEqual(t, true, resolution.UsedDefaultProject)
	requireEqual(t, 1, len(getInputs))
	requireEqual(t, "project-default", getInputs[0].ProjectID)
}

func TestResolveProjectIDForWriteHandlesMissingAndInaccessibleDefault(t *testing.T) {
	t.Run("missing default", func(t *testing.T) {
		service := NewService(nil)
		service.deps.getUserDefault = func(_ context.Context, _ repository.Queryer, _ uuid.UUID) (*string, error) {
			return nil, nil
		}

		_, err := service.ResolveProjectIDForWrite(
			context.Background(),
			uuid.MustParse("00000000-0000-0000-0000-000000000723"),
			models.UserRoleAnalyst,
			"",
		)
		if !errors.Is(err, ErrProjectIDRequiredWhenNoDefaultProject) {
			t.Fatalf("expected ErrProjectIDRequiredWhenNoDefaultProject, got %v", err)
		}
	})

	t.Run("default inaccessible", func(t *testing.T) {
		service := NewService(nil)
		defaultProjectID := "project-default"
		service.deps.getUserDefault = func(_ context.Context, _ repository.Queryer, _ uuid.UUID) (*string, error) {
			return &defaultProjectID, nil
		}
		service.deps.getProjectForActor = func(_ context.Context, _ repository.Queryer, _ repository.ProjectGetInput) (*models.ProjectRecord, error) {
			return nil, nil
		}

		_, err := service.ResolveProjectIDForWrite(
			context.Background(),
			uuid.MustParse("00000000-0000-0000-0000-000000000724"),
			models.UserRoleAnalyst,
			"",
		)
		if !errors.Is(err, ErrDefaultProjectNotAccessible) {
			t.Fatalf("expected ErrDefaultProjectNotAccessible, got %v", err)
		}
	})
}

func TestResolveOwnerUserIDUsesAdminOverrideOnly(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000725")
	overrideUserID := uuid.MustParse("00000000-0000-0000-0000-000000000726")

	if got := resolveOwnerUserID(actorUserID, models.UserRoleAdmin, &overrideUserID); got != overrideUserID {
		t.Fatalf("expected admin override owner %v, got %v", overrideUserID, got)
	}
	if got := resolveOwnerUserID(actorUserID, models.UserRoleAnalyst, &overrideUserID); got != actorUserID {
		t.Fatalf("expected analyst owner %v, got %v", actorUserID, got)
	}
	if got := resolveOwnerUserID(actorUserID, models.UserRoleAdmin, nil); got != actorUserID {
		t.Fatalf("expected missing override to keep actor owner %v, got %v", actorUserID, got)
	}
}

func requireEqual[T any](t *testing.T, expected, actual T) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %#v, got %#v", expected, actual)
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func runCreateProjectAndCaptureInput(
	t *testing.T,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request CreateProjectRequest,
) repository.ProjectCreateInput {
	t.Helper()
	service := NewService(nil)
	captured := repository.ProjectCreateInput{}
	service.deps.createProject = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.ProjectCreateInput,
	) (*models.ProjectRecord, error) {
		captured = input
		return &models.ProjectRecord{ProjectID: input.ProjectID, OwnerUserID: input.OwnerUserID}, nil
	}

	_, err := service.CreateProject(context.Background(), actorUserID, actorRole, request)
	requireNoError(t, err)
	return captured
}

package projects

import (
	"context"
	"strings"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

type serviceDeps struct {
	listProjectsForActor      func(ctx context.Context, db repository.Queryer, input repository.ProjectListInput) ([]models.ProjectRecord, error)
	getProjectForActor        func(ctx context.Context, db repository.Queryer, input repository.ProjectGetInput) (*models.ProjectRecord, error)
	createProject             func(ctx context.Context, db repository.Queryer, input repository.ProjectCreateInput) (*models.ProjectRecord, error)
	ensureProjectExists       func(ctx context.Context, db repository.Queryer, input repository.ProjectEnsureInput) (*models.ProjectRecord, error)
	getUserDefault            func(ctx context.Context, db repository.Queryer, userID uuid.UUID) (*string, error)
	setUserDefault            func(ctx context.Context, db repository.Queryer, userID uuid.UUID, projectID string) (bool, error)
	resolveActorProjectRole   func(ctx context.Context, db repository.Queryer, input repository.ActorProjectRoleInput) (*models.ProjectMemberRole, error)
	listProjectMembers        func(ctx context.Context, db repository.Queryer, input repository.ProjectMemberListInput) ([]models.ProjectMemberRecord, error)
	addOrRestoreProjectMember func(ctx context.Context, db repository.Queryer, input repository.ProjectMemberAddInput) (*models.ProjectMemberRecord, error)
	updateProjectMemberRole   func(ctx context.Context, db repository.Queryer, input repository.ProjectMemberUpdateInput) (*models.ProjectMemberRecord, error)
	removeProjectMember       func(ctx context.Context, db repository.Queryer, input repository.ProjectMemberRemoveInput) (bool, error)
	getProjectMember          func(ctx context.Context, db repository.Queryer, input repository.ProjectMemberGetInput) (*models.ProjectMemberRecord, error)
	createProjectAuditEvent   func(ctx context.Context, db repository.Queryer, input repository.ProjectAuditEventCreateInput) (*models.ProjectAuditEventRecord, error)
	listProjectAuditEvents    func(ctx context.Context, db repository.Queryer, input repository.ProjectAuditEventListInput) ([]models.ProjectAuditEventRecord, error)
	getEngramShareRecord      func(ctx context.Context, db repository.Queryer, engramID uuid.UUID, includeDeleted bool) (*repository.EngramShareRecord, error)
	updateEngramVisibility    func(ctx context.Context, db repository.Queryer, input repository.EngramVisibilityUpdateInput) (*repository.EngramShareRecord, error)
}

func defaultServiceDeps() serviceDeps {
	return serviceDeps{
		listProjectsForActor:      repository.ListProjectsForActor,
		getProjectForActor:        repository.GetProjectForActor,
		createProject:             repository.CreateProject,
		ensureProjectExists:       repository.EnsureProjectExists,
		getUserDefault:            repository.GetUserDefaultProjectID,
		setUserDefault:            repository.SetUserDefaultProjectID,
		resolveActorProjectRole:   repository.ResolveActorProjectRole,
		listProjectMembers:        repository.ListProjectMembers,
		addOrRestoreProjectMember: repository.AddOrRestoreProjectMember,
		updateProjectMemberRole:   repository.UpdateProjectMemberRole,
		removeProjectMember:       repository.RemoveProjectMember,
		getProjectMember:          repository.GetProjectMember,
		createProjectAuditEvent:   repository.CreateProjectAuditEvent,
		listProjectAuditEvents:    repository.ListProjectAuditEvents,
		getEngramShareRecord:      repository.GetEngramShareRecord,
		updateEngramVisibility:    repository.UpdateEngramVisibilityScope,
	}
}

// Service owns project business rules on top of repository operations.
type Service struct {
	db   repository.Queryer
	deps serviceDeps
}

// NewService builds a project service with repository defaults.
func NewService(db repository.Queryer) *Service {
	return &Service{db: db, deps: defaultServiceDeps()}
}

func resolveOwnerUserID(actorUserID uuid.UUID, actorRole models.UserRole, requestedOwnerUserID *uuid.UUID) uuid.UUID {
	if actorRole == models.UserRoleAdmin && requestedOwnerUserID != nil {
		return *requestedOwnerUserID
	}
	return actorUserID
}

func normalizeProjectID(projectID string) string {
	return strings.TrimSpace(projectID)
}

func (service *Service) getVisibleProject(
	ctx context.Context,
	lookup projectVisibilityLookup,
) (*models.ProjectRecord, error) {
	return service.deps.getProjectForActor(
		ctx,
		service.db,
		repository.ProjectGetInput{
			ProjectID:       lookup.ProjectID,
			ActorUserID:     lookup.Actor.UserID,
			ActorRole:       string(lookup.Actor.Role),
			IncludeArchived: lookup.IncludeArchived,
		},
	)
}

func canCreateProject(actorRole models.UserRole) bool {
	return actorRole == models.UserRoleAdmin || actorRole == models.UserRoleAnalyst
}

func canWriteVisibleProject(actorRole models.UserRole, membershipRole *models.ProjectMemberRole) bool {
	if actorRole == models.UserRoleAdmin {
		return true
	}
	if membershipRole == nil {
		return false
	}
	switch *membershipRole {
	case models.ProjectMemberRoleOwner, models.ProjectMemberRoleEditor:
		return true
	default:
		return false
	}
}

func (service *Service) requireProjectMemberManager(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
) (*models.ProjectRecord, error) {
	normalizedProjectID := normalizeProjectID(projectID)
	if normalizedProjectID == "" {
		return nil, ErrProjectIDMustNotBeBlank
	}
	project, err := service.getVisibleProject(
		ctx,
		projectVisibilityLookup{
			Actor:           ActorContext{UserID: actorUserID, Role: actorRole},
			ProjectID:       normalizedProjectID,
			IncludeArchived: true,
		},
	)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}
	if actorRole != models.UserRoleAdmin && project.OwnerUserID != actorUserID {
		return nil, ErrProjectMemberManagementForbidden
	}
	return project, nil
}

func (service *Service) ensureCanonicalOwnerMembership(
	ctx context.Context,
	project models.ProjectRecord,
) error {
	if service.db == nil || service.deps.addOrRestoreProjectMember == nil {
		return nil
	}
	if strings.TrimSpace(project.ProjectID) == "" {
		return nil
	}
	_, err := service.deps.addOrRestoreProjectMember(
		ctx,
		service.db,
		repository.ProjectMemberAddInput{
			ProjectID:     project.ProjectID,
			UserID:        project.OwnerUserID,
			Role:          models.ProjectMemberRoleOwner,
			AddedByUserID: project.OwnerUserID,
		},
	)
	return err
}

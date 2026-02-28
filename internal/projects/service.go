package projects

import (
	"context"
	"errors"
	"strings"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

var (
	// ErrProjectIDMustNotBeBlank indicates project_id was empty after normalization.
	ErrProjectIDMustNotBeBlank = errors.New("project_id must not be blank")
	// ErrProjectNotFound indicates a project is not visible to the caller.
	ErrProjectNotFound = errors.New("Project not found")
	// ErrUserNotFound indicates no user row matched a default-project update.
	ErrUserNotFound = errors.New("User not found")
	// ErrProjectIDRequiredWhenNoDefaultProject indicates a write call needs explicit project context.
	ErrProjectIDRequiredWhenNoDefaultProject = errors.New("project_id is required when no default project is configured")
	// ErrDefaultProjectNotAccessible indicates a default project cannot be used by the caller.
	ErrDefaultProjectNotAccessible = errors.New("default project is not accessible; set a valid default project first")
	// ErrProjectWriteForbidden indicates actor lacks write access for the selected project.
	ErrProjectWriteForbidden = errors.New("project is not writable by actor")
	// ErrProjectMemberManagementForbidden indicates only owner/admin can manage members.
	ErrProjectMemberManagementForbidden = errors.New("project member management requires owner or admin role")
	// ErrProjectAuditForbidden indicates only owner/admin can view project audit events.
	ErrProjectAuditForbidden = errors.New("project audit visibility requires owner or admin role")
	// ErrProjectMemberNotFound indicates project member row does not exist.
	ErrProjectMemberNotFound = errors.New("Project member not found")
	// ErrProjectOwnerMembershipImmutable indicates owner row cannot be modified via member APIs.
	ErrProjectOwnerMembershipImmutable = errors.New("owner membership cannot be modified via member APIs")
	// ErrProjectOwnerRoleNotAssignable indicates owner role cannot be assigned via member APIs.
	ErrProjectOwnerRoleNotAssignable = errors.New("owner role cannot be assigned via member APIs")
	// ErrEngramNotFound indicates target engram was not found.
	ErrEngramNotFound = errors.New("Engram not found")
	// ErrEngramShareForbidden indicates actor cannot share/unshare this engram.
	ErrEngramShareForbidden = errors.New("actor cannot share or unshare this engram")
)

// CreateProjectRequest captures project creation input.
type CreateProjectRequest struct {
	ProjectID   string
	Name        string
	Description string
	OwnerUserID *uuid.UUID
}

// ProjectMemberCreateRequest captures member create payload.
type ProjectMemberCreateRequest struct {
	ProjectID string
	UserID    uuid.UUID
	Role      models.ProjectMemberRole
}

// ProjectMemberUpdateRequest captures member update payload.
type ProjectMemberUpdateRequest struct {
	ProjectID string
	UserID    uuid.UUID
	Role      models.ProjectMemberRole
}

// ProjectMemberRemoveRequest captures member remove payload.
type ProjectMemberRemoveRequest struct {
	ProjectID string
	UserID    uuid.UUID
}

// ProjectAuditListRequest captures audit list filters.
type ProjectAuditListRequest struct {
	ProjectID string
	Limit     int
	Offset    int
}

// Resolution captures resolved project context for write operations.
type Resolution struct {
	ProjectID          string
	UsedDefaultProject bool
}

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
	getProjectMember          func(ctx context.Context, db repository.Queryer, projectID string, userID uuid.UUID, includeRevoked bool) (*models.ProjectMemberRecord, error)
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

// ListProjects returns project rows visible to the actor.
func (service *Service) ListProjects(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	includeArchived bool,
	limit int,
	offset int,
) ([]models.ProjectRecord, error) {
	return service.deps.listProjectsForActor(
		ctx,
		service.db,
		repository.ProjectListInput{
			ActorUserID:     actorUserID,
			ActorRole:       string(actorRole),
			IncludeArchived: includeArchived,
			Limit:           limit,
			Offset:          offset,
		},
	)
}

// GetProject returns a visible project or nil when missing.
func (service *Service) GetProject(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
	includeArchived bool,
) (*models.ProjectRecord, error) {
	normalizedProjectID := normalizeProjectID(projectID)
	if normalizedProjectID == "" {
		return nil, nil
	}
	return service.getVisibleProject(ctx, actorUserID, actorRole, normalizedProjectID, includeArchived)
}

// CreateProject creates a project and applies owner rules.
func (service *Service) CreateProject(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request CreateProjectRequest,
) (*models.ProjectRecord, error) {
	normalizedProjectID := normalizeProjectID(request.ProjectID)
	if normalizedProjectID == "" {
		return nil, ErrProjectIDMustNotBeBlank
	}
	if !canCreateProject(actorRole) {
		return nil, ErrProjectWriteForbidden
	}
	ownerUserID := resolveOwnerUserID(actorUserID, actorRole, request.OwnerUserID)
	created, err := service.deps.createProject(
		ctx,
		service.db,
		repository.ProjectCreateInput{
			ProjectID:   normalizedProjectID,
			Name:        strings.TrimSpace(request.Name),
			Description: strings.TrimSpace(request.Description),
			OwnerUserID: ownerUserID,
		},
	)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return nil, nil
	}
	if err := service.ensureCanonicalOwnerMembership(ctx, *created); err != nil {
		return nil, err
	}
	return created, nil
}

// GetDefaultProjectID returns the caller's default project id when present.
func (service *Service) GetDefaultProjectID(ctx context.Context, actorUserID uuid.UUID) (*string, error) {
	return service.deps.getUserDefault(ctx, service.db, actorUserID)
}

// SetDefaultProjectID updates the caller's default project id after visibility checks.
func (service *Service) SetDefaultProjectID(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
) (string, error) {
	normalizedProjectID := normalizeProjectID(projectID)
	if normalizedProjectID == "" {
		return "", ErrProjectIDMustNotBeBlank
	}
	visible, err := service.getVisibleProject(ctx, actorUserID, actorRole, normalizedProjectID, false)
	if err != nil {
		return "", err
	}
	if visible == nil {
		return "", ErrProjectNotFound
	}
	updated, err := service.deps.setUserDefault(ctx, service.db, actorUserID, normalizedProjectID)
	if err != nil {
		return "", err
	}
	if !updated {
		return "", ErrUserNotFound
	}
	return normalizedProjectID, nil
}

// ResolveProjectIDForWrite resolves explicit or default project context for a write call.
func (service *Service) ResolveProjectIDForWrite(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
) (Resolution, error) {
	explicitProjectID := normalizeProjectID(projectID)
	if explicitProjectID != "" {
		return service.resolveExplicitProjectID(ctx, actorUserID, actorRole, explicitProjectID)
	}
	return service.resolveDefaultProjectID(ctx, actorUserID, actorRole)
}

// ListProjectMembers lists members for a project visible to owner/admin.
func (service *Service) ListProjectMembers(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
	includeRevoked bool,
	limit int,
	offset int,
) ([]models.ProjectMemberRecord, error) {
	project, err := service.requireProjectMemberManager(ctx, actorUserID, actorRole, projectID)
	if err != nil {
		return nil, err
	}
	if err := service.ensureCanonicalOwnerMembership(ctx, *project); err != nil {
		return nil, err
	}
	return service.deps.listProjectMembers(
		ctx,
		service.db,
		repository.ProjectMemberListInput{
			ProjectID:      project.ProjectID,
			IncludeRevoked: includeRevoked,
			Limit:          limit,
			Offset:         offset,
		},
	)
}

// AddProjectMember adds or restores a project member role.
func (service *Service) AddProjectMember(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request ProjectMemberCreateRequest,
) (*models.ProjectMemberRecord, error) {
	if request.Role == models.ProjectMemberRoleOwner {
		return nil, ErrProjectOwnerRoleNotAssignable
	}
	project, err := service.requireProjectMemberManager(ctx, actorUserID, actorRole, request.ProjectID)
	if err != nil {
		return nil, err
	}
	if request.UserID == project.OwnerUserID {
		return nil, ErrProjectOwnerMembershipImmutable
	}
	member, err := service.deps.addOrRestoreProjectMember(
		ctx,
		service.db,
		repository.ProjectMemberAddInput{
			ProjectID:     project.ProjectID,
			UserID:        request.UserID,
			Role:          request.Role,
			AddedByUserID: actorUserID,
		},
	)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrProjectMemberNotFound
	}
	if err := service.emitProjectAuditEvent(
		ctx,
		repository.ProjectAuditEventCreateInput{
			ProjectID:    project.ProjectID,
			ActorUserID:  &actorUserID,
			EventType:    "project.member.add",
			TargetType:   "user",
			TargetUserID: &request.UserID,
			Metadata: map[string]any{
				"role": string(request.Role),
			},
		},
	); err != nil {
		return nil, err
	}
	return member, nil
}

// UpdateProjectMember updates an existing member role.
func (service *Service) UpdateProjectMember(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request ProjectMemberUpdateRequest,
) (*models.ProjectMemberRecord, error) {
	if request.Role == models.ProjectMemberRoleOwner {
		return nil, ErrProjectOwnerRoleNotAssignable
	}
	project, err := service.requireProjectMemberManager(ctx, actorUserID, actorRole, request.ProjectID)
	if err != nil {
		return nil, err
	}
	if request.UserID == project.OwnerUserID {
		return nil, ErrProjectOwnerMembershipImmutable
	}
	current, err := service.deps.getProjectMember(ctx, service.db, project.ProjectID, request.UserID, false)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, ErrProjectMemberNotFound
	}
	updated, err := service.deps.updateProjectMemberRole(
		ctx,
		service.db,
		repository.ProjectMemberUpdateInput{
			ProjectID:       project.ProjectID,
			UserID:          request.UserID,
			Role:            request.Role,
			UpdatedByUserID: actorUserID,
		},
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrProjectMemberNotFound
	}
	if err := service.emitProjectAuditEvent(
		ctx,
		repository.ProjectAuditEventCreateInput{
			ProjectID:    project.ProjectID,
			ActorUserID:  &actorUserID,
			EventType:    "project.member.update",
			TargetType:   "user",
			TargetUserID: &request.UserID,
			Metadata: map[string]any{
				"from_role": string(current.Role),
				"to_role":   string(request.Role),
			},
		},
	); err != nil {
		return nil, err
	}
	return updated, nil
}

// RemoveProjectMember revokes an active member role.
func (service *Service) RemoveProjectMember(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request ProjectMemberRemoveRequest,
) error {
	project, err := service.requireProjectMemberManager(ctx, actorUserID, actorRole, request.ProjectID)
	if err != nil {
		return err
	}
	if request.UserID == project.OwnerUserID {
		return ErrProjectOwnerMembershipImmutable
	}
	removed, err := service.deps.removeProjectMember(
		ctx,
		service.db,
		repository.ProjectMemberRemoveInput{
			ProjectID:       project.ProjectID,
			UserID:          request.UserID,
			RevokedByUserID: actorUserID,
		},
	)
	if err != nil {
		return err
	}
	if !removed {
		return ErrProjectMemberNotFound
	}
	if err := service.emitProjectAuditEvent(
		ctx,
		repository.ProjectAuditEventCreateInput{
			ProjectID:    project.ProjectID,
			ActorUserID:  &actorUserID,
			EventType:    "project.member.remove",
			TargetType:   "user",
			TargetUserID: &request.UserID,
		},
	); err != nil {
		return err
	}
	return nil
}

// ListProjectAuditEvents lists project audit rows for owner/admin.
func (service *Service) ListProjectAuditEvents(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	request ProjectAuditListRequest,
) ([]models.ProjectAuditEventRecord, error) {
	project, err := service.requireProjectMemberManager(ctx, actorUserID, actorRole, request.ProjectID)
	if err != nil {
		if errors.Is(err, ErrProjectMemberManagementForbidden) {
			return nil, ErrProjectAuditForbidden
		}
		return nil, err
	}
	return service.deps.listProjectAuditEvents(
		ctx,
		service.db,
		repository.ProjectAuditEventListInput{
			ProjectID: project.ProjectID,
			Limit:     request.Limit,
			Offset:    request.Offset,
		},
	)
}

// ShareEngram sets engram visibility_scope=project.
func (service *Service) ShareEngram(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engramID uuid.UUID,
) (*models.EngramVisibilityRecord, error) {
	return service.setEngramVisibility(ctx, actorUserID, actorRole, engramID, models.VisibilityScopeProject, "engram.share")
}

// UnshareEngram sets engram visibility_scope=private.
func (service *Service) UnshareEngram(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engramID uuid.UUID,
) (*models.EngramVisibilityRecord, error) {
	return service.setEngramVisibility(ctx, actorUserID, actorRole, engramID, models.VisibilityScopePrivate, "engram.unshare")
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
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
	includeArchived bool,
) (*models.ProjectRecord, error) {
	return service.deps.getProjectForActor(
		ctx,
		service.db,
		repository.ProjectGetInput{
			ProjectID:       projectID,
			ActorUserID:     actorUserID,
			ActorRole:       string(actorRole),
			IncludeArchived: includeArchived,
		},
	)
}

func (service *Service) resolveExplicitProjectID(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	projectID string,
) (Resolution, error) {
	visible, err := service.getVisibleProject(ctx, actorUserID, actorRole, projectID, false)
	if err != nil {
		return Resolution{}, err
	}
	if visible != nil {
		if !canWriteVisibleProject(actorRole, visible.MembershipRole) {
			return Resolution{}, ErrProjectWriteForbidden
		}
		return Resolution{ProjectID: projectID, UsedDefaultProject: false}, nil
	}

	existing, err := service.getVisibleProject(ctx, actorUserID, models.UserRoleAdmin, projectID, true)
	if err != nil {
		return Resolution{}, err
	}
	if existing != nil {
		return Resolution{}, ErrProjectWriteForbidden
	}
	if !canCreateProject(actorRole) {
		return Resolution{}, ErrProjectWriteForbidden
	}
	created, err := service.deps.ensureProjectExists(
		ctx,
		service.db,
		repository.ProjectEnsureInput{ProjectID: projectID, OwnerUserID: actorUserID},
	)
	if err != nil {
		return Resolution{}, err
	}
	if created != nil {
		if err := service.ensureCanonicalOwnerMembership(ctx, *created); err != nil {
			return Resolution{}, err
		}
	}
	return Resolution{ProjectID: projectID, UsedDefaultProject: false}, nil
}

func (service *Service) resolveDefaultProjectID(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
) (Resolution, error) {
	defaultProjectID, err := service.deps.getUserDefault(ctx, service.db, actorUserID)
	if err != nil {
		return Resolution{}, err
	}
	normalizedDefaultProjectID := normalizeOptionalDefaultProjectID(defaultProjectID)
	if normalizedDefaultProjectID == "" {
		return Resolution{}, ErrProjectIDRequiredWhenNoDefaultProject
	}
	visible, err := service.getVisibleProject(ctx, actorUserID, actorRole, normalizedDefaultProjectID, false)
	if err != nil {
		return Resolution{}, err
	}
	if visible == nil {
		return Resolution{}, ErrDefaultProjectNotAccessible
	}
	if !canWriteVisibleProject(actorRole, visible.MembershipRole) {
		return Resolution{}, ErrProjectWriteForbidden
	}
	return Resolution{ProjectID: normalizedDefaultProjectID, UsedDefaultProject: true}, nil
}

func normalizeOptionalDefaultProjectID(defaultProjectID *string) string {
	if defaultProjectID == nil {
		return ""
	}
	return normalizeProjectID(*defaultProjectID)
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
	project, err := service.getVisibleProject(ctx, actorUserID, actorRole, normalizedProjectID, true)
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

func (service *Service) emitProjectAuditEvent(
	ctx context.Context,
	input repository.ProjectAuditEventCreateInput,
) error {
	_, err := service.deps.createProjectAuditEvent(ctx, service.db, input)
	return err
}

func (service *Service) setEngramVisibility(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engramID uuid.UUID,
	targetScope models.VisibilityScope,
	eventType string,
) (*models.EngramVisibilityRecord, error) {
	current, err := service.deps.getEngramShareRecord(ctx, service.db, engramID, false)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, ErrEngramNotFound
	}
	allowed, err := service.canShareEngram(ctx, actorUserID, actorRole, *current)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrEngramShareForbidden
	}
	updated, err := service.deps.updateEngramVisibility(
		ctx,
		service.db,
		repository.EngramVisibilityUpdateInput{
			EngramID:        engramID,
			VisibilityScope: targetScope,
			ActorUserID:     actorUserID,
		},
	)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrEngramNotFound
	}
	targetEngramID := updated.EngramID
	if err := service.emitProjectAuditEvent(
		ctx,
		repository.ProjectAuditEventCreateInput{
			ProjectID:      updated.ProjectID,
			ActorUserID:    &actorUserID,
			EventType:      eventType,
			TargetType:     "engram",
			TargetEngramID: &targetEngramID,
			Metadata: map[string]any{
				"from_visibility_scope": string(current.VisibilityScope),
				"to_visibility_scope":   string(updated.VisibilityScope),
			},
		},
	); err != nil {
		return nil, err
	}
	return &models.EngramVisibilityRecord{
		EngramID:        updated.EngramID,
		ProjectID:       updated.ProjectID,
		VisibilityScope: updated.VisibilityScope,
	}, nil
}

func (service *Service) canShareEngram(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engram repository.EngramShareRecord,
) (bool, error) {
	if actorRole == models.UserRoleAdmin {
		return true, nil
	}
	projectRole, err := service.deps.resolveActorProjectRole(
		ctx,
		service.db,
		repository.ActorProjectRoleInput{
			ProjectID:   engram.ProjectID,
			ActorUserID: actorUserID,
		},
	)
	if err != nil {
		return false, err
	}
	if projectRole == nil {
		return false, nil
	}
	switch *projectRole {
	case models.ProjectMemberRoleOwner:
		return true, nil
	case models.ProjectMemberRoleEditor:
		return engram.OwnerUserID != nil && *engram.OwnerUserID == actorUserID, nil
	default:
		return false, nil
	}
}

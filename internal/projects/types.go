package projects

import (
	"engram/internal/models"

	"github.com/google/uuid"
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

// ActorContext captures actor identity details for project authorization paths.
type ActorContext struct {
	UserID uuid.UUID
	Role   models.UserRole
}

// ProjectListRequest captures list-project filters.
type ProjectListRequest struct {
	IncludeArchived bool
	Limit           int
	Offset          int
}

// ProjectGetRequest captures one-project visibility filters.
type ProjectGetRequest struct {
	ProjectID       string
	IncludeArchived bool
}

// ProjectMemberListRequest captures project-member list filters.
type ProjectMemberListRequest struct {
	ProjectID      string
	IncludeRevoked bool
	Limit          int
	Offset         int
}

// Resolution captures resolved project context for write operations.
type Resolution struct {
	ProjectID          string
	UsedDefaultProject bool
}

type projectVisibilityLookup struct {
	Actor           ActorContext
	ProjectID       string
	IncludeArchived bool
}

type engramVisibilityMutationInput struct {
	Actor       ActorContext
	EngramID    uuid.UUID
	TargetScope models.VisibilityScope
	EventType   string
}

type memberMutationInput struct {
	ProjectID   string
	TargetUser  uuid.UUID
	ActorUserID uuid.UUID
	Role        models.ProjectMemberRole
}

type memberAuditInput struct {
	ProjectID   string
	ActorUserID uuid.UUID
	TargetUser  uuid.UUID
	EventType   string
	Metadata    map[string]any
}

type engramActorVisibilityInput struct {
	Actor       ActorContext
	EngramID    uuid.UUID
	TargetScope models.VisibilityScope
	EventType   string
}

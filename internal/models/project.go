package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ProjectMemberRole captures collaboration membership roles for projects.
type ProjectMemberRole string

const (
	ProjectMemberRoleOwner  ProjectMemberRole = "owner"
	ProjectMemberRoleEditor ProjectMemberRole = "editor"
	ProjectMemberRoleViewer ProjectMemberRole = "viewer"
)

// ParseProjectMemberRole validates and converts a role string into ProjectMemberRole.
func ParseProjectMemberRole(value string) (ProjectMemberRole, error) {
	switch ProjectMemberRole(value) {
	case ProjectMemberRoleOwner, ProjectMemberRoleEditor, ProjectMemberRoleViewer:
		return ProjectMemberRole(value), nil
	default:
		return "", fmt.Errorf("unsupported project member role %q", value)
	}
}

// ProjectRecord models the persisted project row.
type ProjectRecord struct {
	ProjectID      string             `json:"project_id"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	OwnerUserID    uuid.UUID          `json:"owner_user_id"`
	MembershipRole *ProjectMemberRole `json:"membership_role,omitempty"`
	IsArchived     bool               `json:"is_archived"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

// ProjectCreateRequest captures project creation payloads from the API layer.
type ProjectCreateRequest struct {
	ProjectID   string     `json:"project_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	OwnerUserID *uuid.UUID `json:"owner_user_id,omitempty"`
}

// ProjectDefaultResponse mirrors default-project API responses.
type ProjectDefaultResponse struct {
	DefaultProjectID *string `json:"default_project_id"`
}

// ProjectDefaultUpdateRequest captures default-project update payloads.
type ProjectDefaultUpdateRequest struct {
	ProjectID string `json:"project_id"`
}

// ProjectMemberCreateRequest captures membership-create payloads.
type ProjectMemberCreateRequest struct {
	UserID uuid.UUID         `json:"user_id"`
	Role   ProjectMemberRole `json:"role"`
}

// ProjectMemberUpdateRequest captures membership-update payloads.
type ProjectMemberUpdateRequest struct {
	Role ProjectMemberRole `json:"role"`
}

// ProjectMemberRecord models active/revoked project membership rows.
type ProjectMemberRecord struct {
	ProjectID       string            `json:"project_id"`
	UserID          uuid.UUID         `json:"user_id"`
	Role            ProjectMemberRole `json:"role"`
	AddedByUserID   *uuid.UUID        `json:"added_by_user_id,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	RevokedAt       *time.Time        `json:"revoked_at,omitempty"`
	RevokedByUserID *uuid.UUID        `json:"revoked_by_user_id,omitempty"`
}

// ProjectAuditEventRecord models project audit trail records.
type ProjectAuditEventRecord struct {
	EventID        uuid.UUID      `json:"event_id"`
	ProjectID      string         `json:"project_id"`
	ActorUserID    *uuid.UUID     `json:"actor_user_id,omitempty"`
	EventType      string         `json:"event_type"`
	TargetType     string         `json:"target_type"`
	TargetUserID   *uuid.UUID     `json:"target_user_id,omitempty"`
	TargetEngramID *uuid.UUID     `json:"target_engram_id,omitempty"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
}

// EngramVisibilityRecord captures share/unshare visibility outputs.
type EngramVisibilityRecord struct {
	EngramID        uuid.UUID       `json:"engram_id"`
	ProjectID       string          `json:"project_id"`
	VisibilityScope VisibilityScope `json:"visibility_scope"`
}

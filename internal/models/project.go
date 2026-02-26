package models

import (
	"time"

	"github.com/google/uuid"
)

// ProjectRecord models the persisted project row.
type ProjectRecord struct {
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerUserID uuid.UUID `json:"owner_user_id"`
	IsArchived  bool      `json:"is_archived"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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

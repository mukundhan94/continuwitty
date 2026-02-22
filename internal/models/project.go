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

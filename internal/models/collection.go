package models

import (
	"time"

	"github.com/google/uuid"
)

// EngramCollectionRecord models persisted collection rows.
type EngramCollectionRecord struct {
	CollectionID    uuid.UUID  `json:"collection_id"`
	ProjectID       string     `json:"project_id"`
	OwnerUserID     uuid.UUID  `json:"owner_user_id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	DeletedByUserID *uuid.UUID `json:"deleted_by_user_id,omitempty"`
	DeleteReason    *string    `json:"delete_reason,omitempty"`
}

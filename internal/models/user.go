package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UserRole mirrors the Python user role enum.
type UserRole string

const (
	UserRoleAdmin   UserRole = "admin"
	UserRoleAnalyst UserRole = "analyst"
	UserRoleViewer  UserRole = "viewer"
)

// ParseUserRole validates and converts a role string into UserRole.
func ParseUserRole(role string) (UserRole, error) {
	switch UserRole(role) {
	case UserRoleAdmin, UserRoleAnalyst, UserRoleViewer:
		return UserRole(role), nil
	default:
		return "", fmt.Errorf("unsupported user role %q", role)
	}
}

// UserRecord is the public user row shape without password hash.
type UserRecord struct {
	UserID           uuid.UUID `json:"user_id"`
	Username         string    `json:"username"`
	Role             UserRole  `json:"role"`
	IsActive         bool      `json:"is_active"`
	DefaultProjectID *string   `json:"default_project_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// UserAuthRecord includes authentication-only columns.
type UserAuthRecord struct {
	UserID           uuid.UUID `json:"user_id"`
	Username         string    `json:"username"`
	PasswordHash     string    `json:"password_hash"`
	Role             UserRole  `json:"role"`
	IsActive         bool      `json:"is_active"`
	DefaultProjectID *string   `json:"default_project_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

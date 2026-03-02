package models

import (
	"time"

	"github.com/google/uuid"
)

// MCPTokenCreateRequest captures API payload for creating an MCP token.
type MCPTokenCreateRequest struct {
	Name              string   `json:"name"`
	Scope             string   `json:"scope"`
	AllowedTools      []string `json:"allowed_tools"`
	AllowedProjectIDs []string `json:"allowed_project_ids"`
	ExpiresInDays     int      `json:"expires_in_days"`
}

// MCPTokenCreateResponse returns persisted token metadata and plaintext token value.
type MCPTokenCreateResponse struct {
	TokenID           uuid.UUID     `json:"token_id"`
	Name              string        `json:"name"`
	Scope             MCPTokenScope `json:"scope"`
	AllowedTools      []string      `json:"allowed_tools"`
	AllowedProjectIDs []string      `json:"allowed_project_ids"`
	TokenSecretHint   string        `json:"token_secret_hint"`
	Token             string        `json:"token"`
	ExpiresAt         time.Time     `json:"expires_at"`
	CreatedAt         time.Time     `json:"created_at"`
}

// MCPTokenSummary exposes non-secret token metadata.
type MCPTokenSummary struct {
	TokenID           uuid.UUID     `json:"token_id"`
	Name              string        `json:"name"`
	Scope             MCPTokenScope `json:"scope"`
	AllowedTools      []string      `json:"allowed_tools"`
	AllowedProjectIDs []string      `json:"allowed_project_ids"`
	TokenSecretHint   string        `json:"token_secret_hint"`
	ExpiresAt         time.Time     `json:"expires_at"`
	LastUsedAt        *time.Time    `json:"last_used_at,omitempty"`
	RevokedAt         *time.Time    `json:"revoked_at,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	IsActive          bool          `json:"is_active"`
}

// MCPTokenRevokeRequest captures optional revoke metadata from callers.
type MCPTokenRevokeRequest struct {
	Reason *string `json:"reason,omitempty"`
}

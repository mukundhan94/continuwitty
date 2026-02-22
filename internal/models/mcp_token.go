package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	// MCPTokenPrefix prefixes plaintext token values issued by the service layer.
	MCPTokenPrefix = "engram_mcp"
)

// MCPTokenScope controls authorization breadth for MCP tokens.
type MCPTokenScope string

const (
	MCPTokenScopeRead  MCPTokenScope = "read"
	MCPTokenScopeWrite MCPTokenScope = "write"
)

// ParseMCPTokenScope validates MCP token scope values.
func ParseMCPTokenScope(value string) (MCPTokenScope, error) {
	switch MCPTokenScope(value) {
	case MCPTokenScopeRead, MCPTokenScopeWrite:
		return MCPTokenScope(value), nil
	default:
		return "", fmt.Errorf("unsupported mcp token scope %q", value)
	}
}

// MCPTokenRecord models persisted MCP token rows.
type MCPTokenRecord struct {
	TokenID           uuid.UUID     `json:"token_id"`
	OwnerUserID       uuid.UUID     `json:"owner_user_id"`
	Name              string        `json:"name"`
	Scope             MCPTokenScope `json:"scope"`
	AllowedTools      []string      `json:"allowed_tools"`
	AllowedProjectIDs []string      `json:"allowed_project_ids"`
	TokenSecretHash   string        `json:"token_secret_hash"`
	TokenSecretHint   string        `json:"token_secret_hint"`
	ExpiresAt         time.Time     `json:"expires_at"`
	LastUsedAt        *time.Time    `json:"last_used_at,omitempty"`
	RevokedAt         *time.Time    `json:"revoked_at,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
}

// MCPTokenAuthContext captures token claims used for runtime authorization.
type MCPTokenAuthContext struct {
	TokenID           uuid.UUID     `json:"token_id"`
	OwnerUserID       uuid.UUID     `json:"owner_user_id"`
	Scope             MCPTokenScope `json:"scope"`
	AllowedTools      []string      `json:"allowed_tools"`
	AllowedProjectIDs []string      `json:"allowed_project_ids"`
	ExpiresAt         time.Time     `json:"expires_at"`
}

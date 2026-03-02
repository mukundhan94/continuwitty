package mcp

import (
	"engram/internal/models"

	"github.com/google/uuid"
)

// JSONRPCRequest captures inbound MCP JSON-RPC request payloads.
type JSONRPCRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

// IsNotification reports whether the request should be handled as a JSON-RPC notification.
func (request JSONRPCRequest) IsNotification() bool {
	return request.ID == nil
}

// Frame represents a JSON-RPC response/event frame emitted by MCP services.
type Frame map[string]any

// Actor captures authenticated MCP caller identity.
type Actor struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username,omitempty"`
	Role     string    `json:"role"`
}

// ResolvedActor captures actor identity and optional token claims.
type ResolvedActor struct {
	Actor     Actor                       `json:"actor"`
	TokenAuth *models.MCPTokenAuthContext `json:"token_auth,omitempty"`
}

// StreamCallRequest wraps streamable-call input for MCP services.
type StreamCallRequest struct {
	Request   JSONRPCRequest
	Actor     Actor
	TokenAuth *models.MCPTokenAuthContext
}

package mcp

import "context"

// Service defines MCP JSON-RPC dispatch behavior for transport handlers.
type Service interface {
	HandleNotification(ctx context.Context, request JSONRPCRequest) error
	StreamCall(ctx context.Context, request StreamCallRequest) <-chan Frame
}

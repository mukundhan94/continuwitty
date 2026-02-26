package mcp

import (
	"context"
	"strings"

	"engram/internal/models"
)

const (
	mcpProtocolVersion = "2024-11-05"
	mcpServerName      = "engram-vault-mcp"
)

// CompatibilityService provides baseline MCP interop behavior while the full tool catalog migrates.
type CompatibilityService struct {
	serverVersion string
}

// NewCompatibilityService builds a compatibility MCP service with stable initialize/tool-list behavior.
func NewCompatibilityService(serverVersion string) *CompatibilityService {
	trimmed := strings.TrimSpace(serverVersion)
	if trimmed == "" {
		trimmed = "0.1.0"
	}
	return &CompatibilityService{serverVersion: trimmed}
}

// HandleNotification accepts JSON-RPC notifications and intentionally no-ops.
func (service *CompatibilityService) HandleNotification(_ context.Context, _ JSONRPCRequest) error {
	return nil
}

// StreamCall emits a single terminal response frame for the request.
func (service *CompatibilityService) StreamCall(ctx context.Context, request StreamCallRequest) <-chan Frame {
	frames := make(chan Frame, 1)
	go func() {
		defer close(frames)
		response := service.dispatch(request.Request, request.TokenAuth)
		select {
		case <-ctx.Done():
			return
		case frames <- response:
		}
	}()
	return frames
}

func (service *CompatibilityService) dispatch(
	request JSONRPCRequest,
	tokenAuth *models.MCPTokenAuthContext,
) Frame {
	if strings.TrimSpace(request.Method) == "" {
		return invalidRequestFrame(request.ID)
	}

	switch request.Method {
	case "initialize":
		return successFrame(request.ID, map[string]any{
			"protocolVersion": mcpProtocolVersion,
			"serverInfo": map[string]any{
				"name":    mcpServerName,
				"version": service.serverVersion,
			},
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
		})
	case "tools/list":
		return successFrame(request.ID, map[string]any{
			"tools": buildVisiblePublicToolCatalog(tokenAuth),
		})
	case "tools/call":
		return service.dispatchToolsCall(request.ID, request.Params, tokenAuth)
	default:
		return methodNotFoundFrame(request.ID, request.Method)
	}
}

func (service *CompatibilityService) dispatchToolsCall(
	requestID any,
	params map[string]any,
	tokenAuth *models.MCPTokenAuthContext,
) Frame {
	name, ok := requiredToolName(params)
	if !ok {
		return invalidParamsFrame(requestID, map[string]any{"missing": "name"})
	}
	dottedName := toDottedToolName(name)
	if !toolExists(dottedName) {
		return methodNotFoundFrame(requestID, name)
	}
	if policyError := authorizeToolCall(dottedName, tokenAuth); policyError != nil {
		return errorFrame(requestID, policyError.code, policyError.message, policyError.data)
	}
	return errorFrame(
		requestID,
		-32000,
		"Tool not implemented",
		map[string]any{"method": canonicalToolName(dottedName)},
	)
}

func requiredToolName(params map[string]any) (string, bool) {
	rawName, ok := params["name"]
	if !ok {
		return "", false
	}
	name, ok := rawName.(string)
	if !ok {
		return "", false
	}
	name = strings.TrimSpace(name)
	return name, name != ""
}

func successFrame(id any, result map[string]any) Frame {
	return Frame{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
}

func invalidRequestFrame(id any) Frame {
	return errorFrame(id, -32600, "Invalid Request", nil)
}

func invalidParamsFrame(id any, data map[string]any) Frame {
	return errorFrame(id, -32602, "Invalid params", data)
}

func methodNotFoundFrame(id any, method string) Frame {
	return errorFrame(id, -32601, "Method not found", map[string]any{"method": method})
}

func errorFrame(id any, code int, message string, data map[string]any) Frame {
	payload := map[string]any{
		"code":    code,
		"message": message,
	}
	if data != nil {
		payload["data"] = data
	}
	return Frame{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   payload,
	}
}

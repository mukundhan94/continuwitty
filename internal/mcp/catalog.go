package mcp

import (
	"strings"

	"engram/internal/models"
)

var readToolNames = map[string]struct{}{
	"chat.list_sessions":          {},
	"chat.get_session":            {},
	"chat.get_lifecycle_policy":   {},
	"chat.list_messages":          {},
	"chat.list_timeline":          {},
	"chat.list_pinned_engrams":    {},
	"chat.list_pinned_documents":  {},
	"chat.list_project_documents": {},
	"project.list":                {},
	"project.export_bundle":       {},
	"project.get_default":         {},
	"engram.list":                 {},
	"engram.get":                  {},
	"engram.collection_list":      {},
	"engram.query":                {},
	"engram.rehydrate":            {},
	"user.get_profile":            {},
	"user.list_projects":          {},
}

var writeToolNames = map[string]struct{}{
	"chat.create_session":             {},
	"chat.update_lifecycle_policy":    {},
	"chat.send_message":               {},
	"chat.pin_engram":                 {},
	"chat.unpin_engram":               {},
	"chat.pin_document":               {},
	"chat.unpin_document":             {},
	"chat.save_as_engram":             {},
	"chat.continue_session":           {},
	"chat.delete_session":             {},
	"chat.restore_session":            {},
	"project.create":                  {},
	"project.import_bundle":           {},
	"project.set_default":             {},
	"engram.create":                   {},
	"engram.create_from_conversation": {},
	"engram.pin_to_session":           {},
	"engram.update":                   {},
	"engram.move_project":             {},
	"engram.delete":                   {},
	"engram.restore":                  {},
	"engram.collection_create":        {},
	"engram.collection_update":        {},
	"engram.collection_delete":        {},
	"engram.collection_add_items":     {},
	"engram.collection_remove_items":  {},
}

var toolAliases = map[string]string{
	"engram.pin_to_session": "chat.pin_engram",
}

var toolCatalogOrder = []string{
	"chat.create_session",
	"chat.list_sessions",
	"chat.get_session",
	"chat.get_lifecycle_policy",
	"chat.update_lifecycle_policy",
	"chat.list_messages",
	"chat.list_timeline",
	"chat.send_message",
	"chat.list_pinned_engrams",
	"chat.pin_engram",
	"chat.unpin_engram",
	"chat.list_pinned_documents",
	"chat.pin_document",
	"chat.unpin_document",
	"chat.list_project_documents",
	"chat.save_as_engram",
	"chat.continue_session",
	"engram.create",
	"engram.create_from_conversation",
	"engram.query",
	"engram.rehydrate",
	"engram.pin_to_session",
	"chat.delete_session",
	"chat.restore_session",
	"project.list",
	"project.create",
	"project.get_default",
	"project.set_default",
	"project.export_bundle",
	"project.import_bundle",
	"engram.list",
	"engram.get",
	"engram.update",
	"engram.move_project",
	"engram.delete",
	"engram.restore",
	"engram.collection_list",
	"engram.collection_create",
	"engram.collection_update",
	"engram.collection_delete",
	"engram.collection_add_items",
	"engram.collection_remove_items",
	"user.get_profile",
	"user.list_projects",
}

var toolNamespacePrefixes = []string{"chat", "engram", "project", "user"}

func toPublicToolName(canonical string) string {
	return strings.ReplaceAll(canonical, ".", "_")
}

func toDottedToolName(raw string) string {
	if strings.Contains(raw, ".") {
		return raw
	}
	for _, namespace := range toolNamespacePrefixes {
		prefix := namespace + "_"
		if strings.HasPrefix(raw, prefix) {
			return namespace + "." + raw[len(prefix):]
		}
	}
	return raw
}

func canonicalToolName(raw string) string {
	dotted := toDottedToolName(strings.TrimSpace(raw))
	if alias, ok := toolAliases[dotted]; ok {
		return alias
	}
	return dotted
}

func toolExists(name string) bool {
	_, isRead := readToolNames[name]
	_, isWrite := writeToolNames[name]
	return isRead || isWrite
}

func requiresWriteScope(name string) bool {
	_, isWrite := writeToolNames[name]
	return isWrite
}

func buildVisiblePublicToolCatalog(tokenAuth *models.MCPTokenAuthContext) []map[string]any {
	allowedTools := canonicalAllowedTools(tokenAuth)
	tools := make([]map[string]any, 0, len(toolCatalogOrder))
	for _, toolName := range toolCatalogOrder {
		if !toolVisibleForToken(toolName, tokenAuth, allowedTools) {
			continue
		}
		entry := resolvedCatalogEntry(toolName)
		tools = append(tools, map[string]any{
			"name":        toPublicToolName(toolName),
			"description": entry.description,
			"inputSchema": entry.inputSchema,
		})
	}
	return tools
}

func toolVisibleForToken(
	toolName string,
	tokenAuth *models.MCPTokenAuthContext,
	allowedTools map[string]struct{},
) bool {
	if tokenAuth == nil {
		return true
	}
	if tokenAuth.Scope == models.MCPTokenScopeRead && requiresWriteScope(toolName) {
		return false
	}
	if len(allowedTools) == 0 {
		return true
	}
	_, exists := allowedTools[canonicalToolName(toolName)]
	return exists
}

func canonicalAllowedTools(tokenAuth *models.MCPTokenAuthContext) map[string]struct{} {
	if tokenAuth == nil || len(tokenAuth.AllowedTools) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(tokenAuth.AllowedTools))
	for _, item := range tokenAuth.AllowedTools {
		canonical := canonicalToolName(item)
		if canonical == "" {
			continue
		}
		allowed[canonical] = struct{}{}
	}
	return allowed
}

type toolPolicyError struct {
	code    int
	message string
	data    map[string]any
}

func authorizeToolCall(toolName string, tokenAuth *models.MCPTokenAuthContext) *toolPolicyError {
	if tokenAuth == nil {
		return nil
	}
	canonical := canonicalToolName(toolName)
	requiredScope := models.MCPTokenScopeRead
	if requiresWriteScope(canonical) {
		requiredScope = models.MCPTokenScopeWrite
	}
	if tokenAuth.Scope == models.MCPTokenScopeRead && requiredScope == models.MCPTokenScopeWrite {
		return &toolPolicyError{
			code:    -32003,
			message: "Insufficient token scope",
			data: map[string]any{
				"tool":           toolName,
				"required_scope": "write",
				"token_scope":    string(tokenAuth.Scope),
			},
		}
	}
	allowedTools := canonicalAllowedTools(tokenAuth)
	if len(allowedTools) == 0 {
		return nil
	}
	if _, ok := allowedTools[canonical]; ok {
		return nil
	}
	return &toolPolicyError{
		code:    -32003,
		message: "Tool not allowed by token policy",
		data: map[string]any{
			"tool":        toolName,
			"token_scope": string(tokenAuth.Scope),
		},
	}
}

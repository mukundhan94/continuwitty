package mcp

import (
	"strings"

	"engram/internal/models"
)

type toolIdentifier string

func (name toolIdentifier) String() string {
	return string(name)
}

var readToolNames = map[toolIdentifier]struct{}{
	"chat.list_sessions":          {},
	"chat.get_session":            {},
	"chat.get_lifecycle_policy":   {},
	"chat.list_messages":          {},
	"chat.list_timeline":          {},
	"chat.list_pinned_engrams":    {},
	"chat.list_pinned_documents":  {},
	"chat.list_project_documents": {},
	"project.list":                {},
	"project.member_list":         {},
	"project.export_bundle":       {},
	"project.get_default":         {},
	"engram.list":                 {},
	"engram.get":                  {},
	"engram.collection_list":      {},
	"engram.query":                {},
	"engram.rehydrate":            {},
	"engram.link_list":            {},
	"engram.link_suggest":         {},
	"engram.trace_path":           {},
	"user.get_profile":            {},
	"user.list_projects":          {},
}

var writeToolNames = map[toolIdentifier]struct{}{
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
	"project.member_add":              {},
	"project.member_update":           {},
	"project.member_remove":           {},
	"project.import_bundle":           {},
	"project.set_default":             {},
	"engram.create":                   {},
	"engram.create_from_conversation": {},
	"engram.pin_to_session":           {},
	"engram.update":                   {},
	"engram.share":                    {},
	"engram.unshare":                  {},
	"engram.move_project":             {},
	"engram.link_create":              {},
	"engram.link_update":              {},
	"engram.link_archive":             {},
	"engram.delete":                   {},
	"engram.restore":                  {},
	"engram.collection_create":        {},
	"engram.collection_update":        {},
	"engram.collection_delete":        {},
	"engram.collection_add_items":     {},
	"engram.collection_remove_items":  {},
}

var toolAliases = map[toolIdentifier]toolIdentifier{
	"engram.pin_to_session": "chat.pin_engram",
}

var toolCatalogOrder = []toolIdentifier{
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
	"engram.trace_path",
	"engram.link_suggest",
	"engram.pin_to_session",
	"chat.delete_session",
	"chat.restore_session",
	"project.list",
	"project.create",
	"project.member_list",
	"project.member_add",
	"project.member_update",
	"project.member_remove",
	"project.get_default",
	"project.set_default",
	"project.export_bundle",
	"project.import_bundle",
	"engram.list",
	"engram.get",
	"engram.update",
	"engram.link_list",
	"engram.link_create",
	"engram.link_update",
	"engram.link_archive",
	"engram.share",
	"engram.unshare",
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

func toPublicToolName(canonical toolIdentifier) string {
	return strings.ReplaceAll(canonical.String(), ".", "_")
}

func toDottedToolName(raw toolIdentifier) toolIdentifier {
	rawString := raw.String()
	if strings.Contains(rawString, ".") {
		return raw
	}
	for _, namespace := range toolNamespacePrefixes {
		prefix := namespace + "_"
		if strings.HasPrefix(rawString, prefix) {
			return toolIdentifier(namespace + "." + rawString[len(prefix):])
		}
	}
	return raw
}

func canonicalToolName(raw toolIdentifier) toolIdentifier {
	dotted := toDottedToolName(toolIdentifier(strings.TrimSpace(raw.String())))
	if alias, ok := toolAliases[dotted]; ok {
		return alias
	}
	return dotted
}

func toolExists(name toolIdentifier) bool {
	_, isRead := readToolNames[name]
	_, isWrite := writeToolNames[name]
	return isRead || isWrite
}

func requiresWriteScope(name toolIdentifier) bool {
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
		entry := resolvedCatalogEntry(toolName.String())
		tools = append(tools, map[string]any{
			"name":        toPublicToolName(toolName),
			"description": entry.description,
			"inputSchema": entry.inputSchema,
		})
	}
	return tools
}

func toolVisibleForToken(
	toolName toolIdentifier,
	tokenAuth *models.MCPTokenAuthContext,
	allowedTools map[toolIdentifier]struct{},
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

func canonicalAllowedTools(tokenAuth *models.MCPTokenAuthContext) map[toolIdentifier]struct{} {
	if tokenAuth == nil || len(tokenAuth.AllowedTools) == 0 {
		return nil
	}
	allowed := make(map[toolIdentifier]struct{}, len(tokenAuth.AllowedTools))
	for _, item := range tokenAuth.AllowedTools {
		canonical := canonicalToolName(toolIdentifier(item))
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

func authorizeToolCall(toolName toolIdentifier, tokenAuth *models.MCPTokenAuthContext) *toolPolicyError {
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
				"tool":           toolName.String(),
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
			"tool":        toolName.String(),
			"token_scope": string(tokenAuth.Scope),
		},
	}
}

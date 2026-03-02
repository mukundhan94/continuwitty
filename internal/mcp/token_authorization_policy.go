package mcp

import (
	"fmt"
	"strings"

	"engram/internal/models"
)

var optionalProjectTools = map[string]struct{}{
	"engram.query":                {},
	"chat.list_sessions":          {},
	"chat.list_project_documents": {},
	"engram.list":                 {},
	"engram.collection_list":      {},
}

var projectFallbackTools = map[string]struct{}{
	"chat.create_session":             {},
	"chat.save_as_engram":             {},
	"engram.create":                   {},
	"engram.create_from_conversation": {},
	"engram.collection_create":        {},
}

type tokenToolName string

type tokenProjectID string

type tokenProjectSet map[tokenProjectID]struct{}

func normalizeTokenToolParams(
	toolName string,
	params map[string]any,
	tokenAuth *models.MCPTokenAuthContext,
) (map[string]any, *toolPolicyError) {
	allowedProjectSet := newTokenProjectSet(tokenAuth)
	if tokenAuth == nil || len(allowedProjectSet) == 0 {
		return params, nil
	}
	canonicalTool := tokenToolName(canonicalToolName(toolName))
	normalized := cloneToolParams(params)
	projectID, policyError := resolveTokenProjectIDForTool(canonicalTool, normalized, allowedProjectSet)
	if policyError != nil {
		return nil, policyError
	}
	if projectID == "" || allowedProjectSet.contains(projectID) {
		return normalized, nil
	}
	return nil, disallowedTokenProjectPolicyError(toolName, canonicalTool, tokenAuth.Scope, projectID)
}

func newTokenProjectSet(tokenAuth *models.MCPTokenAuthContext) tokenProjectSet {
	if tokenAuth == nil {
		return nil
	}
	set := make(tokenProjectSet, len(tokenAuth.AllowedProjectIDs))
	for _, projectID := range tokenAuth.AllowedProjectIDs {
		normalizedProjectID := tokenProjectID(strings.TrimSpace(projectID))
		if normalizedProjectID == "" {
			continue
		}
		set[normalizedProjectID] = struct{}{}
	}
	return set
}

func cloneToolParams(params map[string]any) map[string]any {
	if params == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(params))
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
}

func resolveTokenProjectIDForTool(
	canonicalTool tokenToolName,
	params map[string]any,
	allowedProjects tokenProjectSet,
) (tokenProjectID, *toolPolicyError) {
	projectID := normalizeTokenProjectIDParam(params)
	if projectID != "" {
		if toolUsesInputProjectIDForTokenPolicy(canonicalTool, params) {
			return projectID, nil
		}
		return tokenProjectID(""), nil
	}
	if !needsProjectAutofillForToken(canonicalTool, params) {
		return projectID, nil
	}
	autofilledProjectID, policyError := resolveProjectAutofillForToken(allowedProjects)
	if policyError != nil {
		return tokenProjectID(""), policyError
	}
	params["project_id"] = string(autofilledProjectID)
	return autofilledProjectID, nil
}

func normalizeTokenProjectIDParam(params map[string]any) tokenProjectID {
	rawProjectID, exists := params["project_id"]
	if !exists || rawProjectID == nil {
		return ""
	}
	return tokenProjectID(strings.TrimSpace(fmt.Sprint(rawProjectID)))
}

func needsProjectAutofillForToken(canonicalTool tokenToolName, params map[string]any) bool {
	if _, exists := optionalProjectTools[string(canonicalTool)]; exists {
		return true
	}
	if _, exists := projectFallbackTools[string(canonicalTool)]; !exists {
		return false
	}
	if canonicalTool != tokenToolName("chat.save_as_engram") {
		return true
	}
	return !hasNonEmptySessionID(params)
}

func toolUsesInputProjectIDForTokenPolicy(canonicalTool tokenToolName, params map[string]any) bool {
	if _, projectInputTool := projectInputTools[string(canonicalTool)]; projectInputTool {
		return true
	}
	if _, optionalTool := optionalProjectTools[string(canonicalTool)]; optionalTool {
		return true
	}
	if _, fallbackTool := projectFallbackTools[string(canonicalTool)]; !fallbackTool {
		return false
	}
	if canonicalTool != tokenToolName("chat.save_as_engram") {
		return true
	}
	return !hasNonEmptySessionID(params)
}

func hasNonEmptySessionID(params map[string]any) bool {
	sessionID, hasSessionID := optionalParamValue(params, "session_id")
	if !hasSessionID {
		return false
	}
	return strings.TrimSpace(fmt.Sprint(sessionID)) != ""
}

func resolveProjectAutofillForToken(allowedProjects tokenProjectSet) (tokenProjectID, *toolPolicyError) {
	if len(allowedProjects) > 1 {
		return "", &toolPolicyError{
			code:    -32602,
			message: "Invalid params",
			data: map[string]any{
				"missing": "project_id",
				"reason":  "token_has_multiple_allowed_projects",
			},
		}
	}
	return allowedProjects.first(), nil
}

func (allowedProjects tokenProjectSet) contains(projectID tokenProjectID) bool {
	_, exists := allowedProjects[projectID]
	return exists
}

func disallowedTokenProjectPolicyError(
	toolName string,
	canonicalTool tokenToolName,
	scope models.MCPTokenScope,
	projectID tokenProjectID,
) *toolPolicyError {
	requiredScope := "read"
	if requiresWriteScope(string(canonicalTool)) {
		requiredScope = "write"
	}
	return &toolPolicyError{
		code:    -32003,
		message: "Project not allowed by token policy",
		data: map[string]any{
			"tool":           toolName,
			"required_scope": requiredScope,
			"token_scope":    string(scope),
			"project_id":     string(projectID),
		},
	}
}

func (allowedProjects tokenProjectSet) first() tokenProjectID {
	for projectID := range allowedProjects {
		return projectID
	}
	return ""
}

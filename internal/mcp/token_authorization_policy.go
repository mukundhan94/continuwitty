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
	"chat.save_as_engram":             {},
	"engram.create":                   {},
	"engram.create_from_conversation": {},
	"engram.collection_create":        {},
}

func normalizeTokenToolParams(
	toolName string,
	params map[string]any,
	tokenAuth *models.MCPTokenAuthContext,
) (map[string]any, *toolPolicyError) {
	if tokenAuth == nil || len(tokenAuth.AllowedProjectIDs) == 0 {
		return params, nil
	}
	canonicalTool := canonicalToolName(toolName)
	normalized := cloneToolParams(params)
	projectID, policyError := resolveTokenProjectIDForTool(canonicalTool, normalized, tokenAuth.AllowedProjectIDs)
	if policyError != nil {
		return nil, policyError
	}
	if projectID == "" || projectAllowedByToken(projectID, tokenAuth.AllowedProjectIDs) {
		return normalized, nil
	}
	return nil, disallowedProjectPolicyError(toolName, canonicalTool, tokenAuth.Scope, projectID)
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
	canonicalTool string,
	params map[string]any,
	allowedProjectIDs []string,
) (string, *toolPolicyError) {
	projectID := normalizeProjectIDParam(params)
	if projectID != "" || !needsProjectAutofillForToken(canonicalTool, params) {
		return projectID, nil
	}
	autofilledProjectID, policyError := resolveProjectAutofillForToken(allowedProjectIDs)
	if policyError != nil {
		return "", policyError
	}
	params["project_id"] = autofilledProjectID
	return autofilledProjectID, nil
}

func normalizeProjectIDParam(params map[string]any) string {
	rawProjectID, exists := params["project_id"]
	if !exists || rawProjectID == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(rawProjectID))
}

func needsProjectAutofillForToken(canonicalTool string, params map[string]any) bool {
	if _, exists := optionalProjectTools[canonicalTool]; exists {
		return true
	}
	if _, exists := projectFallbackTools[canonicalTool]; !exists {
		return false
	}
	if canonicalTool != "chat.save_as_engram" {
		return true
	}
	_, hasSessionID := params["session_id"]
	return !hasSessionID
}

func resolveProjectAutofillForToken(allowedProjectIDs []string) (string, *toolPolicyError) {
	if len(allowedProjectIDs) > 1 {
		return "", &toolPolicyError{
			code:    -32602,
			message: "Invalid params",
			data: map[string]any{
				"missing": "project_id",
				"reason":  "token_has_multiple_allowed_projects",
			},
		}
	}
	return strings.TrimSpace(allowedProjectIDs[0]), nil
}

func projectAllowedByToken(projectID string, allowedProjectIDs []string) bool {
	normalizedProjectID := strings.TrimSpace(projectID)
	for _, allowedProjectID := range allowedProjectIDs {
		if strings.TrimSpace(allowedProjectID) == normalizedProjectID {
			return true
		}
	}
	return false
}

func disallowedProjectPolicyError(
	toolName string,
	canonicalTool string,
	scope models.MCPTokenScope,
	projectID string,
) *toolPolicyError {
	requiredScope := "read"
	if requiresWriteScope(canonicalTool) {
		requiredScope = "write"
	}
	return &toolPolicyError{
		code:    -32003,
		message: "Project not allowed by token policy",
		data: map[string]any{
			"tool":           toolName,
			"required_scope": requiredScope,
			"token_scope":    string(scope),
			"project_id":     projectID,
		},
	}
}

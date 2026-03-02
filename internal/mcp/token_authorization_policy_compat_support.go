package mcp

import (
	"fmt"
	"strings"

	"engram/internal/models"
)

func normalizeProjectIDParam(params map[string]any) string {
	rawProjectID, exists := params["project_id"]
	if !exists || rawProjectID == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(rawProjectID))
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
	return disallowedTokenProjectPolicyError(
		toolName,
		tokenToolName(canonicalTool),
		scope,
		tokenProjectID(projectID),
	)
}

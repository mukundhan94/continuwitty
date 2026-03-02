package mcp

import (
	"encoding/json"
	"sort"
	"strings"

	"engram/internal/models"
)

func normalizedActorRole(actor Actor) models.UserRole {
	return models.UserRole(strings.ToLower(strings.TrimSpace(actor.Role)))
}

func optionalString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func uniqueSortedProjectIDs(sessions []models.ChatSessionRecord) []string {
	seen := map[string]struct{}{}
	for _, session := range sessions {
		projectID := strings.TrimSpace(session.ProjectID)
		if projectID == "" {
			continue
		}
		seen[projectID] = struct{}{}
	}
	projectIDs := make([]string, 0, len(seen))
	for projectID := range seen {
		projectIDs = append(projectIDs, projectID)
	}
	sort.Strings(projectIDs)
	return projectIDs
}

func actorPayload(actor Actor) map[string]any {
	payload := map[string]any{
		"user_id": actor.UserID,
		"role":    strings.ToLower(strings.TrimSpace(actor.Role)),
	}
	if username := strings.TrimSpace(actor.Username); username != "" {
		payload["username"] = username
	}
	return payload
}

func buildToolCallSuccessResult(toolName string, payload map[string]any) map[string]any {
	return map[string]any{
		"tool_name":         toolName,
		"structuredContent": payload,
		"content": []map[string]any{
			{"type": "text", "text": marshalPayloadText(payload)},
		},
		"isError": false,
	}
}

func marshalPayloadText(payload map[string]any) string {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

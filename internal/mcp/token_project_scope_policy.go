package mcp

import (
	"context"
	"strings"

	"engram/internal/models"

	"github.com/google/uuid"
)

var sessionScopedProjectTools = map[string]struct{}{
	"chat.get_session":             {},
	"chat.get_lifecycle_policy":    {},
	"chat.update_lifecycle_policy": {},
	"chat.list_messages":           {},
	"chat.list_timeline":           {},
	"chat.send_message":            {},
	"chat.list_pinned_engrams":     {},
	"chat.pin_engram":              {},
	"chat.unpin_engram":            {},
	"chat.list_pinned_documents":   {},
	"chat.pin_document":            {},
	"chat.unpin_document":          {},
	"chat.continue_session":        {},
	"chat.delete_session":          {},
	"chat.restore_session":         {},
	"chat.save_as_engram":          {},
}

var engramScopedProjectTools = map[string]struct{}{
	"engram.get":          {},
	"engram.update":       {},
	"engram.move_project": {},
	"engram.delete":       {},
	"engram.restore":      {},
}

func (service *CompatibilityService) enforceTokenProjectPolicy(
	ctx context.Context,
	actor Actor,
	toolName string,
	params map[string]any,
	tokenAuth *models.MCPTokenAuthContext,
) *toolPolicyError {
	if tokenAuth == nil || len(tokenAuth.AllowedProjectIDs) == 0 {
		return nil
	}
	canonicalTool := canonicalToolName(toolName)
	projectID, policyError := service.resolveProjectIDForTokenPolicy(ctx, actor, canonicalTool, params)
	if policyError != nil || projectID == "" {
		return policyError
	}
	if projectAllowedByToken(projectID, tokenAuth.AllowedProjectIDs) {
		return nil
	}
	return disallowedProjectPolicyError(toolName, canonicalTool, tokenAuth.Scope, projectID)
}

func (service *CompatibilityService) resolveProjectIDForTokenPolicy(
	ctx context.Context,
	actor Actor,
	canonicalTool string,
	params map[string]any,
) (string, *toolPolicyError) {
	if projectID := normalizeProjectIDParam(params); projectID != "" {
		return projectID, nil
	}
	if _, sessionScoped := sessionScopedProjectTools[canonicalTool]; sessionScoped {
		return service.resolveSessionProjectID(ctx, actor.UserID, params)
	}
	if _, engramScoped := engramScopedProjectTools[canonicalTool]; engramScoped {
		return service.resolveEngramProjectID(ctx, actor, params)
	}
	return "", nil
}

func (service *CompatibilityService) resolveSessionProjectID(
	ctx context.Context,
	actorUserID uuid.UUID,
	params map[string]any,
) (string, *toolPolicyError) {
	sessionID, ok := requiredUUIDParam(params, "session_id")
	if !ok {
		return "", invalidParamPolicyError("session_id")
	}
	if service.sessionGet == nil {
		return "", nil
	}
	session, err := service.sessionGet.GetSession(
		ctx,
		SessionGetRequest{
			ActorUserID: actorUserID,
			SessionID:   sessionID,
		},
	)
	if err != nil || session == nil {
		return "", nil
	}
	return strings.TrimSpace(session.ProjectID), nil
}

func (service *CompatibilityService) resolveEngramProjectID(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (string, *toolPolicyError) {
	engramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return "", invalidParamPolicyError("engram_id")
	}
	if service.engramGet == nil {
		return "", nil
	}
	engram, err := service.engramGet.GetEngram(
		ctx,
		EngramGetRequest{
			ActorUserID:    actor.UserID,
			ActorRole:      normalizedActorRole(actor),
			EngramID:       engramID,
			IncludeDeleted: true,
		},
	)
	if err != nil || engram == nil {
		return "", nil
	}
	return strings.TrimSpace(engram.ProjectID), nil
}

func invalidParamPolicyError(field string) *toolPolicyError {
	return &toolPolicyError{
		code:    -32602,
		message: "Invalid params",
		data: map[string]any{
			"invalid": field,
		},
	}
}

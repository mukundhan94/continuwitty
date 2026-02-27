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
}

var engramScopedProjectTools = map[string]struct{}{
	"engram.get":          {},
	"engram.update":       {},
	"engram.move_project": {},
	"engram.delete":       {},
	"engram.restore":      {},
}

var projectInputTools = map[string]struct{}{
	"chat.create_session":             {},
	"engram.create":                   {},
	"engram.create_from_conversation": {},
	"project.create":                  {},
	"project.export_bundle":           {},
	"project.import_bundle":           {},
	"project.set_default":             {},
	"engram.collection_create":        {},
}

type tokenProjectPolicyRequest struct {
	ctx       context.Context
	actor     Actor
	toolName  string
	params    map[string]any
	tokenAuth *models.MCPTokenAuthContext
}

func (service *CompatibilityService) enforceTokenProjectPolicy(input tokenProjectPolicyRequest) *toolPolicyError {
	if !hasTokenProjectAllowlist(input.tokenAuth) {
		return nil
	}
	canonicalTool := canonicalToolName(input.toolName)
	if policyError := service.enforceResolvedTokenProjectPolicy(
		input,
		canonicalTool,
		service.resolveProjectIDForTokenPolicy,
	); policyError != nil {
		return policyError
	}
	return service.enforceResolvedTokenProjectPolicy(
		input,
		canonicalTool,
		service.resolveSecondaryProjectIDForTokenPolicy,
	)
}

func hasTokenProjectAllowlist(tokenAuth *models.MCPTokenAuthContext) bool {
	return tokenAuth != nil && len(tokenAuth.AllowedProjectIDs) > 0
}

type tokenProjectResolver func(
	ctx context.Context,
	actor Actor,
	canonicalTool string,
	params map[string]any,
) (string, *toolPolicyError)

func (service *CompatibilityService) enforceResolvedTokenProjectPolicy(
	input tokenProjectPolicyRequest,
	canonicalTool string,
	resolver tokenProjectResolver,
) *toolPolicyError {
	projectID, policyError := resolver(
		input.ctx,
		input.actor,
		canonicalTool,
		input.params,
	)
	if policyError != nil {
		return policyError
	}
	if projectID == "" {
		return nil
	}
	return validateTokenProjectPolicy(
		input.toolName,
		canonicalTool,
		input.tokenAuth,
		projectID,
	)
}

func validateTokenProjectPolicy(
	toolName string,
	canonicalTool string,
	tokenAuth *models.MCPTokenAuthContext,
	resolvedValue string,
) *toolPolicyError {
	if projectAllowedByToken(resolvedValue, tokenAuth.AllowedProjectIDs) {
		return nil
	}
	return disallowedProjectPolicyError(toolName, canonicalTool, tokenAuth.Scope, resolvedValue)
}

func (service *CompatibilityService) resolveProjectIDForTokenPolicy(
	ctx context.Context,
	actor Actor,
	canonicalTool string,
	params map[string]any,
) (string, *toolPolicyError) {
	if _, projectInputTool := projectInputTools[canonicalTool]; projectInputTool {
		return normalizeProjectIDParam(params), nil
	}
	if _, sessionScoped := sessionScopedProjectTools[canonicalTool]; sessionScoped {
		return service.resolveSessionProjectID(ctx, actor.UserID, params)
	}
	if canonicalTool == "chat.save_as_engram" {
		return service.resolveSaveAsEngramProjectID(ctx, actor.UserID, params)
	}
	if _, engramScoped := engramScopedProjectTools[canonicalTool]; engramScoped {
		return service.resolveEngramProjectID(ctx, actor, canonicalTool, params)
	}
	if canonicalTool == "engram.rehydrate" {
		return service.resolveEngramProjectID(ctx, actor, canonicalTool, params)
	}
	if _, optionalProjectTool := optionalProjectTools[canonicalTool]; optionalProjectTool {
		return normalizeProjectIDParam(params), nil
	}
	return "", nil
}

func (service *CompatibilityService) resolveSecondaryProjectIDForTokenPolicy(
	ctx context.Context,
	actor Actor,
	canonicalTool string,
	params map[string]any,
) (string, *toolPolicyError) {
	if canonicalTool != "engram.move_project" {
		return "", nil
	}
	if normalizeTargetProjectIDParam(params) == "" {
		return "", nil
	}
	return service.resolveEngramSourceProjectID(ctx, actor, params)
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
	canonicalTool string,
	params map[string]any,
) (string, *toolPolicyError) {
	if _, ok := requiredUUIDParam(params, "engram_id"); !ok {
		return "", invalidParamPolicyError("engram_id")
	}
	if canonicalTool == "engram.move_project" {
		if targetProjectID := normalizeTargetProjectIDParam(params); targetProjectID != "" {
			return targetProjectID, nil
		}
	}
	return service.resolveEngramSourceProjectID(ctx, actor, params)
}

func (service *CompatibilityService) resolveEngramSourceProjectID(
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

func (service *CompatibilityService) resolveSaveAsEngramProjectID(
	ctx context.Context,
	actorUserID uuid.UUID,
	params map[string]any,
) (string, *toolPolicyError) {
	if sessionIDValue, hasSessionID := optionalParamValue(params, "session_id"); hasSessionID {
		if strings.TrimSpace(stringParam(sessionIDValue)) != "" {
			return service.resolveSessionProjectID(ctx, actorUserID, params)
		}
	}
	return normalizeProjectIDParam(params), nil
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

func normalizeTargetProjectIDParam(params map[string]any) string {
	value, found := optionalParamValue(params, "target_project_id")
	if !found {
		return ""
	}
	return strings.TrimSpace(stringParam(value))
}

func stringParam(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

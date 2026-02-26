package api

import (
	"net/http"
	"strings"

	"engram/internal/models"
)

const (
	defaultMCPTokenExpiryDays = 90
	maxMCPTokenExpiryDays     = 3650
	minMCPTokenNameLength     = 3
	maxMCPTokenNameLength     = 120
	maxMCPTokenReasonLength   = 240
)

func (dependencies sessionAuthDependencies) handleCreateMCPToken(writer http.ResponseWriter, request *http.Request) {
	actor, ok := dependencies.requireAdminAPIActor(writer, request)
	if !ok {
		return
	}
	if !dependencies.hasMCPTokenCreateDependencies() {
		writeSessionUserDependenciesError(writer)
		return
	}

	createRequest, ok := decodeMCPTokenCreateRequest(writer, request)
	if !ok {
		return
	}
	created, err := dependencies.createTokenForOwner(
		request.Context(),
		actor.UserID,
		createRequest,
		dependencies.mcpTokenPepper,
	)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	dependencies.logAuditEventRequest(
		request,
		sessionAuditEvent{
			eventType: "mcp_token_created",
			success:   true,
			username:  actor.Username,
			metadata: map[string]any{
				"token_id":            created.TokenID.String(),
				"scope":               string(created.Scope),
				"name":                created.Name,
				"allowed_tools":       created.AllowedTools,
				"allowed_project_ids": created.AllowedProjectIDs,
				"expires_at":          created.ExpiresAt,
			},
		},
	)
	writeJSON(writer, http.StatusCreated, created)
}

func (dependencies sessionAuthDependencies) handleListMCPTokens(writer http.ResponseWriter, request *http.Request) {
	actor, ok := dependencies.requireAdminAPIActor(writer, request)
	if !ok {
		return
	}
	if dependencies.listTokenSummaries == nil {
		writeSessionUserDependenciesError(writer)
		return
	}

	limit, ok := parseOptionalIntQuery(
		writer,
		request,
		"limit",
		intQuerySpec{Default: 200, Min: 1, Max: 500},
	)
	if !ok {
		return
	}
	offset, ok := parseOptionalIntQuery(
		writer,
		request,
		"offset",
		intQuerySpec{Default: 0, Min: 0, Max: 1_000_000},
	)
	if !ok {
		return
	}
	summaries, err := dependencies.listTokenSummaries(request.Context(), actor.UserID, limit, offset)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	writeJSON(writer, http.StatusOK, summaries)
}

func (dependencies sessionAuthDependencies) handleRevokeMCPToken(writer http.ResponseWriter, request *http.Request) {
	actor, ok := dependencies.requireAdminAPIActor(writer, request)
	if !ok {
		return
	}
	if dependencies.revokeTokenForOwner == nil {
		writeSessionUserDependenciesError(writer)
		return
	}

	tokenID, ok := parsePathUUID(writer, request, "token_id")
	if !ok {
		return
	}
	revokeRequest, ok := decodeMCPTokenRevokeRequest(writer, request)
	if !ok {
		return
	}
	revoked, err := dependencies.revokeTokenForOwner(request.Context(), tokenID, actor.UserID)
	if err != nil {
		writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "internal error"})
		return
	}
	if revoked == nil {
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "Token not found"})
		return
	}

	dependencies.logAuditEventRequest(
		request,
		sessionAuditEvent{
			eventType: "mcp_token_revoked",
			success:   true,
			username:  actor.Username,
			metadata: map[string]any{
				"token_id": tokenID.String(),
				"reason":   optionalTrimmedReason(revokeRequest.Reason),
			},
		},
	)
	writeJSON(writer, http.StatusOK, revoked)
}

func (dependencies sessionAuthDependencies) hasMCPTokenCreateDependencies() bool {
	return dependencies.createTokenForOwner != nil && strings.TrimSpace(dependencies.mcpTokenPepper) != ""
}

func decodeMCPTokenCreateRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (models.MCPTokenCreateRequest, bool) {
	payload := models.MCPTokenCreateRequest{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return models.MCPTokenCreateRequest{}, false
	}
	payload.Name = strings.TrimSpace(payload.Name)
	if !isValidMCPTokenName(payload.Name) {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid token name"})
		return models.MCPTokenCreateRequest{}, false
	}
	payload.Scope = resolveMCPTokenScope(payload.Scope)
	if _, err := models.ParseMCPTokenScope(payload.Scope); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid scope"})
		return models.MCPTokenCreateRequest{}, false
	}
	if payload.ExpiresInDays == 0 {
		payload.ExpiresInDays = defaultMCPTokenExpiryDays
	}
	if payload.ExpiresInDays < 1 || payload.ExpiresInDays > maxMCPTokenExpiryDays {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid expires_in_days"})
		return models.MCPTokenCreateRequest{}, false
	}
	if payload.AllowedTools == nil {
		payload.AllowedTools = []string{}
	}
	if payload.AllowedProjectIDs == nil {
		payload.AllowedProjectIDs = []string{}
	}
	return payload, true
}

func decodeMCPTokenRevokeRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (models.MCPTokenRevokeRequest, bool) {
	payload := models.MCPTokenRevokeRequest{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return models.MCPTokenRevokeRequest{}, false
	}
	if !isValidMCPTokenRevokeReason(payload.Reason) {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "invalid reason"})
		return models.MCPTokenRevokeRequest{}, false
	}
	return payload, true
}

func isValidMCPTokenName(name string) bool {
	length := len(name)
	return length >= minMCPTokenNameLength && length <= maxMCPTokenNameLength
}

func resolveMCPTokenScope(scope string) string {
	trimmed := strings.TrimSpace(scope)
	if trimmed == "" {
		return string(models.MCPTokenScopeRead)
	}
	return trimmed
}

func isValidMCPTokenRevokeReason(reason *string) bool {
	if reason == nil {
		return true
	}
	return len(strings.TrimSpace(*reason)) <= maxMCPTokenReasonLength
}

func optionalTrimmedReason(reason *string) any {
	if reason == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*reason)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

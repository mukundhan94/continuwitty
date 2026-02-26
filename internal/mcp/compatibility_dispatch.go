package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"engram/internal/models"
	"engram/internal/projects"
)

const (
	defaultProjectListLimit  = 500
	defaultProjectListOffset = 0
)

type toolDispatchError struct {
	code    int
	message string
	data    map[string]any
}

func (service *CompatibilityService) dispatchImplementedTool(
	ctx context.Context,
	method string,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	switch method {
	case "user.get_profile":
		return map[string]any{"profile": actorPayload(actor)}, true, nil
	case "project.list":
		return service.dispatchProjectListTool(ctx, actor, params)
	case "project.get_default":
		return service.dispatchProjectGetDefaultTool(ctx, actor)
	case "project.set_default":
		return service.dispatchProjectSetDefaultTool(ctx, actor, params)
	default:
		return nil, false, nil
	}
}

func (service *CompatibilityService) dispatchProjectListTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectService == nil {
		return nil, false, nil
	}
	includeArchived, ok := optionalBoolParam(params, "include_archived", false)
	if !ok {
		return nil, true, invalidParamError("include_archived")
	}
	limit, ok := optionalIntParam(params, "limit", defaultProjectListLimit)
	if !ok {
		return nil, true, invalidParamError("limit")
	}
	offset, ok := optionalIntParam(params, "offset", defaultProjectListOffset)
	if !ok {
		return nil, true, invalidParamError("offset")
	}

	projects, err := service.projectService.ListProjects(
		ctx,
		actor.UserID,
		normalizedActorRole(actor),
		includeArchived,
		limit,
		offset,
	)
	if err != nil {
		return nil, true, &toolDispatchError{code: -32603, message: "Internal error"}
	}
	return map[string]any{"projects": projects}, true, nil
}

func (service *CompatibilityService) dispatchProjectGetDefaultTool(
	ctx context.Context,
	actor Actor,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectService == nil {
		return nil, false, nil
	}
	defaultProjectID, err := service.projectService.GetDefaultProjectID(ctx, actor.UserID)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"default_project_id": optionalString(defaultProjectID)}, true, nil
}

func (service *CompatibilityService) dispatchProjectSetDefaultTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.projectService == nil {
		return nil, false, nil
	}
	projectID, ok := requiredStringParam(params, "project_id")
	if !ok {
		return nil, true, invalidParamError("project_id")
	}
	defaultProjectID, err := service.projectService.SetDefaultProjectID(
		ctx,
		actor.UserID,
		normalizedActorRole(actor),
		projectID,
	)
	if err != nil {
		return nil, true, mapProjectServiceError(err)
	}
	return map[string]any{"default_project_id": defaultProjectID}, true, nil
}

func invalidParamError(field string) *toolDispatchError {
	return &toolDispatchError{
		code:    -32602,
		message: "Invalid params",
		data:    map[string]any{"invalid": field},
	}
}

func internalToolDispatchError() *toolDispatchError {
	return &toolDispatchError{
		code:    -32603,
		message: "Internal error",
	}
}

func mapProjectServiceError(err error) *toolDispatchError {
	switch {
	case errors.Is(err, projects.ErrProjectIDMustNotBeBlank):
		return invalidParamError("project_id")
	case errors.Is(err, projects.ErrProjectNotFound):
		return invalidParamsWithStatus(404, "Project not found")
	case errors.Is(err, projects.ErrUserNotFound):
		return invalidParamsWithStatus(404, "User not found")
	default:
		return internalToolDispatchError()
	}
}

func invalidParamsWithStatus(statusCode int, detail string) *toolDispatchError {
	return &toolDispatchError{
		code:    -32602,
		message: "Invalid params",
		data: map[string]any{
			"status_code": statusCode,
			"detail":      detail,
		},
	}
}

func normalizedActorRole(actor Actor) models.UserRole {
	return models.UserRole(strings.ToLower(strings.TrimSpace(actor.Role)))
}

func optionalString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func requiredStringParam(params map[string]any, key string) (string, bool) {
	rawValue, ok := optionalParamValue(params, key)
	if !ok {
		return "", false
	}
	value, ok := rawValue.(string)
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	return value, value != ""
}

func optionalBoolParam(params map[string]any, key string, defaultValue bool) (bool, bool) {
	rawValue, ok := optionalParamValue(params, key)
	if !ok {
		return defaultValue, true
	}
	switch value := rawValue.(type) {
	case bool:
		return value, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

func optionalIntParam(params map[string]any, key string, defaultValue int) (int, bool) {
	rawValue, ok := optionalParamValue(params, key)
	if !ok {
		return defaultValue, true
	}
	switch value := rawValue.(type) {
	case int:
		return value, true
	case int8:
		return int(value), true
	case int16:
		return int(value), true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	case float64:
		intValue := int(value)
		if float64(intValue) != value {
			return 0, false
		}
		return intValue, true
	case json.Number:
		intValue, err := strconv.Atoi(value.String())
		if err != nil {
			return 0, false
		}
		return intValue, true
	case string:
		intValue, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0, false
		}
		return intValue, true
	default:
		return 0, false
	}
}

func optionalParamValue(params map[string]any, key string) (any, bool) {
	if params == nil {
		return nil, false
	}
	rawValue, exists := params[key]
	if !exists || rawValue == nil {
		return nil, false
	}
	return rawValue, true
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

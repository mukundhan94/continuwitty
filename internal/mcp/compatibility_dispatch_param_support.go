package mcp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"engram/internal/models"

	"github.com/google/uuid"
)

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

func requiredUUIDParam(params map[string]any, key string) (uuid.UUID, bool) {
	value, ok := optionalUUIDParam(params, key)
	if !ok || value == nil {
		return uuid.Nil, false
	}
	return *value, true
}

func requiredProjectMemberRoleParam(params map[string]any, key string) (models.ProjectMemberRole, bool) {
	value, ok := requiredStringParam(params, key)
	if !ok {
		return "", false
	}
	parsed, err := models.ParseProjectMemberRole(value)
	if err != nil {
		return "", false
	}
	return parsed, true
}

func optionalProjectIDParam(params map[string]any, key string) *string {
	rawValue, ok := optionalParamValue(params, key)
	if !ok {
		return nil
	}
	text := strings.TrimSpace(fmt.Sprint(rawValue))
	if text == "" {
		return nil
	}
	return &text
}

func stringParamWithDefault(params map[string]any, key string, defaultValue any) string {
	rawValue, ok := optionalParamValue(params, key)
	if !ok {
		return fmt.Sprint(defaultValue)
	}
	return fmt.Sprint(rawValue)
}

func optionalUUIDParam(params map[string]any, key string) (*uuid.UUID, bool) {
	rawValue, ok := optionalParamValue(params, key)
	if !ok {
		return nil, true
	}
	candidate := strings.TrimSpace(fmt.Sprint(rawValue))
	if candidate == "" {
		return nil, false
	}
	parsed, err := uuid.Parse(candidate)
	if err != nil {
		return nil, false
	}
	return &parsed, true
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

type pagingParams struct {
	limit  int
	offset int
}

func parsePagingParams(
	params map[string]any,
	defaultLimit int,
	defaultOffset int,
) (pagingParams, *toolDispatchError) {
	limit, ok := optionalIntParam(params, "limit", defaultLimit)
	if !ok {
		return pagingParams{}, invalidParamError("limit")
	}
	offset, ok := optionalIntParam(params, "offset", defaultOffset)
	if !ok {
		return pagingParams{}, invalidParamError("offset")
	}
	return pagingParams{limit: limit, offset: offset}, nil
}

func optionalIntParam(params map[string]any, key string, defaultValue int) (int, bool) {
	rawValue, ok := optionalParamValue(params, key)
	if !ok {
		return defaultValue, true
	}
	switch value := rawValue.(type) {
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
	default:
		return parseIntValue(rawValue)
	}
}

func parseIntValue(raw any) (int, bool) {
	intValue, err := strconv.Atoi(strings.TrimSpace(fmt.Sprint(raw)))
	if err != nil {
		return 0, false
	}
	return intValue, true
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

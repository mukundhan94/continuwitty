package mcp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type engramLinkParamKey string

func parseOptionalEngramLinkEnumValue[T ~string](
	params map[string]any,
	key engramLinkParamKey,
	defaultValue T,
	parse func(string) (T, error),
) (T, *toolDispatchError) {
	value, ok := optionalStringPointerParam(params, string(key))
	if !ok {
		var zero T
		return zero, invalidParamError(string(key))
	}
	if value == nil {
		return defaultValue, nil
	}
	parsed, err := parse(strings.TrimSpace(*value))
	if err != nil {
		var zero T
		return zero, invalidParamError(string(key))
	}
	return parsed, nil
}

func parseOptionalEngramLinkEnumPointer[T ~string](
	params map[string]any,
	key engramLinkParamKey,
	parse func(string) (T, error),
) (*T, bool) {
	value, ok := optionalStringPointerParam(params, string(key))
	if !ok {
		return nil, false
	}
	if value == nil {
		return nil, true
	}
	parsed, err := parse(strings.TrimSpace(*value))
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func parseBoundedFloat(
	params map[string]any,
	key engramLinkParamKey,
	defaultValue float64,
) (float64, *toolDispatchError) {
	value, ok := optionalFloatParam(params, key)
	if !ok {
		return 0, invalidParamError(string(key))
	}
	if value == nil {
		return defaultValue, nil
	}
	if *value < 0 || *value > 1 {
		return 0, invalidParamError(string(key))
	}
	return *value, nil
}

func optionalBoundedFloatPointer(params map[string]any, key engramLinkParamKey) (*float64, bool) {
	value, ok := optionalFloatParam(params, key)
	if !ok {
		return nil, false
	}
	if value == nil {
		return nil, true
	}
	if *value < 0 || *value > 1 {
		return nil, false
	}
	return value, true
}

func optionalFloatParam(params map[string]any, key engramLinkParamKey) (*float64, bool) {
	value, found := optionalParamValue(params, string(key))
	if !found {
		return nil, true
	}
	parsed, ok := parseFloatValue(value)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func parseFloatValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	case string:
		return parseFloatString(typed)
	default:
		return parseFloatString(fmt.Sprint(value))
	}
}

func parseFloatString(raw string) (float64, bool) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func optionalMapParam(params map[string]any, key engramLinkParamKey) (map[string]any, bool) {
	value, ok := optionalMapPointerParam(params, key)
	if !ok {
		return nil, false
	}
	if value == nil || *value == nil {
		return map[string]any{}, true
	}
	return *value, true
}

func optionalMapPointerParam(params map[string]any, key engramLinkParamKey) (*map[string]any, bool) {
	value, found := optionalParamValue(params, string(key))
	if !found {
		return nil, true
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, false
	}
	return &decoded, true
}

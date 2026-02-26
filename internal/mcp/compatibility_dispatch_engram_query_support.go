package mcp

import (
	"context"
	"strings"
	"time"

	"engram/internal/models"
)

func (service *CompatibilityService) dispatchEngramQueryTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramQuery == nil {
		return nil, false, nil
	}
	request, dispatchErr := buildEngramQueryDispatchRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	results, err := service.engramQuery.QueryEngrams(ctx, request)
	if err != nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{"results": results}, true, nil
}

func buildEngramQueryDispatchRequest(
	actor Actor,
	params map[string]any,
) (EngramQueryDispatchRequest, *toolDispatchError) {
	query, ok := requiredStringParam(params, "query")
	if !ok {
		return EngramQueryDispatchRequest{}, invalidParamError("query")
	}
	topK, dispatchErr := parseEngramQueryTopKParam(params)
	if dispatchErr != nil {
		return EngramQueryDispatchRequest{}, dispatchErr
	}
	tags, ok := optionalStringArrayParam(params, "tags")
	if !ok {
		return EngramQueryDispatchRequest{}, invalidParamError("tags")
	}
	keywords, ok := optionalStringArrayParam(params, "keywords")
	if !ok {
		return EngramQueryDispatchRequest{}, invalidParamError("keywords")
	}
	createdAfter, ok := optionalRFC3339TimeParam(params, "created_after")
	if !ok {
		return EngramQueryDispatchRequest{}, invalidParamError("created_after")
	}
	createdBefore, ok := optionalRFC3339TimeParam(params, "created_before")
	if !ok {
		return EngramQueryDispatchRequest{}, invalidParamError("created_before")
	}
	return EngramQueryDispatchRequest{
		ActorUserID: actor.UserID,
		Payload: models.EngramQueryRequest{
			Query:         query,
			TopK:          topK,
			ProjectID:     optionalProjectIDParam(params, "project_id"),
			Tags:          tags,
			Keywords:      keywords,
			CreatedAfter:  createdAfter,
			CreatedBefore: createdBefore,
		},
	}, nil
}

func parseEngramQueryTopKParam(params map[string]any) (int, *toolDispatchError) {
	topK, ok := optionalIntParam(params, "top_k", 5)
	if !ok {
		return 0, invalidParamError("top_k")
	}
	if topK < 1 {
		return 0, invalidParamError("top_k")
	}
	if topK > 50 {
		return 0, invalidParamError("top_k")
	}
	return topK, nil
}

func optionalStringArrayParam(params map[string]any, key string) ([]string, bool) {
	value, found := optionalParamValue(params, key)
	if !found {
		return []string{}, true
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), true
	case []any:
		return stringArrayFromAnySlice(typed)
	default:
		return nil, false
	}
}

func stringArrayFromAnySlice(values []any) ([]string, bool) {
	result := make([]string, 0, len(values))
	for _, item := range values {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		result = append(result, text)
	}
	return result, true
}

func optionalRFC3339TimeParam(params map[string]any, key string) (*time.Time, bool) {
	value, found := optionalParamValue(params, key)
	if !found {
		return nil, true
	}
	text, ok := value.(string)
	if !ok {
		return nil, false
	}
	return parseRFC3339Pointer(text)
}

func parseRFC3339Pointer(raw string) (*time.Time, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

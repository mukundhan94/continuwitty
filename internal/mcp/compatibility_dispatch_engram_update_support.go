package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"engram/internal/admin"
	"engram/internal/models"

	"github.com/google/uuid"
)

func (service *CompatibilityService) dispatchEngramUpdateTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramUpdate == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramUpdateRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	updated, err := service.engramUpdate.UpdateEngram(ctx, request)
	if err != nil {
		return nil, true, mapEngramUpdateError(err, request.EngramID)
	}
	if updated == nil {
		return nil, true, engramNotFoundDispatchError(request.EngramID)
	}
	return map[string]any{"engram": *updated}, true, nil
}

func parseEngramUpdateRequest(
	actor Actor,
	params map[string]any,
) (EngramUpdateRequest, *toolDispatchError) {
	engramID, ok := requiredUUIDParam(params, "engram_id")
	if !ok {
		return EngramUpdateRequest{}, invalidParamError("engram_id")
	}
	fields, dispatchErr := parseEngramUpdateFields(params)
	if dispatchErr != nil {
		return EngramUpdateRequest{}, dispatchErr
	}
	return EngramUpdateRequest{
		ActorUserID:             actor.UserID,
		ActorRole:               normalizedActorRole(actor),
		EngramID:                engramID,
		ExpectedUpdatedAt:       fields.expectedUpdatedAt,
		Title:                   fields.title,
		Abstract:                fields.abstract,
		DetailedSummaryMarkdown: fields.detailedSummaryMarkdown,
		Tags:                    fields.tags,
		Keywords:                fields.keywords,
		VisibilityScope:         fields.visibilityScope,
		Sources:                 fields.sources,
	}, nil
}

type engramUpdateFields struct {
	expectedUpdatedAt       *time.Time
	title                   *string
	abstract                *string
	detailedSummaryMarkdown *string
	tags                    *[]string
	keywords                *[]string
	visibilityScope         *models.VisibilityScope
	sources                 *[]models.AdminEngramSourceInput
}

func parseEngramUpdateFields(params map[string]any) (engramUpdateFields, *toolDispatchError) {
	fields := engramUpdateFields{}
	parsers := []func() *toolDispatchError{
		func() *toolDispatchError { return assignOptionalUpdateStrings(params, &fields) },
		func() *toolDispatchError {
			value, ok := optionalStringSlicePointerParam(params, "tags")
			return assignOptionalParsedField(value, ok, "tags", &fields.tags)
		},
		func() *toolDispatchError {
			value, ok := optionalStringSlicePointerParam(params, "keywords")
			return assignOptionalParsedField(value, ok, "keywords", &fields.keywords)
		},
		func() *toolDispatchError {
			value, ok := optionalVisibilityScopePointerParam(params, "visibility_scope")
			return assignOptionalParsedField(value, ok, "visibility_scope", &fields.visibilityScope)
		},
		func() *toolDispatchError {
			value, ok := optionalRFC3339TimePointerParam(params, "expected_updated_at")
			return assignOptionalParsedField(value, ok, "expected_updated_at", &fields.expectedUpdatedAt)
		},
		func() *toolDispatchError {
			value, ok := optionalAdminSourceInputsPointerParam(params, "sources")
			return assignOptionalParsedField(value, ok, "sources", &fields.sources)
		},
	}
	for _, parse := range parsers {
		if dispatchErr := parse(); dispatchErr != nil {
			return engramUpdateFields{}, dispatchErr
		}
	}
	return fields, nil
}

func assignOptionalUpdateStrings(
	params map[string]any,
	fields *engramUpdateFields,
) *toolDispatchError {
	assignments := []struct {
		key    string
		target **string
	}{
		{key: "title", target: &fields.title},
		{key: "abstract", target: &fields.abstract},
		{key: "detailed_summary_markdown", target: &fields.detailedSummaryMarkdown},
	}
	for _, assignment := range assignments {
		value, ok := optionalStringPointerParam(params, assignment.key)
		if !ok {
			return invalidParamError(assignment.key)
		}
		*assignment.target = value
	}
	return nil
}

func assignOptionalParsedField[T any](
	value *T,
	ok bool,
	key string,
	target **T,
) *toolDispatchError {
	if !ok {
		return invalidParamError(key)
	}
	*target = value
	return nil
}

func optionalStringSlicePointerParam(params map[string]any, key string) (*[]string, bool) {
	value, found := optionalParamValue(params, key)
	if !found || value == nil {
		return nil, true
	}
	switch typed := value.(type) {
	case []string:
		slice := append([]string(nil), typed...)
		return &slice, true
	case []any:
		parsed, ok := stringArrayFromAnySlice(typed)
		if !ok {
			return nil, false
		}
		return &parsed, true
	default:
		return nil, false
	}
}

func optionalVisibilityScopePointerParam(
	params map[string]any,
	key string,
) (*models.VisibilityScope, bool) {
	value, ok := optionalStringPointerParam(params, key)
	if !ok {
		return nil, false
	}
	if value == nil {
		return nil, true
	}
	parsed, err := models.ParseVisibilityScope(strings.TrimSpace(*value))
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func optionalRFC3339TimePointerParam(params map[string]any, key string) (*time.Time, bool) {
	value, ok := optionalStringPointerParam(params, key)
	if !ok {
		return nil, false
	}
	if value == nil {
		return nil, true
	}
	return parseRFC3339Pointer(*value)
}

func optionalAdminSourceInputsPointerParam(
	params map[string]any,
	key string,
) (*[]models.AdminEngramSourceInput, bool) {
	value, found := optionalParamValue(params, key)
	if !found || value == nil {
		return nil, true
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	sources := []models.AdminEngramSourceInput{}
	if err := json.Unmarshal(raw, &sources); err != nil {
		return nil, false
	}
	return &sources, true
}

func mapEngramUpdateError(err error, engramID uuid.UUID) *toolDispatchError {
	switch {
	case errors.Is(err, admin.ErrEngramNotFound):
		return engramNotFoundDispatchError(engramID)
	case errors.Is(err, admin.ErrEngramStale):
		return invalidParamsWithStatus(409, admin.ErrEngramStale.Error())
	default:
		return internalToolDispatchError()
	}
}

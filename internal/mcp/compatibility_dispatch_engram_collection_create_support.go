package mcp

import (
	"context"
	"errors"
	"strings"

	"engram/internal/admin"
	"engram/internal/projects"
	"engram/internal/repository"
)

func (service *CompatibilityService) dispatchEngramCollectionCreateTool(
	ctx context.Context,
	actor Actor,
	params map[string]any,
) (map[string]any, bool, *toolDispatchError) {
	if service.engramCollectionCreate == nil {
		return nil, false, nil
	}
	request, dispatchErr := parseEngramCollectionCreateRequest(actor, params)
	if dispatchErr != nil {
		return nil, true, dispatchErr
	}
	created, err := service.engramCollectionCreate.CreateCollection(ctx, request)
	if err != nil {
		return nil, true, mapEngramCollectionCreateError(err)
	}
	if created == nil {
		return nil, true, internalToolDispatchError()
	}
	return map[string]any{
		"collection":           created.Collection,
		"resolved_project_id":  created.ResolvedProjectID,
		"used_default_project": created.UsedDefaultProject,
	}, true, nil
}

func parseEngramCollectionCreateRequest(
	actor Actor,
	params map[string]any,
) (EngramCollectionCreateRequest, *toolDispatchError) {
	name := stringParamWithDefault(params, "name", "")
	if strings.TrimSpace(name) == "" {
		return EngramCollectionCreateRequest{}, invalidParamError("name")
	}
	return EngramCollectionCreateRequest{
		ActorUserID: actor.UserID,
		ActorRole:   normalizedActorRole(actor),
		ProjectID:   stringParamWithDefault(params, "project_id", ""),
		Name:        name,
		Description: stringParamWithDefault(params, "description", ""),
	}, nil
}

func mapEngramCollectionCreateError(err error) *toolDispatchError {
	switch {
	case errors.Is(err, projects.ErrProjectIDMustNotBeBlank):
		return invalidParamError("project_id")
	case errors.Is(err, projects.ErrProjectNotFound):
		return invalidParamsWithStatus(404, "Project not found")
	case errors.Is(err, projects.ErrProjectIDRequiredWhenNoDefaultProject):
		return invalidParamsWithStatus(422, projects.ErrProjectIDRequiredWhenNoDefaultProject.Error())
	case errors.Is(err, projects.ErrDefaultProjectNotAccessible):
		return invalidParamsWithStatus(422, projects.ErrDefaultProjectNotAccessible.Error())
	case errors.Is(err, admin.ErrProjectIDRequired):
		return invalidParamError("project_id")
	case errors.Is(err, repository.ErrCollectionNameExists):
		return invalidParamsWithStatus(409, repository.ErrCollectionNameExists.Error())
	default:
		return internalToolDispatchError()
	}
}

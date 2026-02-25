package main

import (
	"context"
	"errors"

	"engram/internal/admin"
	internalapi "engram/internal/api"
	"engram/internal/models"
	"engram/internal/projects"

	"github.com/google/uuid"
)

// projectResolutionService captures the project-service capability we need for runtime adapters.
type projectResolutionService interface {
	ResolveProjectIDForWrite(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		projectID string,
	) (projects.Resolution, error)
}

func resolveProjectIDForWriteDependency(
	projectService projectResolutionService,
) func(ctx context.Context, actorUserID uuid.UUID, actorRole models.UserRole, projectID string) (internalapi.SessionProjectResolution, error) {
	return func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		projectID string,
	) (internalapi.SessionProjectResolution, error) {
		resolution, err := projectService.ResolveProjectIDForWrite(ctx, actorUserID, actorRole, projectID)
		if err != nil {
			return internalapi.SessionProjectResolution{}, mapSessionProjectResolutionError(err)
		}
		return internalapi.SessionProjectResolution{
			ProjectID:          resolution.ProjectID,
			UsedDefaultProject: resolution.UsedDefaultProject,
		}, nil
	}
}

func mapSessionProjectResolutionError(err error) error {
	switch {
	case errors.Is(err, projects.ErrProjectIDRequiredWhenNoDefaultProject):
		return internalapi.ErrProjectIDRequiredWhenNoDefaultProject
	case errors.Is(err, projects.ErrDefaultProjectNotAccessible):
		return internalapi.ErrDefaultProjectNotAccessible
	default:
		return err
	}
}

type adminProjectResolver struct {
	projectService projectResolutionService
}

func newAdminProjectResolver(projectService projectResolutionService) admin.ProjectResolver {
	return adminProjectResolver{projectService: projectService}
}

func (resolver adminProjectResolver) ResolveProjectIDForWrite(
	ctx context.Context,
	input admin.ResolveProjectWriteInput,
) (admin.ResolveProjectWriteResult, error) {
	resolution, err := resolver.projectService.ResolveProjectIDForWrite(
		ctx,
		input.ActorUserID,
		models.UserRole(input.ActorRole),
		input.ProjectID,
	)
	if err != nil {
		return admin.ResolveProjectWriteResult{}, mapAdminProjectResolutionError(err)
	}
	return admin.ResolveProjectWriteResult{
		ProjectID:          resolution.ProjectID,
		UsedDefaultProject: resolution.UsedDefaultProject,
	}, nil
}

func mapAdminProjectResolutionError(err error) error {
	switch {
	case errors.Is(err, projects.ErrProjectIDMustNotBeBlank),
		errors.Is(err, projects.ErrProjectIDRequiredWhenNoDefaultProject),
		errors.Is(err, projects.ErrDefaultProjectNotAccessible):
		return admin.ErrProjectIDRequired
	default:
		return err
	}
}

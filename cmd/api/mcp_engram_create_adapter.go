package main

import (
	"context"

	"engram/internal/mcp"
	"engram/internal/models"
	"engram/internal/projects"
	"engram/internal/repository"
)

type mcpEngramCreateAdapter struct {
	db             repository.Queryer
	projectService *projects.Service
	embeddingDim   int
}

func newMCPEngramCreateAdapter(
	db repository.Queryer,
	projectService *projects.Service,
	embeddingDim int,
) mcp.EngramCreateService {
	if db == nil || projectService == nil {
		return nil
	}
	return mcpEngramCreateAdapter{
		db:             db,
		projectService: projectService,
		embeddingDim:   embeddingDim,
	}
}

func (adapter mcpEngramCreateAdapter) CreateEngram(
	ctx context.Context,
	request mcp.EngramCreateRequest,
) (*models.EngramCreateResponse, error) {
	resolution, err := adapter.projectService.ResolveProjectIDForWrite(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		request.Payload.ProjectID,
	)
	if err != nil {
		return nil, err
	}
	payload := request.Payload
	payload.ProjectID = resolution.ProjectID
	created, err := repository.CreateEngram(
		ctx,
		adapter.db,
		repository.CreateEngramInput{
			Payload:          payload,
			EmbeddingDim:     adapter.embeddingDim,
			OwnerUserID:      &request.ActorUserID,
			EnrichmentOrigin: "mcp.engram.create",
		},
	)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return nil, nil
	}
	resolvedProjectID := resolution.ProjectID
	created.ResolvedProjectID = &resolvedProjectID
	created.UsedDefaultProject = resolution.UsedDefaultProject
	return created, nil
}

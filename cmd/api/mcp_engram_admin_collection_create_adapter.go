package main

import (
	"context"

	"engram/internal/admin"
	"engram/internal/mcp"
	"engram/internal/models"
)

type mcpEngramCollectionCreateAdapter struct {
	service        *admin.Service
	projectService projectResolutionService
}

func newMCPEngramCollectionCreateAdapter(
	service *admin.Service,
	projectService projectResolutionService,
) mcp.EngramCollectionCreateService {
	if service == nil || projectService == nil {
		return nil
	}
	return mcpEngramCollectionCreateAdapter{
		service:        service,
		projectService: projectService,
	}
}

func (adapter mcpEngramCollectionCreateAdapter) CreateCollection(
	ctx context.Context,
	request mcp.EngramCollectionCreateRequest,
) (*mcp.EngramCollectionCreateResponse, error) {
	resolution, err := adapter.projectService.ResolveProjectIDForWrite(
		ctx,
		request.ActorUserID,
		request.ActorRole,
		request.ProjectID,
	)
	if err != nil {
		return nil, err
	}
	collection, err := adapter.service.CreateCollection(
		ctx,
		request.ActorUserID,
		string(request.ActorRole),
		admin.CollectionCreateRequest{
			ProjectID:   resolution.ProjectID,
			Name:        request.Name,
			Description: request.Description,
		},
	)
	if err != nil {
		return nil, err
	}
	if collection == nil {
		return nil, nil
	}
	resolvedProjectID := resolution.ProjectID
	response := &mcp.EngramCollectionCreateResponse{
		Collection:         cloneEngramCollection(*collection),
		ResolvedProjectID:  resolvedProjectID,
		UsedDefaultProject: resolution.UsedDefaultProject,
	}
	return response, nil
}

func cloneEngramCollection(record models.EngramCollectionRecord) models.EngramCollectionRecord {
	clone := record
	if record.DeletedAt != nil {
		deletedAt := *record.DeletedAt
		clone.DeletedAt = &deletedAt
	}
	if record.DeletedByUserID != nil {
		deletedByUserID := *record.DeletedByUserID
		clone.DeletedByUserID = &deletedByUserID
	}
	if record.DeleteReason != nil {
		reason := *record.DeleteReason
		clone.DeleteReason = &reason
	}
	return clone
}

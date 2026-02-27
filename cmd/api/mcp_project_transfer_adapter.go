package main

import (
	"context"
	"encoding/json"

	internalexport "engram/internal/export"
	"engram/internal/mcp"

	"github.com/google/uuid"
)

type mcpProjectTransferAdapter struct {
	service internalexport.Service
}

func newMCPProjectExportAdapter(service internalexport.Service) mcp.ProjectExportService {
	if service == nil {
		return nil
	}
	return mcpProjectTransferAdapter{service: service}
}

func newMCPProjectImportAdapter(service internalexport.Service) mcp.ProjectImportService {
	if service == nil {
		return nil
	}
	return mcpProjectTransferAdapter{service: service}
}

func (adapter mcpProjectTransferAdapter) ExportProjectBundle(
	ctx context.Context,
	request mcp.ProjectExportRequest,
) (*mcp.ProjectExportResponse, error) {
	bundle, err := adapter.service.BuildProjectExportBundle(
		ctx,
		internalexport.ExportProjectRequest{
			ActorUserID:       request.ActorUserID,
			ActorRole:         string(request.ActorRole),
			ProjectID:         request.ProjectID,
			CollectionIDs:     append([]uuid.UUID(nil), request.CollectionIDs...),
			IncludeEmbeddings: request.IncludeEmbeddings,
		},
	)
	if err != nil {
		return nil, err
	}
	payload, err := encodeAnyToMap(bundle)
	if err != nil {
		return nil, err
	}
	return &mcp.ProjectExportResponse{Bundle: payload}, nil
}

func (adapter mcpProjectTransferAdapter) ImportProjectBundle(
	ctx context.Context,
	request mcp.ProjectImportRequest,
) (*mcp.ProjectImportResponse, error) {
	conflictPolicy, err := internalexport.ParseProjectImportConflictPolicy(request.ConflictPolicy)
	if err != nil {
		return nil, err
	}
	summary, err := adapter.service.ImportProjectBundle(
		ctx,
		internalexport.ImportProjectRequest{
			ActorUserID:     request.ActorUserID,
			ActorRole:       string(request.ActorRole),
			TargetProjectID: request.TargetProjectID,
			FileBytes:       append([]byte(nil), request.BundleBytes...),
			ConflictPolicy:  conflictPolicy,
		},
	)
	if err != nil {
		return nil, err
	}
	payload, err := encodeAnyToMap(summary)
	if err != nil {
		return nil, err
	}
	return &mcp.ProjectImportResponse{Summary: payload}, nil
}

func encodeAnyToMap(value any) (map[string]any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

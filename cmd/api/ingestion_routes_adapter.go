package main

import (
	"context"

	internalapi "engram/internal/api"
	"engram/internal/ingestion"
	"engram/internal/models"
)

const (
	defaultAdapterChunkSizeChars    = 1000
	defaultAdapterChunkOverlapChars = 180
	defaultAdapterTopKEngrams       = 4
	defaultAdapterTopKDocumentChunk = 6
)

type ingestionRouteServiceAdapter struct {
	service *ingestion.Service
}

func newIngestionRouteServiceAdapter(service *ingestion.Service) internalapi.IngestionService {
	if service == nil {
		return nil
	}
	return ingestionRouteServiceAdapter{service: service}
}

func (adapter ingestionRouteServiceAdapter) IngestText(
	ctx context.Context,
	request internalapi.IngestionTextRouteRequest,
) (ingestion.DocumentIngestResponse, error) {
	visibilityScope := resolveVisibilityScope(request.Payload.VisibilityScope)
	return adapter.service.IngestText(
		ctx,
		request.ActorUserID,
		ingestion.DocumentIngestTextRequest{
			ProjectID:         request.Payload.ProjectID,
			Title:             request.Payload.Title,
			Text:              request.Payload.Text,
			VisibilityScope:   visibilityScope,
			ChunkSizeChars:    resolveOptionalInt(request.Payload.ChunkSizeChars, defaultAdapterChunkSizeChars),
			ChunkOverlapChars: resolveOptionalInt(request.Payload.ChunkOverlapChars, defaultAdapterChunkOverlapChars),
			Metadata:          request.Payload.Metadata,
		},
	)
}

func (adapter ingestionRouteServiceAdapter) ListDocuments(
	ctx context.Context,
	request internalapi.IngestionListDocumentsRouteRequest,
) ([]models.DocumentRecord, error) {
	return adapter.service.ListDocuments(
		ctx,
		ingestion.ListDocumentsRequest{
			ActorUserID: request.ActorUserID,
			ProjectID:   request.ProjectID,
			Limit:       request.Limit,
			Offset:      request.Offset,
		},
	)
}

func (adapter ingestionRouteServiceAdapter) QueryDocumentChunks(
	ctx context.Context,
	request internalapi.IngestionDocumentQueryRouteRequest,
) ([]models.DocumentChunkQueryResult, error) {
	return adapter.service.QueryDocumentChunks(ctx, request.ActorUserID, request.Payload)
}

func (adapter ingestionRouteServiceAdapter) QueryBlended(
	ctx context.Context,
	request internalapi.IngestionBlendedQueryRouteRequest,
) (ingestion.BlendedRetrievalQueryResponse, error) {
	return adapter.service.QueryBlended(
		ctx,
		request.ActorUserID,
		ingestion.BlendedRetrievalQueryRequest{
			Query:              request.Payload.Query,
			ProjectID:          request.Payload.ProjectID,
			TopKEngrams:        resolveOptionalInt(request.Payload.TopKEngrams, defaultAdapterTopKEngrams),
			TopKDocumentChunks: resolveOptionalInt(request.Payload.TopKDocumentChunks, defaultAdapterTopKDocumentChunk),
		},
	)
}

func resolveVisibilityScope(scope *models.VisibilityScope) models.VisibilityScope {
	if scope == nil {
		return models.VisibilityScopePrivate
	}
	return *scope
}

func resolveOptionalInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

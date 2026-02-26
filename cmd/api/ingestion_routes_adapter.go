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

type resolvedIngestionPayloadOptions struct {
	visibilityScope models.VisibilityScope
	chunkSizeChars  int
	chunkOverlap    int
	metadata        map[string]any
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
	options := resolveIngestionPayloadOptions(
		request.Payload.VisibilityScope,
		request.Payload.ChunkSizeChars,
		request.Payload.ChunkOverlapChars,
		request.Payload.Metadata,
	)
	return adapter.service.IngestText(
		ctx,
		request.ActorUserID,
		ingestion.DocumentIngestTextRequest{
			ProjectID:         request.Payload.ProjectID,
			Title:             request.Payload.Title,
			Text:              request.Payload.Text,
			VisibilityScope:   options.visibilityScope,
			ChunkSizeChars:    options.chunkSizeChars,
			ChunkOverlapChars: options.chunkOverlap,
			Metadata:          options.metadata,
		},
	)
}

func (adapter ingestionRouteServiceAdapter) IngestFile(
	ctx context.Context,
	request internalapi.IngestionFileRouteRequest,
) (ingestion.DocumentIngestResponse, error) {
	filePayload := newFileIngestRequest(request.Upload)
	payloadOptions := resolveIngestionPayloadOptions(
		request.Payload.VisibilityScope,
		request.Payload.ChunkSizeChars,
		request.Payload.ChunkOverlapChars,
		request.Payload.Metadata,
	)
	servicePayload := ingestion.DocumentIngestFileRequest{
		ProjectID:         request.Payload.ProjectID,
		Title:             request.Payload.Title,
		VisibilityScope:   payloadOptions.visibilityScope,
		ChunkSizeChars:    payloadOptions.chunkSizeChars,
		ChunkOverlapChars: payloadOptions.chunkOverlap,
		Metadata:          payloadOptions.metadata,
	}
	response, err := adapter.service.IngestFile(
		ctx,
		request.ActorUserID,
		servicePayload,
		filePayload,
	)
	if err != nil {
		return ingestion.DocumentIngestResponse{}, err
	}
	return response, nil
}

func newFileIngestRequest(upload internalapi.IngestionFileUpload) ingestion.FileIngestRequest {
	return ingestion.FileIngestRequest{
		Filename:     upload.Filename,
		MimeType:     upload.MimeType,
		ContentBytes: upload.ContentBytes,
	}
}

func resolveIngestionPayloadOptions(
	scope *models.VisibilityScope,
	chunkSizeChars *int,
	chunkOverlapChars *int,
	metadata map[string]any,
) resolvedIngestionPayloadOptions {
	return resolvedIngestionPayloadOptions{
		visibilityScope: resolveVisibilityScope(scope),
		chunkSizeChars:  resolveOptionalInt(chunkSizeChars, defaultAdapterChunkSizeChars),
		chunkOverlap:    resolveOptionalInt(chunkOverlapChars, defaultAdapterChunkOverlapChars),
		metadata:        metadata,
	}
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

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"engram/internal/ingestion"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type fakeIngestionService struct {
	ingestTextFn func(
		ctx context.Context,
		request IngestionTextRouteRequest,
	) (ingestion.DocumentIngestResponse, error)
	listDocumentsFn func(
		ctx context.Context,
		request IngestionListDocumentsRouteRequest,
	) ([]models.DocumentRecord, error)
	queryDocumentChunksFn func(
		ctx context.Context,
		request IngestionDocumentQueryRouteRequest,
	) ([]models.DocumentChunkQueryResult, error)
	queryBlendedFn func(
		ctx context.Context,
		request IngestionBlendedQueryRouteRequest,
	) (ingestion.BlendedRetrievalQueryResponse, error)
}

func (service fakeIngestionService) IngestText(
	ctx context.Context,
	request IngestionTextRouteRequest,
) (ingestion.DocumentIngestResponse, error) {
	if service.ingestTextFn == nil {
		return ingestion.DocumentIngestResponse{}, nil
	}
	return service.ingestTextFn(ctx, request)
}

func (service fakeIngestionService) ListDocuments(
	ctx context.Context,
	request IngestionListDocumentsRouteRequest,
) ([]models.DocumentRecord, error) {
	if service.listDocumentsFn == nil {
		return []models.DocumentRecord{}, nil
	}
	return service.listDocumentsFn(ctx, request)
}

func (service fakeIngestionService) QueryDocumentChunks(
	ctx context.Context,
	request IngestionDocumentQueryRouteRequest,
) ([]models.DocumentChunkQueryResult, error) {
	if service.queryDocumentChunksFn == nil {
		return []models.DocumentChunkQueryResult{}, nil
	}
	return service.queryDocumentChunksFn(ctx, request)
}

func (service fakeIngestionService) QueryBlended(
	ctx context.Context,
	request IngestionBlendedQueryRouteRequest,
) (ingestion.BlendedRetrievalQueryResponse, error) {
	if service.queryBlendedFn == nil {
		return ingestion.BlendedRetrievalQueryResponse{}, nil
	}
	return service.queryBlendedFn(ctx, request)
}

func TestMountIngestionRoutesRegistersEndpoints(t *testing.T) {
	router := chi.NewRouter()
	MountIngestionRoutes(router, fakeIngestionService{})
	routes := collectChatRoutes(t, router)

	requiredRoutes := []string{
		"/api/v1/ingestion/text",
		"/api/v1/ingestion/documents",
		"/api/v1/ingestion/query",
		"/api/v1/ingestion/query/blended",
	}
	for _, route := range requiredRoutes {
		requireChatRoute(t, routes, route)
	}
}

func TestIngestTextHandlerWritesCreatedResponse(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000951")
	captured := IngestionTextRouteRequest{}
	documentID := uuid.MustParse("00000000-0000-0000-0000-000000000952")
	service := fakeIngestionService{
		ingestTextFn: func(_ context.Context, request IngestionTextRouteRequest) (ingestion.DocumentIngestResponse, error) {
			captured = request
			timestamp := time.Date(2026, 2, 26, 17, 0, 0, 0, time.UTC)
			return ingestion.DocumentIngestResponse{
				Document: models.DocumentRecord{
					DocumentID:      documentID,
					OwnerUserID:     actorID,
					ProjectID:       request.Payload.ProjectID,
					Title:           request.Payload.Title,
					SourceType:      models.DocumentSourceTypeText,
					VisibilityScope: models.VisibilityScopePrivate,
					ContentHash:     "hash",
					ChunkCount:      3,
					CreatedAt:       timestamp,
					UpdatedAt:       timestamp,
				},
			}, nil
		},
	}
	router := chi.NewRouter()
	MountIngestionRoutes(router, service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/ingestion/text",
		strings.NewReader(`{"project_id":"project-alpha","title":"Doc One","text":"Document body"}`),
	)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "analyst"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	if captured.ActorUserID != actorID {
		t.Fatalf("expected actor id %s, got %s", actorID, captured.ActorUserID)
	}
	if captured.Payload.ChunkSizeChars == nil || *captured.Payload.ChunkSizeChars != defaultIngestChunkSizeChars {
		t.Fatalf("expected default chunk size %d, got %#v", defaultIngestChunkSizeChars, captured.Payload.ChunkSizeChars)
	}
	if captured.Payload.ChunkOverlapChars == nil || *captured.Payload.ChunkOverlapChars != defaultIngestChunkOverlapChars {
		t.Fatalf(
			"expected default chunk overlap %d, got %#v",
			defaultIngestChunkOverlapChars,
			captured.Payload.ChunkOverlapChars,
		)
	}
	var payload ingestion.DocumentIngestResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Document.DocumentID != documentID {
		t.Fatalf("expected document id %s, got %s", documentID, payload.Document.DocumentID)
	}
}

func TestListDocumentsHandlerUsesDefaultPaging(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000953")
	captured := IngestionListDocumentsRouteRequest{}
	service := fakeIngestionService{
		listDocumentsFn: func(_ context.Context, request IngestionListDocumentsRouteRequest) ([]models.DocumentRecord, error) {
			captured = request
			return []models.DocumentRecord{}, nil
		},
	}
	router := chi.NewRouter()
	MountIngestionRoutes(router, service)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/ingestion/documents", nil)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "viewer"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if captured.ActorUserID != actorID {
		t.Fatalf("expected actor id %s, got %s", actorID, captured.ActorUserID)
	}
	if captured.Limit != defaultDocumentListLimit || captured.Offset != defaultDocumentListOffset {
		t.Fatalf(
			"expected defaults limit/offset %d/%d, got %d/%d",
			defaultDocumentListLimit,
			defaultDocumentListOffset,
			captured.Limit,
			captured.Offset,
		)
	}
}

func TestQueryDocumentsHandlerAppliesDefaultTopK(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000954")
	captured := IngestionDocumentQueryRouteRequest{}
	service := fakeIngestionService{
		queryDocumentChunksFn: func(
			_ context.Context,
			request IngestionDocumentQueryRouteRequest,
		) ([]models.DocumentChunkQueryResult, error) {
			captured = request
			return []models.DocumentChunkQueryResult{}, nil
		},
	}
	router := chi.NewRouter()
	MountIngestionRoutes(router, service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/ingestion/query",
		strings.NewReader(`{"query":"incident timeline"}`),
	)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "analyst"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if captured.Payload.TopK != defaultDocumentChunkTopK {
		t.Fatalf("expected default top_k %d, got %d", defaultDocumentChunkTopK, captured.Payload.TopK)
	}
}

func TestQueryBlendedHandlerAppliesDefaultTopK(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000955")
	captured := IngestionBlendedQueryRouteRequest{}
	service := fakeIngestionService{
		queryBlendedFn: func(
			_ context.Context,
			request IngestionBlendedQueryRouteRequest,
		) (ingestion.BlendedRetrievalQueryResponse, error) {
			captured = request
			return ingestion.BlendedRetrievalQueryResponse{}, nil
		},
	}
	router := chi.NewRouter()
	MountIngestionRoutes(router, service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/ingestion/query/blended",
		strings.NewReader(`{"query":"incident timeline"}`),
	)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "analyst"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if captured.Payload.TopKEngrams == nil || *captured.Payload.TopKEngrams != defaultBlendedTopKEngrams {
		t.Fatalf("expected default top_k_engrams %d, got %#v", defaultBlendedTopKEngrams, captured.Payload.TopKEngrams)
	}
	if captured.Payload.TopKDocumentChunks == nil || *captured.Payload.TopKDocumentChunks != defaultBlendedTopKDocumentChunks {
		t.Fatalf(
			"expected default top_k_document_chunks %d, got %#v",
			defaultBlendedTopKDocumentChunks,
			captured.Payload.TopKDocumentChunks,
		)
	}
}

func TestIngestionRoutesRequireAuthenticatedActor(t *testing.T) {
	router := chi.NewRouter()
	MountIngestionRoutes(router, fakeIngestionService{})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/ingestion/documents", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
}

func TestIngestTextHandlerMapsServiceError(t *testing.T) {
	service := fakeIngestionService{
		ingestTextFn: func(_ context.Context, _ IngestionTextRouteRequest) (ingestion.DocumentIngestResponse, error) {
			return ingestion.DocumentIngestResponse{}, ingestion.ServiceError{
				Detail:     "chunk_overlap_chars must be smaller than chunk_size_chars",
				StatusCode: http.StatusUnprocessableEntity,
			}
		},
	}
	router := chi.NewRouter()
	MountIngestionRoutes(router, service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/ingestion/text",
		strings.NewReader(`{"project_id":"project-alpha","title":"Doc One","text":"Document body","chunk_size_chars":500,"chunk_overlap_chars":500}`),
	)
	request = WithAdminActor(
		request,
		AdminActor{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000956"), Role: "analyst"},
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", response.Code)
	}
}

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
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
	ingestFileFn func(
		ctx context.Context,
		request IngestionFileRouteRequest,
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

func (service fakeIngestionService) IngestFile(
	ctx context.Context,
	request IngestionFileRouteRequest,
) (ingestion.DocumentIngestResponse, error) {
	if service.ingestFileFn == nil {
		return ingestion.DocumentIngestResponse{}, nil
	}
	return service.ingestFileFn(ctx, request)
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
	MountIngestionRoutes(router, fakeIngestionService{}, IngestionRouteOptions{})
	routes := collectChatRoutes(t, router)

	requiredRoutes := []string{
		"/api/v1/ingestion/text",
		"/api/v1/ingestion/file",
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
	MountIngestionRoutes(router, service, IngestionRouteOptions{})

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/ingestion/text",
		strings.NewReader(`{"project_id":"project-alpha","title":"Doc One","text":"Document body"}`),
	)
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "analyst"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assertIngestionStatusCode(t, response, http.StatusCreated)
	assertDefaultIngestTextRequest(t, captured, actorID)
	payload := decodeIngestionResponse(t, response)
	if payload.Document.DocumentID != documentID {
		t.Fatalf("expected document id %s, got %s", documentID, payload.Document.DocumentID)
	}
}

func TestIngestFileHandlerWritesCreatedResponse(t *testing.T) {
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000957")
	captured := IngestionFileRouteRequest{}
	documentID := uuid.MustParse("00000000-0000-0000-0000-000000000958")
	service := fakeIngestionService{
		ingestFileFn: func(
			_ context.Context,
			request IngestionFileRouteRequest,
		) (ingestion.DocumentIngestResponse, error) {
			captured = request
			timestamp := time.Date(2026, 2, 26, 17, 15, 0, 0, time.UTC)
			return ingestion.DocumentIngestResponse{
				Document: models.DocumentRecord{
					DocumentID:      documentID,
					OwnerUserID:     actorID,
					ProjectID:       request.Payload.ProjectID,
					Title:           "Imported Notes",
					SourceType:      models.DocumentSourceTypeFile,
					SourceName:      &request.Upload.Filename,
					MimeType:        request.Upload.MimeType,
					VisibilityScope: models.VisibilityScopePrivate,
					ContentHash:     "hash",
					ChunkCount:      2,
					CreatedAt:       timestamp,
					UpdatedAt:       timestamp,
				},
			}, nil
		},
	}
	router := chi.NewRouter()
	MountIngestionRoutes(router, service, IngestionRouteOptions{})

	request := newIngestFileMultipartRequest(t, ingestFileMultipartRequestInput{
		path: "/api/v1/ingestion/file",
		formFields: map[string]string{
			"project_id":    "project-alpha",
			"title":         "Imported Notes",
			"metadata_json": `{"source":"upload"}`,
		},
		filename:    "notes.txt",
		contentType: "text/plain",
		fileBody:    "line one\nline two",
	})
	request = WithAdminActor(request, AdminActor{UserID: actorID, Role: "analyst"})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assertIngestionStatusCode(t, response, http.StatusCreated)
	if captured.Payload.ProjectID != "project-alpha" {
		t.Fatalf("expected project_id project-alpha, got %q", captured.Payload.ProjectID)
	}
	if captured.Payload.Title == nil || *captured.Payload.Title != "Imported Notes" {
		t.Fatalf("expected title Imported Notes, got %#v", captured.Payload.Title)
	}
	if captured.Upload.Filename != "notes.txt" {
		t.Fatalf("expected filename notes.txt, got %q", captured.Upload.Filename)
	}
	if string(captured.Upload.ContentBytes) != "line one\nline two" {
		t.Fatalf("unexpected upload content: %q", string(captured.Upload.ContentBytes))
	}
}

func TestIngestFileHandlerRejectsInvalidMetadataJSON(t *testing.T) {
	testCases := []struct {
		name           string
		actorUserID    uuid.UUID
		routeOptions   IngestionRouteOptions
		metadataJSON   string
		expectedStatus int
	}{
		{
			name:           "invalid json shape",
			actorUserID:    uuid.MustParse("00000000-0000-0000-0000-000000000959"),
			routeOptions:   IngestionRouteOptions{},
			metadataJSON:   "{invalid",
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "metadata too large",
			actorUserID:    uuid.MustParse("00000000-0000-0000-0000-000000000960"),
			routeOptions:   IngestionRouteOptions{MaxMetadataJSONBytes: 8},
			metadataJSON:   `{"long":"value"}`,
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			router := chi.NewRouter()
			MountIngestionRoutes(router, fakeIngestionService{}, testCase.routeOptions)

			request := newIngestFileMultipartRequest(t, ingestFileMultipartRequestInput{
				path: "/api/v1/ingestion/file",
				formFields: map[string]string{
					"project_id":    "project-alpha",
					"metadata_json": testCase.metadataJSON,
				},
				filename:    "notes.txt",
				contentType: "text/plain",
				fileBody:    "line one\nline two",
			})
			request = WithAdminActor(
				request,
				AdminActor{UserID: testCase.actorUserID, Role: "analyst"},
			)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			assertIngestionStatusCode(t, response, testCase.expectedStatus)
		})
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
	MountIngestionRoutes(router, service, IngestionRouteOptions{})

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
	MountIngestionRoutes(router, service, IngestionRouteOptions{})

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
	MountIngestionRoutes(router, service, IngestionRouteOptions{})

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
	MountIngestionRoutes(router, fakeIngestionService{}, IngestionRouteOptions{})

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
	MountIngestionRoutes(router, service, IngestionRouteOptions{})

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

	assertIngestionStatusCode(t, response, http.StatusUnprocessableEntity)
}

func assertIngestionStatusCode(t *testing.T, response *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()
	if response.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, response.Code)
	}
}

func assertDefaultIngestTextRequest(t *testing.T, request IngestionTextRouteRequest, actorID uuid.UUID) {
	t.Helper()
	if request.ActorUserID != actorID {
		t.Fatalf("expected actor id %s, got %s", actorID, request.ActorUserID)
	}
	if request.Payload.ChunkSizeChars == nil || *request.Payload.ChunkSizeChars != defaultIngestChunkSizeChars {
		t.Fatalf("expected default chunk size %d, got %#v", defaultIngestChunkSizeChars, request.Payload.ChunkSizeChars)
	}
	if request.Payload.ChunkOverlapChars == nil || *request.Payload.ChunkOverlapChars != defaultIngestChunkOverlapChars {
		t.Fatalf(
			"expected default chunk overlap %d, got %#v",
			defaultIngestChunkOverlapChars,
			request.Payload.ChunkOverlapChars,
		)
	}
}

func decodeIngestionResponse(
	t *testing.T,
	response *httptest.ResponseRecorder,
) ingestion.DocumentIngestResponse {
	t.Helper()
	payload := ingestion.DocumentIngestResponse{}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return payload
}

type ingestFileMultipartRequestInput struct {
	path        string
	formFields  map[string]string
	filename    string
	contentType string
	fileBody    string
}

func newIngestFileMultipartRequest(t *testing.T, input ingestFileMultipartRequestInput) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, value := range input.formFields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field %q: %v", key, err)
		}
	}
	fileHeader := textproto.MIMEHeader{}
	fileHeader.Set("Content-Disposition", `form-data; name="file"; filename="`+input.filename+`"`)
	fileHeader.Set("Content-Type", input.contentType)
	part, err := writer.CreatePart(fileHeader)
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := part.Write([]byte(input.fileBody)); err != nil {
		t.Fatalf("write file part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, input.path, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"engram/internal/ingestion"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	defaultDocumentListLimit  = 100
	minDocumentListLimit      = 1
	maxDocumentListLimit      = 500
	defaultDocumentListOffset = 0

	defaultDocumentChunkTopK = 6
	minDocumentChunkTopK     = 1
	maxDocumentChunkTopK     = 50

	defaultBlendedTopKEngrams        = 4
	minBlendedTopKEngrams            = 1
	maxBlendedTopKEngrams            = 25
	defaultBlendedTopKDocumentChunks = 6
	minBlendedTopKDocumentChunks     = 1
	maxBlendedTopKDocumentChunks     = 50

	defaultIngestChunkSizeChars    = 1000
	minIngestChunkSizeChars        = 300
	maxIngestChunkSizeChars        = 4000
	defaultIngestChunkOverlapChars = 180
	minIngestChunkOverlapChars     = 0
	maxIngestChunkOverlapChars     = 1600

	defaultIngestMetadataJSONMaxBytes = 20000
	defaultMultipartMaxMemoryBytes    = 32 << 20
)

// IngestionTextPayload captures `/ingestion/text` payload values.
type IngestionTextPayload struct {
	ProjectID         string                  `json:"project_id"`
	Title             string                  `json:"title"`
	Text              string                  `json:"text"`
	VisibilityScope   *models.VisibilityScope `json:"visibility_scope,omitempty"`
	ChunkSizeChars    *int                    `json:"chunk_size_chars,omitempty"`
	ChunkOverlapChars *int                    `json:"chunk_overlap_chars,omitempty"`
	Metadata          map[string]any          `json:"metadata,omitempty"`
}

// IngestionBlendedQueryPayload captures `/ingestion/query/blended` payload values.
type IngestionBlendedQueryPayload struct {
	Query              string  `json:"query"`
	ProjectID          *string `json:"project_id,omitempty"`
	TopKEngrams        *int    `json:"top_k_engrams,omitempty"`
	TopKDocumentChunks *int    `json:"top_k_document_chunks,omitempty"`
}

// IngestionDocumentChunkQueryPayload captures `/ingestion/query` payload values.
type IngestionDocumentChunkQueryPayload struct {
	Query       string      `json:"query"`
	ProjectID   *string     `json:"project_id,omitempty"`
	DocumentIDs []uuid.UUID `json:"document_ids,omitempty"`
	TopK        *int        `json:"top_k,omitempty"`
}

type boundedIntRule struct {
	defaultValue int
	minimumValue int
	maximumValue int
	errorDetail  string
}

// IngestionRouteOptions captures route-level ingestion parser settings.
type IngestionRouteOptions struct {
	MaxMetadataJSONBytes int
}

// IngestionTextRouteRequest captures actor-scoped text-ingestion route input.
type IngestionTextRouteRequest struct {
	ActorUserID uuid.UUID
	Payload     IngestionTextPayload
}

// IngestionFileUpload captures uploaded file details used by ingestion routes.
type IngestionFileUpload struct {
	Filename     string
	MimeType     *string
	ContentBytes []byte
}

// IngestionFilePayload captures `/ingestion/file` payload values.
type IngestionFilePayload struct {
	ProjectID         string                  `json:"project_id"`
	Title             *string                 `json:"title,omitempty"`
	VisibilityScope   *models.VisibilityScope `json:"visibility_scope,omitempty"`
	ChunkSizeChars    *int                    `json:"chunk_size_chars,omitempty"`
	ChunkOverlapChars *int                    `json:"chunk_overlap_chars,omitempty"`
	Metadata          map[string]any          `json:"metadata,omitempty"`
}

// IngestionFileRouteRequest captures actor-scoped file-ingestion route input.
type IngestionFileRouteRequest struct {
	ActorUserID uuid.UUID
	Payload     IngestionFilePayload
	Upload      IngestionFileUpload
}

// IngestionListDocumentsRouteRequest captures actor-scoped list-documents route input.
type IngestionListDocumentsRouteRequest struct {
	ActorUserID uuid.UUID
	ProjectID   *string
	Limit       int
	Offset      int
}

// IngestionDocumentQueryRouteRequest captures actor-scoped document-query route input.
type IngestionDocumentQueryRouteRequest struct {
	ActorUserID uuid.UUID
	Payload     models.DocumentChunkQueryRequest
}

// IngestionBlendedQueryRouteRequest captures actor-scoped blended-query route input.
type IngestionBlendedQueryRouteRequest struct {
	ActorUserID uuid.UUID
	Payload     IngestionBlendedQueryPayload
}

// IngestionService captures ingestion route behavior used by REST handlers.
type IngestionService interface {
	IngestText(ctx context.Context, request IngestionTextRouteRequest) (ingestion.DocumentIngestResponse, error)
	IngestFile(ctx context.Context, request IngestionFileRouteRequest) (ingestion.DocumentIngestResponse, error)
	ListDocuments(ctx context.Context, request IngestionListDocumentsRouteRequest) ([]models.DocumentRecord, error)
	QueryDocumentChunks(
		ctx context.Context,
		request IngestionDocumentQueryRouteRequest,
	) ([]models.DocumentChunkQueryResult, error)
	QueryBlended(
		ctx context.Context,
		request IngestionBlendedQueryRouteRequest,
	) (ingestion.BlendedRetrievalQueryResponse, error)
}

func newIngestionValidationError(detail string) error {
	return ingestion.ServiceError{Detail: detail, StatusCode: http.StatusUnprocessableEntity}
}

func parseOptionalProjectID(rawProjectID string) *string {
	normalized := strings.TrimSpace(rawProjectID)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func writeIngestionRouteError(writer http.ResponseWriter, err error) {
	var serviceError ingestion.ServiceError
	if errors.As(err, &serviceError) {
		writeJSON(
			writer,
			serviceError.StatusCode,
			map[string]string{"detail": serviceError.Detail},
		)
		return
	}
	writeJSON(writer, http.StatusInternalServerError, map[string]string{"detail": "Internal server error"})
}

func normalizeIngestionTextPayload(payload IngestionTextPayload) (IngestionTextPayload, error) {
	payload.ProjectID = strings.TrimSpace(payload.ProjectID)
	payload.Title = strings.TrimSpace(payload.Title)
	payload.Text = strings.TrimSpace(payload.Text)
	if err := validateIngestionTextRequiredFields(payload); err != nil {
		return IngestionTextPayload{}, err
	}
	normalizedPayload, err := normalizeIngestionTextChunkShape(payload)
	if err != nil {
		return IngestionTextPayload{}, err
	}
	return normalizeIngestionTextOptionalFields(normalizedPayload), nil
}

func validateIngestionTextRequiredFields(payload IngestionTextPayload) error {
	switch {
	case payload.ProjectID == "":
		return newIngestionValidationError("project_id is required")
	case payload.Title == "":
		return newIngestionValidationError("title is required")
	case payload.Text == "":
		return newIngestionValidationError("text is required")
	default:
		return nil
	}
}

func normalizeIngestionTextChunkShape(payload IngestionTextPayload) (IngestionTextPayload, error) {
	chunkSizeChars, err := resolveBoundedInt(payload.ChunkSizeChars, boundedIntRule{
		defaultValue: defaultIngestChunkSizeChars,
		minimumValue: minIngestChunkSizeChars,
		maximumValue: maxIngestChunkSizeChars,
		errorDetail:  "chunk_size_chars must be between 300 and 4000",
	})
	if err != nil {
		return IngestionTextPayload{}, err
	}
	chunkOverlapChars, err := resolveBoundedInt(payload.ChunkOverlapChars, boundedIntRule{
		defaultValue: defaultIngestChunkOverlapChars,
		minimumValue: minIngestChunkOverlapChars,
		maximumValue: maxIngestChunkOverlapChars,
		errorDetail:  "chunk_overlap_chars must be between 0 and 1600",
	})
	if err != nil {
		return IngestionTextPayload{}, err
	}
	if chunkOverlapChars >= chunkSizeChars {
		return IngestionTextPayload{}, newIngestionValidationError(
			"chunk_overlap_chars must be smaller than chunk_size_chars",
		)
	}
	payload.ChunkSizeChars = &chunkSizeChars
	payload.ChunkOverlapChars = &chunkOverlapChars
	return payload, nil
}

func normalizeIngestionTextOptionalFields(payload IngestionTextPayload) IngestionTextPayload {
	if payload.VisibilityScope == nil {
		privateScope := models.VisibilityScopePrivate
		payload.VisibilityScope = &privateScope
	}
	if payload.Metadata == nil {
		payload.Metadata = map[string]any{}
	}
	return payload
}

func normalizeIngestionFilePayload(payload IngestionFilePayload) (IngestionFilePayload, error) {
	payload.ProjectID = strings.TrimSpace(payload.ProjectID)
	if payload.ProjectID == "" {
		return IngestionFilePayload{}, newIngestionValidationError("project_id is required")
	}
	chunkSizeChars, err := resolveBoundedInt(payload.ChunkSizeChars, boundedIntRule{
		defaultValue: defaultIngestChunkSizeChars,
		minimumValue: minIngestChunkSizeChars,
		maximumValue: maxIngestChunkSizeChars,
		errorDetail:  "chunk_size_chars must be between 300 and 4000",
	})
	if err != nil {
		return IngestionFilePayload{}, err
	}
	chunkOverlapChars, err := resolveBoundedInt(payload.ChunkOverlapChars, boundedIntRule{
		defaultValue: defaultIngestChunkOverlapChars,
		minimumValue: minIngestChunkOverlapChars,
		maximumValue: maxIngestChunkOverlapChars,
		errorDetail:  "chunk_overlap_chars must be between 0 and 1600",
	})
	if err != nil {
		return IngestionFilePayload{}, err
	}
	if chunkOverlapChars >= chunkSizeChars {
		return IngestionFilePayload{}, newIngestionValidationError(
			"chunk_overlap_chars must be smaller than chunk_size_chars",
		)
	}
	payload.ChunkSizeChars = &chunkSizeChars
	payload.ChunkOverlapChars = &chunkOverlapChars
	if payload.VisibilityScope == nil {
		privateScope := models.VisibilityScopePrivate
		payload.VisibilityScope = &privateScope
	}
	if payload.Metadata == nil {
		payload.Metadata = map[string]any{}
	}
	return payload, nil
}

func resolveBoundedInt(
	value *int,
	rule boundedIntRule,
) (int, error) {
	resolved := rule.defaultValue
	if value != nil {
		resolved = *value
	}
	if resolved < rule.minimumValue || resolved > rule.maximumValue {
		return 0, newIngestionValidationError(rule.errorDetail)
	}
	return resolved, nil
}

func decodeIngestionTextPayload(request *http.Request) (IngestionTextPayload, error) {
	payload := IngestionTextPayload{}
	if err := decodeChatPayload(request, &payload); err != nil {
		return IngestionTextPayload{}, err
	}
	return normalizeIngestionTextPayload(payload)
}

func parseFormIntField(
	formValues map[string][]string,
	fieldName string,
	rule boundedIntRule,
) (int, error) {
	value := strings.TrimSpace(firstFormValue(formValues, fieldName))
	if value == "" {
		return rule.defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, newIngestionValidationError(fieldName + " must be an integer")
	}
	return resolveBoundedInt(&parsed, rule)
}

func parseFormVisibilityScope(formValues map[string][]string) (*models.VisibilityScope, error) {
	rawVisibility := strings.TrimSpace(firstFormValue(formValues, "visibility_scope"))
	if rawVisibility == "" {
		privateScope := models.VisibilityScopePrivate
		return &privateScope, nil
	}
	parsed, err := models.ParseVisibilityScope(rawVisibility)
	if err != nil {
		return nil, newIngestionValidationError("visibility_scope is invalid")
	}
	return &parsed, nil
}

func parseMetadataJSON(metadataJSON string, maxBytes int) (map[string]any, error) {
	trimmed := strings.TrimSpace(metadataJSON)
	if trimmed == "" {
		return map[string]any{}, nil
	}
	if len([]byte(trimmed)) > maxBytes {
		return nil, ingestion.ServiceError{
			Detail:     "metadata_json exceeds max allowed size of " + strconv.Itoa(maxBytes) + " bytes",
			StatusCode: http.StatusRequestEntityTooLarge,
		}
	}
	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil, newIngestionValidationError("metadata_json must be valid JSON")
	}
	parsedMap, ok := parsed.(map[string]any)
	if !ok {
		return nil, newIngestionValidationError("metadata_json must be a JSON object")
	}
	return parsedMap, nil
}

func readIngestionUpload(request *http.Request) (IngestionFileUpload, error) {
	file, header, err := request.FormFile("file")
	if err != nil {
		return IngestionFileUpload{}, newIngestionValidationError("file is required")
	}
	defer file.Close()
	contentBytes, err := io.ReadAll(file)
	if err != nil {
		return IngestionFileUpload{}, err
	}
	mimeTypeValue := strings.TrimSpace(header.Header.Get("Content-Type"))
	var mimeType *string
	if mimeTypeValue != "" {
		mimeType = &mimeTypeValue
	}
	filename := header.Filename
	if strings.TrimSpace(filename) == "" {
		filename = "uploaded.txt"
	}
	return IngestionFileUpload{
		Filename:     filename,
		MimeType:     mimeType,
		ContentBytes: contentBytes,
	}, nil
}

func parseIngestionFilePayload(
	request *http.Request,
	options IngestionRouteOptions,
) (IngestionFilePayload, IngestionFileUpload, error) {
	if err := request.ParseMultipartForm(defaultMultipartMaxMemoryBytes); err != nil {
		return IngestionFilePayload{}, IngestionFileUpload{}, newIngestionValidationError("invalid multipart form")
	}
	formValues := request.MultipartForm.Value
	chunkSizeChars, err := parseFormIntField(formValues, "chunk_size_chars", boundedIntRule{
		defaultValue: defaultIngestChunkSizeChars,
		minimumValue: minIngestChunkSizeChars,
		maximumValue: maxIngestChunkSizeChars,
		errorDetail:  "chunk_size_chars must be between 300 and 4000",
	})
	if err != nil {
		return IngestionFilePayload{}, IngestionFileUpload{}, err
	}
	chunkOverlapChars, err := parseFormIntField(formValues, "chunk_overlap_chars", boundedIntRule{
		defaultValue: defaultIngestChunkOverlapChars,
		minimumValue: minIngestChunkOverlapChars,
		maximumValue: maxIngestChunkOverlapChars,
		errorDetail:  "chunk_overlap_chars must be between 0 and 1600",
	})
	if err != nil {
		return IngestionFilePayload{}, IngestionFileUpload{}, err
	}
	visibilityScope, err := parseFormVisibilityScope(formValues)
	if err != nil {
		return IngestionFilePayload{}, IngestionFileUpload{}, err
	}
	metadata, err := parseMetadataJSON(
		firstFormValue(formValues, "metadata_json"),
		resolveIngestionOptions(options).MaxMetadataJSONBytes,
	)
	if err != nil {
		return IngestionFilePayload{}, IngestionFileUpload{}, err
	}
	title := optionalFormValue(formValues, "title")
	upload, err := readIngestionUpload(request)
	if err != nil {
		return IngestionFilePayload{}, IngestionFileUpload{}, err
	}
	normalizedPayload, err := normalizeIngestionFilePayload(IngestionFilePayload{
		ProjectID:         firstFormValue(formValues, "project_id"),
		Title:             title,
		VisibilityScope:   visibilityScope,
		ChunkSizeChars:    &chunkSizeChars,
		ChunkOverlapChars: &chunkOverlapChars,
		Metadata:          metadata,
	})
	if err != nil {
		return IngestionFilePayload{}, IngestionFileUpload{}, err
	}
	return normalizedPayload, upload, nil
}

func firstFormValue(formValues map[string][]string, key string) string {
	values, ok := formValues[key]
	if !ok || len(values) == 0 {
		return ""
	}
	return values[0]
}

func optionalFormValue(formValues map[string][]string, key string) *string {
	value := strings.TrimSpace(firstFormValue(formValues, key))
	if value == "" {
		return nil
	}
	return &value
}

func resolveIngestionOptions(options IngestionRouteOptions) IngestionRouteOptions {
	if options.MaxMetadataJSONBytes <= 0 {
		options.MaxMetadataJSONBytes = defaultIngestMetadataJSONMaxBytes
	}
	return options
}

func decodeDocumentChunkQueryPayload(request *http.Request) (models.DocumentChunkQueryRequest, error) {
	payload := IngestionDocumentChunkQueryPayload{}
	if err := decodeChatPayload(request, &payload); err != nil {
		return models.DocumentChunkQueryRequest{}, err
	}
	payload.Query = strings.TrimSpace(payload.Query)
	if payload.Query == "" {
		return models.DocumentChunkQueryRequest{}, newIngestionValidationError("query is required")
	}
	topK, err := resolveBoundedInt(
		payload.TopK,
		boundedIntRule{
			defaultValue: defaultDocumentChunkTopK,
			minimumValue: minDocumentChunkTopK,
			maximumValue: maxDocumentChunkTopK,
			errorDetail:  "top_k must be between 1 and 50",
		},
	)
	if err != nil {
		return models.DocumentChunkQueryRequest{}, err
	}
	if payload.DocumentIDs == nil {
		payload.DocumentIDs = []uuid.UUID{}
	}
	return models.DocumentChunkQueryRequest{
		Query:       payload.Query,
		ProjectID:   payload.ProjectID,
		DocumentIDs: payload.DocumentIDs,
		TopK:        topK,
	}, nil
}

func decodeBlendedQueryPayload(request *http.Request) (IngestionBlendedQueryPayload, error) {
	payload := IngestionBlendedQueryPayload{}
	if err := decodeChatPayload(request, &payload); err != nil {
		return IngestionBlendedQueryPayload{}, err
	}
	payload.Query = strings.TrimSpace(payload.Query)
	if payload.Query == "" {
		return IngestionBlendedQueryPayload{}, newIngestionValidationError("query is required")
	}
	topKEngrams, err := resolveBoundedInt(
		payload.TopKEngrams,
		boundedIntRule{
			defaultValue: defaultBlendedTopKEngrams,
			minimumValue: minBlendedTopKEngrams,
			maximumValue: maxBlendedTopKEngrams,
			errorDetail:  "top_k_engrams must be between 1 and 25",
		},
	)
	if err != nil {
		return IngestionBlendedQueryPayload{}, err
	}
	topKDocumentChunks, err := resolveBoundedInt(
		payload.TopKDocumentChunks,
		boundedIntRule{
			defaultValue: defaultBlendedTopKDocumentChunks,
			minimumValue: minBlendedTopKDocumentChunks,
			maximumValue: maxBlendedTopKDocumentChunks,
			errorDetail:  "top_k_document_chunks must be between 1 and 50",
		},
	)
	if err != nil {
		return IngestionBlendedQueryPayload{}, err
	}
	payload.TopKEngrams = &topKEngrams
	payload.TopKDocumentChunks = &topKDocumentChunks
	return payload, nil
}

func ingestionPayloadRouteHandler[Payload any, Result any](
	decodePayload func(request *http.Request) (Payload, error),
	operation func(ctx context.Context, actorUserID uuid.UUID, payload Payload) (Result, error),
	writeSuccess func(writer http.ResponseWriter, result Result),
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorUserID, _, ok := requireProjectActor(writer, request)
		if !ok {
			return
		}
		payload, err := decodePayload(request)
		if err != nil {
			writeIngestionRouteError(writer, err)
			return
		}
		result, err := operation(request.Context(), actorUserID, payload)
		if err != nil {
			writeIngestionRouteError(writer, err)
			return
		}
		writeSuccess(writer, result)
	}
}

func ingestionTypedRouteHandler[Payload any, Request any, Result any](
	serviceOperation func(ctx context.Context, request Request) (Result, error),
	buildRequest func(actorUserID uuid.UUID, payload Payload) Request,
	decodePayload func(request *http.Request) (Payload, error),
	successStatusCode int,
) http.HandlerFunc {
	return ingestionPayloadRouteHandler(
		decodePayload,
		func(
			ctx context.Context,
			actorUserID uuid.UUID,
			payload Payload,
		) (Result, error) {
			return serviceOperation(ctx, buildRequest(actorUserID, payload))
		},
		func(writer http.ResponseWriter, result Result) {
			writeJSON(writer, successStatusCode, result)
		},
	)
}

func ingestTextHandler(service IngestionService) http.HandlerFunc {
	return ingestionTypedRouteHandler(
		service.IngestText,
		func(actorUserID uuid.UUID, payload IngestionTextPayload) IngestionTextRouteRequest {
			return IngestionTextRouteRequest{ActorUserID: actorUserID, Payload: payload}
		},
		decodeIngestionTextPayload,
		http.StatusCreated,
	)
}

func ingestFileHandler(service IngestionService, options IngestionRouteOptions) http.HandlerFunc {
	resolvedOptions := resolveIngestionOptions(options)
	return func(writer http.ResponseWriter, request *http.Request) {
		actorUserID, _, ok := requireProjectActor(writer, request)
		if !ok {
			return
		}
		payload, upload, err := parseIngestionFilePayload(request, resolvedOptions)
		if err != nil {
			writeIngestionRouteError(writer, err)
			return
		}
		result, err := service.IngestFile(
			request.Context(),
			IngestionFileRouteRequest{
				ActorUserID: actorUserID,
				Payload:     payload,
				Upload:      upload,
			},
		)
		if err != nil {
			writeIngestionRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusCreated, result)
	}
}

func listDocumentsHandler(service IngestionService) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		actorUserID, _, ok := requireProjectActor(writer, request)
		if !ok {
			return
		}
		limit, err := parseBoundedIntQueryParam(
			request.URL.Query().Get("limit"),
			defaultDocumentListLimit,
			minDocumentListLimit,
			maxDocumentListLimit,
		)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid query parameters"})
			return
		}
		offset, err := parseBoundedIntQueryParam(
			request.URL.Query().Get("offset"),
			defaultDocumentListOffset,
			defaultDocumentListOffset,
			int(^uint(0)>>1),
		)
		if err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"detail": "Invalid query parameters"})
			return
		}
		records, err := service.ListDocuments(
			request.Context(),
			IngestionListDocumentsRouteRequest{
				ActorUserID: actorUserID,
				ProjectID:   parseOptionalProjectID(request.URL.Query().Get("project_id")),
				Limit:       limit,
				Offset:      offset,
			},
		)
		if err != nil {
			writeIngestionRouteError(writer, err)
			return
		}
		writeJSON(writer, http.StatusOK, records)
	}
}

func ingestionQueryRouteHandler[Payload any, Request any, Result any](
	serviceOperation func(ctx context.Context, request Request) (Result, error),
	decodePayload func(request *http.Request) (Payload, error),
	buildRequest func(actorUserID uuid.UUID, payload Payload) Request,
) http.HandlerFunc {
	return ingestionTypedRouteHandler(
		serviceOperation,
		buildRequest,
		decodePayload,
		http.StatusOK,
	)
}

// MountIngestionRoutes registers ingestion REST routes.
func MountIngestionRoutes(router chi.Router, service IngestionService, options IngestionRouteOptions) {
	if service == nil {
		return
	}
	router.Post("/api/v1/ingestion/text", ingestTextHandler(service))
	router.Post("/api/v1/ingestion/file", ingestFileHandler(service, options))
	router.Get("/api/v1/ingestion/documents", listDocumentsHandler(service))
	router.Post(
		"/api/v1/ingestion/query",
		ingestionQueryRouteHandler(
			service.QueryDocumentChunks,
			decodeDocumentChunkQueryPayload,
			func(
				actorUserID uuid.UUID,
				payload models.DocumentChunkQueryRequest,
			) IngestionDocumentQueryRouteRequest {
				return IngestionDocumentQueryRouteRequest{
					ActorUserID: actorUserID,
					Payload:     payload,
				}
			},
		),
	)
	router.Post(
		"/api/v1/ingestion/query/blended",
		ingestionQueryRouteHandler(
			service.QueryBlended,
			decodeBlendedQueryPayload,
			func(
				actorUserID uuid.UUID,
				payload IngestionBlendedQueryPayload,
			) IngestionBlendedQueryRouteRequest {
				return IngestionBlendedQueryRouteRequest{
					ActorUserID: actorUserID,
					Payload:     payload,
				}
			},
		),
	)
}

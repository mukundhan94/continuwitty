package api

import (
	"context"
	"errors"
	"net/http"
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

// IngestionTextRouteRequest captures actor-scoped text-ingestion route input.
type IngestionTextRouteRequest struct {
	ActorUserID uuid.UUID
	Payload     IngestionTextPayload
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
func MountIngestionRoutes(router chi.Router, service IngestionService) {
	if service == nil {
		return
	}
	router.Post("/api/v1/ingestion/text", ingestTextHandler(service))
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

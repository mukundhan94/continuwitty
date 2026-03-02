package ingestion

import (
	"context"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

const (
	defaultTextMimeType  = "text/plain"
	defaultUntitledTitle = "Untitled Document"
)

// FileIngestRequest captures uploaded file properties used by ingestion.
type FileIngestRequest struct {
	Filename     string
	MimeType     *string
	ContentBytes []byte
}

// DocumentIngestTextRequest captures text-ingestion payload values.
type DocumentIngestTextRequest struct {
	ProjectID         string
	Title             string
	Text              string
	VisibilityScope   models.VisibilityScope
	ChunkSizeChars    int
	ChunkOverlapChars int
	Metadata          map[string]any
}

// DocumentIngestFileRequest captures file-ingestion payload values.
type DocumentIngestFileRequest struct {
	ProjectID         string
	Title             *string
	VisibilityScope   models.VisibilityScope
	ChunkSizeChars    int
	ChunkOverlapChars int
	Metadata          map[string]any
}

func (request DocumentIngestTextRequest) chunkShape() chunkShape {
	return chunkShape{
		sizeChars:    request.ChunkSizeChars,
		overlapChars: request.ChunkOverlapChars,
	}
}

func (request DocumentIngestFileRequest) chunkShape() chunkShape {
	return chunkShape{
		sizeChars:    request.ChunkSizeChars,
		overlapChars: request.ChunkOverlapChars,
	}
}

// BlendedRetrievalQueryRequest captures blended retrieval query values.
type BlendedRetrievalQueryRequest struct {
	Query              string
	ProjectID          *string
	TopKEngrams        int
	TopKDocumentChunks int
}

// BlendedRetrievalQueryResponse captures blended retrieval results.
type BlendedRetrievalQueryResponse struct {
	Engrams        []models.EngramQueryResult        `json:"engrams"`
	DocumentChunks []models.DocumentChunkQueryResult `json:"document_chunks"`
}

// DocumentIngestResponse wraps persisted document output for ingestion writes.
type DocumentIngestResponse struct {
	Document models.DocumentRecord `json:"document"`
}

// ListDocumentsRequest captures list-document retrieval filters.
type ListDocumentsRequest struct {
	ActorUserID uuid.UUID
	ProjectID   *string
	Limit       int
	Offset      int
}

type chunkBuildRequest struct {
	actorUserID       uuid.UUID
	projectID         string
	title             string
	normalizedText    string
	chunkSizeChars    int
	chunkOverlapChars int
	emptyChunksDetail string
}

type chunkShape struct {
	sizeChars    int
	overlapChars int
}

type fileTitleInput struct {
	requestedTitle *string
	filename       string
}

type fileMetadataDetails struct {
	filename string
	mimeType *string
	byteSize int
}

type serviceDeps struct {
	ensureProjectExists      func(ctx context.Context, db repository.Queryer, input repository.ProjectEnsureInput) (*models.ProjectRecord, error)
	upsertDocumentWithChunks func(ctx context.Context, db repository.Queryer, input repository.DocumentUpsertInput) (*models.DocumentRecord, error)
	listDocuments            func(ctx context.Context, db repository.Queryer, input repository.DocumentListInput) ([]models.DocumentRecord, error)
	queryDocumentChunks      func(ctx context.Context, db repository.Queryer, input repository.DocumentChunkQueryInput) ([]models.DocumentChunkQueryResult, error)
	queryEngrams             func(ctx context.Context, db repository.Queryer, input repository.QueryEngramsInput) ([]models.EngramQueryResult, error)
	buildQueryLiteral        func(query string, embeddingDim int) (string, error)
}

func defaultServiceDeps() serviceDeps {
	return serviceDeps{
		ensureProjectExists:      repository.EnsureProjectExists,
		upsertDocumentWithChunks: repository.UpsertDocumentWithChunks,
		listDocuments:            repository.ListDocuments,
		queryDocumentChunks:      repository.QueryDocumentChunks,
		queryEngrams:             repository.QueryEngrams,
		buildQueryLiteral:        repository.BuildLocalQueryLiteral,
	}
}

// Service coordinates document ingestion, chunking, and blended retrieval responses.
type Service struct {
	db           repository.Queryer
	embeddingDim int
	maxFileBytes int
	maxTextChars int
	deps         serviceDeps
}

var allowedFileMimeTypes = map[string]struct{}{
	"application/json": {},
	"application/xml":  {},
	"text/csv":         {},
	"text/markdown":    {},
	"text/plain":       {},
	"text/xml":         {},
}

// NewService builds a document ingestion service.
func NewService(db repository.Queryer, embeddingDim int, maxFileBytes int, maxTextChars int) *Service {
	return &Service{
		db:           db,
		embeddingDim: embeddingDim,
		maxFileBytes: maxFileBytes,
		maxTextChars: maxTextChars,
		deps:         defaultServiceDeps(),
	}
}

// IngestText chunks and persists raw text as a document.
func (service *Service) IngestText(
	ctx context.Context,
	actorUserID uuid.UUID,
	payload DocumentIngestTextRequest,
) (DocumentIngestResponse, error) {
	if err := validateChunkShape(payload.chunkShape()); err != nil {
		return DocumentIngestResponse{}, err
	}
	normalizedText := NormalizeDocumentText(payload.Text)
	if normalizedText == "" {
		return DocumentIngestResponse{}, newDefaultServiceError("Document text cannot be empty")
	}
	if err := service.validateTextSize(normalizedText); err != nil {
		return DocumentIngestResponse{}, err
	}

	contentHash, documentID, chunks, err := service.buildChunksForDocument(
		ctx,
		chunkBuildRequest{
			actorUserID:       actorUserID,
			projectID:         payload.ProjectID,
			title:             payload.Title,
			normalizedText:    normalizedText,
			chunkSizeChars:    payload.ChunkSizeChars,
			chunkOverlapChars: payload.ChunkOverlapChars,
			emptyChunksDetail: "No chunks were produced from document text",
		},
	)
	if err != nil {
		return DocumentIngestResponse{}, err
	}

	document, err := service.deps.upsertDocumentWithChunks(
		ctx,
		service.db,
		repository.DocumentUpsertInput{
			ActorUserID:  actorUserID,
			EmbeddingDim: service.embeddingDim,
			Payload: repository.DocumentUpsertPayload{
				DocumentID:      documentID,
				ProjectID:       payload.ProjectID,
				Title:           strings.TrimSpace(payload.Title),
				SourceType:      models.DocumentSourceTypeText,
				SourceName:      nil,
				MimeType:        stringRef(defaultTextMimeType),
				VisibilityScope: normalizeVisibilityScope(payload.VisibilityScope),
				ContentText:     normalizedText,
				ContentHash:     contentHash,
				Metadata:        cloneMetadata(payload.Metadata),
				Chunks:          toRepositoryChunks(chunks),
			},
		},
	)
	if err != nil {
		return DocumentIngestResponse{}, err
	}
	return DocumentIngestResponse{Document: *document}, nil
}

// IngestFile chunks and persists file contents as a document.
func (service *Service) IngestFile(
	ctx context.Context,
	actorUserID uuid.UUID,
	payload DocumentIngestFileRequest,
	fileRequest FileIngestRequest,
) (DocumentIngestResponse, error) {
	if err := validateChunkShape(payload.chunkShape()); err != nil {
		return DocumentIngestResponse{}, err
	}
	normalizedText, err := service.validateFileInput(fileRequest)
	if err != nil {
		return DocumentIngestResponse{}, err
	}
	resolvedTitle := resolveFileTitle(fileTitleInput{
		requestedTitle: payload.Title,
		filename:       fileRequest.Filename,
	})
	contentHash, documentID, chunks, err := service.buildChunksForDocument(
		ctx,
		chunkBuildRequest{
			actorUserID:       actorUserID,
			projectID:         payload.ProjectID,
			title:             resolvedTitle,
			normalizedText:    normalizedText,
			chunkSizeChars:    payload.ChunkSizeChars,
			chunkOverlapChars: payload.ChunkOverlapChars,
			emptyChunksDetail: "No chunks were produced from file contents",
		},
	)
	if err != nil {
		return DocumentIngestResponse{}, err
	}

	document, err := service.deps.upsertDocumentWithChunks(
		ctx,
		service.db,
		repository.DocumentUpsertInput{
			ActorUserID:  actorUserID,
			EmbeddingDim: service.embeddingDim,
			Payload: repository.DocumentUpsertPayload{
				DocumentID:      documentID,
				ProjectID:       payload.ProjectID,
				Title:           resolvedTitle,
				SourceType:      models.DocumentSourceTypeFile,
				SourceName:      stringRef(fileRequest.Filename),
				MimeType:        fileRequest.MimeType,
				VisibilityScope: normalizeVisibilityScope(payload.VisibilityScope),
				ContentText:     normalizedText,
				ContentHash:     contentHash,
				Metadata: mergeFileMetadata(payload.Metadata, fileMetadataDetails{
					filename: fileRequest.Filename,
					mimeType: fileRequest.MimeType,
					byteSize: len(fileRequest.ContentBytes),
				}),
				Chunks: toRepositoryChunks(chunks),
			},
		},
	)
	if err != nil {
		return DocumentIngestResponse{}, err
	}
	return DocumentIngestResponse{Document: *document}, nil
}

// ListDocuments proxies list-document retrieval with actor visibility constraints.
func (service *Service) ListDocuments(
	ctx context.Context,
	request ListDocumentsRequest,
) ([]models.DocumentRecord, error) {
	return service.deps.listDocuments(
		ctx,
		service.db,
		repository.DocumentListInput{
			ActorUserID: request.ActorUserID,
			ProjectID:   request.ProjectID,
			Limit:       request.Limit,
			Offset:      request.Offset,
		},
	)
}

// QueryDocumentChunks proxies document chunk retrieval with dense + lexical ranking.
func (service *Service) QueryDocumentChunks(
	ctx context.Context,
	actorUserID uuid.UUID,
	payload models.DocumentChunkQueryRequest,
) ([]models.DocumentChunkQueryResult, error) {
	return service.deps.queryDocumentChunks(
		ctx,
		service.db,
		repository.DocumentChunkQueryInput{
			ActorUserID:  actorUserID,
			Request:      payload,
			EmbeddingDim: service.embeddingDim,
		},
	)
}

// QueryBlended combines engram and document-chunk retrieval for a shared query.
func (service *Service) QueryBlended(
	ctx context.Context,
	actorUserID uuid.UUID,
	payload BlendedRetrievalQueryRequest,
) (BlendedRetrievalQueryResponse, error) {
	queryLiteral, err := service.deps.buildQueryLiteral(payload.Query, service.embeddingDim)
	if err != nil {
		return BlendedRetrievalQueryResponse{}, err
	}
	engrams, err := service.deps.queryEngrams(
		ctx,
		service.db,
		repository.QueryEngramsInput{
			Request: models.EngramQueryRequest{
				Query:     payload.Query,
				ProjectID: payload.ProjectID,
				TopK:      payload.TopKEngrams,
			},
			QueryLiteral: queryLiteral,
			ActorUserID:  &actorUserID,
		},
	)
	if err != nil {
		return BlendedRetrievalQueryResponse{}, err
	}
	documentChunks, err := service.deps.queryDocumentChunks(
		ctx,
		service.db,
		repository.DocumentChunkQueryInput{
			ActorUserID: actorUserID,
			Request: models.DocumentChunkQueryRequest{
				Query:       payload.Query,
				ProjectID:   payload.ProjectID,
				DocumentIDs: []uuid.UUID{},
				TopK:        payload.TopKDocumentChunks,
			},
			EmbeddingDim: service.embeddingDim,
		},
	)
	if err != nil {
		return BlendedRetrievalQueryResponse{}, err
	}
	return BlendedRetrievalQueryResponse{Engrams: engrams, DocumentChunks: documentChunks}, nil
}

func validateChunkShape(shape chunkShape) error {
	if shape.overlapChars >= shape.sizeChars {
		return invalidChunkShapeError()
	}
	return nil
}

func (service *Service) validateTextSize(text string) error {
	if len([]rune(text)) > service.maxTextChars {
		return oversizedTextError(service.maxTextChars)
	}
	return nil
}

func (service *Service) validateFileInput(request FileIngestRequest) (string, error) {
	if err := validateFileMimeType(request.MimeType); err != nil {
		return "", err
	}
	if err := service.validateFileSize(request.ContentBytes); err != nil {
		return "", err
	}
	decodedText, err := decodeFileText(request.ContentBytes)
	if err != nil {
		return "", err
	}
	normalized := NormalizeDocumentText(decodedText)
	if normalized == "" {
		return "", newDefaultServiceError("Uploaded file does not contain readable text")
	}
	if err := service.validateTextSize(normalized); err != nil {
		return "", err
	}
	return normalized, nil
}

func (service *Service) validateFileSize(contentBytes []byte) error {
	if len(contentBytes) == 0 {
		return newDefaultServiceError("Uploaded file is empty")
	}
	if len(contentBytes) > service.maxFileBytes {
		return oversizedFileError(service.maxFileBytes)
	}
	return nil
}

func decodeFileText(contentBytes []byte) (string, error) {
	if !utf8.Valid(contentBytes) {
		return "", newServiceError("Only UTF-8 text files are supported in this phase", 415)
	}
	return string(contentBytes), nil
}

func validateFileMimeType(mimeType *string) error {
	normalized := normalizeOptionalString(mimeType)
	if normalized == "" {
		return nil
	}
	if strings.HasPrefix(normalized, "text/") {
		return nil
	}
	if _, ok := allowedFileMimeTypes[normalized]; ok {
		return nil
	}
	return newServiceError("Unsupported file type. Upload a UTF-8 text-based document.", 415)
}

func normalizeOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(*value))
}

func resolveFileTitle(input fileTitleInput) string {
	normalizedRequestedTitle := normalizeOptionalRequestedTitle(input.requestedTitle)
	if normalizedRequestedTitle != "" {
		return normalizedRequestedTitle
	}
	filenameStem := extractFilenameStem(input.filename)
	if filenameStem != "" {
		return filenameStem
	}
	return defaultUntitledTitle
}

func normalizeOptionalRequestedTitle(requestedTitle *string) string {
	if requestedTitle == nil {
		return ""
	}
	return strings.TrimSpace(*requestedTitle)
}

func extractFilenameStem(filename string) string {
	trimmedFilename := strings.TrimSpace(filename)
	if trimmedFilename == "" {
		return ""
	}
	baseFilename := filepath.Base(trimmedFilename)
	filenameExtension := filepath.Ext(baseFilename)
	filenameStem := strings.TrimSuffix(baseFilename, filenameExtension)
	return strings.TrimSpace(filenameStem)
}

func mergeFileMetadata(metadata map[string]any, details fileMetadataDetails) map[string]any {
	merged := cloneMetadata(metadata)
	merged["filename"] = details.filename
	if details.mimeType != nil {
		merged["mime_type"] = *details.mimeType
	} else {
		merged["mime_type"] = nil
	}
	merged["byte_size"] = details.byteSize
	return merged
}

func cloneMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func (service *Service) buildChunksForDocument(
	ctx context.Context,
	request chunkBuildRequest,
) (string, uuid.UUID, []ChunkDraft, error) {
	contentHash := BuildContentHash(request.normalizedText)
	_, err := service.deps.ensureProjectExists(
		ctx,
		service.db,
		repository.ProjectEnsureInput{
			ProjectID:   request.projectID,
			OwnerUserID: request.actorUserID,
		},
	)
	if err != nil {
		return "", uuid.Nil, nil, err
	}
	documentID := BuildDocumentID(request.actorUserID, request.projectID, request.title, contentHash)
	chunks := ChunkDocumentText(ChunkDocumentInput{
		ContentHash:       contentHash,
		Text:              request.normalizedText,
		ChunkSizeChars:    request.chunkSizeChars,
		ChunkOverlapChars: request.chunkOverlapChars,
	})
	if len(chunks) == 0 {
		return "", uuid.Nil, nil, newDefaultServiceError(request.emptyChunksDetail)
	}
	return contentHash, documentID, chunks, nil
}

func toRepositoryChunks(chunks []ChunkDraft) []repository.DocumentChunkDraft {
	repositoryChunks := make([]repository.DocumentChunkDraft, 0, len(chunks))
	for _, chunk := range chunks {
		repositoryChunks = append(repositoryChunks, repository.DocumentChunkDraft{
			ChunkID:       chunk.ChunkID,
			ChunkIndex:    chunk.ChunkIndex,
			ChunkText:     chunk.ChunkText,
			Snippet:       chunk.Snippet,
			CharStart:     chunk.CharStart,
			CharEnd:       chunk.CharEnd,
			TokenEstimate: chunk.TokenEstimate,
			Metadata:      chunk.Metadata,
		})
	}
	return repositoryChunks
}

func normalizeVisibilityScope(scope models.VisibilityScope) models.VisibilityScope {
	if scope == "" {
		return models.VisibilityScopePrivate
	}
	return scope
}

func stringRef(value string) *string {
	return &value
}

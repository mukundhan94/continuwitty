package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"engram/internal/embeddings"
	"engram/internal/models"

	"github.com/google/uuid"
)

const upsertDocumentSQL = `
INSERT INTO documents (
	document_id,
	owner_user_id,
	project_id,
	title,
	source_type,
	source_name,
	mime_type,
	visibility_scope,
	content_text,
	content_hash,
	metadata,
	chunk_count,
	created_at,
	updated_at
)
VALUES (
	$1,
	$2,
	$3,
	$4,
	$5,
	$6,
	$7,
	$8,
	$9,
	$10,
	$11::jsonb,
	$12,
	$13,
	$14
)
ON CONFLICT (document_id)
DO UPDATE SET
	title = EXCLUDED.title,
	source_type = EXCLUDED.source_type,
	source_name = EXCLUDED.source_name,
	mime_type = EXCLUDED.mime_type,
	visibility_scope = EXCLUDED.visibility_scope,
	content_text = EXCLUDED.content_text,
	content_hash = EXCLUDED.content_hash,
	metadata = EXCLUDED.metadata,
	chunk_count = EXCLUDED.chunk_count,
	updated_at = EXCLUDED.updated_at
RETURNING
	document_id,
	owner_user_id,
	project_id,
	title,
	source_type,
	source_name,
	mime_type,
	visibility_scope,
	content_hash,
	chunk_count,
	created_at,
	updated_at
`

const insertDocumentChunkSQL = `
INSERT INTO document_chunks (
	chunk_id,
	document_id,
	chunk_index,
	chunk_text,
	snippet,
	char_start,
	char_end,
	token_estimate,
	metadata,
	embedding_model,
	embed,
	created_at
)
VALUES (
	$1,
	$2,
	$3,
	$4,
	$5,
	$6,
	$7,
	$8,
	$9::jsonb,
	$10,
	$11::vector,
	$12
)
RETURNING chunk_id
`

const queryDocumentChunksSelectTemplate = `
SELECT
	dc.chunk_id,
	dc.document_id,
	d.project_id,
	d.title,
	d.source_name,
	dc.chunk_index,
	dc.snippet,
	dc.created_at,
	d.visibility_scope,
	dc.chunk_text,
	dc.embed <=> $1::vector AS distance
FROM document_chunks dc
JOIN documents d ON d.document_id = dc.document_id
WHERE %s
ORDER BY distance ASC, d.created_at DESC, dc.chunk_index ASC
LIMIT %s
`

var (
	nowDocumentUTC    = func() time.Time { return time.Now().UTC() }
	embedDocumentText = func(text string, dim int) (embeddings.Result, error) {
		return embeddings.EmbedText(text, dim)
	}
	embedDocumentMany = func(texts []string, dim int) ([]embeddings.Result, error) {
		return embeddings.EmbedTexts(texts, dim)
	}
)

// DocumentChunkDraft models chunk data persisted for retrieval.
type DocumentChunkDraft struct {
	ChunkID       uuid.UUID
	ChunkIndex    int
	ChunkText     string
	Snippet       string
	CharStart     int
	CharEnd       int
	TokenEstimate int
	Metadata      map[string]any
}

// DocumentUpsertPayload captures document metadata and replacement chunks.
type DocumentUpsertPayload struct {
	DocumentID      uuid.UUID
	ProjectID       string
	Title           string
	SourceType      models.DocumentSourceType
	SourceName      *string
	MimeType        *string
	VisibilityScope models.VisibilityScope
	ContentText     string
	ContentHash     string
	Metadata        map[string]any
	Chunks          []DocumentChunkDraft
}

// DocumentUpsertInput captures document upsert dependencies.
type DocumentUpsertInput struct {
	ActorUserID   uuid.UUID
	Payload       DocumentUpsertPayload
	EmbeddingDim  int
	CreatedAtHint *time.Time
}

// DocumentListInput captures list-documents filters.
type DocumentListInput struct {
	ActorUserID uuid.UUID
	ProjectID   *string
	Limit       int
	Offset      int
}

// DocumentChunkQueryInput captures vector query dependencies.
type DocumentChunkQueryInput struct {
	ActorUserID  uuid.UUID
	Request      models.DocumentChunkQueryRequest
	EmbeddingDim int
}

type documentChunkReplaceInput struct {
	Payload         DocumentUpsertPayload
	Timestamp       time.Time
	ChunkEmbeddings []embeddings.Result
}

type documentChunkInsertInput struct {
	DocumentID uuid.UUID
	Chunk      DocumentChunkDraft
	Embedding  embeddings.Result
	Timestamp  time.Time
}

// UpsertDocumentWithChunks persists document metadata and atomically replaces chunks.
func UpsertDocumentWithChunks(
	ctx context.Context,
	db Queryer,
	input DocumentUpsertInput,
) (*models.DocumentRecord, error) {
	if _, err := models.ParseDocumentSourceType(string(input.Payload.SourceType)); err != nil {
		return nil, err
	}
	if _, err := models.ParseVisibilityScope(string(input.Payload.VisibilityScope)); err != nil {
		return nil, err
	}

	now := nowDocumentUTC()
	chunkEmbeddings, err := embedDocumentMany(chunkTexts(input.Payload.Chunks), input.EmbeddingDim)
	if err != nil {
		return nil, err
	}
	if len(chunkEmbeddings) != len(input.Payload.Chunks) {
		return nil, fmt.Errorf("chunk embedding count mismatch: expected %d got %d", len(input.Payload.Chunks), len(chunkEmbeddings))
	}

	metadataJSON, err := marshalJSON(orEmptyMap(input.Payload.Metadata))
	if err != nil {
		return nil, err
	}
	row := db.QueryRow(
		ctx,
		upsertDocumentSQL,
		input.Payload.DocumentID,
		input.ActorUserID,
		input.Payload.ProjectID,
		input.Payload.Title,
		string(input.Payload.SourceType),
		input.Payload.SourceName,
		input.Payload.MimeType,
		string(input.Payload.VisibilityScope),
		input.Payload.ContentText,
		input.Payload.ContentHash,
		metadataJSON,
		len(input.Payload.Chunks),
		now,
		now,
	)
	record, err := scanDocumentRecord(row)
	if err != nil {
		return nil, err
	}

	replaceInput := documentChunkReplaceInput{
		Payload:         input.Payload,
		Timestamp:       now,
		ChunkEmbeddings: chunkEmbeddings,
	}
	if err := replaceDocumentChunks(ctx, db, replaceInput); err != nil {
		return nil, err
	}
	return &record, nil
}

// ListDocuments returns visible documents for an actor.
func ListDocuments(
	ctx context.Context,
	db Queryer,
	input DocumentListInput,
) ([]models.DocumentRecord, error) {
	accessClause := buildMembershipReadClause(
		"owner_user_id",
		"visibility_scope",
		"project_id",
		"$1",
		false,
	)
	query := fmt.Sprintf(`
		SELECT
			document_id,
			owner_user_id,
			project_id,
			title,
			source_type,
			source_name,
			mime_type,
			visibility_scope,
			content_hash,
			chunk_count,
			created_at,
			updated_at
		FROM documents
		WHERE %s
	`, accessClause)
	params := []any{input.ActorUserID}
	if input.ProjectID != nil && *input.ProjectID != "" {
		query += " AND project_id = $2"
		params = append(params, *input.ProjectID)
	}
	query += fmt.Sprintf(
		" ORDER BY created_at DESC LIMIT %s OFFSET %s",
		pgxPlaceholder(len(params)+1),
		pgxPlaceholder(len(params)+2),
	)
	params = append(params, input.Limit, input.Offset)

	rows, err := db.Query(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.DocumentRecord, 0)
	for rows.Next() {
		record, scanErr := scanDocumentRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

// QueryDocumentChunks retrieves and reranks visible chunks by dense + lexical score.
func QueryDocumentChunks(
	ctx context.Context,
	db Queryer,
	input DocumentChunkQueryInput,
) ([]models.DocumentChunkQueryResult, error) {
	topK := normalizeDocumentChunkTopK(input.Request.TopK)

	queryEmbedding, err := embedDocumentText(input.Request.Query, input.EmbeddingDim)
	if err != nil {
		return nil, err
	}
	querySQL, params := buildDocumentChunkQuerySQLAndParams(
		input.ActorUserID,
		input.Request,
		queryEmbedding.Vector,
		topK,
	)
	candidates, err := queryDocumentChunkCandidates(ctx, db, querySQL, params)
	if err != nil {
		return nil, err
	}

	reranked := rerankDocumentChunkRows(candidates, input.Request.Query, topK)
	return buildDocumentChunkQueryResults(reranked), nil
}

func replaceDocumentChunks(
	ctx context.Context,
	db Queryer,
	input documentChunkReplaceInput,
) error {
	if err := deleteDocumentChunks(ctx, db, input.Payload.DocumentID); err != nil {
		return err
	}

	for index, chunk := range input.Payload.Chunks {
		if err := insertDocumentChunk(
			ctx,
			db,
			documentChunkInsertInput{
				DocumentID: input.Payload.DocumentID,
				Chunk:      chunk,
				Embedding:  input.ChunkEmbeddings[index],
				Timestamp:  input.Timestamp,
			},
		); err != nil {
			return err
		}
	}
	return nil
}

func normalizeDocumentChunkTopK(requested int) int {
	if requested <= 0 {
		return 6
	}
	return requested
}

func buildDocumentChunkQuerySQLAndParams(
	actorUserID uuid.UUID,
	request models.DocumentChunkQueryRequest,
	queryVector []float64,
	topK int,
) (string, []any) {
	whereSQL, whereParams := buildDocumentChunkWhere(actorUserID, request, 2)
	limitPlaceholder := pgxPlaceholder(len(whereParams) + 2)
	querySQL := fmt.Sprintf(queryDocumentChunksSelectTemplate, whereSQL, limitPlaceholder)

	params := make([]any, 0, len(whereParams)+2)
	params = append(params, vectorLiteral(queryVector))
	params = append(params, whereParams...)
	params = append(params, documentChunkCandidateLimit(topK))
	return querySQL, params
}

func documentChunkCandidateLimit(topK int) int {
	candidateLimit := topK * 4
	if candidateLimit < topK {
		candidateLimit = topK
	}
	if candidateLimit > 200 {
		return 200
	}
	return candidateLimit
}

func queryDocumentChunkCandidates(
	ctx context.Context,
	db Queryer,
	querySQL string,
	params []any,
) ([]documentChunkCandidate, error) {
	rows, err := db.Query(ctx, querySQL, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]documentChunkCandidate, 0)
	for rows.Next() {
		candidate, scanErr := scanDocumentChunkCandidate(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

func buildDocumentChunkQueryResults(rows []documentChunkCandidate) []models.DocumentChunkQueryResult {
	results := make([]models.DocumentChunkQueryResult, 0, len(rows))
	for _, row := range rows {
		results = append(results, models.DocumentChunkQueryResult{
			ChunkID:         row.ChunkID,
			DocumentID:      row.DocumentID,
			ProjectID:       row.ProjectID,
			Title:           row.Title,
			SourceName:      row.SourceName,
			ChunkIndex:      row.ChunkIndex,
			Snippet:         row.Snippet,
			CreatedAt:       row.CreatedAt,
			VisibilityScope: row.VisibilityScope,
			Distance:        row.Distance,
		})
	}
	return results
}

func deleteDocumentChunks(ctx context.Context, db Queryer, documentID uuid.UUID) error {
	rows, err := db.Query(ctx, "DELETE FROM document_chunks WHERE document_id = $1", documentID)
	if err != nil {
		return err
	}
	rows.Close()
	return rows.Err()
}

func insertDocumentChunk(
	ctx context.Context,
	db Queryer,
	input documentChunkInsertInput,
) error {
	metadataJSON, err := marshalJSON(orEmptyMap(input.Chunk.Metadata))
	if err != nil {
		return err
	}
	var chunkID uuid.UUID
	row := db.QueryRow(
		ctx,
		insertDocumentChunkSQL,
		input.Chunk.ChunkID,
		input.DocumentID,
		input.Chunk.ChunkIndex,
		input.Chunk.ChunkText,
		input.Chunk.Snippet,
		input.Chunk.CharStart,
		input.Chunk.CharEnd,
		input.Chunk.TokenEstimate,
		metadataJSON,
		input.Embedding.ProviderID,
		vectorLiteral(input.Embedding.Vector),
		input.Timestamp,
	)
	return row.Scan(&chunkID)
}

func scanDocumentRecord(row interface {
	Scan(dest ...any) error
}) (models.DocumentRecord, error) {
	var (
		record          models.DocumentRecord
		sourceType      string
		visibilityScope string
	)
	err := row.Scan(
		&record.DocumentID,
		&record.OwnerUserID,
		&record.ProjectID,
		&record.Title,
		&sourceType,
		&record.SourceName,
		&record.MimeType,
		&visibilityScope,
		&record.ContentHash,
		&record.ChunkCount,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return models.DocumentRecord{}, err
	}
	parsedSourceType, err := models.ParseDocumentSourceType(strings.TrimSpace(sourceType))
	if err != nil {
		return models.DocumentRecord{}, err
	}
	parsedVisibility, err := models.ParseVisibilityScope(strings.TrimSpace(visibilityScope))
	if err != nil {
		return models.DocumentRecord{}, err
	}
	record.SourceType = parsedSourceType
	record.VisibilityScope = parsedVisibility
	return record, nil
}

func buildDocumentChunkWhere(
	actorUserID uuid.UUID,
	request models.DocumentChunkQueryRequest,
	startPlaceholder int,
) (string, []any) {
	if startPlaceholder <= 0 {
		startPlaceholder = 1
	}
	whereParams := make([]any, 0, 3)
	nextPlaceholder := func() string {
		return pgxPlaceholder(startPlaceholder + len(whereParams))
	}
	actorPlaceholder := nextPlaceholder()
	whereClauses := []string{
		buildMembershipReadClause(
			"d.owner_user_id",
			"d.visibility_scope",
			"d.project_id",
			actorPlaceholder,
			false,
		),
	}
	whereParams = append(whereParams, actorUserID)
	if request.ProjectID != nil && *request.ProjectID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("d.project_id = %s", nextPlaceholder()))
		whereParams = append(whereParams, *request.ProjectID)
	}
	if len(request.DocumentIDs) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("d.document_id = ANY(%s::uuid[])", nextPlaceholder()))
		whereParams = append(whereParams, request.DocumentIDs)
	}
	return strings.Join(whereClauses, " AND "), whereParams
}

func rerankDocumentChunkRows(
	rows []documentChunkCandidate,
	query string,
	topK int,
) []documentChunkCandidate {
	type rankedChunk struct {
		score float64
		row   documentChunkCandidate
	}
	ranked := make([]rankedChunk, 0, len(rows))
	for _, row := range rows {
		lexical := lexicalOverlapScore(query, []string{row.Title, row.Snippet, row.ChunkText})
		ranked = append(ranked, rankedChunk{
			score: combinedRankScore(row.Distance, lexical),
			row:   row,
		})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})
	if topK > len(ranked) {
		topK = len(ranked)
	}
	results := make([]documentChunkCandidate, 0, topK)
	for _, item := range ranked[:topK] {
		results = append(results, item.row)
	}
	return results
}

func scanDocumentChunkCandidate(row interface {
	Scan(dest ...any) error
}) (documentChunkCandidate, error) {
	var (
		candidate       documentChunkCandidate
		snippet         *string
		visibilityScope string
	)
	err := row.Scan(
		&candidate.ChunkID,
		&candidate.DocumentID,
		&candidate.ProjectID,
		&candidate.Title,
		&candidate.SourceName,
		&candidate.ChunkIndex,
		&snippet,
		&candidate.CreatedAt,
		&visibilityScope,
		&candidate.ChunkText,
		&candidate.Distance,
	)
	if err != nil {
		return documentChunkCandidate{}, err
	}
	parsedVisibility, err := models.ParseVisibilityScope(strings.TrimSpace(visibilityScope))
	if err != nil {
		return documentChunkCandidate{}, err
	}
	candidate.Snippet = ""
	if snippet != nil {
		candidate.Snippet = *snippet
	}
	candidate.VisibilityScope = parsedVisibility
	return candidate, nil
}

func chunkTexts(chunks []DocumentChunkDraft) []string {
	results := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		results = append(results, chunk.ChunkText)
	}
	return results
}

func orEmptyMap(values map[string]any) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	return values
}

type documentChunkCandidate struct {
	ChunkID         uuid.UUID
	DocumentID      uuid.UUID
	ProjectID       string
	Title           string
	SourceName      *string
	ChunkIndex      int
	Snippet         string
	CreatedAt       time.Time
	VisibilityScope models.VisibilityScope
	ChunkText       string
	Distance        float64
}

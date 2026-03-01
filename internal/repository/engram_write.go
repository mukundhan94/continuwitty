package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"engram/internal/embeddings"
	"engram/internal/models"

	"github.com/google/uuid"
)

// CreateEngramInput captures write-path dependencies for engram creation.
type CreateEngramInput struct {
	Payload          models.MemoryEngramCreate
	EmbeddingDim     int
	OwnerUserID      *uuid.UUID
	EnrichmentOrigin string
}

type engramInsertRequest struct {
	EngramID         uuid.UUID
	Payload          models.MemoryEngramCreate
	CreatedAt        time.Time
	OwnerUserID      *uuid.UUID
	RetrievalText    string
	EmbeddingModel   string
	EmbeddingLiteral string
	EngramJSON       map[string]any
}

var (
	newEngramUUID   = uuid.New
	newWriteUUID    = uuid.New
	nowUTC          = func() time.Time { return time.Now().UTC() }
	embedEngramText = func(text string, dim int) (embeddings.Result, error) {
		return embeddings.EmbedText(text, dim)
	}
	resolveEnrichedPayload = func(
		payload models.MemoryEngramCreate,
		enrichmentOrigin string,
	) (models.MemoryEngramCreate, map[string]any) {
		return enrichEngramPayloadIfMissing(payload, enrichmentOrigin)
	}
)

// CreateEngramWithReport persists a new engram and returns enrichment metadata.
func CreateEngramWithReport(
	ctx context.Context,
	db Queryer,
	input CreateEngramInput,
) (*models.EngramCreateResponse, map[string]any, error) {
	resolvedPayload, enrichmentReport := resolveEnrichedPayload(input.Payload, input.EnrichmentOrigin)
	resolvedPayload.Tags = normalizeEngramStringSlice(resolvedPayload.Tags)
	resolvedPayload.Keywords = normalizeEngramStringSlice(resolvedPayload.Keywords)
	engramID := newEngramUUID()
	createdAt := nowUTC()
	retrievalText := buildRetrievalText(resolvedPayload)
	embeddingResult, err := embedEngramText(retrievalText, input.EmbeddingDim)
	if err != nil {
		return nil, nil, err
	}
	engramJSON := buildEngramJSONPayload(resolvedPayload, enrichmentReport, createdAt)

	if err := insertEngramRow(
		ctx,
		db,
		engramInsertRequest{
			EngramID:         engramID,
			Payload:          resolvedPayload,
			CreatedAt:        createdAt,
			OwnerUserID:      input.OwnerUserID,
			RetrievalText:    retrievalText,
			EmbeddingModel:   embeddingResult.ProviderID,
			EmbeddingLiteral: vectorLiteral(embeddingResult.Vector),
			EngramJSON:       engramJSON,
		},
	); err != nil {
		return nil, nil, err
	}
	if err := insertClaimSources(ctx, db, engramID, resolvedPayload); err != nil {
		return nil, nil, err
	}
	if err := insertArtifacts(ctx, db, engramID, resolvedPayload); err != nil {
		return nil, nil, err
	}

	response := &models.EngramCreateResponse{
		EngramID:  engramID,
		CreatedAt: createdAt,
	}
	return response, enrichmentReport, nil
}

// CreateEngram persists a new engram without returning enrichment metadata.
func CreateEngram(
	ctx context.Context,
	db Queryer,
	input CreateEngramInput,
) (*models.EngramCreateResponse, error) {
	created, _, err := CreateEngramWithReport(ctx, db, input)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func insertEngramRow(ctx context.Context, db Queryer, request engramInsertRequest) error {
	engramJSON, err := marshalJSON(request.EngramJSON)
	if err != nil {
		return err
	}
	visibilityScope := visibilityScopeValue(request.Payload)
	sourceSessionID := request.Payload.SourceSessionID
	var persistedID uuid.UUID
	row := db.QueryRow(
		ctx,
		`
		INSERT INTO engrams (
			engram_id, project_id, thread_id, created_at, updated_at, schema_version,
			title, abstract, engram_json, engram_markdown, tags, keywords, owner_user_id,
			visibility_scope, source_session_id, retrieval_text, embedding_model, embed
		)
		VALUES (
			$1, $2, $3, $4, $4, '1.0',
			$5, $6, $7::jsonb, $8, $9, $10, $11, $12, $13, $14, $15, $16::vector
		)
		RETURNING engram_id
		`,
		request.EngramID,
		request.Payload.ProjectID,
		threadIDValue(request.Payload),
		request.CreatedAt,
		request.Payload.Title,
		request.Payload.Abstract,
		engramJSON,
		request.Payload.DetailedSummaryMarkdown,
		request.Payload.Tags,
		request.Payload.Keywords,
		request.OwnerUserID,
		visibilityScope,
		sourceSessionID,
		request.RetrievalText,
		request.EmbeddingModel,
		request.EmbeddingLiteral,
	)
	if err := row.Scan(&persistedID); err != nil {
		return err
	}
	return nil
}

func insertClaimSources(
	ctx context.Context,
	db Queryer,
	engramID uuid.UUID,
	payload models.MemoryEngramCreate,
) error {
	for _, claim := range payload.Claims {
		for _, source := range claim.SupportingSources {
			var insertedID uuid.UUID
			row := db.QueryRow(
				ctx,
				`
				INSERT INTO sources (
					source_id, engram_id, captured_at, url, title, snippet, content_text, content_hash
				)
				VALUES ($1, $2, $3, $4, $5, $6, NULL, NULL)
				RETURNING source_id
				`,
				newWriteUUID(),
				engramID,
				source.CapturedAt,
				source.URL,
				source.Title,
				source.Snippet,
			)
			if err := row.Scan(&insertedID); err != nil {
				return err
			}
		}
	}
	return nil
}

func insertArtifacts(
	ctx context.Context,
	db Queryer,
	engramID uuid.UUID,
	payload models.MemoryEngramCreate,
) error {
	for _, artifact := range payload.Artifacts {
		metadataJSON, err := marshalJSON(artifact.Metadata)
		if err != nil {
			return err
		}
		var insertedID uuid.UUID
		row := db.QueryRow(
			ctx,
			`
			INSERT INTO artifacts (artifact_id, engram_id, artifact_type, storage_uri, metadata)
			VALUES ($1, $2, $3, $4, $5::jsonb)
			RETURNING artifact_id
			`,
			newWriteUUID(),
			engramID,
			artifact.ArtifactType,
			artifact.StorageURI,
			metadataJSON,
		)
		if err := row.Scan(&insertedID); err != nil {
			return err
		}
	}
	return nil
}

func defaultEnrichmentReport(enrichmentOrigin string) map[string]any {
	return map[string]any{
		"schema_version":     "1.0",
		"origin":             enrichmentOrigin,
		"enrichment_applied": false,
		"abstract_derived":   false,
		"tags_derived":       false,
		"keywords_derived":   false,
		"auto_tags":          []string{},
		"auto_keywords":      []string{},
		"abstract_source":    nil,
	}
}

func normalizeEngramStringSlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func marshalJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal json: %w", err)
	}
	return string(encoded), nil
}

package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"engram/internal/embeddings"
	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	nowAdminEngramUTC    = func() time.Time { return time.Now().UTC() }
	embedAdminEngramText = func(text string, dim int) (embeddings.Result, error) {
		return embeddings.EmbedText(text, dim)
	}
	newAdminSourceUUID = uuid.New
)

// AdminEngramUpdateInput captures mutable memory-admin engram update dependencies.
type AdminEngramUpdateInput struct {
	EngramID                uuid.UUID
	ActorUserID             uuid.UUID
	Title                   *string
	Abstract                *string
	DetailedSummaryMarkdown *string
	Tags                    *[]string
	Keywords                *[]string
	VisibilityScope         *models.VisibilityScope
	Sources                 *[]models.AdminEngramSourceInput
	EmbeddingDim            int
}

type adminEngramUpdateFields struct {
	Title                   string
	Abstract                string
	DetailedSummaryMarkdown string
	Tags                    []string
	Keywords                []string
	VisibilityScope         models.VisibilityScope
}

// UpdateAdminEngram updates mutable engram fields and optionally replaces sources.
func UpdateAdminEngram(
	ctx context.Context,
	db Queryer,
	input AdminEngramUpdateInput,
) (*models.AdminEngramRecord, error) {
	current, err := GetAdminEngram(ctx, db, input.EngramID, false)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, nil
	}

	updateFields := buildAdminEngramUpdateFields(*current, input)
	retrievalText := buildAdminEngramRetrievalText(updateFields)
	embedding, err := embedAdminEngramText(retrievalText, input.EmbeddingDim)
	if err != nil {
		return nil, err
	}
	engramJSON, err := marshalJSON(buildAdminEngramJSONPayload(*current, updateFields))
	if err != nil {
		return nil, err
	}

	updated, err := updateAdminEngramRecord(ctx, db, input, updateFields, retrievalText, embedding, engramJSON)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, nil
	}

	if err := replaceAdminEngramSourcesIfProvided(ctx, db, input); err != nil {
		return nil, err
	}

	return GetAdminEngram(ctx, db, input.EngramID, false)
}

func updateAdminEngramRecord(
	ctx context.Context,
	db Queryer,
	input AdminEngramUpdateInput,
	updateFields adminEngramUpdateFields,
	retrievalText string,
	embedding embeddings.Result,
	engramJSON string,
) (bool, error) {
	row := db.QueryRow(
		ctx,
		`
		UPDATE engrams
		SET
			title = $1,
			abstract = $2,
			engram_markdown = $3,
			tags = $4,
			keywords = $5,
			visibility_scope = $6,
			retrieval_text = $7,
			embedding_model = $8,
			embed = $9::vector,
			engram_json = $10::jsonb,
			updated_at = now(),
			updated_by_user_id = $11
		WHERE
			engram_id = $12
			AND deleted_at IS NULL
		RETURNING engram_id
		`,
		updateFields.Title,
		updateFields.Abstract,
		updateFields.DetailedSummaryMarkdown,
		updateFields.Tags,
		updateFields.Keywords,
		string(updateFields.VisibilityScope),
		retrievalText,
		embedding.ProviderID,
		vectorLiteral(embedding.Vector),
		engramJSON,
		input.ActorUserID,
		input.EngramID,
	)
	var updatedID uuid.UUID
	err := row.Scan(&updatedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func buildAdminEngramUpdateFields(
	current models.AdminEngramRecord,
	input AdminEngramUpdateInput,
) adminEngramUpdateFields {
	fields := adminEngramUpdateFields{
		Title:                   current.Title,
		Abstract:                current.Abstract,
		DetailedSummaryMarkdown: current.DetailedSummaryMarkdown,
		Tags:                    normalizeStringSlice(current.Tags),
		Keywords:                normalizeStringSlice(current.Keywords),
		VisibilityScope:         current.VisibilityScope,
	}
	if input.Title != nil {
		fields.Title = strings.TrimSpace(*input.Title)
	}
	if input.Abstract != nil {
		fields.Abstract = strings.TrimSpace(*input.Abstract)
	}
	if input.DetailedSummaryMarkdown != nil {
		fields.DetailedSummaryMarkdown = *input.DetailedSummaryMarkdown
	}
	if input.Tags != nil {
		fields.Tags = normalizeStringSlice(*input.Tags)
	}
	if input.Keywords != nil {
		fields.Keywords = normalizeStringSlice(*input.Keywords)
	}
	if input.VisibilityScope != nil {
		fields.VisibilityScope = *input.VisibilityScope
	}
	return fields
}

func buildAdminEngramRetrievalText(fields adminEngramUpdateFields) string {
	parts := []string{
		strings.TrimSpace(fields.Title),
		strings.TrimSpace(fields.Abstract),
		strings.TrimSpace(fields.DetailedSummaryMarkdown),
		strings.TrimSpace(strings.Join(fields.Tags, " ")),
		strings.TrimSpace(strings.Join(fields.Keywords, " ")),
	}
	nonEmpty := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		nonEmpty = append(nonEmpty, part)
	}
	return strings.Join(nonEmpty, " ")
}

func buildAdminEngramJSONPayload(
	current models.AdminEngramRecord,
	fields adminEngramUpdateFields,
) map[string]any {
	return map[string]any{
		"engram_id":                 current.EngramID.String(),
		"project_id":                current.ProjectID,
		"thread_id":                 optionalStringValue(current.ThreadID),
		"title":                     fields.Title,
		"abstract":                  fields.Abstract,
		"detailed_summary_markdown": fields.DetailedSummaryMarkdown,
		"tags":                      fields.Tags,
		"keywords":                  fields.Keywords,
		"owner_user_id":             optionalUUIDString(current.OwnerUserID),
		"visibility_scope":          string(fields.VisibilityScope),
		"source_session_id":         optionalUUIDString(current.SourceSessionID),
		"created_at":                current.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
		"updated_at":                nowAdminEngramUTC().Format("2006-01-02T15:04:05-07:00"),
		"deleted_at":                optionalTimeString(current.DeletedAt),
		"deleted_by_user_id":        optionalUUIDString(current.DeletedByUserID),
		"delete_reason":             optionalStringValue(current.DeleteReason),
		"sources":                   adminSourcePayload(current.Sources),
	}
}

func replaceAdminEngramSources(
	ctx context.Context,
	db Queryer,
	engramID uuid.UUID,
	sources []models.AdminEngramSourceInput,
) error {
	if err := deleteAdminEngramSources(ctx, db, engramID); err != nil {
		return err
	}
	return insertAdminEngramSources(ctx, db, engramID, sources)
}

func replaceAdminEngramSourcesIfProvided(ctx context.Context, db Queryer, input AdminEngramUpdateInput) error {
	if input.Sources == nil {
		return nil
	}
	return replaceAdminEngramSources(ctx, db, input.EngramID, *input.Sources)
}

func deleteAdminEngramSources(ctx context.Context, db Queryer, engramID uuid.UUID) error {
	rows, err := db.Query(
		ctx,
		`DELETE FROM sources WHERE engram_id = $1 RETURNING source_id`,
		engramID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var sourceID uuid.UUID
		if err := rows.Scan(&sourceID); err != nil {
			return err
		}
	}
	return rows.Err()
}

func insertAdminEngramSources(
	ctx context.Context,
	db Queryer,
	engramID uuid.UUID,
	sources []models.AdminEngramSourceInput,
) error {
	for _, source := range sources {
		if err := insertAdminEngramSource(ctx, db, engramID, source); err != nil {
			return err
		}
	}
	return nil
}

func insertAdminEngramSource(
	ctx context.Context,
	db Queryer,
	engramID uuid.UUID,
	source models.AdminEngramSourceInput,
) error {
	row := db.QueryRow(
		ctx,
		`
		INSERT INTO sources (
			source_id,
			engram_id,
			captured_at,
			url,
			title,
			snippet,
			content_text,
			content_hash
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING source_id
		`,
		newAdminSourceUUID(),
		engramID,
		source.CapturedAt,
		source.URL,
		source.Title,
		source.Snippet,
		source.ContentText,
		source.ContentHash,
	)
	var insertedSourceID uuid.UUID
	return row.Scan(&insertedSourceID)
}

func adminSourcePayload(sources []models.AdminEngramSourceRecord) []map[string]any {
	payload := make([]map[string]any, 0, len(sources))
	for _, source := range sources {
		payload = append(
			payload,
			map[string]any{
				"source_id":    source.SourceID.String(),
				"captured_at":  source.CapturedAt.Format("2006-01-02T15:04:05-07:00"),
				"url":          source.URL,
				"title":        optionalStringValue(source.Title),
				"snippet":      optionalStringValue(source.Snippet),
				"content_text": optionalStringValue(source.ContentText),
				"content_hash": optionalStringValue(source.ContentHash),
			},
		)
	}
	return payload
}

func optionalUUIDString(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return value.String()
}

func optionalStringValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func optionalTimeString(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.Format("2006-01-02T15:04:05-07:00")
}

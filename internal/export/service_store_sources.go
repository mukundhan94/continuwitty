package export

import (
	"context"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func replaceEngramSources(
	ctx context.Context,
	db repository.Queryer,
	engramID uuid.UUID,
	sources []models.AdminEngramSourceInput,
) error {
	if err := deleteEngramSources(ctx, db, engramID); err != nil {
		return err
	}
	for _, source := range sources {
		if err := insertEngramSource(ctx, db, engramID, source); err != nil {
			return err
		}
	}
	return nil
}

func deleteEngramSources(ctx context.Context, db repository.Queryer, engramID uuid.UUID) error {
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

func insertEngramSource(
	ctx context.Context,
	db repository.Queryer,
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
		uuid.New(),
		engramID,
		source.CapturedAt,
		source.URL,
		source.Title,
		source.Snippet,
		source.ContentText,
		source.ContentHash,
	)
	var sourceID uuid.UUID
	return row.Scan(&sourceID)
}

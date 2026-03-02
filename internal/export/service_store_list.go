package export

import (
	"context"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func listCollectionItemMap(
	ctx context.Context,
	db repository.Queryer,
	collectionIDs []uuid.UUID,
) (map[uuid.UUID][]uuid.UUID, error) {
	result := make(map[uuid.UUID][]uuid.UUID, len(collectionIDs))
	for _, collectionID := range collectionIDs {
		result[collectionID] = []uuid.UUID{}
	}
	if len(collectionIDs) == 0 {
		return result, nil
	}
	rows, err := db.Query(
		ctx,
		`
		SELECT collection_id, engram_id
		FROM engram_collection_items
		WHERE collection_id = ANY($1::UUID[])
		ORDER BY collection_id, created_at ASC
		`,
		collectionIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var collectionID uuid.UUID
		var engramID uuid.UUID
		if err := rows.Scan(&collectionID, &engramID); err != nil {
			return nil, err
		}
		result[collectionID] = append(result[collectionID], engramID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func listEngramSourcesMap(
	ctx context.Context,
	db repository.Queryer,
	engramIDs []uuid.UUID,
) (map[uuid.UUID][]models.AdminEngramSourceRecord, error) {
	result := make(map[uuid.UUID][]models.AdminEngramSourceRecord, len(engramIDs))
	for _, engramID := range engramIDs {
		result[engramID] = []models.AdminEngramSourceRecord{}
	}
	if len(engramIDs) == 0 {
		return result, nil
	}
	rows, err := db.Query(
		ctx,
		`
		SELECT source_id, engram_id, captured_at, url, title, snippet, content_text, content_hash
		FROM sources
		WHERE engram_id = ANY($1::UUID[])
		ORDER BY engram_id, captured_at DESC
		`,
		engramIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		record, engramID, err := scanSourceMapRow(rows)
		if err != nil {
			return nil, err
		}
		result[engramID] = append(result[engramID], record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func scanSourceMapRow(
	rows interface{ Scan(dest ...any) error },
) (models.AdminEngramSourceRecord, uuid.UUID, error) {
	record := models.AdminEngramSourceRecord{}
	var engramID uuid.UUID
	err := rows.Scan(
		&record.SourceID,
		&engramID,
		&record.CapturedAt,
		&record.URL,
		&record.Title,
		&record.Snippet,
		&record.ContentText,
		&record.ContentHash,
	)
	return record, engramID, err
}

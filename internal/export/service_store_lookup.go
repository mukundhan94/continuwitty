package export

import (
	"context"
	"errors"
	"fmt"

	"engram/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func findExistingEngramID(
	ctx context.Context,
	db repository.Queryer,
	projectID, title, markdown string,
) (*uuid.UUID, error) {
	row := db.QueryRow(
		ctx,
		`
		SELECT engram_id
		FROM engrams
		WHERE project_id = $1
		  AND title = $2
		  AND engram_markdown = $3
		  AND deleted_at IS NULL
		LIMIT 1
		`,
		projectID,
		title,
		markdown,
	)
	return scanOptionalUUID(row)
}

func softDeleteEngram(
	ctx context.Context,
	db repository.Queryer,
	engramID, actorUserID uuid.UUID,
) error {
	row := db.QueryRow(
		ctx,
		`
		UPDATE engrams
		SET deleted_at = now(), deleted_by_user_id = $1, delete_reason = 'import_overwrite'
		WHERE engram_id = $2
		RETURNING engram_id
		`,
		actorUserID,
		engramID,
	)
	_, err := scanOptionalUUID(row)
	return err
}

func findExistingCollectionID(
	ctx context.Context,
	db repository.Queryer,
	projectID, name string,
) (*uuid.UUID, error) {
	row := db.QueryRow(
		ctx,
		`
		SELECT collection_id
		FROM engram_collections
		WHERE project_id = $1
		  AND name = $2
		  AND deleted_at IS NULL
		LIMIT 1
		`,
		projectID,
		name,
	)
	return scanOptionalUUID(row)
}

func buildUniqueName(
	ctx context.Context,
	db repository.Queryer,
	table, column, projectID, baseName string,
) (string, error) {
	candidate := baseName + " (imported)"
	index := 2
	sql := fmt.Sprintf(
		`
		SELECT 1
		FROM %s
		WHERE project_id = $1
		  AND %s = $2
		  AND deleted_at IS NULL
		LIMIT 1
		`,
		table,
		column,
	)
	for {
		exists, err := checkNameExists(ctx, db, sql, projectID, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s (imported %d)", baseName, index)
		index++
	}
}

func checkNameExists(
	ctx context.Context,
	db repository.Queryer,
	sql string,
	projectID string,
	candidate string,
) (bool, error) {
	row := db.QueryRow(ctx, sql, projectID, candidate)
	var exists int
	err := row.Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func scanOptionalUUID(row interface{ Scan(dest ...any) error }) (*uuid.UUID, error) {
	var value uuid.UUID
	err := row.Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

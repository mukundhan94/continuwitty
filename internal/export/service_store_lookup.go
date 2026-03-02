package export

import (
	"context"
	"errors"
	"fmt"

	"engram/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type findExistingEngramIDInput struct {
	ProjectID string
	Title     string
	Markdown  string
}

func findExistingEngramID(
	ctx context.Context,
	db repository.Queryer,
	input findExistingEngramIDInput,
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
		input.ProjectID,
		input.Title,
		input.Markdown,
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

type buildUniqueNameInput struct {
	Table     string
	Column    string
	ProjectID string
	BaseName  string
}

func buildUniqueName(
	ctx context.Context,
	db repository.Queryer,
	input buildUniqueNameInput,
) (string, error) {
	candidate := input.BaseName + " (imported)"
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
		input.Table,
		input.Column,
	)
	for {
		exists, err := checkNameExists(ctx, db, checkNameExistsInput{
			Query:     sql,
			ProjectID: input.ProjectID,
			Candidate: candidate,
		})
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s (imported %d)", input.BaseName, index)
		index++
	}
}

type checkNameExistsInput struct {
	Query     string
	ProjectID string
	Candidate string
}

func checkNameExists(
	ctx context.Context,
	db repository.Queryer,
	input checkNameExistsInput,
) (bool, error) {
	row := db.QueryRow(ctx, input.Query, input.ProjectID, input.Candidate)
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

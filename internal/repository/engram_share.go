package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// EngramShareRecord captures share-related engram state.
type EngramShareRecord struct {
	EngramID        uuid.UUID
	ProjectID       string
	OwnerUserID     *uuid.UUID
	VisibilityScope models.VisibilityScope
	DeletedAt       *time.Time
}

// EngramVisibilityUpdateInput captures share/unshare visibility writes.
type EngramVisibilityUpdateInput struct {
	EngramID        uuid.UUID
	VisibilityScope models.VisibilityScope
	ActorUserID     uuid.UUID
}

// GetEngramShareRecord returns an engram record for share authorization.
func GetEngramShareRecord(
	ctx context.Context,
	db Queryer,
	engramID uuid.UUID,
	includeDeleted bool,
) (*EngramShareRecord, error) {
	query := `
		SELECT
			engram_id,
			project_id,
			owner_user_id,
			visibility_scope,
			deleted_at
		FROM engrams
		WHERE engram_id = $1
	`
	if !includeDeleted {
		query += " AND deleted_at IS NULL"
	}
	query += " LIMIT 1"

	row := db.QueryRow(ctx, query, engramID)
	record, err := scanEngramShareRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateEngramVisibilityScope updates engram visibility and updated-by metadata.
func UpdateEngramVisibilityScope(
	ctx context.Context,
	db Queryer,
	input EngramVisibilityUpdateInput,
) (*EngramShareRecord, error) {
	now := time.Now().UTC()
	row := db.QueryRow(
		ctx,
		`
		UPDATE engrams
		SET
			visibility_scope = $1,
			updated_at = $2,
			updated_by_user_id = $3
		WHERE
			engram_id = $4
			AND deleted_at IS NULL
		RETURNING
			engram_id,
			project_id,
			owner_user_id,
			visibility_scope,
			deleted_at
		`,
		string(input.VisibilityScope),
		now,
		input.ActorUserID,
		input.EngramID,
	)
	record, err := scanEngramShareRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func scanEngramShareRecord(row interface {
	Scan(dest ...any) error
}) (EngramShareRecord, error) {
	record := EngramShareRecord{}
	var visibilityScopeRaw string
	err := row.Scan(
		&record.EngramID,
		&record.ProjectID,
		&record.OwnerUserID,
		&visibilityScopeRaw,
		&record.DeletedAt,
	)
	if err != nil {
		return EngramShareRecord{}, err
	}
	visibilityScope, err := models.ParseVisibilityScope(strings.TrimSpace(visibilityScopeRaw))
	if err != nil {
		return EngramShareRecord{}, err
	}
	record.VisibilityScope = visibilityScope
	return record, nil
}

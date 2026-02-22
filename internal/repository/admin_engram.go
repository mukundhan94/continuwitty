package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const adminEngramColumns = `
	e.engram_id,
	e.project_id,
	e.thread_id,
	e.title,
	e.abstract,
	e.engram_markdown AS detailed_summary_markdown,
	e.tags,
	e.keywords,
	e.owner_user_id,
	e.visibility_scope,
	e.source_session_id,
	e.created_at,
	e.updated_at,
	e.deleted_at,
	e.deleted_by_user_id,
	e.delete_reason
`

// AdminEngramListInput captures list-admin-engrams filters.
type AdminEngramListInput struct {
	IncludeDeleted bool
	ProjectID      *string
	SessionID      *uuid.UUID
	OwnerUserID    *uuid.UUID
	QueryText      *string
	Limit          int
	Offset         int
}

// AdminEngramMoveProjectInput captures move-admin-engram dependencies.
type AdminEngramMoveProjectInput struct {
	EngramID        uuid.UUID
	TargetProjectID string
	ActorUserID     uuid.UUID
}

// AdminEngramSoftDeleteInput captures soft-delete-admin-engram dependencies.
type AdminEngramSoftDeleteInput struct {
	EngramID        uuid.UUID
	DeletedByUserID uuid.UUID
	Reason          *string
}

// ListAdminEngrams returns engrams filtered by memory-admin selectors.
func ListAdminEngrams(
	ctx context.Context,
	db Queryer,
	input AdminEngramListInput,
) ([]models.AdminEngramRecord, error) {
	whereClauses := []string{"1=1"}
	params := make([]any, 0)
	if !input.IncludeDeleted {
		whereClauses = append(whereClauses, "e.deleted_at IS NULL")
	}
	if input.ProjectID != nil && *input.ProjectID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("e.project_id = %s", pgxPlaceholder(len(params)+1)))
		params = append(params, *input.ProjectID)
	}
	if input.SessionID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.source_session_id = %s", pgxPlaceholder(len(params)+1)))
		params = append(params, *input.SessionID)
	}
	if input.OwnerUserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("e.owner_user_id = %s", pgxPlaceholder(len(params)+1)))
		params = append(params, *input.OwnerUserID)
	}
	if input.QueryText != nil && strings.TrimSpace(*input.QueryText) != "" {
		like := "%" + strings.TrimSpace(*input.QueryText) + "%"
		whereClauses = append(
			whereClauses,
			fmt.Sprintf(
				"(e.title ILIKE %s OR e.abstract ILIKE %s OR e.engram_markdown ILIKE %s)",
				pgxPlaceholder(len(params)+1),
				pgxPlaceholder(len(params)+2),
				pgxPlaceholder(len(params)+3),
			),
		)
		params = append(params, like, like, like)
	}

	sql := fmt.Sprintf(
		`
		SELECT %s
		FROM engrams e
		WHERE %s
		ORDER BY e.created_at DESC
		LIMIT %s OFFSET %s
		`,
		adminEngramColumns,
		strings.Join(whereClauses, " AND "),
		pgxPlaceholder(len(params)+1),
		pgxPlaceholder(len(params)+2),
	)
	params = append(params, input.Limit, input.Offset)

	rows, err := db.Query(ctx, sql, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.AdminEngramRecord, 0)
	for rows.Next() {
		record, scanErr := scanAdminEngramRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		record.Sources = []models.AdminEngramSourceRecord{}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

// ListAdminEngramSources returns source rows for an engram ordered by capture time.
func ListAdminEngramSources(
	ctx context.Context,
	db Queryer,
	engramID uuid.UUID,
) ([]models.AdminEngramSourceRecord, error) {
	rows, err := db.Query(
		ctx,
		`
		SELECT
			source_id,
			captured_at,
			url,
			title,
			snippet,
			content_text,
			content_hash
		FROM sources
		WHERE engram_id = $1
		ORDER BY captured_at DESC
		`,
		engramID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.AdminEngramSourceRecord, 0)
	for rows.Next() {
		record, scanErr := scanAdminEngramSourceRecord(rows)
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

// GetAdminEngram fetches an engram by ID with optional deleted filtering and sources.
func GetAdminEngram(
	ctx context.Context,
	db Queryer,
	engramID uuid.UUID,
	includeDeleted bool,
) (*models.AdminEngramRecord, error) {
	sql := fmt.Sprintf(
		`
		SELECT %s
		FROM engrams e
		WHERE e.engram_id = $1
		`,
		adminEngramColumns,
	)
	if !includeDeleted {
		sql += " AND e.deleted_at IS NULL"
	}
	sql += " LIMIT 1"

	row := db.QueryRow(ctx, sql, engramID)
	record, err := scanAdminEngramRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sources, err := ListAdminEngramSources(ctx, db, engramID)
	if err != nil {
		return nil, err
	}
	record.Sources = sources
	return &record, nil
}

// MoveAdminEngramProject moves an engram to another project and detaches cross-project collections.
func MoveAdminEngramProject(
	ctx context.Context,
	db Queryer,
	input AdminEngramMoveProjectInput,
) (*models.AdminEngramRecord, error) {
	row := db.QueryRow(
		ctx,
		`
		UPDATE engrams
		SET
			project_id = $1,
			updated_at = now(),
			updated_by_user_id = $2
		WHERE
			engram_id = $3
			AND deleted_at IS NULL
		RETURNING engram_id
		`,
		input.TargetProjectID,
		input.ActorUserID,
		input.EngramID,
	)
	var updatedID uuid.UUID
	err := row.Scan(&updatedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	cleanupRows, err := db.Query(
		ctx,
		`
		DELETE FROM engram_collection_items item
		USING engram_collections collection, engrams e
		WHERE
			item.collection_id = collection.collection_id
			AND item.engram_id = e.engram_id
			AND item.engram_id = $1
			AND collection.project_id <> e.project_id
		`,
		input.EngramID,
	)
	if err != nil {
		return nil, err
	}
	defer cleanupRows.Close()
	if err := cleanupRows.Err(); err != nil {
		return nil, err
	}

	return GetAdminEngram(ctx, db, input.EngramID, false)
}

// SoftDeleteEngram marks an engram as deleted and records actor metadata.
func SoftDeleteEngram(
	ctx context.Context,
	db Queryer,
	input AdminEngramSoftDeleteInput,
) (bool, error) {
	row := db.QueryRow(
		ctx,
		`
		UPDATE engrams
		SET
			deleted_at = COALESCE(deleted_at, now()),
			deleted_by_user_id = $1,
			delete_reason = $2,
			updated_at = now(),
			updated_by_user_id = $1
		WHERE engram_id = $3
		RETURNING engram_id
		`,
		input.DeletedByUserID,
		input.Reason,
		input.EngramID,
	)
	var deletedID uuid.UUID
	err := row.Scan(&deletedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// RestoreEngram clears engram soft-delete metadata.
func RestoreEngram(
	ctx context.Context,
	db Queryer,
	engramID uuid.UUID,
) (bool, error) {
	row := db.QueryRow(
		ctx,
		`
		UPDATE engrams
		SET
			deleted_at = NULL,
			deleted_by_user_id = NULL,
			delete_reason = NULL,
			updated_at = now()
		WHERE engram_id = $1
		RETURNING engram_id
		`,
		engramID,
	)
	var restoredID uuid.UUID
	err := row.Scan(&restoredID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func scanAdminEngramRecord(row interface {
	Scan(dest ...any) error
}) (models.AdminEngramRecord, error) {
	var (
		record     models.AdminEngramRecord
		tags       []string
		keywords   []string
		visibility string
	)
	err := row.Scan(
		&record.EngramID,
		&record.ProjectID,
		&record.ThreadID,
		&record.Title,
		&record.Abstract,
		&record.DetailedSummaryMarkdown,
		&tags,
		&keywords,
		&record.OwnerUserID,
		&visibility,
		&record.SourceSessionID,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.DeletedAt,
		&record.DeletedByUserID,
		&record.DeleteReason,
	)
	if err != nil {
		return models.AdminEngramRecord{}, err
	}
	parsedVisibility, err := models.ParseVisibilityScope(strings.TrimSpace(visibility))
	if err != nil {
		return models.AdminEngramRecord{}, err
	}
	record.VisibilityScope = parsedVisibility
	record.Tags = normalizeStringSlice(tags)
	record.Keywords = normalizeStringSlice(keywords)
	record.Sources = []models.AdminEngramSourceRecord{}
	return record, nil
}

func scanAdminEngramSourceRecord(row interface {
	Scan(dest ...any) error
}) (models.AdminEngramSourceRecord, error) {
	var record models.AdminEngramSourceRecord
	err := row.Scan(
		&record.SourceID,
		&record.CapturedAt,
		&record.URL,
		&record.Title,
		&record.Snippet,
		&record.ContentText,
		&record.ContentHash,
	)
	if err != nil {
		return models.AdminEngramSourceRecord{}, err
	}
	return record, nil
}

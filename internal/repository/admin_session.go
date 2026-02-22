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

const adminSessionColumns = `
	session_id,
	owner_user_id,
	project_id,
	title,
	provider,
	model_id,
	system_prompt,
	visibility_scope,
	autosave_enabled,
	autosave_strategy,
	autosave_interval_minutes,
	autosave_min_messages,
	retention_days,
	retention_max_snapshots,
	created_at,
	updated_at,
	deleted_at,
	deleted_by_user_id,
	delete_reason
`

// AdminSessionListInput captures list-admin-sessions filters.
type AdminSessionListInput struct {
	IncludeDeleted bool
	ProjectID      *string
	OwnerUserID    *uuid.UUID
	Limit          int
	Offset         int
}

// AdminSessionSoftDeleteInput captures session soft-delete dependencies.
type AdminSessionSoftDeleteInput struct {
	SessionID       uuid.UUID
	DeletedByUserID uuid.UUID
	Reason          *string
}

// SoftDeleteLinkedEngramsInput captures linked-engram soft-delete dependencies.
type SoftDeleteLinkedEngramsInput struct {
	SessionID       uuid.UUID
	DeletedByUserID uuid.UUID
	Reason          *string
}

// ListAdminSessions returns admin session rows filtered by project/owner/deleted state.
func ListAdminSessions(
	ctx context.Context,
	db Queryer,
	input AdminSessionListInput,
) ([]models.AdminChatSessionRecord, error) {
	whereClauses := []string{"1=1"}
	params := make([]any, 0)
	if !input.IncludeDeleted {
		whereClauses = append(whereClauses, "deleted_at IS NULL")
	}
	if input.ProjectID != nil && *input.ProjectID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("project_id = %s", pgxPlaceholder(len(params)+1)))
		params = append(params, *input.ProjectID)
	}
	if input.OwnerUserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("owner_user_id = %s", pgxPlaceholder(len(params)+1)))
		params = append(params, *input.OwnerUserID)
	}

	sql := fmt.Sprintf(
		`
		SELECT %s
		FROM chat_sessions
		WHERE %s
		ORDER BY created_at DESC
		LIMIT %s OFFSET %s
		`,
		adminSessionColumns,
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

	records := make([]models.AdminChatSessionRecord, 0)
	for rows.Next() {
		record, scanErr := scanAdminSessionRecord(rows)
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

// GetAdminSession fetches a session by ID with optional deleted filtering.
func GetAdminSession(
	ctx context.Context,
	db Queryer,
	sessionID uuid.UUID,
	includeDeleted bool,
) (*models.AdminChatSessionRecord, error) {
	sql := fmt.Sprintf(
		`
		SELECT %s
		FROM chat_sessions
		WHERE session_id = $1
		`,
		adminSessionColumns,
	)
	if !includeDeleted {
		sql += " AND deleted_at IS NULL"
	}
	sql += " LIMIT 1"

	row := db.QueryRow(ctx, sql, sessionID)
	record, err := scanAdminSessionRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// SoftDeleteSession soft deletes a session and returns the updated admin row.
func SoftDeleteSession(
	ctx context.Context,
	db Queryer,
	input AdminSessionSoftDeleteInput,
) (*models.AdminChatSessionRecord, error) {
	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			UPDATE chat_sessions
			SET
				deleted_at = COALESCE(deleted_at, now()),
				deleted_by_user_id = $1,
				delete_reason = $2,
				updated_at = now()
			WHERE session_id = $3
			RETURNING %s
			`,
			adminSessionColumns,
		),
		input.DeletedByUserID,
		input.Reason,
		input.SessionID,
	)
	record, err := scanAdminSessionRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// RestoreSession clears soft-delete metadata for a session.
func RestoreSession(
	ctx context.Context,
	db Queryer,
	sessionID uuid.UUID,
) (bool, error) {
	row := db.QueryRow(
		ctx,
		`
		UPDATE chat_sessions
		SET
			deleted_at = NULL,
			deleted_by_user_id = NULL,
			delete_reason = NULL,
			updated_at = now()
		WHERE session_id = $1
		RETURNING session_id
		`,
		sessionID,
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

// SoftDeleteLinkedEngrams soft deletes active engrams linked to a session.
func SoftDeleteLinkedEngrams(
	ctx context.Context,
	db Queryer,
	input SoftDeleteLinkedEngramsInput,
) (int, error) {
	rows, err := db.Query(
		ctx,
		`
		UPDATE engrams
		SET
			deleted_at = COALESCE(deleted_at, now()),
			deleted_by_user_id = $1,
			delete_reason = $2,
			updated_at = now(),
			updated_by_user_id = $1
		WHERE
			source_session_id = $3
			AND deleted_at IS NULL
		RETURNING engram_id
		`,
		input.DeletedByUserID,
		input.Reason,
		input.SessionID,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var engramID uuid.UUID
		if scanErr := rows.Scan(&engramID); scanErr != nil {
			return 0, scanErr
		}
		count += 1
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

func scanAdminSessionRecord(row interface {
	Scan(dest ...any) error
}) (models.AdminChatSessionRecord, error) {
	var (
		record           models.AdminChatSessionRecord
		provider         string
		visibility       string
		autosaveStrategy string
	)
	err := row.Scan(
		&record.SessionID,
		&record.OwnerUserID,
		&record.ProjectID,
		&record.Title,
		&provider,
		&record.ModelID,
		&record.SystemPrompt,
		&visibility,
		&record.AutosaveEnabled,
		&autosaveStrategy,
		&record.AutosaveIntervalMinutes,
		&record.AutosaveMinMessages,
		&record.RetentionDays,
		&record.RetentionMaxSnapshots,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.DeletedAt,
		&record.DeletedByUserID,
		&record.DeleteReason,
	)
	if err != nil {
		return models.AdminChatSessionRecord{}, err
	}
	parsedProvider, err := models.ParseChatProvider(strings.TrimSpace(provider))
	if err != nil {
		return models.AdminChatSessionRecord{}, err
	}
	parsedVisibility, err := models.ParseVisibilityScope(strings.TrimSpace(visibility))
	if err != nil {
		return models.AdminChatSessionRecord{}, err
	}
	parsedAutosaveStrategy, err := models.ParseChatAutosaveStrategy(strings.TrimSpace(autosaveStrategy))
	if err != nil {
		return models.AdminChatSessionRecord{}, err
	}
	record.Provider = parsedProvider
	record.VisibilityScope = parsedVisibility
	record.AutosaveStrategy = parsedAutosaveStrategy
	return record, nil
}

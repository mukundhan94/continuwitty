package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ChatPinEngramInput captures engram pin/unpin dependencies.
type ChatPinEngramInput struct {
	SessionID   uuid.UUID
	EngramID    uuid.UUID
	ActorUserID uuid.UUID
}

// ChatPinDocumentInput captures document pin/unpin dependencies.
type ChatPinDocumentInput struct {
	SessionID   uuid.UUID
	DocumentID  uuid.UUID
	ActorUserID uuid.UUID
}

// ChatPinnedListInput captures list-pinned operation filters.
type ChatPinnedListInput struct {
	SessionID   uuid.UUID
	ActorUserID uuid.UUID
}

type pinnedResourceMutationConfig struct {
	table                string
	idColumn             string
	resourceTable        string
	resourceAccessClause string
}

type pinnedResourceListConfig struct {
	table                string
	idColumn             string
	joinSQL              string
	resourceAccessClause string
	includeActorParam    bool
}

var (
	engramPinMutationConfig = pinnedResourceMutationConfig{
		table:         "session_pinned_engrams",
		idColumn:      "engram_id",
		resourceTable: "engrams",
		resourceAccessClause: `
			r.deleted_at IS NULL
			AND (r.owner_user_id = %s OR r.visibility_scope = 'project' OR r.owner_user_id IS NULL)
		`,
	}
	documentPinMutationConfig = pinnedResourceMutationConfig{
		table:         "session_pinned_documents",
		idColumn:      "document_id",
		resourceTable: "documents",
		resourceAccessClause: `
			(r.owner_user_id = %s OR r.visibility_scope = 'project')
		`,
	}
	engramPinListConfig = pinnedResourceListConfig{
		table:                "session_pinned_engrams",
		idColumn:             "engram_id",
		resourceAccessClause: "1=1",
	}
	documentPinListConfig = pinnedResourceListConfig{
		table:    "session_pinned_documents",
		idColumn: "document_id",
		joinSQL: `
		JOIN documents d
		  ON d.document_id = p.document_id
		`,
		resourceAccessClause: "(d.owner_user_id = %s OR d.visibility_scope = 'project')",
		includeActorParam:    true,
	}
)

// PinEngramToSession pins an engram when both session and engram are visible.
func PinEngramToSession(
	ctx context.Context,
	db Queryer,
	input ChatPinEngramInput,
) (*models.PinnedEngramRecord, error) {
	row, err := pinResourceToSession(
		ctx,
		db,
		engramPinMutationConfig,
		input.SessionID,
		input.EngramID,
		input.ActorUserID,
	)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	record := models.PinnedEngramRecord{
		SessionID:      row.SessionID,
		EngramID:       row.ResourceID,
		PinnedByUserID: row.PinnedByUserID,
		CreatedAt:      row.CreatedAt,
	}
	return &record, nil
}

// PinDocumentToSession pins a document when both session and document are visible.
func PinDocumentToSession(
	ctx context.Context,
	db Queryer,
	input ChatPinDocumentInput,
) (*models.PinnedDocumentRecord, error) {
	row, err := pinResourceToSession(
		ctx,
		db,
		documentPinMutationConfig,
		input.SessionID,
		input.DocumentID,
		input.ActorUserID,
	)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	record := models.PinnedDocumentRecord{
		SessionID:      row.SessionID,
		DocumentID:     row.ResourceID,
		PinnedByUserID: row.PinnedByUserID,
		CreatedAt:      row.CreatedAt,
	}
	return &record, nil
}

// UnpinEngramFromSession removes an engram pin for visible sessions.
func UnpinEngramFromSession(
	ctx context.Context,
	db Queryer,
	input ChatPinEngramInput,
) (bool, error) {
	return unpinResourceFromSession(
		ctx,
		db,
		engramPinMutationConfig,
		input.SessionID,
		input.EngramID,
		input.ActorUserID,
	)
}

// UnpinDocumentFromSession removes a document pin for visible sessions.
func UnpinDocumentFromSession(
	ctx context.Context,
	db Queryer,
	input ChatPinDocumentInput,
) (bool, error) {
	return unpinResourceFromSession(
		ctx,
		db,
		documentPinMutationConfig,
		input.SessionID,
		input.DocumentID,
		input.ActorUserID,
	)
}

// ListPinnedEngrams returns pinned engram rows for a visible session.
func ListPinnedEngrams(
	ctx context.Context,
	db Queryer,
	input ChatPinnedListInput,
) ([]models.PinnedEngramRecord, error) {
	rows, err := listPinnedResources(ctx, db, engramPinListConfig, input)
	if err != nil {
		return nil, err
	}
	records := make([]models.PinnedEngramRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, models.PinnedEngramRecord{
			SessionID:      row.SessionID,
			EngramID:       row.ResourceID,
			PinnedByUserID: row.PinnedByUserID,
			CreatedAt:      row.CreatedAt,
		})
	}
	return records, nil
}

// ListPinnedEngramSummaries returns visible engram summaries from session pins.
func ListPinnedEngramSummaries(
	ctx context.Context,
	db Queryer,
	input ChatPinnedListInput,
) ([]models.EngramSummary, error) {
	rows, err := db.Query(
		ctx,
		`
		SELECT
			e.engram_id,
			e.project_id,
			e.thread_id,
			e.title,
			e.abstract,
			e.created_at,
			e.tags,
			e.keywords,
			e.owner_user_id,
			e.visibility_scope
		FROM session_pinned_engrams p
		JOIN chat_sessions s
		  ON s.session_id = p.session_id
		JOIN engrams e
		  ON e.engram_id = p.engram_id
		WHERE
			p.session_id = $1
			AND s.deleted_at IS NULL
			AND (s.owner_user_id = $2 OR s.visibility_scope = 'project')
			AND e.deleted_at IS NULL
			AND (e.owner_user_id = $3 OR e.visibility_scope = 'project' OR e.owner_user_id IS NULL)
		ORDER BY p.created_at ASC
		`,
		input.SessionID,
		input.ActorUserID,
		input.ActorUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.EngramSummary, 0)
	for rows.Next() {
		record, scanErr := scanEngramSummary(rows)
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

// ListPinnedDocuments returns pinned document rows for a visible session.
func ListPinnedDocuments(
	ctx context.Context,
	db Queryer,
	input ChatPinnedListInput,
) ([]models.PinnedDocumentRecord, error) {
	rows, err := listPinnedResources(ctx, db, documentPinListConfig, input)
	if err != nil {
		return nil, err
	}
	records := make([]models.PinnedDocumentRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, models.PinnedDocumentRecord{
			SessionID:      row.SessionID,
			DocumentID:     row.ResourceID,
			PinnedByUserID: row.PinnedByUserID,
			CreatedAt:      row.CreatedAt,
		})
	}
	return records, nil
}

func pinResourceToSession(
	ctx context.Context,
	db Queryer,
	config pinnedResourceMutationConfig,
	sessionID uuid.UUID,
	resourceID uuid.UUID,
	actorUserID uuid.UUID,
) (*pinnedResourceRow, error) {
	resourceAccessClause := config.resourceAccessClause
	if strings.Contains(resourceAccessClause, "%s") {
		resourceAccessClause = fmt.Sprintf(resourceAccessClause, pgxPlaceholder(4))
	}
	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			WITH accessible_session AS (
				SELECT s.session_id
				FROM chat_sessions s
				WHERE
					s.session_id = $1
					AND s.deleted_at IS NULL
					AND (s.owner_user_id = $2 OR s.visibility_scope = 'project')
			),
			accessible_resource AS (
				SELECT r.%s
				FROM %s r
				WHERE
					r.%s = $3
					AND %s
			)
			INSERT INTO %s (
				session_id,
				%s,
				pinned_by_user_id,
				created_at
			)
			SELECT
				s.session_id,
				r.%s,
				$5,
				$6
			FROM accessible_session s
			CROSS JOIN accessible_resource r
			ON CONFLICT (session_id, %s) DO UPDATE
				SET pinned_by_user_id = EXCLUDED.pinned_by_user_id
			RETURNING
				session_id,
				%s,
				pinned_by_user_id,
				created_at
			`,
			config.idColumn,
			config.resourceTable,
			config.idColumn,
			resourceAccessClause,
			config.table,
			config.idColumn,
			config.idColumn,
			config.idColumn,
			config.idColumn,
		),
		sessionID,
		actorUserID,
		resourceID,
		actorUserID,
		actorUserID,
		nowChatUTC(),
	)
	record, err := scanPinnedResourceRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func unpinResourceFromSession(
	ctx context.Context,
	db Queryer,
	config pinnedResourceMutationConfig,
	sessionID uuid.UUID,
	resourceID uuid.UUID,
	actorUserID uuid.UUID,
) (bool, error) {
	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			DELETE FROM %s p
			USING chat_sessions s
			WHERE
				p.session_id = s.session_id
				AND p.session_id = $1
				AND p.%s = $2
				AND s.deleted_at IS NULL
				AND (s.owner_user_id = $3 OR s.visibility_scope = 'project')
			RETURNING p.session_id
			`,
			config.table,
			config.idColumn,
		),
		sessionID,
		resourceID,
		actorUserID,
	)
	var removedSessionID uuid.UUID
	err := row.Scan(&removedSessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func listPinnedResources(
	ctx context.Context,
	db Queryer,
	config pinnedResourceListConfig,
	input ChatPinnedListInput,
) ([]pinnedResourceRow, error) {
	params := []any{input.SessionID, input.ActorUserID}
	resourceAccessClause := config.resourceAccessClause
	if config.includeActorParam {
		resourceAccessClause = fmt.Sprintf(resourceAccessClause, pgxPlaceholder(len(params)+1))
		params = append(params, input.ActorUserID)
	}

	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`
			SELECT
				p.session_id,
				p.%s,
				p.pinned_by_user_id,
				p.created_at
			FROM %s p
			JOIN chat_sessions s
			  ON s.session_id = p.session_id
			%s
			WHERE
				p.session_id = $1
				AND s.deleted_at IS NULL
				AND (s.owner_user_id = $2 OR s.visibility_scope = 'project')
				AND %s
			ORDER BY p.created_at ASC
			`,
			config.idColumn,
			config.table,
			config.joinSQL,
			resourceAccessClause,
		),
		params...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]pinnedResourceRow, 0)
	for rows.Next() {
		record, scanErr := scanPinnedResourceRow(rows)
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

type pinnedResourceRow struct {
	SessionID      uuid.UUID
	ResourceID     uuid.UUID
	PinnedByUserID uuid.UUID
	CreatedAt      time.Time
}

func scanPinnedResourceRow(row interface {
	Scan(dest ...any) error
}) (pinnedResourceRow, error) {
	var record pinnedResourceRow
	err := row.Scan(
		&record.SessionID,
		&record.ResourceID,
		&record.PinnedByUserID,
		&record.CreatedAt,
	)
	if err != nil {
		return pinnedResourceRow{}, err
	}
	return record, nil
}

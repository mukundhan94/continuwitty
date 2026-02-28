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

type chatPinResourceInput struct {
	SessionID   uuid.UUID
	ResourceID  uuid.UUID
	ActorUserID uuid.UUID
}

type pinnedResourceMutationInput struct {
	Config   pinnedResourceMutationConfig
	Resource chatPinResourceInput
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
		resourceAccessClause: fmt.Sprintf(
			`r.deleted_at IS NULL AND %s`,
			buildMembershipReadClause(
				"r.owner_user_id",
				"r.visibility_scope",
				"r.project_id",
				"%s",
				true,
			),
		),
	}
	documentPinMutationConfig = pinnedResourceMutationConfig{
		table:         "session_pinned_documents",
		idColumn:      "document_id",
		resourceTable: "documents",
		resourceAccessClause: buildMembershipReadClause(
			"r.owner_user_id",
			"r.visibility_scope",
			"r.project_id",
			"%s",
			false,
		),
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
		resourceAccessClause: buildMembershipReadClause(
			"d.owner_user_id",
			"d.visibility_scope",
			"d.project_id",
			"%s",
			false,
		),
		includeActorParam: true,
	}
)

// PinEngramToSession pins an engram when both session and engram are visible.
func PinEngramToSession(
	ctx context.Context,
	db Queryer,
	input ChatPinEngramInput,
) (*models.PinnedEngramRecord, error) {
	record, err := pinResourceRecord(
		ctx,
		db,
		engramMutationInput(input.resourceInput()),
		pinnedEngramRecordFromRow,
	)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, nil
	}
	if err := emitEngramPinAuditEvent(ctx, db, input, "engram.pin"); err != nil {
		return nil, err
	}
	return record, nil
}

// PinDocumentToSession pins a document when both session and document are visible.
func PinDocumentToSession(
	ctx context.Context,
	db Queryer,
	input ChatPinDocumentInput,
) (*models.PinnedDocumentRecord, error) {
	return pinResourceRecord(
		ctx,
		db,
		documentMutationInput(input.resourceInput()),
		pinnedDocumentRecordFromRow,
	)
}

// UnpinEngramFromSession removes an engram pin for visible sessions.
func UnpinEngramFromSession(
	ctx context.Context,
	db Queryer,
	input ChatPinEngramInput,
) (bool, error) {
	removed, err := unpinResourceFromSession(ctx, db, engramMutationInput(input.resourceInput()))
	if err != nil {
		return false, err
	}
	if !removed {
		return false, nil
	}
	if err := emitEngramPinAuditEvent(ctx, db, input, "engram.unpin"); err != nil {
		return false, err
	}
	return true, nil
}

// UnpinDocumentFromSession removes a document pin for visible sessions.
func UnpinDocumentFromSession(
	ctx context.Context,
	db Queryer,
	input ChatPinDocumentInput,
) (bool, error) {
	return unpinResourceFromSession(ctx, db, documentMutationInput(input.resourceInput()))
}

// ListPinnedEngrams returns pinned engram rows for a visible session.
func ListPinnedEngrams(
	ctx context.Context,
	db Queryer,
	input ChatPinnedListInput,
) ([]models.PinnedEngramRecord, error) {
	return listPinnedResourceRecords(ctx, db, engramPinListConfig, input, pinnedEngramRecordFromRow)
}

// ListPinnedEngramSummaries returns visible engram summaries from session pins.
func ListPinnedEngramSummaries(
	ctx context.Context,
	db Queryer,
	input ChatPinnedListInput,
) ([]models.EngramSummary, error) {
	sessionAccessClause := buildMembershipReadClause(
		"s.owner_user_id",
		"s.visibility_scope",
		"s.project_id",
		"$2",
		false,
	)
	engramAccessClause := buildMembershipReadClause(
		"e.owner_user_id",
		"e.visibility_scope",
		"e.project_id",
		"$3",
		true,
	)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
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
			AND %s
			AND e.deleted_at IS NULL
			AND %s
		ORDER BY p.created_at ASC
		`,
			sessionAccessClause,
			engramAccessClause,
		),
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
	return listPinnedResourceRecords(ctx, db, documentPinListConfig, input, pinnedDocumentRecordFromRow)
}

func pinResourceToSession(
	ctx context.Context,
	db Queryer,
	input pinnedResourceMutationInput,
) (*pinnedResourceRow, error) {
	config := input.Config
	resourceAccessClause := config.resourceAccessClause
	if strings.Contains(resourceAccessClause, "%s") {
		resourceAccessClause = fmt.Sprintf(resourceAccessClause, pgxPlaceholder(4))
	}
	sessionAccessClause := buildMembershipReadClause(
		"s.owner_user_id",
		"s.visibility_scope",
		"s.project_id",
		"$2",
		false,
	)
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
					AND %s
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
			sessionAccessClause,
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
		input.Resource.SessionID,
		input.Resource.ActorUserID,
		input.Resource.ResourceID,
		input.Resource.ActorUserID,
		input.Resource.ActorUserID,
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
	input pinnedResourceMutationInput,
) (bool, error) {
	config := input.Config
	sessionAccessClause := buildMembershipReadClause(
		"s.owner_user_id",
		"s.visibility_scope",
		"s.project_id",
		"$3",
		false,
	)
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
				AND %s
			RETURNING p.session_id
			`,
			config.table,
			config.idColumn,
			sessionAccessClause,
		),
		input.Resource.SessionID,
		input.Resource.ResourceID,
		input.Resource.ActorUserID,
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
	sessionAccessClause := buildMembershipReadClause(
		"s.owner_user_id",
		"s.visibility_scope",
		"s.project_id",
		"$2",
		false,
	)

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
				AND %s
				AND %s
			ORDER BY p.created_at ASC
			`,
			config.idColumn,
			config.table,
			config.joinSQL,
			sessionAccessClause,
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

func (input ChatPinEngramInput) resourceInput() chatPinResourceInput {
	return chatPinResourceInput{
		SessionID:   input.SessionID,
		ResourceID:  input.EngramID,
		ActorUserID: input.ActorUserID,
	}
}

func (input ChatPinDocumentInput) resourceInput() chatPinResourceInput {
	return chatPinResourceInput{
		SessionID:   input.SessionID,
		ResourceID:  input.DocumentID,
		ActorUserID: input.ActorUserID,
	}
}

func engramMutationInput(resource chatPinResourceInput) pinnedResourceMutationInput {
	return pinnedResourceMutationInput{
		Config:   engramPinMutationConfig,
		Resource: resource,
	}
}

func documentMutationInput(resource chatPinResourceInput) pinnedResourceMutationInput {
	return pinnedResourceMutationInput{
		Config:   documentPinMutationConfig,
		Resource: resource,
	}
}

func pinResourceRecord[T any](
	ctx context.Context,
	db Queryer,
	input pinnedResourceMutationInput,
	mapper func(pinnedResourceRow) T,
) (*T, error) {
	row, err := pinResourceToSession(ctx, db, input)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	record := mapper(*row)
	return &record, nil
}

func listPinnedResourceRecords[T any](
	ctx context.Context,
	db Queryer,
	config pinnedResourceListConfig,
	input ChatPinnedListInput,
	mapper func(pinnedResourceRow) T,
) ([]T, error) {
	rows, err := listPinnedResources(ctx, db, config, input)
	if err != nil {
		return nil, err
	}
	return mapPinnedResourceRows(rows, mapper), nil
}

func mapPinnedResourceRows[T any](
	rows []pinnedResourceRow,
	mapper func(pinnedResourceRow) T,
) []T {
	records := make([]T, 0, len(rows))
	for _, row := range rows {
		records = append(records, mapper(row))
	}
	return records
}

func pinnedEngramRecordFromRow(row pinnedResourceRow) models.PinnedEngramRecord {
	return models.PinnedEngramRecord{
		SessionID:      row.SessionID,
		EngramID:       row.ResourceID,
		PinnedByUserID: row.PinnedByUserID,
		CreatedAt:      row.CreatedAt,
	}
}

func pinnedDocumentRecordFromRow(row pinnedResourceRow) models.PinnedDocumentRecord {
	return models.PinnedDocumentRecord{
		SessionID:      row.SessionID,
		DocumentID:     row.ResourceID,
		PinnedByUserID: row.PinnedByUserID,
		CreatedAt:      row.CreatedAt,
	}
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

func emitEngramPinAuditEvent(
	ctx context.Context,
	db Queryer,
	input ChatPinEngramInput,
	eventType string,
) error {
	projectID, err := lookupSessionProjectID(ctx, db, input.SessionID)
	if err != nil || strings.TrimSpace(projectID) == "" {
		return err
	}
	actorUserID := input.ActorUserID
	targetEngramID := input.EngramID
	_, err = CreateProjectAuditEvent(
		ctx,
		db,
		ProjectAuditEventCreateInput{
			ProjectID:      projectID,
			ActorUserID:    &actorUserID,
			EventType:      eventType,
			TargetType:     "engram",
			TargetEngramID: &targetEngramID,
			Metadata: map[string]any{
				"session_id": input.SessionID.String(),
			},
		},
	)
	return err
}

func lookupSessionProjectID(ctx context.Context, db Queryer, sessionID uuid.UUID) (string, error) {
	row := db.QueryRow(
		ctx,
		`
		SELECT project_id
		FROM chat_sessions
		WHERE session_id = $1
		LIMIT 1
		`,
		sessionID,
	)
	var projectID string
	if err := row.Scan(&projectID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return projectID, nil
}

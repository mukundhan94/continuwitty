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

const chatSessionColumns = `
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
	updated_at
`

var (
	newChatUUID = uuid.New
	nowChatUTC  = func() time.Time { return time.Now().UTC() }
)

// ChatSessionCreateInput captures create-session dependencies.
type ChatSessionCreateInput struct {
	OwnerUserID uuid.UUID
	Payload     models.ChatSessionCreateRequest
}

// ChatSessionListInput captures list-session filters.
type ChatSessionListInput struct {
	ActorUserID uuid.UUID
	ProjectID   *string
	Limit       int
	Offset      int
}

// ChatSessionGetInput captures get-session lookup parameters.
type ChatSessionGetInput struct {
	SessionID   uuid.UUID
	ActorUserID uuid.UUID
}

// ChatSessionUpdateInput captures update-session request context.
type ChatSessionUpdateInput struct {
	SessionID   uuid.UUID
	ActorUserID uuid.UUID
	Payload     models.ChatSessionUpdateRequest
}

// CreateChatSession persists a new chat session row.
func CreateChatSession(
	ctx context.Context,
	db Queryer,
	input ChatSessionCreateInput,
) (*models.ChatSessionRecord, error) {
	payload, err := normalizeCreatePayload(input.Payload)
	if err != nil {
		return nil, err
	}

	sessionID := newChatUUID()
	now := nowChatUTC()
	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			INSERT INTO chat_sessions (
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
				updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $15)
			RETURNING %s
			`,
			chatSessionColumns,
		),
		sessionID,
		input.OwnerUserID,
		payload.ProjectID,
		payload.Title,
		string(payload.Provider),
		payload.ModelID,
		payload.SystemPrompt,
		string(payload.VisibilityScope),
		payload.AutosaveEnabled,
		string(payload.AutosaveStrategy),
		payload.AutosaveIntervalMinutes,
		payload.AutosaveMinMessages,
		payload.RetentionDays,
		payload.RetentionMaxSnapshots,
		now,
	)
	record, err := scanChatSessionRecord(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ListChatSessions returns visible chat sessions for an actor.
func ListChatSessions(
	ctx context.Context,
	db Queryer,
	input ChatSessionListInput,
) ([]models.ChatSessionRecord, error) {
	query := fmt.Sprintf(
		`
		SELECT %s
		FROM chat_sessions
		WHERE
			deleted_at IS NULL
			AND (owner_user_id = $1 OR visibility_scope = 'project')
		`,
		chatSessionColumns,
	)
	params := []any{input.ActorUserID}
	if input.ProjectID != nil && *input.ProjectID != "" {
		query += " AND project_id = $2"
		params = append(params, *input.ProjectID)
	}
	query += fmt.Sprintf(
		" ORDER BY created_at DESC LIMIT %s OFFSET %s",
		pgxPlaceholder(len(params)+1),
		pgxPlaceholder(len(params)+2),
	)
	params = append(params, input.Limit, input.Offset)

	rows, err := db.Query(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.ChatSessionRecord, 0)
	for rows.Next() {
		record, scanErr := scanChatSessionRecord(rows)
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

// GetChatSession returns a visible chat session by ID.
func GetChatSession(
	ctx context.Context,
	db Queryer,
	input ChatSessionGetInput,
) (*models.ChatSessionRecord, error) {
	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			SELECT %s
			FROM chat_sessions
			WHERE
				session_id = $1
				AND deleted_at IS NULL
				AND (owner_user_id = $2 OR visibility_scope = 'project')
			`,
			chatSessionColumns,
		),
		input.SessionID,
		input.ActorUserID,
	)
	record, err := scanChatSessionRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetChatSessionAdminRecord returns session ownership/deletion state without actor scoping.
func GetChatSessionAdminRecord(
	ctx context.Context,
	db Queryer,
	sessionID uuid.UUID,
) (*models.ChatSessionAdminRecord, error) {
	row := db.QueryRow(
		ctx,
		`
		SELECT
			session_id,
			owner_user_id,
			project_id,
			deleted_at
		FROM chat_sessions
		WHERE session_id = $1
		LIMIT 1
		`,
		sessionID,
	)
	var record models.ChatSessionAdminRecord
	err := row.Scan(
		&record.SessionID,
		&record.OwnerUserID,
		&record.ProjectID,
		&record.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateChatSession updates owner-scoped mutable chat session fields.
func UpdateChatSession(
	ctx context.Context,
	db Queryer,
	input ChatSessionUpdateInput,
) (*models.ChatSessionRecord, error) {
	payload, err := normalizeUpdatePayload(input.Payload)
	if err != nil {
		return nil, err
	}

	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			UPDATE chat_sessions
			SET
				title = COALESCE($1, title),
				provider = COALESCE($2, provider),
				model_id = COALESCE($3, model_id),
				system_prompt = COALESCE($4, system_prompt),
				visibility_scope = COALESCE($5, visibility_scope),
				autosave_enabled = COALESCE($6, autosave_enabled),
				autosave_strategy = COALESCE($7, autosave_strategy),
				autosave_interval_minutes = COALESCE($8, autosave_interval_minutes),
				autosave_min_messages = COALESCE($9, autosave_min_messages),
				retention_days = COALESCE($10, retention_days),
				retention_max_snapshots = COALESCE($11, retention_max_snapshots),
				updated_at = $12
			WHERE
				session_id = $13
				AND owner_user_id = $14
				AND deleted_at IS NULL
			RETURNING %s
			`,
			chatSessionColumns,
		),
		payload.Title,
		payload.Provider,
		payload.ModelID,
		payload.SystemPrompt,
		payload.VisibilityScope,
		payload.AutosaveEnabled,
		payload.AutosaveStrategy,
		payload.AutosaveIntervalMinutes,
		payload.AutosaveMinMessages,
		payload.RetentionDays,
		payload.RetentionMaxSnapshots,
		nowChatUTC(),
		input.SessionID,
		input.ActorUserID,
	)
	record, err := scanChatSessionRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func normalizeCreatePayload(payload models.ChatSessionCreateRequest) (models.ChatSessionCreateRequest, error) {
	if payload.Provider == "" {
		payload.Provider = models.ChatProviderOpenAI
	}
	if payload.ModelID == "" {
		payload.ModelID = "gpt-4o-mini"
	}
	if payload.VisibilityScope == "" {
		payload.VisibilityScope = models.VisibilityScopePrivate
	}
	if payload.AutosaveStrategy == "" {
		payload.AutosaveStrategy = models.ChatAutosaveStrategyOff
	}
	if payload.AutosaveIntervalMinutes == 0 {
		payload.AutosaveIntervalMinutes = 30
	}
	if payload.AutosaveMinMessages == 0 {
		payload.AutosaveMinMessages = 6
	}
	if payload.RetentionDays == 0 {
		payload.RetentionDays = 30
	}
	if payload.RetentionMaxSnapshots == 0 {
		payload.RetentionMaxSnapshots = 60
	}
	if _, err := models.ParseChatProvider(string(payload.Provider)); err != nil {
		return models.ChatSessionCreateRequest{}, err
	}
	if _, err := models.ParseVisibilityScope(string(payload.VisibilityScope)); err != nil {
		return models.ChatSessionCreateRequest{}, err
	}
	if _, err := models.ParseChatAutosaveStrategy(string(payload.AutosaveStrategy)); err != nil {
		return models.ChatSessionCreateRequest{}, err
	}
	return payload, nil
}

func normalizeUpdatePayload(payload models.ChatSessionUpdateRequest) (chatSessionUpdateValues, error) {
	values := chatSessionUpdateValues{
		Title:                   payload.Title,
		ModelID:                 payload.ModelID,
		SystemPrompt:            payload.SystemPrompt,
		AutosaveEnabled:         payload.AutosaveEnabled,
		AutosaveIntervalMinutes: payload.AutosaveIntervalMinutes,
		AutosaveMinMessages:     payload.AutosaveMinMessages,
		RetentionDays:           payload.RetentionDays,
		RetentionMaxSnapshots:   payload.RetentionMaxSnapshots,
	}
	if payload.Provider != nil {
		if _, err := models.ParseChatProvider(string(*payload.Provider)); err != nil {
			return chatSessionUpdateValues{}, err
		}
		provider := string(*payload.Provider)
		values.Provider = &provider
	}
	if payload.VisibilityScope != nil {
		if _, err := models.ParseVisibilityScope(string(*payload.VisibilityScope)); err != nil {
			return chatSessionUpdateValues{}, err
		}
		scope := string(*payload.VisibilityScope)
		values.VisibilityScope = &scope
	}
	if payload.AutosaveStrategy != nil {
		if _, err := models.ParseChatAutosaveStrategy(string(*payload.AutosaveStrategy)); err != nil {
			return chatSessionUpdateValues{}, err
		}
		strategy := string(*payload.AutosaveStrategy)
		values.AutosaveStrategy = &strategy
	}
	return values, nil
}

func scanChatSessionRecord(row interface {
	Scan(dest ...any) error
}) (models.ChatSessionRecord, error) {
	var (
		record           models.ChatSessionRecord
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
	)
	if err != nil {
		return models.ChatSessionRecord{}, err
	}
	parsedProvider, err := models.ParseChatProvider(strings.TrimSpace(provider))
	if err != nil {
		return models.ChatSessionRecord{}, err
	}
	parsedVisibility, err := models.ParseVisibilityScope(strings.TrimSpace(visibility))
	if err != nil {
		return models.ChatSessionRecord{}, err
	}
	parsedAutosaveStrategy, err := models.ParseChatAutosaveStrategy(strings.TrimSpace(autosaveStrategy))
	if err != nil {
		return models.ChatSessionRecord{}, err
	}
	record.Provider = parsedProvider
	record.VisibilityScope = parsedVisibility
	record.AutosaveStrategy = parsedAutosaveStrategy
	return record, nil
}

type chatSessionUpdateValues struct {
	Title                   *string
	Provider                *string
	ModelID                 *string
	SystemPrompt            *string
	VisibilityScope         *string
	AutosaveEnabled         *bool
	AutosaveStrategy        *string
	AutosaveIntervalMinutes *int
	AutosaveMinMessages     *int
	RetentionDays           *int
	RetentionMaxSnapshots   *int
}

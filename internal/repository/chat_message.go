package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const chatMessageColumns = `
	message_id,
	session_id,
	role,
	content_text,
	provider,
	model_id,
	token_usage_json,
	used_engram_ids,
	created_at
`

var newChatMessageUUID = uuid.New

// ChatMessageMetadata captures optional write-time metadata for chat messages.
type ChatMessageMetadata struct {
	Provider       *string
	ModelID        *string
	TokenUsageJSON map[string]any
	UsedEngramIDs  []uuid.UUID
}

// ChatMessageCreateInput captures create-message dependencies.
type ChatMessageCreateInput struct {
	SessionID   uuid.UUID
	ActorUserID uuid.UUID
	Role        string
	ContentText string
	Metadata    *ChatMessageMetadata
}

// ChatMessageListInput captures list-message filters.
type ChatMessageListInput struct {
	SessionID   uuid.UUID
	ActorUserID uuid.UUID
	Limit       int
	Offset      int
}

// SessionLinkedEngramsListInput captures linked-engram listing filters.
type SessionLinkedEngramsListInput struct {
	SessionID   uuid.UUID
	ActorUserID uuid.UUID
	Limit       int
	Offset      int
}

// SessionMessageRoleCountInput captures role count lookup filters.
type SessionMessageRoleCountInput struct {
	SessionID   uuid.UUID
	ActorUserID uuid.UUID
	Role        string
}

// SessionAutosaveDeleteInput captures autosave prune request context.
type SessionAutosaveDeleteInput struct {
	SessionID   uuid.UUID
	ActorUserID uuid.UUID
	EngramIDs   []uuid.UUID
}

// CreateChatMessage writes a message when the actor can access the session.
func CreateChatMessage(
	ctx context.Context,
	db Queryer,
	input ChatMessageCreateInput,
) (*models.ChatMessageRecord, error) {
	metadata := resolveMessageMetadata(input.Metadata)
	sessionAccessClause := buildMembershipReadClause(membershipReadClauseInput{ownerColumn: "s.owner_user_id", visibilityColumn: "s.visibility_scope", projectColumn: "s.project_id", actorPlaceholder: "$10", includeOwnerless: false})
	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			INSERT INTO chat_messages (
				message_id,
				session_id,
				role,
				content_text,
				provider,
				model_id,
				token_usage_json,
				used_engram_ids,
				created_at
			)
			SELECT
				$1,
				s.session_id,
				$2,
				$3,
				$4,
				$5,
				$6,
				$7,
				$8
			FROM chat_sessions s
			WHERE
				s.session_id = $9
				AND s.deleted_at IS NULL
				AND %s
			RETURNING %s
			`,
			sessionAccessClause,
			chatMessageColumns,
		),
		newChatMessageUUID(),
		input.Role,
		input.ContentText,
		metadata.Provider,
		metadata.ModelID,
		metadata.TokenUsageJSON,
		metadata.UsedEngramIDs,
		nowChatUTC(),
		input.SessionID,
		input.ActorUserID,
	)
	record, err := scanChatMessageRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ListChatMessages returns visible messages for a chat session.
func ListChatMessages(
	ctx context.Context,
	db Queryer,
	input ChatMessageListInput,
) ([]models.ChatMessageRecord, error) {
	sessionAccessClause := buildMembershipReadClause(membershipReadClauseInput{ownerColumn: "s.owner_user_id", visibilityColumn: "s.visibility_scope", projectColumn: "s.project_id", actorPlaceholder: "$2", includeOwnerless: false})
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`
		SELECT
			m.message_id,
			m.session_id,
			m.role,
			m.content_text,
			m.provider,
			m.model_id,
			m.token_usage_json,
			m.used_engram_ids,
			m.created_at
		FROM chat_messages m
		JOIN chat_sessions s
		  ON s.session_id = m.session_id
		WHERE
			m.session_id = $1
			AND s.deleted_at IS NULL
			AND %s
		ORDER BY m.created_at ASC
		LIMIT $3 OFFSET $4
		`,
			sessionAccessClause,
		),
		input.SessionID,
		input.ActorUserID,
		input.Limit,
		input.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.ChatMessageRecord, 0)
	for rows.Next() {
		record, scanErr := scanChatMessageRecord(rows)
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

// ListSessionLinkedEngrams returns engrams created from the session.
func ListSessionLinkedEngrams(
	ctx context.Context,
	db Queryer,
	input SessionLinkedEngramsListInput,
) ([]models.EngramSummary, error) {
	sessionAccessClause := buildMembershipReadClause(membershipReadClauseInput{ownerColumn: "s.owner_user_id", visibilityColumn: "s.visibility_scope", projectColumn: "s.project_id", actorPlaceholder: "$2", includeOwnerless: false})
	engramAccessClause := buildMembershipReadClause(membershipReadClauseInput{ownerColumn: "e.owner_user_id", visibilityColumn: "e.visibility_scope", projectColumn: "e.project_id", actorPlaceholder: "$3", includeOwnerless: true})
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
		FROM engrams e
		JOIN chat_sessions s
		  ON s.session_id = e.source_session_id
		WHERE
			s.session_id = $1
			AND s.deleted_at IS NULL
			AND %s
			AND e.deleted_at IS NULL
			AND %s
		ORDER BY e.created_at DESC
		LIMIT $4 OFFSET $5
		`,
			sessionAccessClause,
			engramAccessClause,
		),
		input.SessionID,
		input.ActorUserID,
		input.ActorUserID,
		input.Limit,
		input.Offset,
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

// CountSessionMessagesByRole counts messages in a session by role.
func CountSessionMessagesByRole(
	ctx context.Context,
	db Queryer,
	input SessionMessageRoleCountInput,
) (int, error) {
	sessionAccessClause := buildMembershipReadClause(membershipReadClauseInput{ownerColumn: "s.owner_user_id", visibilityColumn: "s.visibility_scope", projectColumn: "s.project_id", actorPlaceholder: "$3", includeOwnerless: false})
	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
		SELECT COUNT(*)::INT
		FROM chat_messages m
		JOIN chat_sessions s
		  ON s.session_id = m.session_id
		WHERE
			m.session_id = $1
			AND m.role = $2
			AND s.deleted_at IS NULL
			AND %s
		`,
			sessionAccessClause,
		),
		input.SessionID,
		input.Role,
		input.ActorUserID,
	)
	count := 0
	err := row.Scan(&count)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteSessionAutosaveEngrams deletes autosave snapshot engrams scoped to a session.
func DeleteSessionAutosaveEngrams(
	ctx context.Context,
	db Queryer,
	input SessionAutosaveDeleteInput,
) ([]uuid.UUID, error) {
	if len(input.EngramIDs) == 0 {
		return []uuid.UUID{}, nil
	}
	sessionAccessClause := buildMembershipReadClause(membershipReadClauseInput{ownerColumn: "s.owner_user_id", visibilityColumn: "s.visibility_scope", projectColumn: "s.project_id", actorPlaceholder: "$3", includeOwnerless: false})
	engramAccessClause := buildMembershipReadClause(membershipReadClauseInput{ownerColumn: "e.owner_user_id", visibilityColumn: "e.visibility_scope", projectColumn: "e.project_id", actorPlaceholder: "$4", includeOwnerless: true})
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`
		DELETE FROM engrams e
		USING chat_sessions s
		WHERE
			e.source_session_id = s.session_id
			AND s.session_id = $1
			AND e.engram_id = ANY($2::UUID[])
			AND e.deleted_at IS NULL
			AND e.tags @> ARRAY['autosave_snapshot']::TEXT[]
			AND s.deleted_at IS NULL
			AND %s
			AND %s
		RETURNING e.engram_id
		`,
			sessionAccessClause,
			engramAccessClause,
		),
		input.SessionID,
		input.EngramIDs,
		input.ActorUserID,
		input.ActorUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	removed := make([]uuid.UUID, 0)
	for rows.Next() {
		var engramID uuid.UUID
		if scanErr := rows.Scan(&engramID); scanErr != nil {
			return nil, scanErr
		}
		removed = append(removed, engramID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return removed, nil
}

func resolveMessageMetadata(metadata *ChatMessageMetadata) ChatMessageMetadata {
	resolved := ChatMessageMetadata{
		TokenUsageJSON: map[string]any{},
		UsedEngramIDs:  []uuid.UUID{},
	}
	if metadata == nil {
		return resolved
	}
	resolved.Provider = metadata.Provider
	resolved.ModelID = metadata.ModelID
	if metadata.TokenUsageJSON != nil {
		resolved.TokenUsageJSON = metadata.TokenUsageJSON
	}
	if metadata.UsedEngramIDs != nil {
		resolved.UsedEngramIDs = metadata.UsedEngramIDs
	}
	return resolved
}

func scanChatMessageRecord(row interface {
	Scan(dest ...any) error
}) (models.ChatMessageRecord, error) {
	var (
		record        models.ChatMessageRecord
		provider      *string
		modelID       *string
		tokenUsageRaw any
		usedEngramIDs []uuid.UUID
	)
	err := row.Scan(
		&record.MessageID,
		&record.SessionID,
		&record.Role,
		&record.ContentText,
		&provider,
		&modelID,
		&tokenUsageRaw,
		&usedEngramIDs,
		&record.CreatedAt,
	)
	if err != nil {
		return models.ChatMessageRecord{}, err
	}
	tokenUsageJSON, err := decodeTokenUsageJSON(tokenUsageRaw)
	if err != nil {
		return models.ChatMessageRecord{}, err
	}
	if usedEngramIDs == nil {
		usedEngramIDs = []uuid.UUID{}
	}
	record.Provider = provider
	record.ModelID = modelID
	record.TokenUsageJSON = tokenUsageJSON
	record.UsedEngramIDs = usedEngramIDs
	return record, nil
}

func decodeTokenUsageJSON(value any) (map[string]any, error) {
	switch typed := value.(type) {
	case nil:
		return map[string]any{}, nil
	case map[string]any:
		if typed == nil {
			return map[string]any{}, nil
		}
		return typed, nil
	case []byte:
		return unmarshalTokenUsageJSON(typed)
	case string:
		return unmarshalTokenUsageJSON([]byte(typed))
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("marshal token usage json: %w", err)
		}
		return unmarshalTokenUsageJSON(encoded)
	}
}

func unmarshalTokenUsageJSON(encoded []byte) (map[string]any, error) {
	if len(bytes.TrimSpace(encoded)) == 0 {
		return map[string]any{}, nil
	}
	if bytes.Equal(bytes.TrimSpace(encoded), []byte("null")) {
		return map[string]any{}, nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("decode token usage json: %w", err)
	}
	if decoded == nil {
		return map[string]any{}, nil
	}
	return decoded, nil
}

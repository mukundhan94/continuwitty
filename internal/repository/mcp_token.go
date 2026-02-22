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

const mcpTokenColumns = `
	token_id,
	owner_user_id,
	name,
	scope,
	allowed_tools,
	allowed_project_ids,
	token_secret_hash,
	token_secret_hint,
	expires_at,
	last_used_at,
	revoked_at,
	created_at
`

var (
	newMCPTokenUUID = uuid.New
	nowMCPTokenUTC  = func() time.Time { return time.Now().UTC() }
)

// MCPTokenCreateInput captures create-token dependencies.
type MCPTokenCreateInput struct {
	TokenID           *uuid.UUID
	OwnerUserID       uuid.UUID
	Name              string
	Scope             models.MCPTokenScope
	AllowedTools      []string
	AllowedProjectIDs []string
	TokenSecretHash   string
	TokenSecretHint   string
	ExpiresAt         time.Time
}

// MCPTokenListInput captures list-token filters.
type MCPTokenListInput struct {
	OwnerUserID uuid.UUID
	Limit       int
	Offset      int
}

// MCPTokenRevokeInput captures revoke-token authorization dependencies.
type MCPTokenRevokeInput struct {
	TokenID     uuid.UUID
	OwnerUserID uuid.UUID
}

// CreateMCPToken persists a new MCP token record.
func CreateMCPToken(
	ctx context.Context,
	db Queryer,
	input MCPTokenCreateInput,
) (*models.MCPTokenRecord, error) {
	if _, err := models.ParseMCPTokenScope(string(input.Scope)); err != nil {
		return nil, err
	}
	createdTokenID := newMCPTokenUUID()
	if input.TokenID != nil {
		createdTokenID = *input.TokenID
	}
	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			INSERT INTO mcp_tokens (
				token_id,
				owner_user_id,
				name,
				scope,
				allowed_tools,
				allowed_project_ids,
				token_secret_hash,
				token_secret_hint,
				expires_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING %s
			`,
			mcpTokenColumns,
		),
		createdTokenID,
		input.OwnerUserID,
		input.Name,
		string(input.Scope),
		normalizeStringSlice(input.AllowedTools),
		normalizeStringSlice(input.AllowedProjectIDs),
		input.TokenSecretHash,
		input.TokenSecretHint,
		input.ExpiresAt,
	)
	record, err := scanMCPTokenRecord(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ListMCPTokens returns MCP tokens for an owner ordered by most-recent creation.
func ListMCPTokens(
	ctx context.Context,
	db Queryer,
	input MCPTokenListInput,
) ([]models.MCPTokenRecord, error) {
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`
			SELECT %s
			FROM mcp_tokens
			WHERE owner_user_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
			`,
			mcpTokenColumns,
		),
		input.OwnerUserID,
		input.Limit,
		input.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]models.MCPTokenRecord, 0)
	for rows.Next() {
		record, scanErr := scanMCPTokenRecord(rows)
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

// GetMCPTokenByID returns a token row by ID.
func GetMCPTokenByID(
	ctx context.Context,
	db Queryer,
	tokenID uuid.UUID,
) (*models.MCPTokenRecord, error) {
	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			SELECT %s
			FROM mcp_tokens
			WHERE token_id = $1
			`,
			mcpTokenColumns,
		),
		tokenID,
	)
	record, err := scanMCPTokenRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// RevokeMCPToken revokes a token for its owner.
func RevokeMCPToken(
	ctx context.Context,
	db Queryer,
	input MCPTokenRevokeInput,
) (*models.MCPTokenRecord, error) {
	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			UPDATE mcp_tokens
			SET revoked_at = COALESCE(revoked_at, now())
			WHERE token_id = $1
			  AND owner_user_id = $2
			RETURNING %s
			`,
			mcpTokenColumns,
		),
		input.TokenID,
		input.OwnerUserID,
	)
	record, err := scanMCPTokenRecord(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// TouchMCPTokenLastUsed updates token usage timestamp.
func TouchMCPTokenLastUsed(
	ctx context.Context,
	db Queryer,
	tokenID uuid.UUID,
) error {
	rows, err := db.Query(
		ctx,
		`
		UPDATE mcp_tokens
		SET last_used_at = $1
		WHERE token_id = $2
		`,
		nowMCPTokenUTC(),
		tokenID,
	)
	if err != nil {
		return err
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func scanMCPTokenRecord(row interface {
	Scan(dest ...any) error
}) (models.MCPTokenRecord, error) {
	var (
		record            models.MCPTokenRecord
		scope             string
		allowedTools      []string
		allowedProjectIDs []string
	)
	err := row.Scan(
		&record.TokenID,
		&record.OwnerUserID,
		&record.Name,
		&scope,
		&allowedTools,
		&allowedProjectIDs,
		&record.TokenSecretHash,
		&record.TokenSecretHint,
		&record.ExpiresAt,
		&record.LastUsedAt,
		&record.RevokedAt,
		&record.CreatedAt,
	)
	if err != nil {
		return models.MCPTokenRecord{}, err
	}
	parsedScope, err := models.ParseMCPTokenScope(strings.TrimSpace(scope))
	if err != nil {
		return models.MCPTokenRecord{}, err
	}
	record.Scope = parsedScope
	record.AllowedTools = normalizeStringSlice(allowedTools)
	record.AllowedProjectIDs = normalizeStringSlice(allowedProjectIDs)
	return record, nil
}

func normalizeStringSlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

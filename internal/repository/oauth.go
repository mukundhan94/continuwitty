package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const oauthClientColumns = `
	client_id,
	client_name,
	redirect_uris,
	grant_types,
	response_types,
	token_endpoint_auth_method,
	client_secret_hash,
	metadata_json,
	created_at
`

const oauthAuthorizationCodeColumns = `
	code_id,
	code_hash,
	client_id,
	user_id,
	redirect_uri,
	code_challenge,
	code_challenge_method,
	requested_scope,
	resource,
	expires_at,
	consumed_at,
	created_at
`

// OAuthClientCreateInput captures dependencies for OAuth client creation.
type OAuthClientCreateInput struct {
	ClientID                string
	ClientName              string
	RedirectURIs            []string
	GrantTypes              []string
	ResponseTypes           []string
	TokenEndpointAuthMethod string
	ClientSecretHash        *string
	MetadataJSON            map[string]any
}

// OAuthAuthorizationCodeCreateInput captures dependencies for auth-code creation.
type OAuthAuthorizationCodeCreateInput struct {
	CodeID              uuid.UUID
	CodeHash            string
	ClientID            string
	UserID              uuid.UUID
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	RequestedScope      string
	Resource            *string
	ExpiresAt           time.Time
}

// OAuthAuthorizationCodeConsumeInput captures auth-code consume dependencies.
type OAuthAuthorizationCodeConsumeInput struct {
	CodeID     uuid.UUID
	ConsumedAt time.Time
}

// CreateOAuthClient inserts a new OAuth client row.
func CreateOAuthClient(
	ctx context.Context,
	db Queryer,
	input OAuthClientCreateInput,
) (*models.OAuthClientRecord, error) {
	metadataJSON, err := marshalJSON(orEmptyMap(input.MetadataJSON))
	if err != nil {
		return nil, err
	}
	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			INSERT INTO oauth_clients (
				client_id,
				client_name,
				redirect_uris,
				grant_types,
				response_types,
				token_endpoint_auth_method,
				client_secret_hash,
				metadata_json
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
			RETURNING %s
			`,
			oauthClientColumns,
		),
		input.ClientID,
		input.ClientName,
		normalizeStringSlice(input.RedirectURIs),
		normalizeStringSlice(input.GrantTypes),
		normalizeStringSlice(input.ResponseTypes),
		input.TokenEndpointAuthMethod,
		input.ClientSecretHash,
		metadataJSON,
	)
	record, err := scanOAuthClientRecord(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetOAuthClient fetches an OAuth client by ID.
func GetOAuthClient(
	ctx context.Context,
	db Queryer,
	clientID string,
) (*models.OAuthClientRecord, error) {
	return queryOptionalRecord(
		ctx,
		db,
		fmt.Sprintf(
			`
			SELECT %s
			FROM oauth_clients
			WHERE client_id = $1
			`,
			oauthClientColumns,
		),
		scanOAuthClientRecord,
		clientID,
	)
}

// CreateOAuthAuthorizationCode inserts a new OAuth authorization code row.
func CreateOAuthAuthorizationCode(
	ctx context.Context,
	db Queryer,
	input OAuthAuthorizationCodeCreateInput,
) (*models.OAuthAuthorizationCodeRecord, error) {
	row := db.QueryRow(
		ctx,
		fmt.Sprintf(
			`
			INSERT INTO oauth_authorization_codes (
				code_id,
				code_hash,
				client_id,
				user_id,
				redirect_uri,
				code_challenge,
				code_challenge_method,
				requested_scope,
				resource,
				expires_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING %s
			`,
			oauthAuthorizationCodeColumns,
		),
		input.CodeID,
		input.CodeHash,
		input.ClientID,
		input.UserID,
		input.RedirectURI,
		input.CodeChallenge,
		input.CodeChallengeMethod,
		input.RequestedScope,
		input.Resource,
		input.ExpiresAt,
	)
	record, err := scanOAuthAuthorizationCodeRecord(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetOAuthAuthorizationCodeByHash fetches an auth code by hash.
func GetOAuthAuthorizationCodeByHash(
	ctx context.Context,
	db Queryer,
	codeHash string,
) (*models.OAuthAuthorizationCodeRecord, error) {
	return queryOptionalRecord(
		ctx,
		db,
		fmt.Sprintf(
			`
			SELECT %s
			FROM oauth_authorization_codes
			WHERE code_hash = $1
			`,
			oauthAuthorizationCodeColumns,
		),
		scanOAuthAuthorizationCodeRecord,
		codeHash,
	)
}

// ConsumeOAuthAuthorizationCode marks an auth code as consumed when still active.
func ConsumeOAuthAuthorizationCode(
	ctx context.Context,
	db Queryer,
	input OAuthAuthorizationCodeConsumeInput,
) (*models.OAuthAuthorizationCodeRecord, error) {
	return queryOptionalRecord(
		ctx,
		db,
		fmt.Sprintf(
			`
			UPDATE oauth_authorization_codes
			SET consumed_at = $1
			WHERE code_id = $2
			  AND consumed_at IS NULL
			RETURNING %s
			`,
			oauthAuthorizationCodeColumns,
		),
		scanOAuthAuthorizationCodeRecord,
		input.ConsumedAt,
		input.CodeID,
	)
}

func queryOptionalRecord[T any](
	ctx context.Context,
	db Queryer,
	query string,
	scan func(row interface {
		Scan(dest ...any) error
	}) (T, error),
	args ...any,
) (*T, error) {
	row := db.QueryRow(ctx, query, args...)
	record, err := scan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func scanOAuthClientRecord(row interface {
	Scan(dest ...any) error
}) (models.OAuthClientRecord, error) {
	var (
		record          models.OAuthClientRecord
		redirectURIs    []string
		grantTypes      []string
		responseTypes   []string
		metadataJSONRaw any
	)
	err := row.Scan(
		&record.ClientID,
		&record.ClientName,
		&redirectURIs,
		&grantTypes,
		&responseTypes,
		&record.TokenEndpointAuthMethod,
		&record.ClientSecretHash,
		&metadataJSONRaw,
		&record.CreatedAt,
	)
	if err != nil {
		return models.OAuthClientRecord{}, err
	}
	metadataJSON, err := decodeJSONObject(metadataJSONRaw)
	if err != nil {
		return models.OAuthClientRecord{}, err
	}
	record.RedirectURIs = normalizeStringSlice(redirectURIs)
	record.GrantTypes = normalizeStringSlice(grantTypes)
	record.ResponseTypes = normalizeStringSlice(responseTypes)
	record.MetadataJSON = metadataJSON
	return record, nil
}

func scanOAuthAuthorizationCodeRecord(row interface {
	Scan(dest ...any) error
}) (models.OAuthAuthorizationCodeRecord, error) {
	var record models.OAuthAuthorizationCodeRecord
	err := row.Scan(
		&record.CodeID,
		&record.CodeHash,
		&record.ClientID,
		&record.UserID,
		&record.RedirectURI,
		&record.CodeChallenge,
		&record.CodeChallengeMethod,
		&record.RequestedScope,
		&record.Resource,
		&record.ExpiresAt,
		&record.ConsumedAt,
		&record.CreatedAt,
	)
	if err != nil {
		return models.OAuthAuthorizationCodeRecord{}, err
	}
	if record.RequestedScope == "" {
		record.RequestedScope = ""
	}
	return record, nil
}

func decodeJSONObject(value any) (map[string]any, error) {
	switch typed := value.(type) {
	case nil:
		return map[string]any{}, nil
	case map[string]any:
		if typed == nil {
			return map[string]any{}, nil
		}
		return typed, nil
	case []byte:
		return unmarshalJSONObject(typed)
	case string:
		return unmarshalJSONObject([]byte(typed))
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("marshal oauth json: %w", err)
		}
		return unmarshalJSONObject(encoded)
	}
}

func unmarshalJSONObject(encoded []byte) (map[string]any, error) {
	if len(bytes.TrimSpace(encoded)) == 0 {
		return map[string]any{}, nil
	}
	if bytes.Equal(bytes.TrimSpace(encoded), []byte("null")) {
		return map[string]any{}, nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, fmt.Errorf("decode oauth json: %w", err)
	}
	if decoded == nil {
		return map[string]any{}, nil
	}
	return decoded, nil
}

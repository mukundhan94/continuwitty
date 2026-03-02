package models

import (
	"time"

	"github.com/google/uuid"
)

// OAuthClientRecord models persisted OAuth dynamic client metadata.
type OAuthClientRecord struct {
	ClientID                string         `json:"client_id"`
	ClientName              string         `json:"client_name"`
	RedirectURIs            []string       `json:"redirect_uris"`
	GrantTypes              []string       `json:"grant_types"`
	ResponseTypes           []string       `json:"response_types"`
	TokenEndpointAuthMethod string         `json:"token_endpoint_auth_method"`
	ClientSecretHash        *string        `json:"client_secret_hash,omitempty"`
	MetadataJSON            map[string]any `json:"metadata_json"`
	CreatedAt               time.Time      `json:"created_at"`
}

// OAuthAuthorizationCodeRecord models persisted authorization code rows.
type OAuthAuthorizationCodeRecord struct {
	CodeID              uuid.UUID  `json:"code_id"`
	CodeHash            string     `json:"code_hash"`
	ClientID            string     `json:"client_id"`
	UserID              uuid.UUID  `json:"user_id"`
	RedirectURI         string     `json:"redirect_uri"`
	CodeChallenge       string     `json:"code_challenge"`
	CodeChallengeMethod string     `json:"code_challenge_method"`
	RequestedScope      string     `json:"requested_scope"`
	Resource            *string    `json:"resource,omitempty"`
	ExpiresAt           time.Time  `json:"expires_at"`
	ConsumedAt          *time.Time `json:"consumed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

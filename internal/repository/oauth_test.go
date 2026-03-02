package repository

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestCreateOAuthClientReturnsCreatedRecord(t *testing.T) {
	createdAt := time.Date(2026, 2, 22, 15, 0, 0, 0, time.UTC)
	secretHash := "secret-hash"
	metadataJSON := map[string]any{"contacts": []any{"ops@example.com"}}
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: oauthClientRowValues(oauthClientRowFixture{
			ClientID:                "client-123",
			ClientName:              "Codex Agent",
			RedirectURIs:            []string{"https://example.com/callback"},
			GrantTypes:              []string{"authorization_code"},
			ResponseTypes:           []string{"code"},
			TokenEndpointAuthMethod: "none",
			ClientSecretHash:        &secretHash,
			MetadataJSON:            metadataJSON,
			CreatedAt:               createdAt,
		})},
	}

	record, err := CreateOAuthClient(
		context.Background(),
		db,
		OAuthClientCreateInput{
			ClientID:                "client-123",
			ClientName:              "Codex Agent",
			RedirectURIs:            []string{"https://example.com/callback"},
			GrantTypes:              []string{"authorization_code"},
			ResponseTypes:           []string{"code"},
			TokenEndpointAuthMethod: "none",
			ClientSecretHash:        &secretHash,
			MetadataJSON:            metadataJSON,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, "client-123", record.ClientID)
	requireEqual(t, "Codex Agent", record.ClientName)
	requireEqual(t, "none", record.TokenEndpointAuthMethod)
	if record.ClientSecretHash == nil || *record.ClientSecretHash != secretHash {
		t.Fatalf("expected client_secret_hash to round-trip")
	}
	if !reflect.DeepEqual(metadataJSON, record.MetadataJSON) {
		t.Fatalf("expected metadata %#v, got %#v", metadataJSON, record.MetadataJSON)
	}

	requireEqual(t, 1, len(db.queryRowArgs))
	if gotMetadata, ok := db.queryRowArgs[0][7].(string); !ok || !strings.Contains(gotMetadata, "contacts") {
		t.Fatalf("expected marshaled metadata json arg, got %#v", db.queryRowArgs[0][7])
	}
}

func TestGetOAuthClientReturnsNilWhenMissing(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}

	record, err := GetOAuthClient(context.Background(), db, "missing-client")
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil client when missing")
	}
}

func TestCreateOAuthAuthorizationCodeReturnsRecord(t *testing.T) {
	codeID := uuid.MustParse("00000000-0000-0000-0000-000000000a01")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000a02")
	expiresAt := time.Date(2026, 2, 22, 15, 10, 0, 0, time.UTC)
	createdAt := time.Date(2026, 2, 22, 15, 0, 0, 0, time.UTC)
	resource := "https://engram.local/mcp"
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: oauthAuthorizationCodeRowValues(oauthAuthorizationCodeRowFixture{
			CodeID:              codeID,
			CodeHash:            "hash-abc",
			ClientID:            "client-123",
			UserID:              userID,
			RedirectURI:         "https://example.com/callback",
			CodeChallenge:       "challenge",
			CodeChallengeMethod: "S256",
			RequestedScope:      "read write",
			Resource:            &resource,
			ExpiresAt:           expiresAt,
			ConsumedAt:          nil,
			CreatedAt:           createdAt,
		})},
	}

	record, err := CreateOAuthAuthorizationCode(
		context.Background(),
		db,
		OAuthAuthorizationCodeCreateInput{
			CodeID:              codeID,
			CodeHash:            "hash-abc",
			ClientID:            "client-123",
			UserID:              userID,
			RedirectURI:         "https://example.com/callback",
			CodeChallenge:       "challenge",
			CodeChallengeMethod: "S256",
			RequestedScope:      "read write",
			Resource:            &resource,
			ExpiresAt:           expiresAt,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, codeID, record.CodeID)
	requireEqual(t, "hash-abc", record.CodeHash)
	requireEqual(t, "S256", record.CodeChallengeMethod)
	if record.Resource == nil || *record.Resource != resource {
		t.Fatalf("expected resource to round-trip")
	}
	expectedArgs := []any{
		codeID,
		"hash-abc",
		"client-123",
		userID,
		"https://example.com/callback",
		"challenge",
		"S256",
		"read write",
		&resource,
		expiresAt,
	}
	if !reflect.DeepEqual(expectedArgs, db.queryRowArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}
}

func TestGetOAuthAuthorizationCodeByHashReturnsRecord(t *testing.T) {
	codeID := uuid.MustParse("00000000-0000-0000-0000-000000000a11")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000a12")
	expiresAt := time.Date(2026, 2, 22, 15, 20, 0, 0, time.UTC)
	createdAt := time.Date(2026, 2, 22, 15, 5, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: oauthAuthorizationCodeRowValues(oauthAuthorizationCodeRowFixture{
			CodeID:              codeID,
			CodeHash:            "hash-def",
			ClientID:            "client-123",
			UserID:              userID,
			RedirectURI:         "https://example.com/callback",
			CodeChallenge:       "challenge",
			CodeChallengeMethod: "S256",
			RequestedScope:      "",
			Resource:            nil,
			ExpiresAt:           expiresAt,
			ConsumedAt:          nil,
			CreatedAt:           createdAt,
		})},
	}

	record, err := GetOAuthAuthorizationCodeByHash(context.Background(), db, "hash-def")
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, codeID, record.CodeID)
	requireEqual(t, "", record.RequestedScope)
}

func TestConsumeOAuthAuthorizationCodeReturnsNilWhenAlreadyConsumed(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}

	record, err := ConsumeOAuthAuthorizationCode(
		context.Background(),
		db,
		OAuthAuthorizationCodeConsumeInput{
			CodeID:     uuid.MustParse("00000000-0000-0000-0000-000000000a21"),
			ConsumedAt: time.Date(2026, 2, 22, 15, 30, 0, 0, time.UTC),
		},
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil record when consume affected no rows")
	}
}

type oauthClientRowFixture struct {
	ClientID                string
	ClientName              string
	RedirectURIs            []string
	GrantTypes              []string
	ResponseTypes           []string
	TokenEndpointAuthMethod string
	ClientSecretHash        *string
	MetadataJSON            map[string]any
	CreatedAt               time.Time
}

func oauthClientRowValues(fixture oauthClientRowFixture) []any {
	var secretHashValue any
	if fixture.ClientSecretHash != nil {
		secretHashValue = *fixture.ClientSecretHash
	}
	return []any{
		fixture.ClientID,
		fixture.ClientName,
		fixture.RedirectURIs,
		fixture.GrantTypes,
		fixture.ResponseTypes,
		fixture.TokenEndpointAuthMethod,
		secretHashValue,
		fixture.MetadataJSON,
		fixture.CreatedAt,
	}
}

type oauthAuthorizationCodeRowFixture struct {
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
	ConsumedAt          *time.Time
	CreatedAt           time.Time
}

func oauthAuthorizationCodeRowValues(fixture oauthAuthorizationCodeRowFixture) []any {
	var resourceValue any
	if fixture.Resource != nil {
		resourceValue = *fixture.Resource
	}
	var consumedAtValue any
	if fixture.ConsumedAt != nil {
		consumedAtValue = *fixture.ConsumedAt
	}
	return []any{
		fixture.CodeID,
		fixture.CodeHash,
		fixture.ClientID,
		fixture.UserID,
		fixture.RedirectURI,
		fixture.CodeChallenge,
		fixture.CodeChallengeMethod,
		fixture.RequestedScope,
		resourceValue,
		fixture.ExpiresAt,
		consumedAtValue,
		fixture.CreatedAt,
	}
}

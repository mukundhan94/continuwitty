package repository

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestCreateMCPTokenUsesGeneratedIDAndReturnsRecord(t *testing.T) {
	tokenID := uuid.MustParse("00000000-0000-0000-0000-000000000901")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000902")
	expiresAt := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 2, 22, 14, 0, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowResult: &fakeRow{values: mcpTokenRowValues(
			tokenID,
			ownerUserID,
			"Agent write",
			"write",
			[]string{"chat.send", "engram.create"},
			[]string{"project-docs"},
			"hash-value",
			"...abcd",
			expiresAt,
			nil,
			nil,
			createdAt,
		)},
	}

	originalTokenUUID := newMCPTokenUUID
	newMCPTokenUUID = func() uuid.UUID { return tokenID }
	t.Cleanup(func() { newMCPTokenUUID = originalTokenUUID })

	record, err := CreateMCPToken(
		context.Background(),
		db,
		MCPTokenCreateInput{
			OwnerUserID:       ownerUserID,
			Name:              "Agent write",
			Scope:             models.MCPTokenScopeWrite,
			AllowedTools:      []string{"chat.send", "engram.create"},
			AllowedProjectIDs: []string{"project-docs"},
			TokenSecretHash:   "hash-value",
			TokenSecretHint:   "...abcd",
			ExpiresAt:         expiresAt,
		},
	)
	requireNoError(t, err)
	requireNotNil(t, record)
	requireEqual(t, tokenID, record.TokenID)
	requireEqual(t, models.MCPTokenScopeWrite, record.Scope)
	if !reflect.DeepEqual([]string{"chat.send", "engram.create"}, record.AllowedTools) {
		t.Fatalf("expected allowed_tools to round-trip, got %#v", record.AllowedTools)
	}
	requireEqual(t, 1, len(db.queryRowArgs))
	expectedArgs := []any{
		tokenID,
		ownerUserID,
		"Agent write",
		"write",
		[]string{"chat.send", "engram.create"},
		[]string{"project-docs"},
		"hash-value",
		"...abcd",
		expiresAt,
	}
	if !reflect.DeepEqual(expectedArgs, db.queryRowArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryRowArgs[0])
	}
}

func TestCreateMCPTokenRejectsInvalidScope(t *testing.T) {
	db := &fakeQueryer{}
	record, err := CreateMCPToken(
		context.Background(),
		db,
		MCPTokenCreateInput{
			OwnerUserID:     uuid.MustParse("00000000-0000-0000-0000-000000000911"),
			Name:            "Bad token",
			Scope:           models.MCPTokenScope("invalid"),
			TokenSecretHash: "hash",
			TokenSecretHint: "...",
			ExpiresAt:       time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC),
		},
	)
	if err == nil {
		t.Fatalf("expected invalid scope error")
	}
	if !strings.Contains(err.Error(), "unsupported mcp token scope") {
		t.Fatalf("expected scope validation error, got %v", err)
	}
	if record != nil {
		t.Fatalf("expected nil record on validation error")
	}
	if len(db.queryRowSQL) != 0 {
		t.Fatalf("expected no db calls on validation error")
	}
}

func TestListMCPTokensReturnsDefaultsForNilArrays(t *testing.T) {
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000921")
	tokenID := uuid.MustParse("00000000-0000-0000-0000-000000000922")
	expiresAt := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 2, 22, 14, 5, 0, 0, time.UTC)
	db := &fakeQueryer{
		queryRowsResult: &fakeRows{values: [][]any{
			mcpTokenRowValues(
				tokenID,
				ownerUserID,
				"Read token",
				"read",
				nil,
				nil,
				"hash",
				"...hint",
				expiresAt,
				nil,
				nil,
				createdAt,
			),
		}},
	}

	records, err := ListMCPTokens(
		context.Background(),
		db,
		MCPTokenListInput{OwnerUserID: ownerUserID, Limit: 50, Offset: 10},
	)
	requireNoError(t, err)
	requireEqual(t, 1, len(records))
	requireEqual(t, 0, len(records[0].AllowedTools))
	requireEqual(t, 0, len(records[0].AllowedProjectIDs))

	expectedArgs := []any{ownerUserID, 50, 10}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func TestGetMCPTokenByIDReturnsNilWhenMissing(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}

	record, err := GetMCPTokenByID(
		context.Background(),
		db,
		uuid.MustParse("00000000-0000-0000-0000-000000000931"),
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil token when missing")
	}
}

func TestRevokeMCPTokenReturnsNilWhenMissing(t *testing.T) {
	db := &fakeQueryer{queryRowResult: &fakeRow{err: pgx.ErrNoRows}}

	record, err := RevokeMCPToken(
		context.Background(),
		db,
		MCPTokenRevokeInput{
			TokenID:     uuid.MustParse("00000000-0000-0000-0000-000000000941"),
			OwnerUserID: uuid.MustParse("00000000-0000-0000-0000-000000000942"),
		},
	)
	requireNoError(t, err)
	if record != nil {
		t.Fatalf("expected nil token when revoke affects no rows")
	}
}

func TestTouchMCPTokenLastUsedUsesCurrentTimestamp(t *testing.T) {
	tokenID := uuid.MustParse("00000000-0000-0000-0000-000000000951")
	touchedAt := time.Date(2026, 2, 22, 14, 10, 0, 0, time.UTC)
	db := &fakeQueryer{}

	originalNow := nowMCPTokenUTC
	nowMCPTokenUTC = func() time.Time { return touchedAt }
	t.Cleanup(func() { nowMCPTokenUTC = originalNow })

	err := TouchMCPTokenLastUsed(context.Background(), db, tokenID)
	requireNoError(t, err)
	requireEqual(t, 1, len(db.queryArgs))
	expectedArgs := []any{touchedAt, tokenID}
	if !reflect.DeepEqual(expectedArgs, db.queryArgs[0]) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, db.queryArgs[0])
	}
}

func mcpTokenRowValues(
	tokenID uuid.UUID,
	ownerUserID uuid.UUID,
	name string,
	scope string,
	allowedTools []string,
	allowedProjectIDs []string,
	tokenSecretHash string,
	tokenSecretHint string,
	expiresAt time.Time,
	lastUsedAt *time.Time,
	revokedAt *time.Time,
	createdAt time.Time,
) []any {
	var lastUsedAtValue any
	if lastUsedAt != nil {
		lastUsedAtValue = *lastUsedAt
	}
	var revokedAtValue any
	if revokedAt != nil {
		revokedAtValue = *revokedAt
	}
	return []any{
		tokenID,
		ownerUserID,
		name,
		scope,
		allowedTools,
		allowedProjectIDs,
		tokenSecretHash,
		tokenSecretHint,
		expiresAt,
		lastUsedAtValue,
		revokedAtValue,
		createdAt,
	}
}

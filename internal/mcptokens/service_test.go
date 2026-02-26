package mcptokens

import (
	"context"
	"testing"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestIssueTokenParseAndVerifyRoundTrip(t *testing.T) {
	tokenID := uuid.MustParse("00000000-0000-0000-0000-000000000981")
	issued, err := issueTokenWithDays(tokenID, 90, "pepper-value")
	requireNoErrorMCPToken(t, err)

	parsedID, parsedSecret, err := ParsePlaintextToken(issued.plaintext)
	requireNoErrorMCPToken(t, err)
	requireEqualMCPToken(t, tokenID, parsedID)
	if !VerifyTokenSecret(parsedID, parsedSecret, "pepper-value", issued.hash) {
		t.Fatalf("expected parsed token secret to verify")
	}
}

func TestTokenIsActiveHandlesExpiryAndRevocation(t *testing.T) {
	now := time.Date(2026, 2, 26, 3, 0, 0, 0, time.UTC)
	active := testTokenRecord(now.Add(24*time.Hour), nil)
	if !TokenIsActive(active, now) {
		t.Fatalf("expected active token to be active")
	}

	expired := testTokenRecord(now.Add(-time.Second), nil)
	if TokenIsActive(expired, now) {
		t.Fatalf("expected expired token to be inactive")
	}

	revokedAt := now.Add(-time.Minute)
	revoked := testTokenRecord(now.Add(24*time.Hour), &revokedAt)
	if TokenIsActive(revoked, now) {
		t.Fatalf("expected revoked token to be inactive")
	}
}

func TestCreateTokenForOwnerNormalizesLists(t *testing.T) {
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000982")
	tokenID := uuid.MustParse("00000000-0000-0000-0000-000000000983")
	expiresAt := time.Date(2026, 3, 28, 3, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 2, 26, 3, 10, 0, 0, time.UTC)

	capturedIssue := struct {
		tokenID       uuid.UUID
		expiresInDays int
		pepper        string
	}{}
	capturedCreateInput := repository.MCPTokenCreateInput{}

	service := NewService(nil)
	service.deps.newTokenID = func() uuid.UUID { return tokenID }
	service.deps.issueNewToken = func(
		issuedTokenID uuid.UUID,
		expiresInDays int,
		pepper string,
	) (issuedToken, error) {
		capturedIssue.tokenID = issuedTokenID
		capturedIssue.expiresInDays = expiresInDays
		capturedIssue.pepper = pepper
		return issuedToken{
			plaintext: "engram_mcp_plain",
			hash:      "hashed-value",
			hint:      "secret...hint",
			expiresAt: expiresAt,
		}, nil
	}
	service.deps.createMCPToken = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.MCPTokenCreateInput,
	) (*models.MCPTokenRecord, error) {
		capturedCreateInput = input
		recordTokenID := tokenID
		if input.TokenID != nil {
			recordTokenID = *input.TokenID
		}
		return &models.MCPTokenRecord{
			TokenID:           recordTokenID,
			OwnerUserID:       input.OwnerUserID,
			Name:              input.Name,
			Scope:             input.Scope,
			AllowedTools:      input.AllowedTools,
			AllowedProjectIDs: input.AllowedProjectIDs,
			TokenSecretHash:   input.TokenSecretHash,
			TokenSecretHint:   input.TokenSecretHint,
			ExpiresAt:         input.ExpiresAt,
			CreatedAt:         createdAt,
		}, nil
	}

	created, err := service.CreateTokenForOwner(
		context.Background(),
		ownerUserID,
		models.MCPTokenCreateRequest{
			Name:              "  CLI Token  ",
			Scope:             "read",
			AllowedTools:      []string{" engram.query ", "", "engram.query", "chat.list_sessions"},
			AllowedProjectIDs: []string{"engram-vault", " engram-vault ", "phase31"},
			ExpiresInDays:     30,
		},
		"pepper-1",
	)
	requireNoErrorMCPToken(t, err)
	requireNotNilMCPToken(t, created)
	requireEqualMCPToken(t, "engram_mcp_plain", created.Token)
	requireEqualMCPToken(t, models.MCPTokenScopeRead, created.Scope)
	requireEqualStringSliceMCPToken(t, []string{"engram.query", "chat.list_sessions"}, created.AllowedTools)
	requireEqualStringSliceMCPToken(t, []string{"engram-vault", "phase31"}, created.AllowedProjectIDs)
	requireEqualMCPToken(t, "pepper-1", capturedIssue.pepper)
	requireEqualMCPToken(t, tokenID, capturedIssue.tokenID)
	requireEqualMCPToken(t, 30, capturedIssue.expiresInDays)
	requireEqualMCPToken(t, "CLI Token", capturedCreateInput.Name)
	requireEqualStringSliceMCPToken(t, []string{"engram.query", "chat.list_sessions"}, capturedCreateInput.AllowedTools)
	requireEqualStringSliceMCPToken(t, []string{"engram-vault", "phase31"}, capturedCreateInput.AllowedProjectIDs)
}

func TestListTokenSummariesAndRevokeTokenForOwner(t *testing.T) {
	now := time.Date(2026, 2, 26, 5, 0, 0, 0, time.UTC)
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000984")
	activeRecord := testTokenRecord(now.Add(48*time.Hour), nil)
	revokedAt := now.Add(-5 * time.Minute)
	revokedRecord := testTokenRecord(now.Add(48*time.Hour), &revokedAt)

	service := NewService(nil)
	service.deps.nowUTC = func() time.Time { return now }
	service.deps.listMCPTokens = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.MCPTokenListInput,
	) ([]models.MCPTokenRecord, error) {
		return []models.MCPTokenRecord{activeRecord, revokedRecord}, nil
	}
	service.deps.revokeMCPToken = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.MCPTokenRevokeInput,
	) (*models.MCPTokenRecord, error) {
		return &revokedRecord, nil
	}

	listed, err := service.ListTokenSummaries(context.Background(), ownerUserID, 200, 0)
	requireNoErrorMCPToken(t, err)
	requireEqualMCPToken(t, 2, len(listed))
	if !listed[0].IsActive {
		t.Fatalf("expected first summary to be active")
	}
	if listed[1].IsActive {
		t.Fatalf("expected second summary to be inactive")
	}
	requireEqualMCPToken(t, models.MCPTokenScopeRead, listed[0].Scope)

	revoked, err := service.RevokeTokenForOwner(context.Background(), uuid.New(), ownerUserID)
	requireNoErrorMCPToken(t, err)
	requireNotNilMCPToken(t, revoked)
	if revoked.IsActive {
		t.Fatalf("expected revoked summary to be inactive")
	}

	service.deps.revokeMCPToken = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.MCPTokenRevokeInput,
	) (*models.MCPTokenRecord, error) {
		return nil, nil
	}
	missing, err := service.RevokeTokenForOwner(context.Background(), uuid.New(), ownerUserID)
	requireNoErrorMCPToken(t, err)
	if missing != nil {
		t.Fatalf("expected nil revoke summary when token is missing")
	}
}

func testTokenRecord(expiresAt time.Time, revokedAt *time.Time) models.MCPTokenRecord {
	return models.MCPTokenRecord{
		TokenID:           uuid.New(),
		OwnerUserID:       uuid.New(),
		Name:              "token",
		Scope:             models.MCPTokenScopeRead,
		AllowedTools:      []string{},
		AllowedProjectIDs: []string{},
		TokenSecretHash:   "hash",
		TokenSecretHint:   "hint",
		ExpiresAt:         expiresAt,
		LastUsedAt:        nil,
		RevokedAt:         revokedAt,
		CreatedAt:         time.Date(2026, 2, 26, 4, 0, 0, 0, time.UTC),
	}
}

func requireNoErrorMCPToken(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func requireNotNilMCPToken(t *testing.T, value any) {
	t.Helper()
	if value == nil {
		t.Fatalf("expected value to be non-nil")
	}
}

func requireEqualMCPToken[T comparable](t *testing.T, expected T, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

func requireEqualStringSliceMCPToken(t *testing.T, expected []string, actual []string) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Fatalf("expected len %d, got %d", len(expected), len(actual))
	}
	for index := range expected {
		if expected[index] != actual[index] {
			t.Fatalf("expected[%d]=%q, got %q", index, expected[index], actual[index])
		}
	}
}

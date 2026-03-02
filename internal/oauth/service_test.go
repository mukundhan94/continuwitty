package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

func TestNormalizeScopeDefaultsToMCPWrite(t *testing.T) {
	requireEqualString(t, "mcp:write", NormalizeScope(""))
	requireEqualString(t, "mcp:write", NormalizeScope("   "))
}

func TestNormalizeScopeAliasesAndDeduplicates(t *testing.T) {
	requireEqualString(t, "mcp:read mcp:write custom", NormalizeScope("read write write custom"))
}

func TestTokenScopeMapsWriteScope(t *testing.T) {
	requireEqualString(t, "write", TokenScopeFromOAuthScope("mcp:write"))
	requireEqualString(t, "read", TokenScopeFromOAuthScope("mcp:read"))
}

func TestValidatePKCES256Only(t *testing.T) {
	verifier := "pkce-verifier-example"
	requireTrue(t, ValidatePKCE(PKCEValidationInput{
		CodeVerifier:        verifier,
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}))
	requireFalse(t, ValidatePKCE(PKCEValidationInput{
		CodeVerifier:        "wrong",
		CodeChallenge:       s256Challenge(verifier),
		CodeChallengeMethod: "S256",
	}))
	requireFalse(t, ValidatePKCE(PKCEValidationInput{
		CodeVerifier:        verifier,
		CodeChallenge:       verifier,
		CodeChallengeMethod: "plain",
	}))
}

func TestOAuthSecretHashAndVerifyRoundTrip(t *testing.T) {
	hash := OAuthSecretHash(SecretHashInput{
		Identifier: "client-id",
		Secret:     "client-secret",
		Pepper:     "pepper",
	})
	requireTrue(t, VerifyOAuthSecret(SecretVerificationInput{
		Identifier:   "client-id",
		Secret:       "client-secret",
		Pepper:       "pepper",
		ExpectedHash: hash,
	}))
	requireFalse(t, VerifyOAuthSecret(SecretVerificationInput{
		Identifier:   "client-id",
		Secret:       "wrong",
		Pepper:       "pepper",
		ExpectedHash: hash,
	}))
}

func TestAuthorizationCodeIsActiveHonorsExpiryAndConsumption(t *testing.T) {
	now := time.Date(2026, 2, 25, 13, 0, 0, 0, time.UTC)
	notConsumed := models.OAuthAuthorizationCodeRecord{
		CodeID:    uuid.MustParse("00000000-0000-0000-0000-000000000a11"),
		ExpiresAt: now.Add(5 * time.Minute),
	}
	requireTrue(t, AuthorizationCodeIsActive(notConsumed, now))

	consumedAt := now
	consumed := notConsumed
	consumed.ConsumedAt = &consumedAt
	requireFalse(t, AuthorizationCodeIsActive(consumed, now))

	expired := notConsumed
	expired.ExpiresAt = now.Add(-1 * time.Minute)
	requireFalse(t, AuthorizationCodeIsActive(expired, now))
}

func TestGeneratedClientCredentialsAreNonEmpty(t *testing.T) {
	clientID := BuildOAuthClientID()
	if !strings.HasPrefix(clientID, "engram_client_") {
		t.Fatalf("expected client id prefix, got %q", clientID)
	}
	secret, err := IssueOAuthClientSecret()
	requireNoError(t, err)
	if strings.TrimSpace(secret) == "" {
		t.Fatalf("expected non-empty oauth client secret")
	}
	code, err := GenerateAuthorizationCode()
	requireNoError(t, err)
	if strings.TrimSpace(code) == "" {
		t.Fatalf("expected non-empty authorization code")
	}
}

func s256Challenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func requireTrue(t *testing.T, value bool) {
	t.Helper()
	if !value {
		t.Fatalf("expected true")
	}
}

func requireFalse(t *testing.T, value bool) {
	t.Helper()
	if value {
		t.Fatalf("expected false")
	}
}

func requireEqualString[T ~string](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}

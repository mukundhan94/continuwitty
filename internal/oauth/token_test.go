package oauth

import (
	"context"
	"strings"
	"testing"
	"time"

	"engram/internal/config"
	"engram/internal/mcptokens"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestHandleTokenRejectsUnsupportedGrantType(t *testing.T) {
	service := newTokenService(defaultTokenDeps())

	result, err := service.HandleToken(
		context.Background(),
		config.Settings{},
		TokenRequest{GrantType: "client_credentials"},
	)
	requireOAuthTokenNoError(t, err)
	if result.ErrorCode != "unsupported_grant_type" || result.StatusCode != 400 {
		t.Fatalf("expected unsupported_grant_type 400, got %#v", result)
	}
}

func TestHandleTokenReturnsInvalidClientWhenMissing(t *testing.T) {
	deps := defaultTokenDeps()
	deps.getOAuthClient = func(
		_ context.Context,
		_ repository.Queryer,
		_ string,
	) (*models.OAuthClientRecord, error) {
		return nil, nil
	}
	service := newTokenService(deps)

	result, err := service.HandleToken(
		context.Background(),
		config.Settings{},
		TokenRequest{GrantType: "authorization_code", ClientID: "missing-client"},
	)
	requireOAuthTokenNoError(t, err)
	if result.ErrorCode != "invalid_client" || result.StatusCode != 401 {
		t.Fatalf("expected invalid_client 401, got %#v", result)
	}
}

func TestHandleTokenRejectsInvalidClientSecret(t *testing.T) {
	clientSecretHash := "expected-hash"
	deps := defaultTokenDeps()
	deps.getOAuthClient = func(
		_ context.Context,
		_ repository.Queryer,
		_ string,
	) (*models.OAuthClientRecord, error) {
		return &models.OAuthClientRecord{
			ClientID:                "client-1",
			TokenEndpointAuthMethod: "client_secret_post",
			ClientSecretHash:        &clientSecretHash,
		}, nil
	}
	deps.verifyOAuthSecret = func(_ SecretVerificationInput) bool {
		return false
	}
	service := newTokenService(deps)

	result, err := service.HandleToken(
		context.Background(),
		config.Settings{},
		TokenRequest{
			GrantType:    "authorization_code",
			ClientID:     "client-1",
			ClientSecret: "wrong-secret",
		},
	)
	requireOAuthTokenNoError(t, err)
	if result.ErrorCode != "invalid_client" || result.StatusCode != 401 {
		t.Fatalf("expected invalid_client 401, got %#v", result)
	}
}

func TestHandleTokenIssuesAccessToken(t *testing.T) {
	nowUTC := time.Date(2026, 2, 26, 22, 0, 0, 0, time.UTC)
	codeID := uuid.MustParse("00000000-0000-0000-0000-000000000981")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000982")
	capturedTokenInput := repository.MCPTokenCreateInput{}
	service := newTokenService(
		buildSuccessfulTokenDeps(nowUTC, codeID, userID, &capturedTokenInput),
	)

	result, err := service.HandleToken(
		context.Background(),
		config.Settings{
			MCPTokenPepper:             "pepper",
			OAuthAccessTokenTTLSeconds: 3600,
		},
		TokenRequest{
			GrantType:    "authorization_code",
			Code:         "raw-code",
			RedirectURI:  "https://client.example/callback",
			ClientID:     "client-1",
			CodeVerifier: "verifier",
		},
	)
	requireOAuthTokenNoError(t, err)
	assertIssuedTokenResult(t, result, capturedTokenInput, userID)
}

func requireOAuthTokenNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestBuildOAuthTokenSecretHintFormatsLongSecret(t *testing.T) {
	hint := buildOAuthTokenSecretHint("abcdefghijklmnopqrstuvwxyz")
	if !strings.HasPrefix(hint, "abcdef") || !strings.HasSuffix(hint, "wxyz") {
		t.Fatalf("unexpected hint %q", hint)
	}
}

func TestIssueOAuthAccessTokenProducesHashVerifiedByMCPTokenVerifier(t *testing.T) {
	tokenID := uuid.MustParse("00000000-0000-0000-0000-0000000009a1")
	issued, err := issueOAuthAccessToken(tokenID, time.Now().UTC().Add(time.Hour), "pepper")
	requireOAuthTokenNoError(t, err)

	parsedTokenID, tokenSecret, err := mcptokens.ParsePlaintextToken(issued.plaintext)
	requireOAuthTokenNoError(t, err)
	if parsedTokenID != tokenID {
		t.Fatalf("expected parsed token id %s, got %s", tokenID, parsedTokenID)
	}
	if !mcptokens.VerifyTokenSecret(
		mcptokens.TokenSecretVerificationInput{
			TokenID:      tokenID,
			TokenSecret:  tokenSecret,
			Pepper:       "pepper",
			ExpectedHash: issued.hash,
		},
	) {
		t.Fatalf("expected issued hash to validate against parsed token secret")
	}
}

func newTokenService(deps tokenDeps) *TokenService {
	return &TokenService{deps: deps}
}

func buildSuccessfulTokenDeps(
	nowUTC time.Time,
	codeID uuid.UUID,
	userID uuid.UUID,
	capturedTokenInput *repository.MCPTokenCreateInput,
) tokenDeps {
	deps := defaultTokenDeps()
	deps.nowUTC = func() time.Time { return nowUTC }
	deps.getOAuthClient = func(
		_ context.Context,
		_ repository.Queryer,
		_ string,
	) (*models.OAuthClientRecord, error) {
		return &models.OAuthClientRecord{
			ClientID:                "client-1",
			TokenEndpointAuthMethod: "none",
		}, nil
	}
	deps.getAuthorizationCodeByHash = func(
		_ context.Context,
		_ repository.Queryer,
		_ string,
	) (*models.OAuthAuthorizationCodeRecord, error) {
		return &models.OAuthAuthorizationCodeRecord{
			CodeID:              codeID,
			ClientID:            "client-1",
			UserID:              userID,
			RedirectURI:         "https://client.example/callback",
			CodeChallenge:       "challenge",
			CodeChallengeMethod: "S256",
			RequestedScope:      "mcp:write",
			ExpiresAt:           nowUTC.Add(30 * time.Second),
		}, nil
	}
	deps.consumeAuthorizationCode = func(
		_ context.Context,
		_ repository.Queryer,
		_ repository.OAuthAuthorizationCodeConsumeInput,
	) (*models.OAuthAuthorizationCodeRecord, error) {
		return &models.OAuthAuthorizationCodeRecord{}, nil
	}
	deps.authorizationCodeHash = func(_ AuthorizationCodeHashInput) string {
		return "hashed-code"
	}
	deps.validatePKCE = func(_ PKCEValidationInput) bool {
		return true
	}
	deps.issueToken = func(
		_ uuid.UUID,
		expiresAt time.Time,
		_ string,
	) (issuedToken, error) {
		return issuedToken{
			plaintext: "engram_mcp_token",
			hash:      "token-hash",
			hint:      "abc123...xyz9",
			expiresAt: expiresAt,
		}, nil
	}
	deps.createMCPToken = func(
		_ context.Context,
		_ repository.Queryer,
		input repository.MCPTokenCreateInput,
	) (*models.MCPTokenRecord, error) {
		*capturedTokenInput = input
		return &models.MCPTokenRecord{
			TokenID:   *input.TokenID,
			ExpiresAt: input.ExpiresAt,
		}, nil
	}
	return deps
}

func assertIssuedTokenResult(
	t *testing.T,
	result TokenResult,
	capturedTokenInput repository.MCPTokenCreateInput,
	userID uuid.UUID,
) {
	t.Helper()
	if result.AccessToken != "engram_mcp_token" {
		t.Fatalf("expected issued access token, got %q", result.AccessToken)
	}
	if result.TokenType != "Bearer" {
		t.Fatalf("expected bearer token type, got %q", result.TokenType)
	}
	if result.ExpiresInSeconds != 3600 {
		t.Fatalf("expected expires_in 3600, got %d", result.ExpiresInSeconds)
	}
	if result.Scope != "mcp:write" {
		t.Fatalf("expected normalized scope mcp:write, got %q", result.Scope)
	}
	if capturedTokenInput.Name != "oauth:client-1" {
		t.Fatalf("expected oauth-prefixed token name, got %q", capturedTokenInput.Name)
	}
	if capturedTokenInput.OwnerUserID != userID {
		t.Fatalf("expected token owner %s, got %s", userID, capturedTokenInput.OwnerUserID)
	}
	if capturedTokenInput.Scope != models.MCPTokenScopeWrite {
		t.Fatalf("expected write scope, got %q", capturedTokenInput.Scope)
	}
}

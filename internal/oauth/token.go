package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"

	"engram/internal/config"
	"engram/internal/mcptokens"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

const (
	minOAuthAccessTokenTTLSeconds = 60
	defaultTokenTypeBearer        = "Bearer"
)

// TokenRequest captures OAuth token endpoint form payload values.
type TokenRequest struct {
	GrantType    string
	Code         string
	RedirectURI  string
	ClientID     string
	CodeVerifier string
	ClientSecret string
}

// TokenResult captures either token issuance output or OAuth error values.
type TokenResult struct {
	AccessToken      string `json:"access_token,omitempty"`
	TokenType        string `json:"token_type,omitempty"`
	ExpiresInSeconds int    `json:"expires_in,omitempty"`
	Scope            string `json:"scope,omitempty"`
	ErrorCode        string
	ErrorDescription string
	StatusCode       int
}

// TokenService handles OAuth authorization-code token exchanges.
type TokenService struct {
	db   repository.Queryer
	deps tokenDeps
}

type tokenExchangeContext struct {
	client     models.OAuthClientRecord
	codeRecord models.OAuthAuthorizationCodeRecord
}

type authorizationCodeExchangeInput struct {
	request    TokenRequest
	client     models.OAuthClientRecord
	codeRecord models.OAuthAuthorizationCodeRecord
	now        time.Time
}

type tokenDeps struct {
	nowUTC                     func() time.Time
	getOAuthClient             func(ctx context.Context, db repository.Queryer, clientID string) (*models.OAuthClientRecord, error)
	getAuthorizationCodeByHash func(
		ctx context.Context,
		db repository.Queryer,
		codeHash string,
	) (*models.OAuthAuthorizationCodeRecord, error)
	consumeAuthorizationCode func(
		ctx context.Context,
		db repository.Queryer,
		input repository.OAuthAuthorizationCodeConsumeInput,
	) (*models.OAuthAuthorizationCodeRecord, error)
	createMCPToken func(
		ctx context.Context,
		db repository.Queryer,
		input repository.MCPTokenCreateInput,
	) (*models.MCPTokenRecord, error)
	authorizationCodeHash func(input AuthorizationCodeHashInput) string
	verifyOAuthSecret     func(input SecretVerificationInput) bool
	validatePKCE          func(input PKCEValidationInput) bool
	issueToken            func(tokenID uuid.UUID, expiresAt time.Time, pepper string) (issuedToken, error)
}

type issuedToken struct {
	plaintext string
	hash      string
	hint      string
	expiresAt time.Time
}

func defaultTokenDeps() tokenDeps {
	return tokenDeps{
		nowUTC:                     func() time.Time { return time.Now().UTC() },
		getOAuthClient:             repository.GetOAuthClient,
		getAuthorizationCodeByHash: repository.GetOAuthAuthorizationCodeByHash,
		consumeAuthorizationCode:   repository.ConsumeOAuthAuthorizationCode,
		createMCPToken:             repository.CreateMCPToken,
		authorizationCodeHash:      AuthorizationCodeHash,
		verifyOAuthSecret:          VerifyOAuthSecret,
		validatePKCE:               ValidatePKCE,
		issueToken:                 issueOAuthAccessToken,
	}
}

// NewTokenService builds a token service with repository defaults.
func NewTokenService(db repository.Queryer) *TokenService {
	return &TokenService{db: db, deps: defaultTokenDeps()}
}

// HandleToken validates and exchanges authorization codes for MCP bearer tokens.
func (service *TokenService) HandleToken(
	ctx context.Context,
	settings config.Settings,
	request TokenRequest,
) (TokenResult, error) {
	if result := validateTokenGrantType(request.GrantType); result != nil {
		return *result, nil
	}
	exchangeContext, result, err := service.resolveTokenExchangeContext(ctx, settings, request)
	if err != nil {
		return TokenResult{}, err
	}
	if result != nil {
		return *result, nil
	}
	if result, err := service.consumeAuthorizationCode(ctx, exchangeContext.codeRecord.CodeID); err != nil || result != nil {
		if err != nil {
			return TokenResult{}, err
		}
		return *result, nil
	}
	return service.issueAccessToken(ctx, settings, exchangeContext.client, exchangeContext.codeRecord)
}

func (service *TokenService) resolveTokenExchangeContext(
	ctx context.Context,
	settings config.Settings,
	request TokenRequest,
) (tokenExchangeContext, *TokenResult, error) {
	client, result, err := service.resolveTokenClient(ctx, settings, request)
	if err != nil {
		return tokenExchangeContext{}, nil, err
	}
	if result != nil {
		return tokenExchangeContext{}, result, nil
	}
	codeRecord, result, err := service.resolveAuthorizationCodeRecord(ctx, settings, request)
	if err != nil {
		return tokenExchangeContext{}, nil, err
	}
	if result != nil {
		return tokenExchangeContext{}, result, nil
	}
	if result := validateAuthorizationCodeExchange(
		service.deps,
		authorizationCodeExchangeInput{
			request:    request,
			client:     *client,
			codeRecord: *codeRecord,
			now:        service.deps.nowUTC(),
		},
	); result != nil {
		return tokenExchangeContext{}, result, nil
	}
	return tokenExchangeContext{
		client:     *client,
		codeRecord: *codeRecord,
	}, nil, nil
}

func validateTokenGrantType(grantType string) *TokenResult {
	if strings.TrimSpace(grantType) == "authorization_code" {
		return nil
	}
	return tokenErrorResult(
		400,
		"unsupported_grant_type",
		"Only authorization_code is supported.",
	)
}

func (service *TokenService) resolveTokenClient(
	ctx context.Context,
	settings config.Settings,
	request TokenRequest,
) (*models.OAuthClientRecord, *TokenResult, error) {
	client, err := service.deps.getOAuthClient(ctx, service.db, strings.TrimSpace(request.ClientID))
	if err != nil {
		return nil, nil, err
	}
	if client == nil {
		return nil, tokenErrorResult(401, "invalid_client", "Unknown client_id."), nil
	}
	if result := validateTokenClientSecret(service.deps, settings, *client, request.ClientSecret); result != nil {
		return nil, result, nil
	}
	return client, nil, nil
}

func validateTokenClientSecret(
	deps tokenDeps,
	settings config.Settings,
	client models.OAuthClientRecord,
	clientSecret string,
) *TokenResult {
	if strings.TrimSpace(client.TokenEndpointAuthMethod) != "client_secret_post" {
		return nil
	}
	if strings.TrimSpace(clientSecret) == "" || client.ClientSecretHash == nil {
		return tokenErrorResult(401, "invalid_client", "client_secret is required.")
	}
	if deps.verifyOAuthSecret(
		SecretVerificationInput{
			Identifier:   client.ClientID,
			Secret:       clientSecret,
			Pepper:       settings.OAuthClientSecretPepper,
			ExpectedHash: *client.ClientSecretHash,
		},
	) {
		return nil
	}
	return tokenErrorResult(401, "invalid_client", "client_secret is invalid.")
}

func (service *TokenService) resolveAuthorizationCodeRecord(
	ctx context.Context,
	settings config.Settings,
	request TokenRequest,
) (*models.OAuthAuthorizationCodeRecord, *TokenResult, error) {
	hashedCode := service.deps.authorizationCodeHash(
		AuthorizationCodeHashInput{
			Code:   strings.TrimSpace(request.Code),
			Pepper: settings.MCPTokenPepper,
		},
	)
	codeRecord, err := service.deps.getAuthorizationCodeByHash(ctx, service.db, hashedCode)
	if err != nil {
		return nil, nil, err
	}
	if codeRecord == nil {
		return nil, tokenErrorResult(400, "invalid_grant", "Authorization code is invalid."), nil
	}
	return codeRecord, nil, nil
}

func validateAuthorizationCodeExchange(
	deps tokenDeps,
	input authorizationCodeExchangeInput,
) *TokenResult {
	if input.codeRecord.ClientID != input.client.ClientID {
		return tokenErrorResult(
			400,
			"invalid_grant",
			"Authorization code does not belong to this client.",
		)
	}
	if strings.TrimSpace(input.codeRecord.RedirectURI) != strings.TrimSpace(input.request.RedirectURI) {
		return tokenErrorResult(400, "invalid_grant", "redirect_uri mismatch.")
	}
	if !AuthorizationCodeIsActive(input.codeRecord, input.now) {
		return tokenErrorResult(400, "invalid_grant", "Authorization code expired or already consumed.")
	}
	if deps.validatePKCE(
		PKCEValidationInput{
			CodeVerifier:        input.request.CodeVerifier,
			CodeChallenge:       input.codeRecord.CodeChallenge,
			CodeChallengeMethod: input.codeRecord.CodeChallengeMethod,
		},
	) {
		return nil
	}
	return tokenErrorResult(400, "invalid_grant", "PKCE validation failed.")
}

func (service *TokenService) consumeAuthorizationCode(
	ctx context.Context,
	codeID uuid.UUID,
) (*TokenResult, error) {
	consumed, err := service.deps.consumeAuthorizationCode(
		ctx,
		service.db,
		repository.OAuthAuthorizationCodeConsumeInput{
			CodeID:     codeID,
			ConsumedAt: service.deps.nowUTC(),
		},
	)
	if err != nil {
		return nil, err
	}
	if consumed != nil {
		return nil, nil
	}
	return tokenErrorResult(400, "invalid_grant", "Authorization code already consumed."), nil
}

func (service *TokenService) issueAccessToken(
	ctx context.Context,
	settings config.Settings,
	client models.OAuthClientRecord,
	codeRecord models.OAuthAuthorizationCodeRecord,
) (TokenResult, error) {
	now := service.deps.nowUTC()
	ttl := resolveAccessTokenTTL(settings.OAuthAccessTokenTTLSeconds)
	tokenID := uuid.New()
	issued, err := service.deps.issueToken(tokenID, now.Add(ttl), settings.MCPTokenPepper)
	if err != nil {
		return TokenResult{}, err
	}
	_, err = service.deps.createMCPToken(
		ctx,
		service.db,
		repository.MCPTokenCreateInput{
			TokenID:           &tokenID,
			OwnerUserID:       codeRecord.UserID,
			Name:              "oauth:" + client.ClientID,
			Scope:             normalizeIssuedTokenScope(codeRecord.RequestedScope),
			AllowedTools:      []string{},
			AllowedProjectIDs: []string{},
			TokenSecretHash:   issued.hash,
			TokenSecretHint:   issued.hint,
			ExpiresAt:         issued.expiresAt,
		},
	)
	if err != nil {
		return TokenResult{}, err
	}
	expiresIn := int(issued.expiresAt.Sub(now).Seconds())
	if expiresIn < 1 {
		expiresIn = 1
	}
	return TokenResult{
		AccessToken:      issued.plaintext,
		TokenType:        defaultTokenTypeBearer,
		ExpiresInSeconds: expiresIn,
		Scope:            NormalizeScope(codeRecord.RequestedScope),
	}, nil
}

func resolveAccessTokenTTL(ttlSeconds int) time.Duration {
	if ttlSeconds < minOAuthAccessTokenTTLSeconds {
		ttlSeconds = minOAuthAccessTokenTTLSeconds
	}
	return time.Duration(ttlSeconds) * time.Second
}

func normalizeIssuedTokenScope(requestedScope string) models.MCPTokenScope {
	scopeValue := TokenScopeFromOAuthScope(requestedScope)
	parsedScope, err := models.ParseMCPTokenScope(scopeValue)
	if err != nil {
		return models.MCPTokenScopeRead
	}
	return parsedScope
}

func tokenErrorResult(statusCode int, errorCode string, errorDescription string) *TokenResult {
	return &TokenResult{
		ErrorCode:        errorCode,
		ErrorDescription: errorDescription,
		StatusCode:       statusCode,
	}
}

// TokenRequestHasRequiredFields validates required token exchange fields.
func TokenRequestHasRequiredFields(request TokenRequest) (bool, string) {
	if strings.TrimSpace(request.GrantType) == "" {
		return false, "grant_type is required"
	}
	if strings.TrimSpace(request.Code) == "" {
		return false, "code is required"
	}
	if strings.TrimSpace(request.RedirectURI) == "" {
		return false, "redirect_uri is required"
	}
	if strings.TrimSpace(request.ClientID) == "" {
		return false, "client_id is required"
	}
	if strings.TrimSpace(request.CodeVerifier) == "" {
		return false, "code_verifier is required"
	}
	return true, ""
}

// TokenResultStatus returns response status for oauth token error outputs.
func TokenResultStatus(result TokenResult) int {
	if result.StatusCode > 0 {
		return result.StatusCode
	}
	return 400
}

func issueOAuthAccessToken(
	tokenID uuid.UUID,
	expiresAt time.Time,
	pepper string,
) (issuedToken, error) {
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return issuedToken{}, err
	}
	tokenSecret := base64.RawURLEncoding.EncodeToString(secretBytes)
	return issuedToken{
		plaintext: buildOAuthPlaintextToken(tokenID, tokenSecret),
		hash: mcptokens.TokenHash(
			mcptokens.TokenHashInput{
				TokenID:     tokenID,
				TokenSecret: tokenSecret,
				Pepper:      pepper,
			},
		),
		hint:      buildOAuthTokenSecretHint(tokenSecret),
		expiresAt: expiresAt,
	}, nil
}

func buildOAuthPlaintextToken(tokenID uuid.UUID, tokenSecret string) string {
	return models.MCPTokenPrefix + "_" + strings.ReplaceAll(tokenID.String(), "-", "") + "_" + tokenSecret
}

func buildOAuthTokenSecretHint(tokenSecret string) string {
	if len(tokenSecret) <= 10 {
		return tokenSecret
	}
	return tokenSecret[:6] + "..." + tokenSecret[len(tokenSecret)-4:]
}

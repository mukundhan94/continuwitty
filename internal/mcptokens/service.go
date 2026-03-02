package mcptokens

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

const (
	defaultTokenExpiryDays = 90
	defaultTokenListLimit  = 200
)

// Service orchestrates MCP token CRUD and token material handling.
type Service struct {
	db   repository.Queryer
	deps serviceDeps
}

type serviceDeps struct {
	nowUTC         func() time.Time
	newTokenID     func() uuid.UUID
	issueNewToken  func(input tokenIssueInput) (issuedToken, error)
	createMCPToken func(context.Context, repository.Queryer, repository.MCPTokenCreateInput) (*models.MCPTokenRecord, error)
	listMCPTokens  func(context.Context, repository.Queryer, repository.MCPTokenListInput) ([]models.MCPTokenRecord, error)
	revokeMCPToken func(context.Context, repository.Queryer, repository.MCPTokenRevokeInput) (*models.MCPTokenRecord, error)
}

type issuedToken struct {
	plaintext string
	hash      string
	hint      string
	expiresAt time.Time
}

// TokenHashInput captures token hashing materials.
type TokenHashInput struct {
	TokenID     uuid.UUID
	TokenSecret string
	Pepper      string
}

// TokenSecretVerificationInput captures secret verification materials.
type TokenSecretVerificationInput struct {
	TokenID      uuid.UUID
	TokenSecret  string
	Pepper       string
	ExpectedHash string
}

type tokenIssueInput struct {
	tokenID       uuid.UUID
	expiresInDays int
	pepper        string
}

// TokenListRequest captures list filters for token summaries.
type TokenListRequest struct {
	OwnerUserID uuid.UUID
	Limit       int
	Offset      int
}

func defaultServiceDeps() serviceDeps {
	return serviceDeps{
		nowUTC:        func() time.Time { return time.Now().UTC() },
		newTokenID:    uuid.New,
		issueNewToken: issueTokenWithDays,
		createMCPToken: func(
			ctx context.Context,
			db repository.Queryer,
			input repository.MCPTokenCreateInput,
		) (*models.MCPTokenRecord, error) {
			return repository.CreateMCPToken(ctx, db, input)
		},
		listMCPTokens: func(
			ctx context.Context,
			db repository.Queryer,
			input repository.MCPTokenListInput,
		) ([]models.MCPTokenRecord, error) {
			return repository.ListMCPTokens(ctx, db, input)
		},
		revokeMCPToken: func(
			ctx context.Context,
			db repository.Queryer,
			input repository.MCPTokenRevokeInput,
		) (*models.MCPTokenRecord, error) {
			return repository.RevokeMCPToken(ctx, db, input)
		},
	}
}

// NewService builds an MCP token service with repository defaults.
func NewService(db repository.Queryer) *Service {
	return &Service{db: db, deps: defaultServiceDeps()}
}

// NormalizeStringList trims, drops blanks, and deduplicates while preserving order.
func NormalizeStringList(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// TokenHash computes the persisted hash for an MCP token secret.
func TokenHash(input TokenHashInput) string {
	payload := strings.ReplaceAll(input.TokenID.String(), "-", "") + ":" + input.TokenSecret
	mac := hmac.New(sha256.New, []byte(input.Pepper))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// TokenSecretHint formats an operator-facing token hint.
func TokenSecretHint(tokenSecret string) string {
	if len(tokenSecret) <= 10 {
		return tokenSecret
	}
	return tokenSecret[:6] + "..." + tokenSecret[len(tokenSecret)-4:]
}

// BuildPlaintextToken composes the token string returned to the caller.
func BuildPlaintextToken(tokenID uuid.UUID, tokenSecret string) string {
	tokenIDHex := strings.ReplaceAll(tokenID.String(), "-", "")
	return models.MCPTokenPrefix + "_" + tokenIDHex + "_" + tokenSecret
}

// ParsePlaintextToken splits an MCP token into identifier and secret.
func ParsePlaintextToken(raw string) (uuid.UUID, string, error) {
	text := strings.TrimSpace(raw)
	prefix := models.MCPTokenPrefix + "_"
	if !strings.HasPrefix(text, prefix) {
		return uuid.UUID{}, "", errTokenPrefixMismatch
	}

	payload := strings.TrimPrefix(text, prefix)
	parts := strings.SplitN(payload, "_", 2)
	if len(parts) != 2 {
		return uuid.UUID{}, "", errTokenPayloadFormatMismatch
	}

	tokenIDHex := parts[0]
	tokenSecret := parts[1]
	if len(tokenIDHex) != 32 {
		return uuid.UUID{}, "", errTokenIdentifierLengthMismatch
	}
	tokenID, err := uuid.Parse(tokenIDHex)
	if err != nil {
		return uuid.UUID{}, "", errTokenIdentifierInvalid
	}
	if strings.TrimSpace(tokenSecret) == "" {
		return uuid.UUID{}, "", errTokenSecretMissing
	}
	return tokenID, tokenSecret, nil
}

// VerifyTokenSecret validates plaintext secret material against a persisted hash.
func VerifyTokenSecret(
	input TokenSecretVerificationInput,
) bool {
	candidateHash := TokenHash(
		TokenHashInput{
			TokenID:     input.TokenID,
			TokenSecret: input.TokenSecret,
			Pepper:      input.Pepper,
		},
	)
	return hmac.Equal([]byte(candidateHash), []byte(input.ExpectedHash))
}

// TokenIsActive evaluates token expiry and revocation state.
func TokenIsActive(record models.MCPTokenRecord, now time.Time) bool {
	if !now.IsZero() && record.ExpiresAt.IsZero() {
		return false
	}
	if record.RevokedAt != nil {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return record.ExpiresAt.After(now)
}

// BuildAuthContext maps token storage record to authorization claims.
func BuildAuthContext(record models.MCPTokenRecord) models.MCPTokenAuthContext {
	return models.MCPTokenAuthContext{
		TokenID:           record.TokenID,
		OwnerUserID:       record.OwnerUserID,
		Scope:             record.Scope,
		AllowedTools:      NormalizeStringList(record.AllowedTools),
		AllowedProjectIDs: NormalizeStringList(record.AllowedProjectIDs),
		ExpiresAt:         record.ExpiresAt,
	}
}

// CreateTokenForOwner issues and persists a new MCP token for an owner.
func (service *Service) CreateTokenForOwner(
	ctx context.Context,
	ownerUserID uuid.UUID,
	payload models.MCPTokenCreateRequest,
	pepper string,
) (*models.MCPTokenCreateResponse, error) {
	scopeValue := resolveCreateScope(payload)
	parsedScope, err := models.ParseMCPTokenScope(scopeValue)
	if err != nil {
		return nil, err
	}
	tokenID := service.deps.newTokenID()
	issued, err := service.deps.issueNewToken(
		tokenIssueInput{
			tokenID:       tokenID,
			expiresInDays: resolveExpiresInDays(payload),
			pepper:        pepper,
		},
	)
	if err != nil {
		return nil, err
	}
	created, err := service.deps.createMCPToken(
		ctx,
		service.db,
		repository.MCPTokenCreateInput{
			TokenID:           &tokenID,
			OwnerUserID:       ownerUserID,
			Name:              strings.TrimSpace(payload.Name),
			Scope:             parsedScope,
			AllowedTools:      NormalizeStringList(payload.AllowedTools),
			AllowedProjectIDs: NormalizeStringList(payload.AllowedProjectIDs),
			TokenSecretHash:   issued.hash,
			TokenSecretHint:   issued.hint,
			ExpiresAt:         issued.expiresAt,
		},
	)
	if err != nil {
		return nil, err
	}
	return &models.MCPTokenCreateResponse{
		TokenID:           created.TokenID,
		Name:              created.Name,
		Scope:             created.Scope,
		AllowedTools:      created.AllowedTools,
		AllowedProjectIDs: created.AllowedProjectIDs,
		TokenSecretHint:   created.TokenSecretHint,
		Token:             issued.plaintext,
		ExpiresAt:         created.ExpiresAt,
		CreatedAt:         created.CreatedAt,
	}, nil
}

// ListTokenSummaries returns non-secret token metadata for an owner.
func (service *Service) ListTokenSummaries(
	ctx context.Context,
	request TokenListRequest,
) ([]models.MCPTokenSummary, error) {
	normalizedRequest := normalizeTokenListRequest(request)
	records, err := service.deps.listMCPTokens(
		ctx,
		service.db,
		repository.MCPTokenListInput{
			OwnerUserID: normalizedRequest.OwnerUserID,
			Limit:       normalizedRequest.Limit,
			Offset:      normalizedRequest.Offset,
		},
	)
	if err != nil {
		return nil, err
	}
	now := service.deps.nowUTC()
	summaries := make([]models.MCPTokenSummary, 0, len(records))
	for _, record := range records {
		summaries = append(summaries, buildTokenSummary(record, now))
	}
	return summaries, nil
}

// RevokeTokenForOwner revokes a token and returns the updated summary.
func (service *Service) RevokeTokenForOwner(
	ctx context.Context,
	tokenID uuid.UUID,
	ownerUserID uuid.UUID,
) (*models.MCPTokenSummary, error) {
	revoked, err := service.deps.revokeMCPToken(
		ctx,
		service.db,
		repository.MCPTokenRevokeInput{
			TokenID:     tokenID,
			OwnerUserID: ownerUserID,
		},
	)
	if err != nil {
		return nil, err
	}
	if revoked == nil {
		return nil, nil
	}
	summary := buildTokenSummary(*revoked, service.deps.nowUTC())
	return &summary, nil
}

func buildTokenSummary(record models.MCPTokenRecord, now time.Time) models.MCPTokenSummary {
	return models.MCPTokenSummary{
		TokenID:           record.TokenID,
		Name:              record.Name,
		Scope:             record.Scope,
		AllowedTools:      NormalizeStringList(record.AllowedTools),
		AllowedProjectIDs: NormalizeStringList(record.AllowedProjectIDs),
		TokenSecretHint:   record.TokenSecretHint,
		ExpiresAt:         record.ExpiresAt,
		LastUsedAt:        record.LastUsedAt,
		RevokedAt:         record.RevokedAt,
		CreatedAt:         record.CreatedAt,
		IsActive:          TokenIsActive(record, now),
	}
}

func resolveExpiresInDays(payload models.MCPTokenCreateRequest) int {
	if payload.ExpiresInDays <= 0 {
		return defaultTokenExpiryDays
	}
	return payload.ExpiresInDays
}

func normalizeTokenListRequest(request TokenListRequest) TokenListRequest {
	if request.Limit <= 0 {
		request.Limit = defaultTokenListLimit
	}
	if request.Offset < 0 {
		request.Offset = 0
	}
	return request
}

func resolveCreateScope(payload models.MCPTokenCreateRequest) string {
	trimmed := strings.TrimSpace(payload.Scope)
	if trimmed == "" {
		return string(models.MCPTokenScopeRead)
	}
	return trimmed
}

func issueTokenWithDays(input tokenIssueInput) (issuedToken, error) {
	expiresAt := time.Now().UTC().Add(time.Duration(input.expiresInDays) * 24 * time.Hour)
	return issueTokenWithExpiry(input, expiresAt)
}

func issueTokenWithExpiry(input tokenIssueInput, expiresAt time.Time) (issuedToken, error) {
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return issuedToken{}, err
	}
	tokenSecret := base64.RawURLEncoding.EncodeToString(secretBytes)
	return issuedToken{
		plaintext: BuildPlaintextToken(input.tokenID, tokenSecret),
		hash: TokenHash(
			TokenHashInput{
				TokenID:     input.tokenID,
				TokenSecret: tokenSecret,
				Pepper:      input.pepper,
			},
		),
		hint:      TokenSecretHint(tokenSecret),
		expiresAt: expiresAt,
	}, nil
}

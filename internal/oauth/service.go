package oauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"engram/internal/config"
	"engram/internal/models"

	"github.com/google/uuid"
)

const (
	defaultScope = "mcp:read"
)

var scopeAliasMap = map[string]string{
	"read":  "mcp:read",
	"write": "mcp:write",
}

// SecretHashInput captures hash inputs for OAuth secrets.
type SecretHashInput struct {
	Identifier string
	Secret     string
	Pepper     string
}

// SecretVerificationInput captures hash verification inputs for OAuth secrets.
type SecretVerificationInput struct {
	Identifier   string
	Secret       string
	Pepper       string
	ExpectedHash string
}

// AuthorizationCodeHashInput captures authorization code hash values.
type AuthorizationCodeHashInput struct {
	Code   string
	Pepper string
}

// RedirectURIAllowedInput captures redirect URI allowlist checks.
type RedirectURIAllowedInput struct {
	RedirectURI string
	AllowedURIs []string
}

// PKCEValidationInput captures PKCE verifier/challenge values.
type PKCEValidationInput struct {
	CodeVerifier        string
	CodeChallenge       string
	CodeChallengeMethod string
}

// IssuerURLForRequest resolves issuer URL using explicit config or request-derived origin.
func IssuerURLForRequest(request *http.Request, settings config.Settings) string {
	configured := strings.TrimSpace(settings.OAuthIssuerURL)
	if configured != "" {
		return strings.TrimRight(configured, "/")
	}
	if request == nil {
		return ""
	}
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	return strings.TrimRight(scheme+"://"+request.Host, "/")
}

// NormalizeScope aliases and deduplicates scope values, defaulting to mcp:read.
func NormalizeScope(scope string) string {
	tokens := strings.Split(scope, " ")
	normalized := make([]string, 0, len(tokens))
	seen := make(map[string]struct{}, len(tokens))
	for _, raw := range tokens {
		token := strings.TrimSpace(raw)
		if token == "" {
			continue
		}
		mapped := mapScopeAlias(token)
		if _, exists := seen[mapped]; exists {
			continue
		}
		seen[mapped] = struct{}{}
		normalized = append(normalized, mapped)
	}
	if len(normalized) == 0 {
		return defaultScope
	}
	return strings.Join(normalized, " ")
}

func mapScopeAlias(token string) string {
	if mapped, ok := scopeAliasMap[token]; ok {
		return mapped
	}
	return token
}

// TokenScopeFromOAuthScope maps OAuth scope tokens into MCP token scope.
func TokenScopeFromOAuthScope(scope string) string {
	normalized := NormalizeScope(scope)
	tokens := strings.Fields(normalized)
	for _, token := range tokens {
		if token == "mcp:write" {
			return "write"
		}
	}
	return "read"
}

// OAuthSecretHash returns a peppered HMAC-SHA256 hash for OAuth secrets.
func OAuthSecretHash(input SecretHashInput) string {
	payload := input.Identifier + ":" + input.Secret
	mac := hmac.New(sha256.New, []byte(input.Pepper))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyOAuthSecret compares candidate and expected secret hashes in constant time.
func VerifyOAuthSecret(input SecretVerificationInput) bool {
	candidate := OAuthSecretHash(SecretHashInput{
		Identifier: input.Identifier,
		Secret:     input.Secret,
		Pepper:     input.Pepper,
	})
	return hmac.Equal([]byte(candidate), []byte(input.ExpectedHash))
}

// BuildOAuthClientID creates a new client_id value.
func BuildOAuthClientID() string {
	compactUUID := strings.ReplaceAll(uuid.NewString(), "-", "")
	return "engram_client_" + compactUUID
}

// IssueOAuthClientSecret returns a URL-safe random client secret.
func IssueOAuthClientSecret() (string, error) {
	return issueURLSafeToken(32)
}

// GenerateAuthorizationCode returns a URL-safe random authorization code.
func GenerateAuthorizationCode() (string, error) {
	return issueURLSafeToken(48)
}

func issueURLSafeToken(byteLength int) (string, error) {
	buffer := make([]byte, byteLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

// AuthorizationCodeHash returns a peppered hash for authorization code storage.
func AuthorizationCodeHash(input AuthorizationCodeHashInput) string {
	return OAuthSecretHash(SecretHashInput{
		Identifier: "authorization_code",
		Secret:     input.Code,
		Pepper:     input.Pepper,
	})
}

// AuthorizationCodeIsActive reports whether an authorization code can still be used.
func AuthorizationCodeIsActive(record models.OAuthAuthorizationCodeRecord, now time.Time) bool {
	if record.ConsumedAt != nil {
		return false
	}
	return record.ExpiresAt.After(now)
}

// RedirectURIAllowed reports whether a redirect URI is present in the allowlist.
func RedirectURIAllowed(input RedirectURIAllowedInput) bool {
	candidate := strings.TrimSpace(input.RedirectURI)
	if candidate == "" {
		return false
	}
	for _, rawAllowedURI := range input.AllowedURIs {
		allowedURI := strings.TrimSpace(rawAllowedURI)
		if allowedURI == "" {
			continue
		}
		if candidate == allowedURI {
			return true
		}
	}
	return false
}

// ValidatePKCE validates an S256 PKCE verifier/challenge pair.
func ValidatePKCE(input PKCEValidationInput) bool {
	method := strings.TrimSpace(input.CodeChallengeMethod)
	if method == "" {
		method = "S256"
	}
	verifier := strings.TrimSpace(input.CodeVerifier)
	challenge := strings.TrimSpace(input.CodeChallenge)
	if verifier == "" || challenge == "" {
		return false
	}
	if method != "S256" {
		return false
	}
	digest := sha256.Sum256([]byte(verifier))
	encoded := base64.RawURLEncoding.EncodeToString(digest[:])
	return hmac.Equal([]byte(encoded), []byte(challenge))
}

// ClientSupportsAuthorizationCode reports whether the client allows authorization_code grant.
func ClientSupportsAuthorizationCode(record models.OAuthClientRecord) bool {
	for _, grantType := range record.GrantTypes {
		if grantType == "authorization_code" {
			return true
		}
	}
	return false
}

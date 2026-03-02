package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"engram/internal/config"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	defaultOIDCScopes        = "openid profile email"
	defaultOIDCUsernameClaim = "email"
)

var (
	errOIDCProviderDisabled = errors.New("oidc provider is disabled")
	errOIDCTokenMissingID   = errors.New("oidc token response did not include id_token")
	errOIDCNonceMismatch    = errors.New("oidc nonce did not match")
	errOIDCIdentityMissing  = errors.New("oidc identity did not include a usable username")
)

type oidcState string
type oidcNonce string
type oidcAuthCode string
type oidcScopeConfig string
type oidcUsernameClaim string
type oidcClaimKey string
type oidcClaimValue string

// OIDCIdentity stores normalized identity claims extracted from an ID token.
type OIDCIdentity struct {
	Subject  string
	Username string
	Email    string
}

// OIDCLoginProvider handles start/exchange operations for OIDC login.
type OIDCLoginProvider interface {
	Enabled() bool
	AuthCodeURL(state string, nonce string) (string, error)
	AuthenticateCode(ctx context.Context, code string, expectedNonce string) (*OIDCIdentity, error)
}

type disabledOIDCLoginProvider struct{}

func (disabledOIDCLoginProvider) Enabled() bool {
	return false
}

func (disabledOIDCLoginProvider) AuthCodeURL(string, string) (string, error) {
	return "", errOIDCProviderDisabled
}

func (disabledOIDCLoginProvider) AuthenticateCode(context.Context, string, string) (*OIDCIdentity, error) {
	return nil, errOIDCProviderDisabled
}

type oidcLoginProvider struct {
	oauthConfig   oauth2.Config
	verifier      *oidc.IDTokenVerifier
	usernameClaim string
}

// NewOIDCLoginProvider builds an OIDC provider from runtime settings.
func NewOIDCLoginProvider(ctx context.Context, settings config.Settings) (OIDCLoginProvider, error) {
	if !settings.OIDCEnabled {
		return disabledOIDCLoginProvider{}, nil
	}
	if err := validateOIDCProviderSettings(settings); err != nil {
		return nil, err
	}
	provider, err := oidc.NewProvider(ctx, strings.TrimSpace(settings.OIDCIssuerURL))
	if err != nil {
		return nil, fmt.Errorf("initialize oidc provider: %w", err)
	}
	return &oidcLoginProvider{
		oauthConfig: oauth2.Config{
			ClientID:     strings.TrimSpace(settings.OIDCClientID),
			ClientSecret: strings.TrimSpace(settings.OIDCClientSecret),
			RedirectURL:  strings.TrimSpace(settings.OIDCRedirectURL),
			Endpoint:     provider.Endpoint(),
			Scopes:       parseOIDCScopes(oidcScopeConfig(settings.OIDCScopes)),
		},
		verifier: provider.Verifier(&oidc.Config{
			ClientID: strings.TrimSpace(settings.OIDCClientID),
		}),
		usernameClaim: resolveOIDCUsernameClaim(oidcUsernameClaim(settings.OIDCUsernameClaim)),
	}, nil
}

func validateOIDCProviderSettings(settings config.Settings) error {
	missing := make([]string, 0, 4)
	if strings.TrimSpace(settings.OIDCIssuerURL) == "" {
		missing = append(missing, "OIDC_ISSUER_URL")
	}
	if strings.TrimSpace(settings.OIDCClientID) == "" {
		missing = append(missing, "OIDC_CLIENT_ID")
	}
	if strings.TrimSpace(settings.OIDCClientSecret) == "" {
		missing = append(missing, "OIDC_CLIENT_SECRET")
	}
	if strings.TrimSpace(settings.OIDCRedirectURL) == "" {
		missing = append(missing, "OIDC_REDIRECT_URL")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("missing oidc settings: %s", strings.Join(missing, ", "))
}

func parseOIDCScopes(rawScopes oidcScopeConfig) []string {
	scopes := strings.Fields(strings.TrimSpace(string(rawScopes)))
	if len(scopes) == 0 {
		scopes = strings.Fields(defaultOIDCScopes)
	}
	seen := make(map[string]struct{}, len(scopes))
	normalized := make([]string, 0, len(scopes)+1)
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if _, hasOpenID := seen["openid"]; !hasOpenID {
		normalized = append([]string{"openid"}, normalized...)
	}
	return normalized
}

func resolveOIDCUsernameClaim(rawClaim oidcUsernameClaim) string {
	claim := strings.TrimSpace(string(rawClaim))
	if claim == "" {
		return defaultOIDCUsernameClaim
	}
	return claim
}

func (provider *oidcLoginProvider) Enabled() bool {
	return provider != nil
}

func (provider *oidcLoginProvider) AuthCodeURL(state string, nonce string) (string, error) {
	return provider.buildAuthCodeURL(oidcState(state), oidcNonce(nonce))
}

func (provider *oidcLoginProvider) buildAuthCodeURL(state oidcState, nonce oidcNonce) (string, error) {
	if provider == nil {
		return "", errOIDCProviderDisabled
	}
	trimmedState := strings.TrimSpace(string(state))
	if trimmedState == "" {
		return "", errors.New("oidc state is required")
	}
	return provider.oauthConfig.AuthCodeURL(trimmedState, oidc.Nonce(strings.TrimSpace(string(nonce)))), nil
}

func (provider *oidcLoginProvider) AuthenticateCode(
	ctx context.Context,
	code string,
	expectedNonce string,
) (*OIDCIdentity, error) {
	if provider == nil {
		return nil, errOIDCProviderDisabled
	}
	token, err := provider.exchangeOIDCToken(ctx, oidcAuthCode(code))
	if err != nil {
		return nil, err
	}
	idToken, err := provider.verifyOIDCIDToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if err := validateOIDCNonce(idToken, oidcNonce(expectedNonce)); err != nil {
		return nil, err
	}
	claims, err := decodeOIDCClaims(idToken)
	if err != nil {
		return nil, err
	}
	identity, err := provider.identityFromOIDCClaims(claims)
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

func (provider *oidcLoginProvider) exchangeOIDCToken(
	ctx context.Context,
	code oidcAuthCode,
) (*oauth2.Token, error) {
	token, err := provider.oauthConfig.Exchange(ctx, strings.TrimSpace(string(code)))
	if err != nil {
		return nil, fmt.Errorf("exchange oidc code: %w", err)
	}
	return token, nil
}

func (provider *oidcLoginProvider) verifyOIDCIDToken(
	ctx context.Context,
	token *oauth2.Token,
) (*oidc.IDToken, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || strings.TrimSpace(rawIDToken) == "" {
		return nil, errOIDCTokenMissingID
	}
	idToken, err := provider.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("verify oidc id_token: %w", err)
	}
	return idToken, nil
}

func validateOIDCNonce(idToken *oidc.IDToken, expectedNonce oidcNonce) error {
	if strings.TrimSpace(idToken.Nonce) == strings.TrimSpace(string(expectedNonce)) {
		return nil
	}
	return errOIDCNonceMismatch
}

func decodeOIDCClaims(idToken *oidc.IDToken) (map[string]any, error) {
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("decode oidc claims: %w", err)
	}
	return claims, nil
}

func (provider *oidcLoginProvider) identityFromOIDCClaims(claims map[string]any) (OIDCIdentity, error) {
	identity := OIDCIdentity{
		Subject:  string(claimString(claims, oidcClaimKey("sub"))),
		Email:    string(claimString(claims, oidcClaimKey("email"))),
		Username: string(claimString(claims, oidcClaimKey(provider.usernameClaim))),
	}
	if strings.TrimSpace(identity.Username) == "" {
		identity.Username = firstNonEmptyString(
			claimString(claims, oidcClaimKey("email")),
			claimString(claims, oidcClaimKey("preferred_username")),
			claimString(claims, oidcClaimKey("upn")),
			claimString(claims, oidcClaimKey("sub")),
		)
	}
	if strings.TrimSpace(identity.Username) == "" {
		return OIDCIdentity{}, errOIDCIdentityMissing
	}
	return identity, nil
}

func claimString(claims map[string]any, key oidcClaimKey) oidcClaimValue {
	value, exists := claims[string(key)]
	if !exists {
		return oidcClaimValue("")
	}
	text, ok := value.(string)
	if !ok {
		return oidcClaimValue("")
	}
	return oidcClaimValue(strings.TrimSpace(text))
}

func firstNonEmptyString(candidates ...oidcClaimValue) string {
	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(string(candidate))
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

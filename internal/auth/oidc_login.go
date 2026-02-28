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
			Scopes:       parseOIDCScopes(settings.OIDCScopes),
		},
		verifier: provider.Verifier(&oidc.Config{
			ClientID: strings.TrimSpace(settings.OIDCClientID),
		}),
		usernameClaim: resolveOIDCUsernameClaim(settings.OIDCUsernameClaim),
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

func parseOIDCScopes(rawScopes string) []string {
	scopes := strings.Fields(strings.TrimSpace(rawScopes))
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

func resolveOIDCUsernameClaim(rawClaim string) string {
	claim := strings.TrimSpace(rawClaim)
	if claim == "" {
		return defaultOIDCUsernameClaim
	}
	return claim
}

func (provider *oidcLoginProvider) Enabled() bool {
	return provider != nil
}

func (provider *oidcLoginProvider) AuthCodeURL(state string, nonce string) (string, error) {
	if provider == nil {
		return "", errOIDCProviderDisabled
	}
	trimmedState := strings.TrimSpace(state)
	if trimmedState == "" {
		return "", errors.New("oidc state is required")
	}
	return provider.oauthConfig.AuthCodeURL(trimmedState, oidc.Nonce(strings.TrimSpace(nonce))), nil
}

func (provider *oidcLoginProvider) AuthenticateCode(
	ctx context.Context,
	code string,
	expectedNonce string,
) (*OIDCIdentity, error) {
	if provider == nil {
		return nil, errOIDCProviderDisabled
	}
	token, err := provider.oauthConfig.Exchange(ctx, strings.TrimSpace(code))
	if err != nil {
		return nil, fmt.Errorf("exchange oidc code: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || strings.TrimSpace(rawIDToken) == "" {
		return nil, errOIDCTokenMissingID
	}
	idToken, err := provider.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("verify oidc id_token: %w", err)
	}
	if strings.TrimSpace(idToken.Nonce) != strings.TrimSpace(expectedNonce) {
		return nil, errOIDCNonceMismatch
	}
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("decode oidc claims: %w", err)
	}
	identity := OIDCIdentity{
		Subject:  claimString(claims, "sub"),
		Email:    claimString(claims, "email"),
		Username: claimString(claims, provider.usernameClaim),
	}
	if strings.TrimSpace(identity.Username) == "" {
		identity.Username = firstNonEmptyString(
			claimString(claims, "email"),
			claimString(claims, "preferred_username"),
			claimString(claims, "upn"),
			claimString(claims, "sub"),
		)
	}
	if strings.TrimSpace(identity.Username) == "" {
		return nil, errOIDCIdentityMissing
	}
	return &identity, nil
}

func claimString(claims map[string]any, key string) string {
	value, exists := claims[key]
	if !exists {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func firstNonEmptyString(candidates ...string) string {
	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

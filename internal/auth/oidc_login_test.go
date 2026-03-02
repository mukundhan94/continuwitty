package auth

import (
	"context"
	"strings"
	"testing"

	"engram/internal/config"
)

func TestParseOIDCScopesUsesDefaultAndInjectsOpenID(t *testing.T) {
	scopes := parseOIDCScopes("")
	if len(scopes) == 0 {
		t.Fatalf("expected default oidc scopes")
	}
	if scopes[0] != "openid" {
		t.Fatalf("expected openid to be first scope, got %q", scopes[0])
	}
}

func TestParseOIDCScopesDeduplicatesAndKeepsOrder(t *testing.T) {
	scopes := parseOIDCScopes("email profile email")
	expected := []string{"openid", "email", "profile"}
	if len(scopes) != len(expected) {
		t.Fatalf("expected %d scopes, got %d (%v)", len(expected), len(scopes), scopes)
	}
	for index, scope := range expected {
		if scopes[index] != scope {
			t.Fatalf("expected scope %q at index %d, got %q", scope, index, scopes[index])
		}
	}
}

func TestResolveOIDCUsernameClaimDefaultsToEmail(t *testing.T) {
	if claim := resolveOIDCUsernameClaim(""); claim != "email" {
		t.Fatalf("expected default username claim email, got %q", claim)
	}
	if claim := resolveOIDCUsernameClaim("preferred_username"); claim != "preferred_username" {
		t.Fatalf("expected explicit claim to be preserved, got %q", claim)
	}
}

func TestNewOIDCLoginProviderReturnsDisabledProviderWhenFeatureDisabled(t *testing.T) {
	provider, err := NewOIDCLoginProvider(context.Background(), config.Settings{OIDCEnabled: false})
	if err != nil {
		t.Fatalf("expected disabled provider without error, got %v", err)
	}
	if provider.Enabled() {
		t.Fatalf("expected provider to be disabled")
	}
	if _, err := provider.AuthCodeURL("state", "nonce"); err == nil {
		t.Fatalf("expected auth code url error on disabled provider")
	}
	if _, err := provider.AuthenticateCode(context.Background(), "code", "nonce"); err == nil {
		t.Fatalf("expected authenticate error on disabled provider")
	}
}

func TestNewOIDCLoginProviderRejectsMissingRequiredSettings(t *testing.T) {
	_, err := NewOIDCLoginProvider(context.Background(), config.Settings{
		OIDCEnabled: true,
	})
	if err == nil {
		t.Fatalf("expected missing-setting error")
	}
	if !strings.Contains(err.Error(), "OIDC_ISSUER_URL") {
		t.Fatalf("expected issuer setting in error, got %q", err.Error())
	}
}

func TestFirstNonEmptyStringReturnsFirstValue(t *testing.T) {
	result := firstNonEmptyString("", "  ", "alpha", "beta")
	if result != "alpha" {
		t.Fatalf("expected alpha, got %q", result)
	}
}

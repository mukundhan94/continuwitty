package oauth

import (
	"context"
	"strings"
	"testing"
	"time"

	"engram/internal/config"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestHandleAuthorizeReturnsJSONErrorWhenClientMissing(t *testing.T) {
	service := authorizationServiceForTest(
		authorizationDeps{
			getOAuthClient: func(
				_ context.Context,
				_ repository.Queryer,
				_ string,
			) (*models.OAuthClientRecord, error) {
				return nil, nil
			},
		},
	)

	result, err := service.HandleAuthorize(
		context.Background(),
		config.Settings{},
		AuthorizationRequest{
			ResponseType: "code",
			ClientID:     "missing-client",
			RedirectURI:  "https://client.example/callback",
		},
	)
	requireOAuthNoError(t, err)
	if result.StatusCode != 401 || result.ErrorCode != "invalid_client" {
		t.Fatalf("expected invalid_client 401, got %#v", result)
	}
}

func TestHandleAuthorizeRedirectsOnUnsupportedResponseType(t *testing.T) {
	service := authorizationServiceForTest(
		authorizationDeps{
			getOAuthClient: func(
				_ context.Context,
				_ repository.Queryer,
				_ string,
			) (*models.OAuthClientRecord, error) {
				return &models.OAuthClientRecord{
					ClientID:                "client-1",
					RedirectURIs:            []string{"https://client.example/callback"},
					GrantTypes:              []string{"authorization_code"},
					TokenEndpointAuthMethod: "none",
				}, nil
			},
		},
	)

	result, err := service.HandleAuthorize(
		context.Background(),
		config.Settings{},
		AuthorizationRequest{
			ResponseType: "token",
			ClientID:     "client-1",
			RedirectURI:  "https://client.example/callback",
			State:        "abc-state",
		},
	)
	requireOAuthNoError(t, err)
	if !strings.Contains(result.RedirectURL, "error=unsupported_response_type") {
		t.Fatalf("expected unsupported_response_type redirect, got %q", result.RedirectURL)
	}
	if !strings.Contains(result.RedirectURL, "state=abc-state") {
		t.Fatalf("expected state to be preserved, got %q", result.RedirectURL)
	}
}

func TestHandleAuthorizeRedirectsToLoginWhenSessionMissing(t *testing.T) {
	service := authorizationServiceForTest(
		authorizationDeps{
			getOAuthClient: func(
				_ context.Context,
				_ repository.Queryer,
				_ string,
			) (*models.OAuthClientRecord, error) {
				return &models.OAuthClientRecord{
					ClientID:                "client-1",
					RedirectURIs:            []string{"https://client.example/callback"},
					GrantTypes:              []string{"authorization_code"},
					TokenEndpointAuthMethod: "client_secret_post",
				}, nil
			},
		},
	)

	result, err := service.HandleAuthorize(
		context.Background(),
		config.Settings{},
		AuthorizationRequest{
			RequestPathWithQuery: "/oauth/authorize?client_id=client-1",
			ResponseType:         "code",
			ClientID:             "client-1",
			RedirectURI:          "https://client.example/callback",
		},
	)
	requireOAuthNoError(t, err)
	if result.RedirectURL != "/login?next=%2Foauth%2Fauthorize%3Fclient_id%3Dclient-1" {
		t.Fatalf("expected login redirect, got %q", result.RedirectURL)
	}
}

func TestHandleAuthorizeCreatesCodeAndReturnsRedirect(t *testing.T) {
	createdInput := repository.OAuthAuthorizationCodeCreateInput{}
	nowUTC := time.Date(2026, 2, 26, 21, 0, 0, 0, time.UTC)
	service := authorizationServiceForTest(
		authorizationDeps{
			nowUTC: func() time.Time { return nowUTC },
			generateAuthorizationCode: func() (string, error) {
				return "generated-code", nil
			},
			authorizationCodeHash: func(_ AuthorizationCodeHashInput) string {
				return "hashed-code"
			},
			getOAuthClient: func(
				_ context.Context,
				_ repository.Queryer,
				_ string,
			) (*models.OAuthClientRecord, error) {
				return &models.OAuthClientRecord{
					ClientID:                "client-1",
					RedirectURIs:            []string{"https://client.example/callback"},
					GrantTypes:              []string{"authorization_code"},
					TokenEndpointAuthMethod: "none",
				}, nil
			},
			createOAuthAuthorizationCode: func(
				_ context.Context,
				_ repository.Queryer,
				input repository.OAuthAuthorizationCodeCreateInput,
			) (*models.OAuthAuthorizationCodeRecord, error) {
				createdInput = input
				return &models.OAuthAuthorizationCodeRecord{
					CodeID: input.CodeID,
				}, nil
			},
		},
	)

	result, err := service.HandleAuthorize(
		context.Background(),
		config.Settings{
			MCPTokenPepper:                   "pepper",
			OAuthAuthorizationCodeTTLSeconds: 300,
		},
		AuthorizationRequest{
			RequestPathWithQuery: "/oauth/authorize?client_id=client-1",
			Issuer:               "https://auth.example.com",
			ResponseType:         "code",
			ClientID:             "client-1",
			RedirectURI:          "https://client.example/callback",
			State:                "xyz-state",
			Scope:                "read",
			CodeChallenge:        "pkce-challenge",
			CodeChallengeMethod:  "S256",
			SessionUser: &SessionUser{
				UserID: uuid.MustParse("00000000-0000-0000-0000-000000000972").String(),
				Role:   "analyst",
			},
		},
	)
	requireOAuthNoError(t, err)
	if !strings.Contains(result.RedirectURL, "code=generated-code") {
		t.Fatalf("expected generated code in redirect, got %q", result.RedirectURL)
	}
	if !strings.Contains(result.RedirectURL, "iss=https%3A%2F%2Fauth.example.com") {
		t.Fatalf("expected issuer in redirect, got %q", result.RedirectURL)
	}
	if createdInput.RequestedScope != "mcp:read" {
		t.Fatalf("expected normalized scope mcp:read, got %q", createdInput.RequestedScope)
	}
	if createdInput.CodeHash != "hashed-code" {
		t.Fatalf("expected hashed code to be persisted, got %q", createdInput.CodeHash)
	}
	if createdInput.ExpiresAt != nowUTC.Add(300*time.Second) {
		t.Fatalf("unexpected expires_at %s", createdInput.ExpiresAt)
	}
}

func authorizationServiceForTest(overrides authorizationDeps) *AuthorizationService {
	deps := defaultAuthorizationDeps()
	if overrides.nowUTC != nil {
		deps.nowUTC = overrides.nowUTC
	}
	if overrides.generateAuthorizationCode != nil {
		deps.generateAuthorizationCode = overrides.generateAuthorizationCode
	}
	if overrides.authorizationCodeHash != nil {
		deps.authorizationCodeHash = overrides.authorizationCodeHash
	}
	if overrides.getOAuthClient != nil {
		deps.getOAuthClient = overrides.getOAuthClient
	}
	if overrides.createOAuthAuthorizationCode != nil {
		deps.createOAuthAuthorizationCode = overrides.createOAuthAuthorizationCode
	}
	return &AuthorizationService{deps: deps}
}

func requireOAuthNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

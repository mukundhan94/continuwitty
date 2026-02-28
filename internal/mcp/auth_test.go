package mcp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"engram/internal/config"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

func TestActorResolverFallsBackToSessionActor(t *testing.T) {
	actorID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	resolver := NewActorResolver(
		config.Settings{OAuthEnabled: false},
		nil,
		func(_ *http.Request) (Actor, error) { return Actor{UserID: actorID, Role: "Viewer"}, nil },
	)
	request := httptest.NewRequest("POST", "http://api.local/api/v1/mcp/stream", nil)

	resolved, err := resolver.ResolveActor(request)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resolved.Actor.UserID != actorID {
		t.Fatalf("expected actor id %s, got %s", actorID, resolved.Actor.UserID)
	}
	if resolved.Actor.Role != "viewer" {
		t.Fatalf("expected normalized role viewer, got %q", resolved.Actor.Role)
	}
	if resolved.TokenAuth != nil {
		t.Fatalf("expected nil token auth for session fallback")
	}
}

func TestActorResolverMissingSessionReturnsUnauthorizedWithOAuthHeader(t *testing.T) {
	resolver := NewActorResolver(
		config.Settings{
			OAuthEnabled:     true,
			OAuthIssuerURL:   "https://auth.example.com",
			MCPTokenPepper:   "pepper-value",
			AppSessionSecret: "session-secret",
		},
		nil,
		func(_ *http.Request) (Actor, error) {
			return Actor{}, errors.New("missing session actor")
		},
	)
	request := httptest.NewRequest("POST", "http://api.local/api/v1/mcp/stream", nil)

	_, err := resolver.ResolveActor(request)
	authErr := requireAuthError(t, err, 401)
	if authErr.Detail != "Authentication required" {
		t.Fatalf("expected authentication required detail, got %q", authErr.Detail)
	}
	headerValue := authErr.Headers["WWW-Authenticate"]
	if !strings.Contains(headerValue, "resource_metadata=\"https://auth.example.com/.well-known/oauth-protected-resource\"") {
		t.Fatalf("expected resource metadata in authenticate header, got %q", headerValue)
	}
	if !strings.Contains(headerValue, "scope=\"mcp:write\"") {
		t.Fatalf("expected mcp:write challenge scope, got %q", headerValue)
	}
}

func TestActorResolverRejectsInvalidAuthorizationScheme(t *testing.T) {
	resolver := NewActorResolver(config.Settings{}, nil, nil)
	request := httptest.NewRequest("POST", "http://api.local/api/v1/mcp/stream", nil)
	request.Header.Set("Authorization", "Basic abc123")

	_, err := resolver.ResolveActor(request)
	requireAuthError(t, err, 401)
}

func TestActorResolverResolvesBearerTokenActorAndClaims(t *testing.T) {
	fixture := newBearerResolverFixture(t)
	resolved, err := fixture.resolve()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !fixture.touchCalled {
		t.Fatalf("expected touch token last used to be called")
	}
	if resolved.Actor.UserID != fixture.ownerID {
		t.Fatalf("expected owner id %s, got %s", fixture.ownerID, resolved.Actor.UserID)
	}
	if resolved.Actor.Role != "analyst" {
		t.Fatalf("expected normalized role analyst, got %q", resolved.Actor.Role)
	}
	if resolved.TokenAuth == nil {
		t.Fatalf("expected token auth context")
	}
	if resolved.TokenAuth.TokenID != fixture.tokenID {
		t.Fatalf("expected token auth id %s, got %s", fixture.tokenID, resolved.TokenAuth.TokenID)
	}
}

type bearerResolverFixture struct {
	resolver    *ActorResolver
	request     *http.Request
	now         time.Time
	tokenID     uuid.UUID
	ownerID     uuid.UUID
	touchCalled bool
}

func newBearerResolverFixture(t *testing.T) *bearerResolverFixture {
	t.Helper()
	fixture := &bearerResolverFixture{
		now:     time.Date(2026, time.February, 26, 14, 0, 0, 0, time.UTC),
		tokenID: uuid.MustParse("20000000-0000-0000-0000-000000000002"),
		ownerID: uuid.MustParse("30000000-0000-0000-0000-000000000003"),
	}
	fixture.resolver = newBearerResolverWithDependencies(t, fixture)
	fixture.request = httptest.NewRequest("POST", "http://api.local/api/v1/mcp/stream", nil)
	fixture.request.Header.Set("Authorization", "Bearer engram_mcp_token")
	return fixture
}

func newBearerResolverWithDependencies(t *testing.T, fixture *bearerResolverFixture) *ActorResolver {
	t.Helper()
	tokenRecord := fixture.tokenRecord()
	userRecord := fixture.userRecord()

	resolver := NewActorResolver(
		config.Settings{
			MCPTokenPepper: "pepper-value",
			OAuthEnabled:   false,
		},
		nil,
		nil,
	)
	resolver.deps.nowUTC = func() time.Time { return fixture.now }
	resolver.deps.parsePlaintextToken = parsePlaintextTokenForFixture(t, fixture.tokenID)
	resolver.deps.getTokenByID = func(
		_ context.Context,
		_ repository.Queryer,
		tokenID uuid.UUID,
	) (*models.MCPTokenRecord, error) {
		assertUUIDEqual(t, fixture.tokenID, tokenID, "token id")
		copied := tokenRecord
		return &copied, nil
	}
	resolver.deps.tokenIsActive = tokenIsActiveForFixture(t, fixture.tokenID, fixture.now)
	resolver.deps.verifyTokenSecret = verifyTokenSecretForFixture(t, fixture.tokenID)
	resolver.deps.getUserByID = func(
		_ context.Context,
		_ repository.Queryer,
		userID uuid.UUID,
	) (*models.UserAuthRecord, error) {
		assertUUIDEqual(t, fixture.ownerID, userID, "owner id")
		copied := userRecord
		return &copied, nil
	}
	resolver.deps.touchTokenLastUsed = touchTokenLastUsedForFixture(t, fixture)
	resolver.deps.buildAuthContext = buildAuthContextForFixture
	return resolver
}

func parsePlaintextTokenForFixture(t *testing.T, tokenID uuid.UUID) func(string) (uuid.UUID, string, error) {
	t.Helper()
	return func(raw string) (uuid.UUID, string, error) {
		if raw != "engram_mcp_token" {
			t.Fatalf("expected token payload, got %q", raw)
		}
		return tokenID, "token-secret", nil
	}
}

func tokenIsActiveForFixture(
	t *testing.T,
	tokenID uuid.UUID,
	now time.Time,
) func(models.MCPTokenRecord, time.Time) bool {
	t.Helper()
	return func(record models.MCPTokenRecord, observed time.Time) bool {
		if record.TokenID != tokenID {
			t.Fatalf("expected token record")
		}
		if !observed.Equal(now) {
			t.Fatalf("expected now %s, got %s", now, observed)
		}
		return true
	}
}

func verifyTokenSecretForFixture(
	t *testing.T,
	tokenID uuid.UUID,
) func(uuid.UUID, string, string, string) bool {
	t.Helper()
	return func(token uuid.UUID, secret string, pepper string, expectedHash string) bool {
		assertUUIDEqual(t, tokenID, token, "token id")
		assertStringEqual(t, "token-secret", secret, "token secret")
		assertStringEqual(t, "pepper-value", pepper, "token pepper")
		assertStringEqual(t, "expected-hash", expectedHash, "token hash")
		return true
	}
}

func touchTokenLastUsedForFixture(
	t *testing.T,
	fixture *bearerResolverFixture,
) func(context.Context, repository.Queryer, uuid.UUID) error {
	t.Helper()
	return func(_ context.Context, _ repository.Queryer, touchedTokenID uuid.UUID) error {
		if touchedTokenID != fixture.tokenID {
			t.Fatalf("expected touched token id %s, got %s", fixture.tokenID, touchedTokenID)
		}
		fixture.touchCalled = true
		return nil
	}
}

func buildAuthContextForFixture(record models.MCPTokenRecord) models.MCPTokenAuthContext {
	return models.MCPTokenAuthContext{
		TokenID:           record.TokenID,
		OwnerUserID:       record.OwnerUserID,
		Scope:             record.Scope,
		AllowedTools:      record.AllowedTools,
		AllowedProjectIDs: record.AllowedProjectIDs,
		ExpiresAt:         record.ExpiresAt,
	}
}

func (fixture *bearerResolverFixture) tokenRecord() models.MCPTokenRecord {
	return models.MCPTokenRecord{
		TokenID:           fixture.tokenID,
		OwnerUserID:       fixture.ownerID,
		Name:              "CLI token",
		Scope:             models.MCPTokenScopeRead,
		AllowedTools:      []string{"chat.list_sessions"},
		AllowedProjectIDs: []string{"engram-vault"},
		TokenSecretHash:   "expected-hash",
		ExpiresAt:         fixture.now.Add(2 * time.Hour),
		CreatedAt:         fixture.now,
	}
}

func (fixture *bearerResolverFixture) userRecord() models.UserAuthRecord {
	return models.UserAuthRecord{
		UserID:   fixture.ownerID,
		Username: "alice",
		Role:     models.UserRoleAnalyst,
		IsActive: true,
	}
}

func (fixture *bearerResolverFixture) resolve() (ResolvedActor, error) {
	return fixture.resolver.ResolveActor(fixture.request)
}

func assertUUIDEqual(t *testing.T, expected uuid.UUID, actual uuid.UUID, field string) {
	t.Helper()
	if actual != expected {
		t.Fatalf("expected %s %s, got %s", field, expected, actual)
	}
}

func assertStringEqual(t *testing.T, expected string, actual string, field string) {
	t.Helper()
	if actual != expected {
		t.Fatalf("expected %s %q, got %q", field, expected, actual)
	}
}

func requireAuthError(t *testing.T, err error, expectedStatus int) *AuthError {
	t.Helper()
	authErr := &AuthError{}
	if !errors.As(err, &authErr) {
		t.Fatalf("expected auth error, got %v", err)
	}
	if authErr.StatusCode != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, authErr.StatusCode)
	}
	return authErr
}

package mcp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"engram/internal/config"
	"engram/internal/mcptokens"
	"engram/internal/models"
	"engram/internal/oauth"
	"engram/internal/repository"

	"github.com/google/uuid"
)

// HTTPActorResolver resolves MCP caller identity from HTTP request context.
type HTTPActorResolver interface {
	ResolveActor(request *http.Request) (ResolvedActor, error)
}

// SessionActorResolver resolves actor identity from a session-authenticated request context.
type SessionActorResolver func(request *http.Request) (Actor, error)

// AuthError captures transport-safe auth failures.
type AuthError struct {
	StatusCode int
	Detail     string
	Headers    map[string]string
}

func (err *AuthError) Error() string {
	return err.Detail
}

// ActorResolver resolves MCP actor context from bearer token or session actor fallback.
type ActorResolver struct {
	settings            config.Settings
	db                  repository.Queryer
	resolveSessionActor SessionActorResolver
	deps                actorResolverDeps
}

type actorResolverDeps struct {
	nowUTC              func() time.Time
	parsePlaintextToken func(raw string) (uuid.UUID, string, error)
	getTokenByID        func(context.Context, repository.Queryer, uuid.UUID) (*models.MCPTokenRecord, error)
	tokenIsActive       func(models.MCPTokenRecord, time.Time) bool
	verifyTokenSecret   func(uuid.UUID, string, string, string) bool
	getUserByID         func(context.Context, repository.Queryer, uuid.UUID) (*models.UserAuthRecord, error)
	touchTokenLastUsed  func(context.Context, repository.Queryer, uuid.UUID) error
	buildAuthContext    func(models.MCPTokenRecord) models.MCPTokenAuthContext
}

// NewActorResolver builds an MCP actor resolver with repository-backed dependencies.
func NewActorResolver(
	settings config.Settings,
	db repository.Queryer,
	resolveSessionActor SessionActorResolver,
) *ActorResolver {
	return &ActorResolver{
		settings:            settings,
		db:                  db,
		resolveSessionActor: resolveSessionActor,
		deps: actorResolverDeps{
			nowUTC:              func() time.Time { return time.Now().UTC() },
			parsePlaintextToken: mcptokens.ParsePlaintextToken,
			getTokenByID:        repository.GetMCPTokenByID,
			tokenIsActive:       mcptokens.TokenIsActive,
			verifyTokenSecret:   mcptokens.VerifyTokenSecret,
			getUserByID:         repository.GetUserAuthRecordByID,
			touchTokenLastUsed:  repository.TouchMCPTokenLastUsed,
			buildAuthContext:    mcptokens.BuildAuthContext,
		},
	}
}

// ResolveActor resolves MCP actor identity from bearer token or session actor fallback.
func (resolver *ActorResolver) ResolveActor(request *http.Request) (ResolvedActor, error) {
	token, hasBearerToken, err := parseBearerToken(request)
	if err != nil {
		return ResolvedActor{}, resolver.unauthorized(request)
	}
	if !hasBearerToken {
		return resolver.resolveSessionActorContext(request)
	}
	return resolver.resolveBearerActorContext(request, token)
}

func parseBearerToken(request *http.Request) (string, bool, error) {
	if request == nil {
		return "", false, nil
	}
	authorization := strings.TrimSpace(request.Header.Get("Authorization"))
	if authorization == "" {
		return "", false, nil
	}
	if !strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		return "", true, errBearerTokenMalformed
	}
	token := strings.TrimSpace(authorization[len("Bearer "):])
	if token == "" {
		return "", true, errBearerTokenMissing
	}
	return token, true, nil
}

func (resolver *ActorResolver) resolveSessionActorContext(request *http.Request) (ResolvedActor, error) {
	if resolver.resolveSessionActor == nil {
		return ResolvedActor{}, resolver.unauthorized(request)
	}
	actor, err := resolver.resolveSessionActor(request)
	if err != nil || actor.UserID == uuid.Nil {
		return ResolvedActor{}, resolver.unauthorized(request)
	}
	return ResolvedActor{Actor: normalizeActor(actor)}, nil
}

func (resolver *ActorResolver) resolveBearerActorContext(
	request *http.Request,
	token string,
) (ResolvedActor, error) {
	tokenID, tokenSecret, err := resolver.deps.parsePlaintextToken(token)
	if err != nil {
		return ResolvedActor{}, resolver.unauthorized(request)
	}

	record, err := resolver.deps.getTokenByID(request.Context(), resolver.db, tokenID)
	if err != nil {
		return ResolvedActor{}, resolver.internalError()
	}
	if record == nil || !resolver.deps.tokenIsActive(*record, resolver.deps.nowUTC()) {
		return ResolvedActor{}, resolver.unauthorized(request)
	}

	if !resolver.deps.verifyTokenSecret(
		record.TokenID,
		tokenSecret,
		resolver.settings.MCPTokenPepper,
		record.TokenSecretHash,
	) {
		return ResolvedActor{}, resolver.unauthorized(request)
	}

	user, err := resolver.deps.getUserByID(request.Context(), resolver.db, record.OwnerUserID)
	if err != nil {
		return ResolvedActor{}, resolver.internalError()
	}
	if user == nil || !user.IsActive {
		return ResolvedActor{}, resolver.unauthorized(request)
	}

	if err := resolver.deps.touchTokenLastUsed(request.Context(), resolver.db, record.TokenID); err != nil {
		return ResolvedActor{}, resolver.internalError()
	}

	tokenAuth := resolver.deps.buildAuthContext(*record)
	return ResolvedActor{
		Actor: normalizeActor(Actor{
			UserID:   user.UserID,
			Username: strings.TrimSpace(user.Username),
			Role:     string(user.Role),
		}),
		TokenAuth: &tokenAuth,
	}, nil
}

func normalizeActor(actor Actor) Actor {
	normalized := actor
	normalized.Role = strings.ToLower(strings.TrimSpace(actor.Role))
	normalized.Username = strings.TrimSpace(actor.Username)
	return normalized
}

func (resolver *ActorResolver) unauthorized(request *http.Request) error {
	headers := map[string]string{}
	if resolver.settings.OAuthEnabled {
		issuer := oauth.IssuerURLForRequest(request, resolver.settings)
		if issuer != "" {
			metadataURL := fmt.Sprintf("%s/.well-known/oauth-protected-resource", strings.TrimRight(issuer, "/"))
			headers["WWW-Authenticate"] = fmt.Sprintf(
				`Bearer realm="engram-mcp", resource_metadata="%s", scope="mcp:write"`,
				metadataURL,
			)
		}
	}
	return &AuthError{
		StatusCode: http.StatusUnauthorized,
		Detail:     "Authentication required",
		Headers:    headers,
	}
}

func (resolver *ActorResolver) internalError() error {
	return &AuthError{
		StatusCode: http.StatusInternalServerError,
		Detail:     "Internal server error",
	}
}

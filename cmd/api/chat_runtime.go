package main

import (
	"context"
	"net/http"
	"time"

	internalapi "engram/internal/api"
	"engram/internal/chat"
	"engram/internal/config"
	"engram/internal/models"
	"engram/internal/providers"
	"engram/internal/repository"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func buildChatRouter(
	settings config.Settings,
	pool *pgxpool.Pool,
	observability chat.ObservabilityRecorder,
) chi.Router {
	sessionService := chat.NewSessionOperationsService(pool)
	messageService := buildChatService(settings, pool, observability)
	return internalapi.CreateChatRouter(
		sessionService,
		messageService,
		sessionService,
		chatActorResolverFromContext,
	)
}

func buildChatService(
	settings config.Settings,
	pool *pgxpool.Pool,
	observability chat.ObservabilityRecorder,
) *chat.ChatService {
	messageRuntime := chat.NewChatMessageRuntime(buildChatMessageRuntimeDependencies(settings, pool))
	return chat.NewChatService(
		chat.ChatServiceDependencies{
			Runtime:                        messageRuntime,
			ResolveProvider:                resolveChatProviderDependency(settings),
			RunSessionLifecycleMaintenance: runSessionLifecycleMaintenanceDependency(settings, pool),
			ProviderFallback:               buildProviderFallbackStrategy(settings),
			CircuitPolicy:                  buildProviderCircuitPolicy(),
			Observability:                  observability,
			ResolveTraceID:                 resolveChatTraceID,
			NowUTC:                         func() time.Time { return time.Now().UTC() },
		},
	)
}

func buildProviderFallbackStrategy(settings config.Settings) chat.ProviderFallbackStrategy {
	return chat.NewStaticProviderFallbackStrategy(
		chat.ProviderFallbackStrategyOptions{
			Enabled:                true,
			FallbackOrder:          providerFallbackOrder(settings),
			DefaultFallbackModelID: settings.DefaultChatModel,
		},
	)
}

func providerFallbackOrder(settings config.Settings) []models.ChatProvider {
	ordered := []models.ChatProvider{
		models.ChatProviderOpenAI,
		models.ChatProviderAnthropic,
		models.ChatProviderBedrock,
	}
	if parsed, err := models.ParseChatProvider(settings.DefaultChatProvider); err == nil {
		ordered = append([]models.ChatProvider{parsed}, ordered...)
	}
	seen := make(map[models.ChatProvider]struct{}, len(ordered))
	unique := make([]models.ChatProvider, 0, len(ordered))
	for _, provider := range ordered {
		if _, exists := seen[provider]; exists {
			continue
		}
		seen[provider] = struct{}{}
		unique = append(unique, provider)
	}
	return unique
}

func buildProviderCircuitPolicy() chat.ProviderCircuitPolicy {
	return chat.NewSimpleProviderCircuitPolicy(chat.ProviderCircuitPolicyOptions{
		Enabled: true,
	})
}

func resolveChatTraceID(ctx context.Context) string {
	return middleware.GetReqID(ctx)
}

func buildChatMessageRuntimeDependencies(
	settings config.Settings,
	pool *pgxpool.Pool,
) chat.ChatMessageRuntimeDependencies {
	chatContextDependencies := chat.DefaultChatContextDependencies(pool)
	return chat.ChatMessageRuntimeDependencies{
		EmbeddingDim:              settings.EmbeddingDim,
		ChatDebugEnabled:          settings.ChatDebugEnabled,
		ChatDebugIncludeRawOutput: settings.ChatDebugIncludeRawText,
		GetSession:                getChatSessionDependency(pool),
		CreateChatMessage:         createChatMessageDependency(pool),
		ListChatMessages:          listChatMessagesDependency(pool),
		AssembleChatContext: func(ctx context.Context, request chat.ChatContextRequest) (chat.AssembledChatContext, error) {
			return chat.AssembleChatContext(ctx, request, chatContextDependencies)
		},
	}
}

func getChatSessionDependency(
	pool *pgxpool.Pool,
) func(ctx context.Context, actorUserID uuid.UUID, sessionID uuid.UUID) (*models.ChatSessionRecord, error) {
	return func(ctx context.Context, actorUserID uuid.UUID, sessionID uuid.UUID) (*models.ChatSessionRecord, error) {
		return repository.GetChatSession(
			ctx,
			pool,
			repository.ChatSessionGetInput{
				SessionID:   sessionID,
				ActorUserID: actorUserID,
			},
		)
	}
}

func createChatMessageDependency(
	pool *pgxpool.Pool,
) func(ctx context.Context, input chat.RuntimeMessageCreateInput) (*models.ChatMessageRecord, error) {
	return func(ctx context.Context, input chat.RuntimeMessageCreateInput) (*models.ChatMessageRecord, error) {
		return repository.CreateChatMessage(
			ctx,
			pool,
			repository.ChatMessageCreateInput{
				SessionID:   input.SessionID,
				ActorUserID: input.ActorUserID,
				Role:        input.Role,
				ContentText: input.ContentText,
				Metadata:    mapRuntimeMessageMetadata(input.Metadata),
			},
		)
	}
}

func listChatMessagesDependency(
	pool *pgxpool.Pool,
) func(
	ctx context.Context,
	sessionID uuid.UUID,
	actorUserID uuid.UUID,
	limit int,
	offset int,
) ([]models.ChatMessageRecord, error) {
	return pagedChatSessionQueryDependency(pool, repository.ListChatMessages, buildChatMessageListInput)
}

func mapRuntimeMessageMetadata(metadata *chat.RuntimeMessageMetadata) *repository.ChatMessageMetadata {
	if metadata == nil {
		return nil
	}
	return &repository.ChatMessageMetadata{
		Provider:       metadata.Provider,
		ModelID:        metadata.ModelID,
		TokenUsageJSON: intMapToAny(metadata.TokenUsage),
		UsedEngramIDs:  append([]uuid.UUID(nil), metadata.UsedEngramIDs...),
	}
}

func intMapToAny(input map[string]int) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func resolveChatProviderDependency(
	settings config.Settings,
) func(provider models.ChatProvider) (providers.ChatProviderAdapter, error) {
	return func(provider models.ChatProvider) (providers.ChatProviderAdapter, error) {
		return providers.GetProviderAdapter(provider, settings)
	}
}

func runSessionLifecycleMaintenanceDependency(
	settings config.Settings,
	pool *pgxpool.Pool,
) func(ctx context.Context, actorUserID uuid.UUID, session models.ChatSessionRecord) error {
	lifecycleDependencies := buildSessionLifecycleDependencies(pool)
	return func(ctx context.Context, actorUserID uuid.UUID, session models.ChatSessionRecord) error {
		_, err := chat.RunSessionLifecycleMaintenance(
			ctx,
			actorUserID,
			session,
			settings.EmbeddingDim,
			lifecycleDependencies,
		)
		return err
	}
}

func buildSessionLifecycleDependencies(pool *pgxpool.Pool) chat.SessionLifecycleDependencies {
	return chat.SessionLifecycleDependencies{
		ListSessionLinkedEngrams:     listSessionLinkedEngramsDependency(pool),
		ListChatMessages:             listChatMessagesDependency(pool),
		CountSessionMessagesByRole:   countSessionMessagesByRoleDependency(pool),
		DeleteSessionAutosaveEngrams: deleteSessionAutosaveEngramsDependency(pool),
		CreateEngram:                 createLifecycleEngramDependency(pool),
	}
}

func listSessionLinkedEngramsDependency(pool *pgxpool.Pool) func(
	ctx context.Context,
	sessionID uuid.UUID,
	actorUserID uuid.UUID,
	limit int,
	offset int,
) ([]models.EngramSummary, error) {
	return pagedChatSessionQueryDependency(
		pool,
		repository.ListSessionLinkedEngrams,
		func(
			sessionID uuid.UUID,
			actorUserID uuid.UUID,
			limit int,
			offset int,
		) repository.SessionLinkedEngramsListInput {
			return toSessionLinkedEngramsListInput(
				buildChatMessageListInput(sessionID, actorUserID, limit, offset),
			)
		},
	)
}

func pagedChatSessionQueryDependency[Input any, Record any](
	pool *pgxpool.Pool,
	operation func(ctx context.Context, db repository.Queryer, input Input) ([]Record, error),
	buildInput func(sessionID uuid.UUID, actorUserID uuid.UUID, limit int, offset int) Input,
) func(
	ctx context.Context,
	sessionID uuid.UUID,
	actorUserID uuid.UUID,
	limit int,
	offset int,
) ([]Record, error) {
	return func(
		ctx context.Context,
		sessionID uuid.UUID,
		actorUserID uuid.UUID,
		limit int,
		offset int,
	) ([]Record, error) {
		return operation(
			ctx,
			pool,
			buildInput(sessionID, actorUserID, limit, offset),
		)
	}
}

func buildChatMessageListInput(
	sessionID uuid.UUID,
	actorUserID uuid.UUID,
	limit int,
	offset int,
) repository.ChatMessageListInput {
	return repository.ChatMessageListInput{
		SessionID:   sessionID,
		ActorUserID: actorUserID,
		Limit:       limit,
		Offset:      offset,
	}
}

func toSessionLinkedEngramsListInput(
	input repository.ChatMessageListInput,
) repository.SessionLinkedEngramsListInput {
	return repository.SessionLinkedEngramsListInput{
		SessionID:   input.SessionID,
		ActorUserID: input.ActorUserID,
		Limit:       input.Limit,
		Offset:      input.Offset,
	}
}

func countSessionMessagesByRoleDependency(pool *pgxpool.Pool) func(
	ctx context.Context,
	sessionID uuid.UUID,
	actorUserID uuid.UUID,
	role string,
) (int, error) {
	return func(
		ctx context.Context,
		sessionID uuid.UUID,
		actorUserID uuid.UUID,
		role string,
	) (int, error) {
		return repository.CountSessionMessagesByRole(
			ctx,
			pool,
			repository.SessionMessageRoleCountInput{
				SessionID:   sessionID,
				ActorUserID: actorUserID,
				Role:        role,
			},
		)
	}
}

func deleteSessionAutosaveEngramsDependency(pool *pgxpool.Pool) func(
	ctx context.Context,
	sessionID uuid.UUID,
	actorUserID uuid.UUID,
	engramIDs []uuid.UUID,
) ([]uuid.UUID, error) {
	return func(
		ctx context.Context,
		sessionID uuid.UUID,
		actorUserID uuid.UUID,
		engramIDs []uuid.UUID,
	) ([]uuid.UUID, error) {
		return repository.DeleteSessionAutosaveEngrams(
			ctx,
			pool,
			repository.SessionAutosaveDeleteInput{
				SessionID:   sessionID,
				ActorUserID: actorUserID,
				EngramIDs:   append([]uuid.UUID(nil), engramIDs...),
			},
		)
	}
}

func createLifecycleEngramDependency(pool *pgxpool.Pool) func(
	ctx context.Context,
	payload models.MemoryEngramCreate,
	embeddingDim int,
	ownerUserID uuid.UUID,
	enrichmentOrigin string,
) (*models.EngramCreateResponse, error) {
	return func(
		ctx context.Context,
		payload models.MemoryEngramCreate,
		embeddingDim int,
		ownerUserID uuid.UUID,
		enrichmentOrigin string,
	) (*models.EngramCreateResponse, error) {
		return repository.CreateEngram(
			ctx,
			pool,
			repository.CreateEngramInput{
				Payload:          payload,
				EmbeddingDim:     embeddingDim,
				OwnerUserID:      &ownerUserID,
				EnrichmentOrigin: enrichmentOrigin,
			},
		)
	}
}

func chatActorResolverFromContext(request *http.Request) (map[string]any, error) {
	actor, err := internalapi.RequireAdminActorFromContext(request)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"user_id": actor.UserID.String(),
		"role":    actor.Role,
	}, nil
}

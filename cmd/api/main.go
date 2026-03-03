package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"engram/internal/admin"
	internalapi "engram/internal/api"
	"engram/internal/audit"
	"engram/internal/auth"
	"engram/internal/chat"
	"engram/internal/config"
	"engram/internal/db"
	"engram/internal/embeddings"
	internalexport "engram/internal/export"
	"engram/internal/graph"
	"engram/internal/ingestion"
	"engram/internal/mcp"
	"engram/internal/mcptokens"
	"engram/internal/models"
	"engram/internal/oauth"
	"engram/internal/projects"
	"engram/internal/repository"
	"engram/internal/workflow"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sessionAuthRuntimeDependencies struct {
	settings          config.Settings
	pool              *pgxpool.Pool
	sessionManager    *auth.SessionManager
	oidcProvider      auth.OIDCLoginProvider
	loginAttemptGuard *auth.LoginAttemptGuard
	auditLogger       *audit.Logger
	projectService    projectResolutionService
	mcpTokenService   *mcptokens.Service
}

type mcpCompatibilityRuntimeDependencies struct {
	pool               *pgxpool.Pool
	projectService     *projects.Service
	memoryAdminService *admin.Service
	exportService      internalexport.Service
}

type runtimeServices struct {
	projectService           *projects.Service
	oauthRegistrationService *oauth.RegistrationService
	oauthAuthorization       *oauth.AuthorizationService
	oauthTokenService        *oauth.TokenService
	memoryAdminService       *admin.Service
	mcpTokenService          *mcptokens.Service
	exportService            internalexport.Service
	mcpService               *mcp.CompatibilityService
	mcpTransportLimiter      *auth.RequestRateLimiter
	mcpActorResolver         *mcp.ActorResolver
	agentWorkflowService     *workflow.Service
	ingestionService         *ingestion.Service
	loginAttemptGuard        *auth.LoginAttemptGuard
	auditLogger              *audit.Logger
}

var errMissingSessionActorForMCP = errors.New("session actor missing for mcp")

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	startupCtx := context.Background()
	settings := loadSettingsOrExit(logger)
	pool := initDBPoolOrExit(startupCtx, logger, settings)
	defer pool.Close()

	handler := buildHandlerOrExit(startupCtx, logger, settings, pool)
	server := buildServer(settings, handler)
	runServerOrExit(logger, server)
}

func loadSettingsOrExit(logger *slog.Logger) config.Settings {
	settings, err := config.LoadSettings()
	if err != nil {
		logger.Error("failed to load settings", "error", err)
		os.Exit(1)
	}
	return settings
}

func initDBPoolOrExit(ctx context.Context, logger *slog.Logger, settings config.Settings) *pgxpool.Pool {
	pool, err := db.NewPool(ctx, settings)
	if err != nil {
		logger.Error("failed to initialize db pool", "error", err)
		os.Exit(1)
	}
	if err := db.Ping(ctx, pool); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	if err := db.EnsureSchemaInitialized(
		ctx,
		db.PgxPoolBeginner{Pool: pool},
		db.SchemaInitializationOptions{
			Settings:     settings,
			SchemaPath:   db.DefaultSchemaPath(),
			HashPassword: hashPasswordWithRandomSalt,
		},
	); err != nil {
		logger.Error("failed to initialize database schema", "error", err)
		os.Exit(1)
	}
	return pool
}

func buildHandlerOrExit(
	ctx context.Context,
	logger *slog.Logger,
	settings config.Settings,
	pool *pgxpool.Pool,
) http.Handler {
	configureEmbeddingsOrExit(logger, settings)
	sessionManager, err := auth.NewSessionManager(settings.AppSessionSecret, auth.DefaultSessionCookieName)
	if err != nil {
		logger.Error("failed to initialize session manager", "error", err)
		os.Exit(1)
	}
	oidcProvider, err := auth.NewOIDCLoginProvider(ctx, settings)
	if err != nil {
		logger.Error("failed to initialize oidc login provider", "error", err)
		os.Exit(1)
	}
	requestLogger := resolveRequestLogger(logger, settings)
	requestMetrics := resolveRequestMetrics(settings)
	chatObservability := resolveChatObservabilityRecorder(requestMetrics)
	services := initializeRuntimeServices(settings, pool, chatObservability)

	routerDependencies := internalapi.RouterDependencies{
		MemoryAdminService: services.memoryAdminService,
		RequireAdminActor:  internalapi.RequireAdminActorFromContext,
		SessionAuth: buildSessionAuthDependencies(
			sessionAuthRuntimeDependencies{
				settings:          settings,
				pool:              pool,
				sessionManager:    sessionManager,
				oidcProvider:      oidcProvider,
				loginAttemptGuard: services.loginAttemptGuard,
				auditLogger:       services.auditLogger,
				projectService:    services.projectService,
				mcpTokenService:   services.mcpTokenService,
			},
		),
		ProjectsService:  newProjectRouteServiceAdapter(services.projectService),
		IngestionService: newIngestionRouteServiceAdapter(services.ingestionService),
		IngestionOptions: internalapi.IngestionRouteOptions{
			MaxMetadataJSONBytes: settings.IngestionMaxMetadataJSONBytes,
		},
		OAuthRegistration:   services.oauthRegistrationService,
		OAuthAuthorization:  services.oauthAuthorization,
		OAuthToken:          services.oauthTokenService,
		ChatRouter:          buildChatRouter(settings, pool, chatObservability),
		AgentWorkflow:       services.agentWorkflowService,
		ExportService:       services.exportService,
		MCPService:          services.mcpService,
		MCPActorResolver:    services.mcpActorResolver,
		MCPTransportLimiter: services.mcpTransportLimiter,
		RequestLogger:       requestLogger,
		RequestMetrics:      requestMetrics,
	}
	handler := internalapi.NewRouterWithDependencies(settings, routerDependencies)
	handler = internalapi.SessionActorMiddleware(sessionManager, lookupSessionUser(pool))(handler)
	logger.Info("session-authenticated actor context enabled")
	return handler
}

func configureEmbeddingsOrExit(logger *slog.Logger, settings config.Settings) {
	timeout := time.Duration(settings.EmbeddingTimeoutSeconds * float64(time.Second))
	if err := embeddings.ConfigureDefault(
		embeddings.RuntimeConfig{
			Provider:        settings.EmbeddingProvider,
			Model:           settings.EmbeddingModel,
			FallbackToLocal: settings.EmbeddingFallbackToLocal,
			OpenAIAPIKey:    settings.OpenAIAPIKey,
			OpenAIBaseURL:   settings.OpenAIBaseURL,
			Timeout:         timeout,
		},
	); err != nil {
		logger.Error("failed to configure embeddings provider", "error", err)
		os.Exit(1)
	}
}

func initializeRuntimeServices(
	settings config.Settings,
	pool *pgxpool.Pool,
	chatObservability chat.ObservabilityRecorder,
) runtimeServices {
	projectService := projects.NewService(pool)
	memoryAdminService := admin.NewService(pool, settings.EmbeddingDim, newAdminProjectResolver(projectService))
	exportService := internalexport.NewService(pool, projectService, memoryAdminService, settings.EmbeddingDim)
	return runtimeServices{
		projectService:           projectService,
		oauthRegistrationService: oauth.NewRegistrationService(pool),
		oauthAuthorization:       oauth.NewAuthorizationService(pool),
		oauthTokenService:        oauth.NewTokenService(pool),
		memoryAdminService:       memoryAdminService,
		mcpTokenService:          mcptokens.NewService(pool),
		exportService:            exportService,
		mcpService: newMCPCompatibilityService(
			settings,
			mcpCompatibilityRuntimeDependencies{
				pool:               pool,
				projectService:     projectService,
				memoryAdminService: memoryAdminService,
				exportService:      exportService,
			},
			chatObservability,
		),
		mcpTransportLimiter: newMCPTransportRateLimiter(settings, pool),
		mcpActorResolver: mcp.NewActorResolver(
			settings,
			pool,
			resolveMCPActorFromSessionContext,
		),
		agentWorkflowService: workflow.NewService(newWorkflowEngramCreator(pool, settings.EmbeddingDim)),
		ingestionService: ingestion.NewService(
			pool,
			settings.EmbeddingDim,
			settings.IngestionMaxFileBytes,
			settings.IngestionMaxTextChars,
		),
		loginAttemptGuard: newLoginAttemptGuard(settings, pool),
		auditLogger:       newSessionAuditLogger(settings),
	}
}

func resolveRequestLogger(logger *slog.Logger, settings config.Settings) *slog.Logger {
	if !settings.APIRequestLogEnabled {
		return nil
	}
	return logger
}

func resolveRequestMetrics(settings config.Settings) internalapi.RequestMetricsRecorder {
	if !settings.APIMetricsEnabled {
		return nil
	}
	return internalapi.NewInMemoryRequestMetrics()
}

func newMCPCompatibilityService(
	settings config.Settings,
	dependencies mcpCompatibilityRuntimeDependencies,
	chatObservability chat.ObservabilityRecorder,
) *mcp.CompatibilityService {
	messageSend, messageStream := resolveMCPMessageServices(
		settings,
		dependencies.pool,
		chatObservability,
	)
	compatibilityDeps := buildMCPCompatibilityDependencies(
		settings,
		dependencies,
		messageSend,
		messageStream,
	)

	return mcp.NewCompatibilityServiceWithDependencies(
		settings.AppSemanticVersion,
		compatibilityDeps,
	)
}

func buildMCPCompatibilityDependencies(
	settings config.Settings,
	dependencies mcpCompatibilityRuntimeDependencies,
	messageSend mcp.MessageSendService,
	messageStream mcp.MessageStreamService,
) mcp.CompatibilityServiceDependencies {
	compatibilityDeps := mcp.CompatibilityServiceDependencies{
		MCPToolPolicyVersion:   settings.MCPToolPolicyVersion,
		EvalSuiteVersion:       settings.EvalSuiteVersion,
		ProjectService:         dependencies.projectService,
		ProjectExport:          newMCPProjectExportAdapter(dependencies.exportService),
		ProjectImport:          newMCPProjectImportAdapter(dependencies.exportService),
		SessionService:         newMCPSessionListAdapter(dependencies.pool),
		SessionGet:             newMCPSessionGetAdapter(dependencies.pool),
		SessionCreate:          newMCPSessionCreateAdapter(dependencies.pool),
		SessionContinue:        newMCPSessionContinueAdapter(dependencies.pool),
		SessionSaveAsEngram:    newMCPSaveSessionAsEngramAdapter(dependencies.pool),
		SessionDelete:          newMCPSessionDeleteAdapter(dependencies.memoryAdminService),
		SessionRestore:         newMCPSessionRestoreAdapter(dependencies.memoryAdminService),
		LifecyclePolicyUpdate:  newMCPLifecyclePolicyUpdateAdapter(dependencies.pool),
		MessageService:         newMCPMessageListAdapter(dependencies.pool),
		MessageSend:            messageSend,
		MessageStream:          messageStream,
		TimelineService:        newMCPTimelineListAdapter(dependencies.pool),
		PinnedEngramService:    newMCPPinnedEngramListAdapter(dependencies.pool),
		PinnedDocumentService:  newMCPPinnedDocumentListAdapter(dependencies.pool),
		ProjectDocumentService: newMCPProjectDocumentListAdapter(dependencies.pool),
		PinEngramService:       newMCPPinEngramAdapter(dependencies.pool),
		UnpinEngramService:     newMCPUnpinEngramAdapter(dependencies.pool),
		PinDocumentService:     newMCPPinDocumentAdapter(dependencies.pool),
		UnpinDocumentService:   newMCPUnpinDocumentAdapter(dependencies.pool),
	}
	applyMCPEngramCompatibilityDependencies(&compatibilityDeps, settings, dependencies)
	return compatibilityDeps
}

func applyMCPEngramCompatibilityDependencies(
	compatibilityDeps *mcp.CompatibilityServiceDependencies,
	settings config.Settings,
	dependencies mcpCompatibilityRuntimeDependencies,
) {
	if compatibilityDeps == nil {
		return
	}
	compatibilityDeps.EngramCreate = newMCPEngramCreateAdapter(
		dependencies.pool,
		dependencies.projectService,
		settings.EmbeddingDim,
	)
	compatibilityDeps.EngramCreateConversation = newMCPEngramCreateFromConversationAdapter(
		dependencies.pool,
		dependencies.projectService,
		settings.EmbeddingDim,
	)
	compatibilityDeps.EngramList = newMCPEngramListAdapter(dependencies.memoryAdminService)
	compatibilityDeps.EngramGet = newMCPEngramGetAdapter(dependencies.memoryAdminService)
	compatibilityDeps.EngramQuery = newMCPEngramQueryAdapter(dependencies.pool, settings.EmbeddingDim)
	compatibilityDeps.EngramRehydrate = newMCPEngramRehydrateAdapter(dependencies.pool)
	compatibilityDeps.EngramLinkGet = newMCPEngramLinkGetAdapter(dependencies.pool)
	compatibilityDeps.EngramLinkCreate = newMCPEngramLinkCreateAdapter(dependencies.pool)
	compatibilityDeps.EngramLinkList = newMCPEngramLinkListAdapter(dependencies.pool)
	compatibilityDeps.EngramLinkUpdate = newMCPEngramLinkUpdateAdapter(dependencies.pool)
	compatibilityDeps.EngramLinkArchive = newMCPEngramLinkArchiveAdapter(dependencies.pool)
	compatibilityDeps.EngramLinkSuggest = newMCPEngramLinkSuggestAdapter(dependencies.pool, settings.EmbeddingDim)
	compatibilityDeps.EngramTracePath = newMCPEngramTracePathAdapter(dependencies.pool)
	compatibilityDeps.EngramUpdate = newMCPEngramUpdateAdapter(dependencies.memoryAdminService)
	compatibilityDeps.EngramFeedback = newMCPEngramFeedbackAdapter(dependencies.pool)
	compatibilityDeps.EngramFreshnessRefresh = newMCPEngramFreshnessRefreshAdapter(dependencies.memoryAdminService)
	compatibilityDeps.EngramConsolidationRefresh = newMCPEngramConsolidationRefreshAdapter(
		dependencies.memoryAdminService,
	)
	compatibilityDeps.EngramConsolidationList = newMCPEngramConsolidationListAdapter(
		dependencies.memoryAdminService,
	)
	compatibilityDeps.EngramConsolidationAction = newMCPEngramConsolidationActionAdapter(
		dependencies.memoryAdminService,
	)
	compatibilityDeps.EngramContradictionRefresh = newMCPEngramContradictionRefreshAdapter(
		dependencies.memoryAdminService,
	)
	compatibilityDeps.EngramContradictionList = newMCPEngramContradictionListAdapter(
		dependencies.memoryAdminService,
	)
	compatibilityDeps.EngramContradictionResolve = newMCPEngramContradictionResolveAdapter(
		dependencies.memoryAdminService,
	)
	compatibilityDeps.EngramCurationList = newMCPEngramCurationListAdapter(
		dependencies.memoryAdminService,
	)
	compatibilityDeps.EngramCurationRefresh = newMCPEngramCurationRefreshAdapter(
		dependencies.memoryAdminService,
	)
	compatibilityDeps.EngramCurationAction = newMCPEngramCurationActionAdapter(
		dependencies.memoryAdminService,
	)
	compatibilityDeps.EngramMove = newMCPEngramMoveAdapter(dependencies.memoryAdminService)
	compatibilityDeps.EngramDelete = newMCPEngramDeleteAdapter(dependencies.memoryAdminService)
	compatibilityDeps.EngramRestore = newMCPEngramRestoreAdapter(dependencies.memoryAdminService)
	compatibilityDeps.EngramCollectionList = newMCPEngramCollectionListAdapter(dependencies.memoryAdminService)
	compatibilityDeps.EngramCollectionGet = newMCPEngramCollectionGetAdapter(dependencies.memoryAdminService)
	compatibilityDeps.EngramCollectionCreate = newMCPEngramCollectionCreateAdapter(
		dependencies.memoryAdminService,
		dependencies.projectService,
	)
	compatibilityDeps.EngramCollectionUpdate = newMCPEngramCollectionUpdateAdapter(dependencies.memoryAdminService)
	compatibilityDeps.EngramCollectionDelete = newMCPEngramCollectionDeleteAdapter(dependencies.memoryAdminService)
	compatibilityDeps.EngramCollectionAddItems = newMCPEngramCollectionAddItemsAdapter(dependencies.memoryAdminService)
	compatibilityDeps.EngramCollectionRemove = newMCPEngramCollectionRemoveItemAdapter(
		dependencies.memoryAdminService,
	)
}

func resolveMCPMessageServices(
	settings config.Settings,
	pool *pgxpool.Pool,
	chatObservability chat.ObservabilityRecorder,
) (mcp.MessageSendService, mcp.MessageStreamService) {
	messageAdapter := newMCPMessageAdapter(settings, pool, chatObservability)
	if messageAdapter == nil {
		return nil, nil
	}
	return messageAdapter, messageAdapter
}

func resolveChatObservabilityRecorder(
	requestMetrics internalapi.RequestMetricsRecorder,
) chat.ObservabilityRecorder {
	recorder, ok := requestMetrics.(chat.ObservabilityRecorder)
	if !ok {
		return nil
	}
	return recorder
}

func resolveMCPActorFromSessionContext(request *http.Request) (mcp.Actor, error) {
	if request == nil {
		return mcp.Actor{}, errMissingSessionActorForMCP
	}
	actor, ok := internalapi.AdminActorFromContext(request.Context())
	if !ok {
		return mcp.Actor{}, errMissingSessionActorForMCP
	}
	return mcp.Actor{
		UserID: actor.UserID,
		Role:   actor.Role,
	}, nil
}

func newLoginAttemptGuard(settings config.Settings, pool *pgxpool.Pool) *auth.LoginAttemptGuard {
	loginAttemptGuard := auth.NewLoginAttemptGuard(
		settings.LoginRateLimitMaxAttempts,
		settings.LoginRateLimitWindowSeconds,
		settings.LoginLockoutSeconds,
	)
	rateLimitStore := auth.NewPGXRateLimitStore(pool)
	loginAttemptGuard.SetDistributedStore(auth.DefaultLoginAttemptNamespace, rateLimitStore)
	return loginAttemptGuard
}

func newMCPTransportRateLimiter(settings config.Settings, pool *pgxpool.Pool) *auth.RequestRateLimiter {
	limiter := auth.NewRequestRateLimiter(
		settings.MCPTransportRateLimitMaxRequests,
		settings.MCPTransportRateLimitWindowSeconds,
		settings.MCPTransportRateLimitBlockSeconds,
		"mcp_transport",
	)
	limiter.SetDistributedStore(auth.NewPGXRateLimitStore(pool))
	return limiter
}

func newSessionAuditLogger(settings config.Settings) *audit.Logger {
	return audit.NewLogger(audit.LoggerOptions{
		Path:          settings.AuditLogPath,
		StdoutEnabled: settings.AuditLogStdoutEnabled || config.IsProductionEnv(settings),
		MaxEventBytes: settings.AuditLogMaxEventBytes,
		SinkURL:       settings.AuditSinkURL,
		SinkAuthToken: settings.AuditSinkAuthToken,
		SinkRequired:  settings.AuditSinkRequired,
		SinkTimeout:   time.Duration(settings.AuditSinkTimeoutSeconds * float64(time.Second)),
	})
}

func buildSessionAuthDependencies(runtimeDependencies sessionAuthRuntimeDependencies) internalapi.SessionAuthDependencies {
	linkAdapter := newSessionEngramLinkRepositoryAdapter(runtimeDependencies.pool)
	return internalapi.SessionAuthDependencies{
		SessionManager:           runtimeDependencies.sessionManager,
		OIDCProvider:             runtimeDependencies.oidcProvider,
		LookupUserByUsername:     lookupSessionUserByUsername(runtimeDependencies.pool),
		LookupUserByID:           lookupSessionUser(runtimeDependencies.pool),
		VerifyPassword:           auth.VerifyPassword,
		GenerateCSRFToken:        auth.GenerateCSRFToken,
		HashPassword:             hashPasswordWithRandomSalt,
		ListUsers:                listUsersDependency(runtimeDependencies.pool),
		CreateUser:               createUserDependency(runtimeDependencies.pool),
		UpdateUser:               updateUserDependency(runtimeDependencies.pool),
		ResolveProjectIDForWrite: resolveProjectIDForWriteDependency(runtimeDependencies.projectService),
		CreateEngram:             createEngramDependency(runtimeDependencies.pool, runtimeDependencies.settings.EmbeddingDim),
		ListEngrams:              listEngramsDependency(runtimeDependencies.pool),
		QueryEngrams:             queryEngramsDependency(runtimeDependencies.pool, runtimeDependencies.settings.EmbeddingDim),
		GetRehydrationBundle:     getRehydrationBundleDependency(runtimeDependencies.pool),
		GetEngramSources:         getEngramSourcesDependency(runtimeDependencies.pool),
		SubmitEngramFeedback:     submitEngramFeedbackDependency(runtimeDependencies.pool),
		ShareEngram:              shareEngramDependency(runtimeDependencies.projectService),
		UnshareEngram:            unshareEngramDependency(runtimeDependencies.projectService),
		CreateEngramLink:         linkAdapter.create,
		ListEngramLinks:          linkAdapter.list,
		UpdateEngramLink:         linkAdapter.update,
		ArchiveEngramLink:        linkAdapter.archive,
		SuggestEngramLinks:       suggestEngramLinksDependency(runtimeDependencies.pool, runtimeDependencies.settings.EmbeddingDim),
		HygieneEngramLinks:       hygieneEngramLinksDependency(runtimeDependencies.pool),
		TraceEngramLinks:         linkAdapter.trace,
		CreateTokenForOwner:      createMCPTokenForOwnerDependency(runtimeDependencies.mcpTokenService),
		ListTokenSummaries:       listMCPTokenSummariesDependency(runtimeDependencies.mcpTokenService),
		RevokeTokenForOwner:      revokeMCPTokenForOwnerDependency(runtimeDependencies.mcpTokenService),
		MCPTokenPepper:           runtimeDependencies.settings.MCPTokenPepper,
		CookieSecure:             config.IsProductionEnv(runtimeDependencies.settings),
		LoginAttemptGuard:        runtimeDependencies.loginAttemptGuard,
		LogAuditEvent:            auditEventLogger(runtimeDependencies.auditLogger),
	}
}

func hashPasswordWithRandomSalt(password string) (string, error) {
	return auth.HashPassword(password, nil)
}

func listUsersDependency(pool *pgxpool.Pool) func(ctx context.Context, limit, offset int) ([]models.UserRecord, error) {
	return func(ctx context.Context, limit, offset int) ([]models.UserRecord, error) {
		return repository.ListUsers(ctx, pool, limit, offset)
	}
}

func createUserDependency(
	pool *pgxpool.Pool,
) func(ctx context.Context, input internalapi.SessionUserCreateInput) (*models.UserRecord, error) {
	return func(ctx context.Context, input internalapi.SessionUserCreateInput) (*models.UserRecord, error) {
		return repository.CreateUser(
			ctx,
			pool,
			repository.CreateUserInput{
				Username:     input.Username,
				PasswordHash: input.PasswordHash,
				Role:         input.Role,
				IsActive:     input.IsActive,
			},
		)
	}
}

func updateUserDependency(
	pool *pgxpool.Pool,
) func(ctx context.Context, userID uuid.UUID, input internalapi.SessionUserUpdateInput) (*models.UserRecord, error) {
	return func(
		ctx context.Context,
		userID uuid.UUID,
		input internalapi.SessionUserUpdateInput,
	) (*models.UserRecord, error) {
		return repository.UpdateUser(
			ctx,
			pool,
			userID,
			repository.UserUpdateInput{
				Role:         input.Role,
				IsActive:     input.IsActive,
				PasswordHash: input.PasswordHash,
			},
		)
	}
}

func createEngramDependency(
	pool *pgxpool.Pool,
	embeddingDim int,
) func(ctx context.Context, payload models.MemoryEngramCreate, ownerUserID uuid.UUID) (*models.EngramCreateResponse, error) {
	return func(
		ctx context.Context,
		payload models.MemoryEngramCreate,
		ownerUserID uuid.UUID,
	) (*models.EngramCreateResponse, error) {
		return repository.CreateEngram(
			ctx,
			pool,
			repository.CreateEngramInput{
				Payload:          payload,
				EmbeddingDim:     embeddingDim,
				OwnerUserID:      &ownerUserID,
				EnrichmentOrigin: "api.engrams.create",
			},
		)
	}
}

func listEngramsDependency(
	pool *pgxpool.Pool,
) func(ctx context.Context, projectID *string, limit, offset int, actorUserID uuid.UUID) ([]models.EngramSummary, error) {
	return func(
		ctx context.Context,
		projectID *string,
		limit int,
		offset int,
		actorUserID uuid.UUID,
	) ([]models.EngramSummary, error) {
		return repository.ListEngrams(
			ctx,
			pool,
			repository.ListEngramsInput{
				ProjectID:   projectID,
				Limit:       limit,
				Offset:      offset,
				ActorUserID: &actorUserID,
			},
		)
	}
}

func queryEngramsDependency(
	pool *pgxpool.Pool,
	embeddingDim int,
) func(ctx context.Context, request models.EngramQueryRequest, actorUserID uuid.UUID) ([]models.EngramQueryResult, error) {
	return func(
		ctx context.Context,
		request models.EngramQueryRequest,
		actorUserID uuid.UUID,
	) ([]models.EngramQueryResult, error) {
		queryLiteral, err := repository.BuildLocalQueryLiteral(request.Query, embeddingDim)
		if err != nil {
			return nil, err
		}
		return repository.QueryEngrams(
			ctx,
			pool,
			repository.QueryEngramsInput{
				Request:      request,
				QueryLiteral: queryLiteral,
				ActorUserID:  &actorUserID,
			},
		)
	}
}

func getRehydrationBundleDependency(
	pool *pgxpool.Pool,
) func(ctx context.Context, engramID uuid.UUID, actorUserID uuid.UUID) (*models.RehydrationBundle, error) {
	return func(
		ctx context.Context,
		engramID uuid.UUID,
		actorUserID uuid.UUID,
	) (*models.RehydrationBundle, error) {
		return repository.GetRehydrationBundle(
			ctx,
			pool,
			repository.RehydrationInput{
				EngramID:    engramID,
				ActorUserID: &actorUserID,
			},
		)
	}
}

func getEngramSourcesDependency(
	pool *pgxpool.Pool,
) func(ctx context.Context, engramID uuid.UUID, limit int, actorUserID uuid.UUID) ([]models.EngramSourceRecord, error) {
	return func(
		ctx context.Context,
		engramID uuid.UUID,
		limit int,
		actorUserID uuid.UUID,
	) ([]models.EngramSourceRecord, error) {
		return repository.GetEngramSources(
			ctx,
			pool,
			repository.EngramSourceListInput{
				EngramID:    engramID,
				Limit:       limit,
				ActorUserID: &actorUserID,
			},
		)
	}
}

func submitEngramFeedbackDependency(
	pool *pgxpool.Pool,
) func(
	ctx context.Context,
	input internalapi.SessionEngramFeedbackInput,
) (*models.EngramFeedbackRecord, error) {
	return func(
		ctx context.Context,
		input internalapi.SessionEngramFeedbackInput,
	) (*models.EngramFeedbackRecord, error) {
		return repository.RecordEngramFeedback(
			ctx,
			pool,
			repository.EngramFeedbackCreateInput{
				EngramID:     input.EngramID,
				ActorUserID:  input.ActorUserID,
				FeedbackType: input.FeedbackType,
				Note:         input.Note,
			},
		)
	}
}

func shareEngramDependency(
	projectService projectResolutionService,
) func(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engramID uuid.UUID,
) (*models.EngramVisibilityRecord, error) {
	return buildEngramVisibilityDependency(projectService, (*projects.Service).ShareEngram)
}

func unshareEngramDependency(
	projectService projectResolutionService,
) func(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engramID uuid.UUID,
) (*models.EngramVisibilityRecord, error) {
	return buildEngramVisibilityDependency(projectService, (*projects.Service).UnshareEngram)
}

type engramVisibilityMutation func(
	service *projects.Service,
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engramID uuid.UUID,
) (*models.EngramVisibilityRecord, error)

func buildEngramVisibilityDependency(
	projectService projectResolutionService,
	mutation engramVisibilityMutation,
) func(
	ctx context.Context,
	actorUserID uuid.UUID,
	actorRole models.UserRole,
	engramID uuid.UUID,
) (*models.EngramVisibilityRecord, error) {
	concreteService, ok := projectService.(*projects.Service)
	if !ok || concreteService == nil {
		return nil
	}
	return func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		engramID uuid.UUID,
	) (*models.EngramVisibilityRecord, error) {
		return mutation(concreteService, ctx, actorUserID, actorRole, engramID)
	}
}

type sessionEngramLinkRepositoryAdapter struct {
	pool repository.Queryer
}

func newSessionEngramLinkRepositoryAdapter(pool repository.Queryer) sessionEngramLinkRepositoryAdapter {
	return sessionEngramLinkRepositoryAdapter{pool: pool}
}

func (adapter sessionEngramLinkRepositoryAdapter) create(
	ctx context.Context,
	input internalapi.SessionEngramLinkCreateInput,
) (*models.EngramLinkRecord, error) {
	return repository.CreateEngramLink(
		ctx,
		adapter.pool,
		repository.EngramLinkCreateInput{
			SourceEngramID:   input.SourceEngramID,
			TargetEngramID:   input.TargetEngramID,
			RelationType:     input.RelationType,
			Weight:           input.Weight,
			TemporalWeight:   input.TemporalWeight,
			Confidence:       input.Confidence,
			Origin:           input.Origin,
			Status:           input.Status,
			EvidenceJSON:     input.EvidenceJSON,
			CreatedByUserID:  input.CreatedByUserID,
			ActorUserID:      input.ActorUserID,
			LastReinforcedAt: input.LastReinforcedAt,
		},
	)
}

func (adapter sessionEngramLinkRepositoryAdapter) list(
	ctx context.Context,
	input internalapi.SessionEngramLinkListInput,
) ([]models.EngramLinkRecord, error) {
	return repository.ListEngramLinks(
		ctx,
		adapter.pool,
		repository.EngramLinkListInput{
			SourceEngramID:  input.SourceEngramID,
			ActorUserID:     input.ActorUserID,
			RelationType:    input.RelationType,
			IncludeArchived: input.IncludeArchived,
			Limit:           input.Limit,
			Offset:          input.Offset,
		},
	)
}

func (adapter sessionEngramLinkRepositoryAdapter) update(
	ctx context.Context,
	input internalapi.SessionEngramLinkUpdateInput,
) (*models.EngramLinkRecord, error) {
	return repository.UpdateEngramLink(
		ctx,
		adapter.pool,
		repository.EngramLinkUpdateInput{
			LinkID:           input.LinkID,
			ActorUserID:      input.ActorUserID,
			Weight:           input.Weight,
			TemporalWeight:   input.TemporalWeight,
			Confidence:       input.Confidence,
			Status:           input.Status,
			EvidenceJSON:     input.EvidenceJSON,
			LastReinforcedAt: input.LastReinforcedAt,
		},
	)
}

func (adapter sessionEngramLinkRepositoryAdapter) archive(
	ctx context.Context,
	input internalapi.SessionEngramLinkArchiveInput,
) (*models.EngramLinkRecord, error) {
	return repository.ArchiveEngramLink(
		ctx,
		adapter.pool,
		repository.EngramLinkArchiveInput{
			LinkID:      input.LinkID,
			ActorUserID: input.ActorUserID,
		},
	)
}

func (adapter sessionEngramLinkRepositoryAdapter) trace(
	ctx context.Context,
	input internalapi.SessionEngramTraceInput,
) ([]models.EngramLinkTraversalStep, error) {
	return repository.TraverseEngramLinks(
		ctx,
		adapter.pool,
		repository.EngramLinkTraverseInput{
			RootEngramID:    input.RootEngramID,
			ActorUserID:     input.ActorUserID,
			MaxDepth:        input.MaxDepth,
			MaxNeighbors:    input.MaxNeighbors,
			IncludeArchived: input.IncludeArchived,
		},
	)
}

func suggestEngramLinksDependency(
	pool repository.Queryer,
	embeddingDim int,
) func(
	ctx context.Context,
	input internalapi.SessionEngramLinkSuggestInput,
) ([]models.EngramLinkSuggestion, error) {
	service := graph.NewLinkSuggestionService(pool, embeddingDim)
	if service == nil {
		return nil
	}
	return func(
		ctx context.Context,
		input internalapi.SessionEngramLinkSuggestInput,
	) ([]models.EngramLinkSuggestion, error) {
		return service.SuggestLinks(
			ctx,
			graph.LinkSuggestionInput{
				SourceEngramID:  input.SourceEngramID,
				ActorUserID:     input.ActorUserID,
				Limit:           input.Limit,
				MaxCandidates:   input.MaxCandidates,
				MinimumScore:    input.MinimumScore,
				IncludeArchived: input.IncludeArchived,
			},
		)
	}
}

func hygieneEngramLinksDependency(
	pool repository.Queryer,
) func(
	ctx context.Context,
	input internalapi.SessionEngramLinkHygieneInput,
) ([]models.EngramLinkHygieneRecommendation, error) {
	service := graph.NewLinkHygieneService(pool)
	if service == nil {
		return nil
	}
	return func(
		ctx context.Context,
		input internalapi.SessionEngramLinkHygieneInput,
	) ([]models.EngramLinkHygieneRecommendation, error) {
		return service.Recommend(
			ctx,
			graph.LinkHygieneInput{
				SourceEngramID:    input.SourceEngramID,
				ActorUserID:       input.ActorUserID,
				IncludeArchived:   input.IncludeArchived,
				Limit:             input.Limit,
				StaleAfterDays:    input.StaleAfterDays,
				LowValueThreshold: input.LowValueThreshold,
			},
		)
	}
}

func createMCPTokenForOwnerDependency(
	service *mcptokens.Service,
) func(
	ctx context.Context,
	ownerUserID uuid.UUID,
	payload models.MCPTokenCreateRequest,
	pepper string,
) (*models.MCPTokenCreateResponse, error) {
	if service == nil {
		return nil
	}
	return func(
		ctx context.Context,
		ownerUserID uuid.UUID,
		payload models.MCPTokenCreateRequest,
		pepper string,
	) (*models.MCPTokenCreateResponse, error) {
		return service.CreateTokenForOwner(ctx, ownerUserID, payload, pepper)
	}
}

func listMCPTokenSummariesDependency(
	service *mcptokens.Service,
) func(
	ctx context.Context,
	ownerUserID uuid.UUID,
	limit int,
	offset int,
) ([]models.MCPTokenSummary, error) {
	if service == nil {
		return nil
	}
	return func(
		ctx context.Context,
		ownerUserID uuid.UUID,
		limit int,
		offset int,
	) ([]models.MCPTokenSummary, error) {
		return service.ListTokenSummaries(
			ctx,
			mcptokens.TokenListRequest{
				OwnerUserID: ownerUserID,
				Limit:       limit,
				Offset:      offset,
			},
		)
	}
}

func revokeMCPTokenForOwnerDependency(
	service *mcptokens.Service,
) func(
	ctx context.Context,
	tokenID uuid.UUID,
	ownerUserID uuid.UUID,
) (*models.MCPTokenSummary, error) {
	if service == nil {
		return nil
	}
	return func(
		ctx context.Context,
		tokenID uuid.UUID,
		ownerUserID uuid.UUID,
	) (*models.MCPTokenSummary, error) {
		return service.RevokeTokenForOwner(ctx, tokenID, ownerUserID)
	}
}

func newWorkflowEngramCreator(
	pool *pgxpool.Pool,
	embeddingDim int,
) workflow.EngramCreator {
	return func(
		ctx context.Context,
		payload models.MemoryEngramCreate,
		enrichmentOrigin string,
	) (*models.EngramCreateResponse, error) {
		return repository.CreateEngram(
			ctx,
			pool,
			repository.CreateEngramInput{
				Payload:          payload,
				EmbeddingDim:     embeddingDim,
				EnrichmentOrigin: enrichmentOrigin,
			},
		)
	}
}

func auditEventLogger(logger *audit.Logger) internalapi.SessionAuditLogger {
	return func(
		request *http.Request,
		eventType string,
		success bool,
		username string,
		detail string,
		metadata map[string]any,
	) {
		_ = logger.LogRequestEvent(audit.RequestEvent{
			Request:   request,
			EventType: eventType,
			Success:   success,
			Username:  username,
			Detail:    detail,
			Metadata:  metadata,
		})
	}
}

func lookupSessionUser(pool *pgxpool.Pool) internalapi.SessionUserLookup {
	return func(ctx context.Context, userID uuid.UUID) (*models.UserAuthRecord, error) {
		return repository.GetUserAuthRecordByID(ctx, pool, userID)
	}
}

func lookupSessionUserByUsername(pool *pgxpool.Pool) internalapi.SessionUserByUsernameLookup {
	return func(ctx context.Context, username string) (*models.UserAuthRecord, error) {
		return repository.GetUserAuthRecord(ctx, pool, username)
	}
}

func buildServer(settings config.Settings, handler http.Handler) *http.Server {
	address := net.JoinHostPort(settings.APIHost, strconv.Itoa(settings.APIPort))
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func runServerOrExit(logger *slog.Logger, server *http.Server) {
	logger.Info("starting go-api", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}

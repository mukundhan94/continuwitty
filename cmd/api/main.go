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
	"engram/internal/config"
	"engram/internal/db"
	internalexport "engram/internal/export"
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

var errMissingSessionActorForMCP = errors.New("session actor missing for mcp")

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	startupCtx := context.Background()
	settings := loadSettingsOrExit(logger)
	pool := initDBPoolOrExit(startupCtx, logger, settings)
	defer pool.Close()

	handler := buildHandlerOrExit(logger, settings, pool)
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

func buildHandlerOrExit(logger *slog.Logger, settings config.Settings, pool *pgxpool.Pool) http.Handler {
	sessionManager, err := auth.NewSessionManager(settings.AppSessionSecret, auth.DefaultSessionCookieName)
	if err != nil {
		logger.Error("failed to initialize session manager", "error", err)
		os.Exit(1)
	}
	projectService := projects.NewService(pool)
	oauthRegistrationService := oauth.NewRegistrationService(pool)
	oauthAuthorizationService := oauth.NewAuthorizationService(pool)
	oauthTokenService := oauth.NewTokenService(pool)
	memoryAdminService := admin.NewService(pool, settings.EmbeddingDim, newAdminProjectResolver(projectService))
	mcpTokenService := mcptokens.NewService(pool)
	exportService := internalexport.NewService(pool, projectService, memoryAdminService, settings.EmbeddingDim)
	mcpService := newMCPCompatibilityService(
		settings,
		mcpCompatibilityRuntimeDependencies{
			pool:               pool,
			projectService:     projectService,
			memoryAdminService: memoryAdminService,
			exportService:      exportService,
		},
	)
	mcpTransportLimiter := newMCPTransportRateLimiter(settings, pool)
	mcpActorResolver := mcp.NewActorResolver(
		settings,
		pool,
		resolveMCPActorFromSessionContext,
	)
	agentWorkflowService := workflow.NewService(newWorkflowEngramCreator(pool, settings.EmbeddingDim))
	ingestionService := ingestion.NewService(
		pool,
		settings.EmbeddingDim,
		settings.IngestionMaxFileBytes,
		settings.IngestionMaxTextChars,
	)
	loginAttemptGuard := newLoginAttemptGuard(settings, pool)
	auditLogger := newSessionAuditLogger(settings)

	routerDependencies := internalapi.RouterDependencies{
		MemoryAdminService: memoryAdminService,
		RequireAdminActor:  internalapi.RequireAdminActorFromContext,
		SessionAuth: buildSessionAuthDependencies(
			sessionAuthRuntimeDependencies{
				settings:          settings,
				pool:              pool,
				sessionManager:    sessionManager,
				loginAttemptGuard: loginAttemptGuard,
				auditLogger:       auditLogger,
				projectService:    projectService,
				mcpTokenService:   mcpTokenService,
			},
		),
		ProjectsService:  newProjectRouteServiceAdapter(projectService),
		IngestionService: newIngestionRouteServiceAdapter(ingestionService),
		IngestionOptions: internalapi.IngestionRouteOptions{
			MaxMetadataJSONBytes: settings.IngestionMaxMetadataJSONBytes,
		},
		OAuthRegistration:   oauthRegistrationService,
		OAuthAuthorization:  oauthAuthorizationService,
		OAuthToken:          oauthTokenService,
		ChatRouter:          buildChatRouter(settings, pool),
		AgentWorkflow:       agentWorkflowService,
		ExportService:       exportService,
		MCPService:          mcpService,
		MCPActorResolver:    mcpActorResolver,
		MCPTransportLimiter: mcpTransportLimiter,
	}
	handler := internalapi.NewRouterWithDependencies(settings, routerDependencies)
	handler = internalapi.SessionActorMiddleware(sessionManager, lookupSessionUser(pool))(handler)
	logger.Info("session-authenticated actor context enabled")
	return handler
}

func newMCPCompatibilityService(
	settings config.Settings,
	dependencies mcpCompatibilityRuntimeDependencies,
) *mcp.CompatibilityService {
	messageAdapter := newMCPMessageAdapter(settings, dependencies.pool)
	var messageSend mcp.MessageSendService
	var messageStream mcp.MessageStreamService
	if messageAdapter != nil {
		messageSend = messageAdapter
		messageStream = messageAdapter
	}

	return mcp.NewCompatibilityServiceWithDependencies(
		settings.AppSemanticVersion,
		mcp.CompatibilityServiceDependencies{
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
			EngramCreate: newMCPEngramCreateAdapter(
				dependencies.pool,
				dependencies.projectService,
				settings.EmbeddingDim,
			),
			EngramCreateConversation: newMCPEngramCreateFromConversationAdapter(
				dependencies.pool,
				dependencies.projectService,
				settings.EmbeddingDim,
			),
			EngramList:           newMCPEngramListAdapter(dependencies.memoryAdminService),
			EngramGet:            newMCPEngramGetAdapter(dependencies.memoryAdminService),
			EngramQuery:          newMCPEngramQueryAdapter(dependencies.pool, settings.EmbeddingDim),
			EngramRehydrate:      newMCPEngramRehydrateAdapter(dependencies.pool),
			EngramUpdate:         newMCPEngramUpdateAdapter(dependencies.memoryAdminService),
			EngramMove:           newMCPEngramMoveAdapter(dependencies.memoryAdminService),
			EngramDelete:         newMCPEngramDeleteAdapter(dependencies.memoryAdminService),
			EngramRestore:        newMCPEngramRestoreAdapter(dependencies.memoryAdminService),
			EngramCollectionList: newMCPEngramCollectionListAdapter(dependencies.memoryAdminService),
			EngramCollectionGet:  newMCPEngramCollectionGetAdapter(dependencies.memoryAdminService),
			EngramCollectionCreate: newMCPEngramCollectionCreateAdapter(
				dependencies.memoryAdminService,
				dependencies.projectService,
			),
			EngramCollectionUpdate:   newMCPEngramCollectionUpdateAdapter(dependencies.memoryAdminService),
			EngramCollectionDelete:   newMCPEngramCollectionDeleteAdapter(dependencies.memoryAdminService),
			EngramCollectionAddItems: newMCPEngramCollectionAddItemsAdapter(dependencies.memoryAdminService),
			EngramCollectionRemove:   newMCPEngramCollectionRemoveItemAdapter(dependencies.memoryAdminService),
			PinEngramService:         newMCPPinEngramAdapter(dependencies.pool),
			UnpinEngramService:       newMCPUnpinEngramAdapter(dependencies.pool),
			PinDocumentService:       newMCPPinDocumentAdapter(dependencies.pool),
			UnpinDocumentService:     newMCPUnpinDocumentAdapter(dependencies.pool),
		},
	)
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
	})
}

func buildSessionAuthDependencies(runtimeDependencies sessionAuthRuntimeDependencies) internalapi.SessionAuthDependencies {
	return internalapi.SessionAuthDependencies{
		SessionManager:           runtimeDependencies.sessionManager,
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
		ShareEngram:              shareEngramDependency(runtimeDependencies.projectService),
		UnshareEngram:            unshareEngramDependency(runtimeDependencies.projectService),
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
		return repository.GetEngramSources(ctx, pool, engramID, limit, &actorUserID)
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
		return concreteService.ShareEngram(ctx, actorUserID, actorRole, engramID)
	}
}

func unshareEngramDependency(
	projectService projectResolutionService,
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
		return concreteService.UnshareEngram(ctx, actorUserID, actorRole, engramID)
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
		return service.ListTokenSummaries(ctx, ownerUserID, limit, offset)
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
		_ = logger.LogRequestEvent(
			request,
			eventType,
			success,
			username,
			detail,
			metadata,
		)
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

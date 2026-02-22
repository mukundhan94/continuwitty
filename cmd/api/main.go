package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"engram/internal/admin"
	internalapi "engram/internal/api"
	"engram/internal/audit"
	"engram/internal/auth"
	"engram/internal/config"
	"engram/internal/db"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sessionAuthRuntimeDependencies struct {
	settings          config.Settings
	pool              *pgxpool.Pool
	sessionManager    *auth.SessionManager
	loginAttemptGuard *auth.LoginAttemptGuard
	auditLogger       *audit.Logger
}

type projectWriteResolutionRequest struct {
	ctx         context.Context
	db          repository.Queryer
	actorUserID uuid.UUID
	actorRole   models.UserRole
	projectID   string
}

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
	return pool
}

func buildHandlerOrExit(logger *slog.Logger, settings config.Settings, pool *pgxpool.Pool) http.Handler {
	sessionManager, err := auth.NewSessionManager(settings.AppSessionSecret, auth.DefaultSessionCookieName)
	if err != nil {
		logger.Error("failed to initialize session manager", "error", err)
		os.Exit(1)
	}
	loginAttemptGuard := newLoginAttemptGuard(settings, pool)
	auditLogger := newSessionAuditLogger(settings)

	routerDependencies := internalapi.RouterDependencies{
		MemoryAdminService: admin.NewService(pool, settings.EmbeddingDim, admin.PassthroughProjectResolver{}),
		RequireAdminActor:  internalapi.RequireAdminActorFromContext,
		SessionAuth: buildSessionAuthDependencies(
			sessionAuthRuntimeDependencies{
				settings:          settings,
				pool:              pool,
				sessionManager:    sessionManager,
				loginAttemptGuard: loginAttemptGuard,
				auditLogger:       auditLogger,
			},
		),
	}
	handler := internalapi.NewRouterWithDependencies(settings, routerDependencies)
	handler = internalapi.SessionActorMiddleware(sessionManager, lookupSessionUser(pool))(handler)
	logger.Info("session-authenticated actor context enabled")
	return handler
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
		ResolveProjectIDForWrite: resolveProjectIDForWriteDependency(runtimeDependencies.pool),
		CreateEngram:             createEngramDependency(runtimeDependencies.pool, runtimeDependencies.settings.EmbeddingDim),
		ListEngrams:              listEngramsDependency(runtimeDependencies.pool),
		QueryEngrams:             queryEngramsDependency(runtimeDependencies.pool, runtimeDependencies.settings.EmbeddingDim),
		GetRehydrationBundle:     getRehydrationBundleDependency(runtimeDependencies.pool),
		GetEngramSources:         getEngramSourcesDependency(runtimeDependencies.pool),
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

func resolveProjectIDForWriteDependency(
	pool *pgxpool.Pool,
) func(ctx context.Context, actorUserID uuid.UUID, actorRole models.UserRole, projectID string) (internalapi.SessionProjectResolution, error) {
	return func(
		ctx context.Context,
		actorUserID uuid.UUID,
		actorRole models.UserRole,
		projectID string,
	) (internalapi.SessionProjectResolution, error) {
		return resolveProjectIDForWrite(
			projectWriteResolutionRequest{
				ctx:         ctx,
				db:          pool,
				actorUserID: actorUserID,
				actorRole:   actorRole,
				projectID:   projectID,
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

func resolveProjectIDForWrite(input projectWriteResolutionRequest) (internalapi.SessionProjectResolution, error) {
	normalizedProjectID := strings.TrimSpace(input.projectID)
	if normalizedProjectID != "" {
		return resolveExplicitProjectWrite(input, normalizedProjectID)
	}
	return resolveDefaultProjectWrite(input)
}

func resolveExplicitProjectWrite(
	input projectWriteResolutionRequest,
	projectID string,
) (internalapi.SessionProjectResolution, error) {
	visibleProject, err := getVisibleProjectForActor(input, projectID)
	if err != nil {
		return internalapi.SessionProjectResolution{}, err
	}
	if visibleProject == nil {
		if err := ensureProjectForActor(input, projectID); err != nil {
			return internalapi.SessionProjectResolution{}, err
		}
	}
	return internalapi.SessionProjectResolution{
		ProjectID:          projectID,
		UsedDefaultProject: false,
	}, nil
}

func resolveDefaultProjectWrite(input projectWriteResolutionRequest) (internalapi.SessionProjectResolution, error) {
	defaultProjectID, err := repository.GetUserDefaultProjectID(input.ctx, input.db, input.actorUserID)
	if err != nil {
		return internalapi.SessionProjectResolution{}, err
	}
	if defaultProjectID == nil {
		return internalapi.SessionProjectResolution{}, internalapi.ErrProjectIDRequiredWhenNoDefaultProject
	}
	normalizedDefaultProjectID := strings.TrimSpace(*defaultProjectID)
	if normalizedDefaultProjectID == "" {
		return internalapi.SessionProjectResolution{}, internalapi.ErrProjectIDRequiredWhenNoDefaultProject
	}

	visibleDefaultProject, err := getVisibleProjectForActor(input, normalizedDefaultProjectID)
	if err != nil {
		return internalapi.SessionProjectResolution{}, err
	}
	if visibleDefaultProject == nil {
		return internalapi.SessionProjectResolution{}, internalapi.ErrDefaultProjectNotAccessible
	}
	return internalapi.SessionProjectResolution{
		ProjectID:          normalizedDefaultProjectID,
		UsedDefaultProject: true,
	}, nil
}

func getVisibleProjectForActor(
	input projectWriteResolutionRequest,
	projectID string,
) (*models.ProjectRecord, error) {
	return repository.GetProjectForActor(
		input.ctx,
		input.db,
		repository.ProjectGetInput{
			ProjectID:       projectID,
			ActorUserID:     input.actorUserID,
			ActorRole:       string(input.actorRole),
			IncludeArchived: false,
		},
	)
}

func ensureProjectForActor(input projectWriteResolutionRequest, projectID string) error {
	_, err := repository.EnsureProjectExists(
		input.ctx,
		input.db,
		repository.ProjectEnsureInput{
			ProjectID:   projectID,
			OwnerUserID: input.actorUserID,
		},
	)
	return err
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

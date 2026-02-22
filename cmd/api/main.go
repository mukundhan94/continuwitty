package main

import (
	"context"
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
		SessionManager:       runtimeDependencies.sessionManager,
		LookupUserByUsername: lookupSessionUserByUsername(runtimeDependencies.pool),
		LookupUserByID:       lookupSessionUser(runtimeDependencies.pool),
		VerifyPassword:       auth.VerifyPassword,
		GenerateCSRFToken:    auth.GenerateCSRFToken,
		HashPassword: func(password string) (string, error) {
			return auth.HashPassword(password, nil)
		},
		ListUsers: func(ctx context.Context, limit, offset int) ([]models.UserRecord, error) {
			return repository.ListUsers(ctx, runtimeDependencies.pool, limit, offset)
		},
		CreateUser: func(ctx context.Context, input internalapi.SessionUserCreateInput) (*models.UserRecord, error) {
			return repository.CreateUser(
				ctx,
				runtimeDependencies.pool,
				repository.CreateUserInput{
					Username:     input.Username,
					PasswordHash: input.PasswordHash,
					Role:         input.Role,
					IsActive:     input.IsActive,
				},
			)
		},
		UpdateUser: func(
			ctx context.Context,
			userID uuid.UUID,
			input internalapi.SessionUserUpdateInput,
		) (*models.UserRecord, error) {
			return repository.UpdateUser(
				ctx,
				runtimeDependencies.pool,
				userID,
				repository.UserUpdateInput{
					Role:         input.Role,
					IsActive:     input.IsActive,
					PasswordHash: input.PasswordHash,
				},
			)
		},
		CookieSecure:      config.IsProductionEnv(runtimeDependencies.settings),
		LoginAttemptGuard: runtimeDependencies.loginAttemptGuard,
		LogAuditEvent: func(
			request *http.Request,
			eventType string,
			success bool,
			username string,
			detail string,
			metadata map[string]any,
		) {
			_ = runtimeDependencies.auditLogger.LogRequestEvent(
				request,
				eventType,
				success,
				username,
				detail,
				metadata,
			)
		},
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

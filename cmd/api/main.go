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
	"engram/internal/auth"
	"engram/internal/config"
	"engram/internal/db"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	routerDependencies := internalapi.RouterDependencies{
		MemoryAdminService: admin.NewService(pool, settings.EmbeddingDim, admin.PassthroughProjectResolver{}),
		RequireAdminActor:  internalapi.RequireAdminActorFromContext,
	}
	handler := internalapi.NewRouterWithDependencies(settings, routerDependencies)

	sessionManager, err := auth.NewSessionManager(settings.AppSessionSecret, auth.DefaultSessionCookieName)
	if err != nil {
		logger.Error("failed to initialize session manager", "error", err)
		os.Exit(1)
	}
	handler = internalapi.SessionActorMiddleware(sessionManager, lookupSessionUser(pool))(handler)

	if config.IsProductionEnv(settings) {
		logger.Info("memory-admin routes require session-authenticated context actor")
		return handler
	}
	logger.Info("enabling migration-time header admin actor bridge")
	return internalapi.AdminActorHeaderBridge(handler)
}

func lookupSessionUser(pool *pgxpool.Pool) internalapi.SessionUserLookup {
	return func(ctx context.Context, userID uuid.UUID) (*models.UserAuthRecord, error) {
		return repository.GetUserAuthRecordByID(ctx, pool, userID)
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

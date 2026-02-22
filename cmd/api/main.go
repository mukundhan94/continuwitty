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
	"engram/internal/config"
	"engram/internal/db"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	settings, err := config.LoadSettings()
	if err != nil {
		logger.Error("failed to load settings", "error", err)
		os.Exit(1)
	}

	startupCtx := context.Background()
	pool, err := db.NewPool(startupCtx, settings)
	if err != nil {
		logger.Error("failed to initialize db pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Ping(startupCtx, pool); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	routerDependencies := internalapi.RouterDependencies{}
	if config.IsProductionEnv(settings) {
		logger.Warn("memory-admin routes disabled in production until auth migration is complete")
	} else {
		logger.Info("enabling memory-admin header actor bridge")
		routerDependencies = internalapi.RouterDependencies{
			MemoryAdminService: admin.NewService(pool, settings.EmbeddingDim, admin.PassthroughProjectResolver{}),
			RequireAdminActor:  internalapi.RequireAdminActorFromHeaders,
		}
	}

	handler := internalapi.NewRouterWithDependencies(settings, routerDependencies)
	address := net.JoinHostPort(settings.APIHost, strconv.Itoa(settings.APIPort))
	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("starting go-api", "addr", address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}

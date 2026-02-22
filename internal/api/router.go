package api

import (
	"encoding/json"
	"net/http"

	"engram/internal/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds the API router for the Go migration path.
func NewRouter(settings config.Settings) http.Handler {
	return NewRouterWithDependencies(settings, RouterDependencies{})
}

// RouterDependencies captures optional services required by composed API routes.
type RouterDependencies struct {
	MemoryAdminService MemoryAdminService
	RequireAdminActor  RequireAdminActor
	SessionAuth        SessionAuthDependencies
}

// NewRouterWithDependencies builds the API router and mounts dependency-backed routes.
func NewRouterWithDependencies(settings config.Settings, dependencies RouterDependencies) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)

	router.Get("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})

	router.Route("/api/v1", func(versioned chi.Router) {
		versioned.Get("/version", func(writer http.ResponseWriter, _ *http.Request) {
			writeJSON(writer, http.StatusOK, map[string]string{
				"semantic_version": settings.AppSemanticVersion,
				"release":          "v" + settings.AppSemanticVersion,
				"commit_id":        settings.AppCommitSHA,
			})
		})
	})
	if dependencies.MemoryAdminService != nil && dependencies.RequireAdminActor != nil {
		MountMemoryAdminRoutes(router, dependencies.MemoryAdminService, dependencies.RequireAdminActor)
	}
	if dependencies.SessionAuth.SessionManager != nil {
		MountSessionAuthRoutes(router, dependencies.SessionAuth)
	}

	return router
}

func writeJSON(writer http.ResponseWriter, statusCode int, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(body)
}

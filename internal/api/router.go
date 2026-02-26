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
	ProjectsService    ProjectService
	IngestionService   IngestionService
	IngestionOptions   IngestionRouteOptions
	OAuthRegistration  OAuthRegistrationRouteService
	OAuthAuthorization OAuthAuthorizationRouteService
	ChatRouter         chi.Router
}

// NewRouterWithDependencies builds the API router and mounts dependency-backed routes.
func NewRouterWithDependencies(settings config.Settings, dependencies RouterDependencies) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)

	mountHealthRoute(router)
	mountVersionRoute(router, settings)
	mountDependencyRoutes(router, settings, dependencies)
	return router
}

func mountHealthRoute(router chi.Router) {
	router.Get("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})
}

func mountVersionRoute(router chi.Router, settings config.Settings) {
	router.Route("/api/v1", func(versioned chi.Router) {
		versioned.Get("/version", func(writer http.ResponseWriter, _ *http.Request) {
			writeJSON(writer, http.StatusOK, map[string]string{
				"semantic_version": settings.AppSemanticVersion,
				"release":          "v" + settings.AppSemanticVersion,
				"commit_id":        settings.AppCommitSHA,
			})
		})
	})
}

func mountDependencyRoutes(router chi.Router, settings config.Settings, dependencies RouterDependencies) {
	mountMemoryAdminDependencyRoutes(router, dependencies)
	mountSessionDependencyRoutes(router, dependencies)
	mountOAuthDependencyRoutes(router, settings, dependencies)
	mountProjectDependencyRoutes(router, dependencies)
	mountIngestionDependencyRoutes(router, dependencies)
	mountChatDependencyRoutes(router, dependencies)
}

func mountMemoryAdminDependencyRoutes(router chi.Router, dependencies RouterDependencies) {
	if dependencies.MemoryAdminService == nil || dependencies.RequireAdminActor == nil {
		return
	}
	MountMemoryAdminRoutes(router, dependencies.MemoryAdminService, dependencies.RequireAdminActor)
}

func mountSessionDependencyRoutes(router chi.Router, dependencies RouterDependencies) {
	if dependencies.SessionAuth.SessionManager == nil {
		return
	}
	MountSessionAuthRoutes(router, dependencies.SessionAuth)
	MountSessionUIRoutes(router, dependencies.SessionAuth)
}

func mountOAuthDependencyRoutes(router chi.Router, settings config.Settings, dependencies RouterDependencies) {
	if dependencies.OAuthRegistration != nil {
		MountOAuthRoutes(router, settings, dependencies.OAuthRegistration)
	}
	if dependencies.OAuthAuthorization != nil {
		MountOAuthAuthorizationRoutes(router, settings, dependencies.OAuthAuthorization)
	}
}

func mountProjectDependencyRoutes(router chi.Router, dependencies RouterDependencies) {
	if dependencies.ProjectsService == nil {
		return
	}
	MountProjectRoutes(router, dependencies.ProjectsService)
}

func mountIngestionDependencyRoutes(router chi.Router, dependencies RouterDependencies) {
	if dependencies.IngestionService == nil {
		return
	}
	MountIngestionRoutes(router, dependencies.IngestionService, dependencies.IngestionOptions)
}

func mountChatDependencyRoutes(router chi.Router, dependencies RouterDependencies) {
	if dependencies.ChatRouter == nil {
		return
	}
	router.Handle("/api/v1/chat", dependencies.ChatRouter)
	router.Handle("/api/v1/chat/*", dependencies.ChatRouter)
}

func writeJSON(writer http.ResponseWriter, statusCode int, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(body)
}

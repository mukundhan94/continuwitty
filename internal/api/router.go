package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"engram/internal/config"
	"engram/internal/export"
	"engram/internal/mcp"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds the API router for the Go migration path.
func NewRouter(settings config.Settings) http.Handler {
	return NewRouterWithDependencies(settings, RouterDependencies{})
}

// RouterDependencies captures optional services required by composed API routes.
type RouterDependencies struct {
	MemoryAdminService  MemoryAdminService
	RequireAdminActor   RequireAdminActor
	SessionAuth         SessionAuthDependencies
	ProjectsService     ProjectService
	IngestionService    IngestionService
	IngestionOptions    IngestionRouteOptions
	OAuthRegistration   OAuthRegistrationRouteService
	OAuthAuthorization  OAuthAuthorizationRouteService
	OAuthToken          OAuthTokenRouteService
	ChatRouter          chi.Router
	AgentWorkflow       AgentWorkflowRouteService
	ExportService       export.Service
	MCPService          mcp.Service
	MCPActorResolver    mcp.HTTPActorResolver
	MCPTransportLimiter MCPTransportRateLimiter
	RequestLogger       *slog.Logger
	RequestMetrics      RequestMetricsRecorder
}

// NewRouterWithDependencies builds the API router and mounts dependency-backed routes.
func NewRouterWithDependencies(settings config.Settings, dependencies RouterDependencies) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	if dependencies.RequestLogger != nil || dependencies.RequestMetrics != nil {
		router.Use(newRequestTelemetryMiddleware(dependencies.RequestLogger, dependencies.RequestMetrics))
	}

	mountHealthRoute(router)
	mountVersionRoute(router, settings)
	mountObservabilityRoutes(router, dependencies)
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
				"semantic_version":           settings.AppSemanticVersion,
				"release":                    "v" + settings.AppSemanticVersion,
				"commit_id":                  settings.AppCommitSHA,
				"chat_prompt_policy_version": settings.ChatPromptPolicyVersion,
				"mcp_tool_policy_version":    settings.MCPToolPolicyVersion,
				"eval_suite_version":         settings.EvalSuiteVersion,
			})
		})
	})
}

func mountObservabilityRoutes(router chi.Router, dependencies RouterDependencies) {
	if dependencies.RequestMetrics == nil {
		return
	}
	router.Get("/api/v1/metrics", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, dependencies.RequestMetrics.Snapshot())
	})
}

func mountDependencyRoutes(router chi.Router, settings config.Settings, dependencies RouterDependencies) {
	mountMemoryAdminDependencyRoutes(router, dependencies)
	mountSessionDependencyRoutes(router, dependencies)
	mountOAuthDependencyRoutes(router, settings, dependencies)
	mountProjectDependencyRoutes(router, dependencies)
	mountIngestionDependencyRoutes(router, dependencies)
	mountChatDependencyRoutes(router, dependencies)
	mountAgentWorkflowDependencyRoutes(router, dependencies)
	mountExportDependencyRoutes(router, dependencies)
	mountMCPDependencyRoutes(router, dependencies)
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
	if dependencies.OAuthToken != nil {
		MountOAuthTokenRoutes(router, settings, dependencies.OAuthToken)
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

func mountAgentWorkflowDependencyRoutes(router chi.Router, dependencies RouterDependencies) {
	if dependencies.AgentWorkflow == nil {
		return
	}
	MountAgentWorkflowRoutes(router, dependencies.AgentWorkflow)
}

func mountExportDependencyRoutes(router chi.Router, dependencies RouterDependencies) {
	if dependencies.ExportService == nil {
		return
	}
	MountExportRoutes(router, dependencies.ExportService)
}

func mountMCPDependencyRoutes(router chi.Router, dependencies RouterDependencies) {
	if dependencies.MCPService == nil || dependencies.MCPActorResolver == nil {
		return
	}
	MountMCPRoutes(
		router,
		dependencies.MCPService,
		dependencies.MCPActorResolver,
		dependencies.MCPTransportLimiter,
	)
}

func writeJSON(writer http.ResponseWriter, statusCode int, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(body)
}

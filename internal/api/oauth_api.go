package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"engram/internal/config"
	internaloauth "engram/internal/oauth"

	"github.com/go-chi/chi/v5"
)

const defaultOAuthClientName = "Engram MCP Client"

var (
	oauthCodeChallengeMethodsSupported = []string{"S256"}
	oauthGrantTypesSupported           = []string{"authorization_code"}
	oauthResponseTypesSupported        = []string{"code"}
	oauthScopesSupported               = []string{"mcp:read", "mcp:write"}
	oauthTokenAuthMethodsSupported     = []string{"client_secret_post", "none"}
)

// OAuthRegistrationRouteService captures OAuth dynamic registration route behavior.
type OAuthRegistrationRouteService interface {
	HandleRegister(
		ctx context.Context,
		settings config.Settings,
		payload internaloauth.RegistrationRequest,
		sessionUser *internaloauth.SessionUser,
	) (internaloauth.RegistrationResponse, error)
}

type oauthRegistrationRoutePayload struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

type oauthRouteDependencies struct {
	settings            config.Settings
	registrationService OAuthRegistrationRouteService
}

// MountOAuthRoutes registers OAuth metadata + registration routes.
func MountOAuthRoutes(
	router chi.Router,
	settings config.Settings,
	registrationService OAuthRegistrationRouteService,
) {
	dependencies := oauthRouteDependencies{
		settings:            settings,
		registrationService: registrationService,
	}
	router.Get(
		"/.well-known/oauth-authorization-server",
		dependencies.oauthMetadataHandler(
			func(issuer string, _ *http.Request) map[string]any {
				return oauthServerMetadata(issuer)
			},
		),
	)
	router.Get(
		"/.well-known/openid-configuration",
		dependencies.oauthMetadataHandler(
			func(issuer string, _ *http.Request) map[string]any {
				metadata := oauthServerMetadata(issuer)
				metadata["claims_supported"] = []string{}
				metadata["subject_types_supported"] = []string{"public"}
				return metadata
			},
		),
	)
	router.Get(
		"/.well-known/oauth-protected-resource",
		dependencies.oauthMetadataHandler(
			func(issuer string, _ *http.Request) map[string]any {
				return oauthProtectedResourceMetadata(issuer, "")
			},
		),
	)
	router.Get(
		"/.well-known/oauth-protected-resource/*",
		dependencies.oauthMetadataHandler(
			func(issuer string, request *http.Request) map[string]any {
				resourcePath := strings.TrimSpace(chi.URLParam(request, "*"))
				return oauthProtectedResourceMetadata(issuer, resourcePath)
			},
		),
	)
	router.Post("/oauth/register", dependencies.handleOAuthRegisterClient)
}

func (dependencies oauthRouteDependencies) handleOAuthRegisterClient(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if dependencies.registrationService == nil {
		writeOAuthError(writer, http.StatusInternalServerError, "server_error", "OAuth registration service unavailable.")
		return
	}
	payload, ok := decodeOAuthRegistrationPayload(writer, request)
	if !ok {
		return
	}
	sessionUser := oauthSessionUserFromRequest(request)
	responsePayload, err := dependencies.registrationService.HandleRegister(
		request.Context(),
		dependencies.settings,
		payload,
		sessionUser,
	)
	if err != nil {
		writeOAuthRegistrationError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, responsePayload)
}

func decodeOAuthRegistrationPayload(
	writer http.ResponseWriter,
	request *http.Request,
) (internaloauth.RegistrationRequest, bool) {
	payload := oauthRegistrationRoutePayload{}
	if !decodeJSONAllowEmpty(writer, request, &payload) {
		return internaloauth.RegistrationRequest{}, false
	}
	return internaloauth.RegistrationRequest{
		RedirectURIs:            payload.RedirectURIs,
		ClientName:              resolveOAuthClientName(payload.ClientName),
		GrantTypes:              payload.GrantTypes,
		ResponseTypes:           payload.ResponseTypes,
		TokenEndpointAuthMethod: strings.TrimSpace(payload.TokenEndpointAuthMethod),
	}, true
}

func resolveOAuthClientName(clientName string) string {
	normalized := strings.TrimSpace(clientName)
	if normalized == "" {
		return defaultOAuthClientName
	}
	return normalized
}

func oauthSessionUserFromRequest(request *http.Request) *internaloauth.SessionUser {
	actor, ok := AdminActorFromContext(request.Context())
	if !ok {
		return nil
	}
	return &internaloauth.SessionUser{
		UserID: actor.UserID.String(),
		Role:   actor.Role,
	}
}

func writeOAuthRegistrationError(writer http.ResponseWriter, err error) {
	registrationError := internaloauth.RegistrationError{}
	if errors.As(err, &registrationError) {
		writeOAuthError(
			writer,
			registrationError.StatusCode,
			registrationError.ErrorCode,
			registrationError.Description,
		)
		return
	}
	writeOAuthError(writer, http.StatusInternalServerError, "server_error", "Internal server error.")
}

func writeOAuthError(
	writer http.ResponseWriter,
	statusCode int,
	errorCode string,
	description string,
) {
	writeJSON(
		writer,
		statusCode,
		map[string]string{
			"error":             errorCode,
			"error_description": description,
		},
	)
}

func oauthServerMetadata(issuer string) map[string]any {
	return map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/oauth/authorize",
		"token_endpoint":                        issuer + "/oauth/token",
		"registration_endpoint":                 issuer + "/oauth/register",
		"response_types_supported":              oauthResponseTypesSupported,
		"grant_types_supported":                 oauthGrantTypesSupported,
		"token_endpoint_auth_methods_supported": oauthTokenAuthMethodsSupported,
		"scopes_supported":                      oauthScopesSupported,
		"code_challenge_methods_supported":      oauthCodeChallengeMethodsSupported,
	}
}

func oauthProtectedResourceMetadata(issuer string, resourcePath string) map[string]any {
	resource := issuer + "/api/v1/mcp/stream"
	normalizedPath := strings.TrimLeft(strings.TrimSpace(resourcePath), "/")
	if normalizedPath != "" {
		resource = issuer + "/" + normalizedPath
	}
	return map[string]any{
		"resource":                 resource,
		"authorization_servers":    []string{issuer},
		"bearer_methods_supported": []string{"header"},
		"scopes_supported":         oauthScopesSupported,
	}
}

func (dependencies oauthRouteDependencies) requireOAuthEnabled(writer http.ResponseWriter) bool {
	if dependencies.settings.OAuthEnabled {
		return true
	}
	writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "OAuth endpoints disabled"})
	return false
}

func (dependencies oauthRouteDependencies) oauthMetadataHandler(
	buildMetadata func(issuer string, request *http.Request) map[string]any,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if !dependencies.requireOAuthEnabled(writer) {
			return
		}
		issuer := internaloauth.IssuerURLForRequest(request, dependencies.settings)
		writeJSON(writer, http.StatusOK, buildMetadata(issuer, request))
	}
}

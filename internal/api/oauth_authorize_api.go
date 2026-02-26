package api

import (
	"context"
	"net/http"

	"engram/internal/config"
	internaloauth "engram/internal/oauth"

	"github.com/go-chi/chi/v5"
)

// OAuthAuthorizationRouteService captures OAuth authorization route behavior.
type OAuthAuthorizationRouteService interface {
	HandleAuthorize(
		ctx context.Context,
		settings config.Settings,
		request internaloauth.AuthorizationRequest,
	) (internaloauth.AuthorizationResult, error)
}

type oauthAuthorizationDependencies struct {
	settings             config.Settings
	authorizationService OAuthAuthorizationRouteService
}

// MountOAuthAuthorizationRoutes registers OAuth authorization-code entrypoint routes.
func MountOAuthAuthorizationRoutes(
	router chi.Router,
	settings config.Settings,
	authorizationService OAuthAuthorizationRouteService,
) {
	dependencies := oauthAuthorizationDependencies{
		settings:             settings,
		authorizationService: authorizationService,
	}
	router.Get("/oauth/authorize", dependencies.handleOAuthAuthorize)
}

func (dependencies oauthAuthorizationDependencies) handleOAuthAuthorize(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !dependencies.settings.OAuthEnabled {
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "OAuth endpoints disabled"})
		return
	}
	if dependencies.authorizationService == nil {
		writeOAuthError(writer, http.StatusInternalServerError, "server_error", "OAuth authorization service unavailable.")
		return
	}
	payload := dependencies.decodeOAuthAuthorizeRequest(request)
	if ok, detail := internaloauth.AuthorizationRequestHasRequiredFields(payload); !ok {
		writeOAuthError(writer, http.StatusBadRequest, "invalid_request", detail)
		return
	}
	result, err := dependencies.authorizationService.HandleAuthorize(
		request.Context(),
		dependencies.settings,
		payload,
	)
	if err != nil {
		writeOAuthError(writer, http.StatusInternalServerError, "server_error", "Internal server error.")
		return
	}
	if result.RedirectURL != "" {
		http.Redirect(writer, request, result.RedirectURL, http.StatusSeeOther)
		return
	}
	writeOAuthError(
		writer,
		internaloauth.AuthorizationResultStatus(result),
		internaloauth.AuthorizationResultCode(result),
		internaloauth.AuthorizationResultErrorMessage(result),
	)
}

func (dependencies oauthAuthorizationDependencies) decodeOAuthAuthorizeRequest(
	request *http.Request,
) internaloauth.AuthorizationRequest {
	query := request.URL.Query()
	return internaloauth.AuthorizationRequest{
		RequestPathWithQuery: internaloauth.AuthorizationRequestPathWithQuery(request.URL.Path, request.URL.RawQuery),
		Issuer:               internaloauth.IssuerURLForRequest(request, dependencies.settings),
		ResponseType:         query.Get("response_type"),
		ClientID:             query.Get("client_id"),
		RedirectURI:          query.Get("redirect_uri"),
		State:                query.Get("state"),
		Scope:                query.Get("scope"),
		CodeChallenge:        query.Get("code_challenge"),
		CodeChallengeMethod:  query.Get("code_challenge_method"),
		Resource:             query.Get("resource"),
		SessionUser:          oauthSessionUserFromRequest(request),
	}
}

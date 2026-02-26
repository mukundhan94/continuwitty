package api

import (
	"context"
	"net/http"

	"engram/internal/config"
	internaloauth "engram/internal/oauth"

	"github.com/go-chi/chi/v5"
)

// OAuthTokenRouteService captures OAuth token route behavior.
type OAuthTokenRouteService interface {
	HandleToken(
		ctx context.Context,
		settings config.Settings,
		request internaloauth.TokenRequest,
	) (internaloauth.TokenResult, error)
}

type oauthTokenDependencies struct {
	settings     config.Settings
	tokenService OAuthTokenRouteService
}

// MountOAuthTokenRoutes registers OAuth token exchange route.
func MountOAuthTokenRoutes(
	router chi.Router,
	settings config.Settings,
	tokenService OAuthTokenRouteService,
) {
	dependencies := oauthTokenDependencies{
		settings:     settings,
		tokenService: tokenService,
	}
	router.Post("/oauth/token", dependencies.handleOAuthToken)
}

func (dependencies oauthTokenDependencies) handleOAuthToken(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !dependencies.settings.OAuthEnabled {
		writeJSON(writer, http.StatusNotFound, map[string]string{"detail": "OAuth endpoints disabled"})
		return
	}
	if dependencies.tokenService == nil {
		writeOAuthError(writer, http.StatusInternalServerError, "server_error", "OAuth token service unavailable.")
		return
	}
	payload, ok := decodeOAuthTokenRequest(writer, request)
	if !ok {
		return
	}
	result, err := dependencies.tokenService.HandleToken(
		request.Context(),
		dependencies.settings,
		payload,
	)
	if err != nil {
		writeOAuthError(writer, http.StatusInternalServerError, "server_error", "Internal server error.")
		return
	}
	if result.ErrorCode != "" {
		writeOAuthError(
			writer,
			internaloauth.TokenResultStatus(result),
			result.ErrorCode,
			result.ErrorDescription,
		)
		return
	}
	writeJSON(writer, http.StatusOK, result)
}

func decodeOAuthTokenRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (internaloauth.TokenRequest, bool) {
	if err := request.ParseForm(); err != nil {
		writeOAuthError(writer, http.StatusBadRequest, "invalid_request", "Invalid token request payload.")
		return internaloauth.TokenRequest{}, false
	}
	payload := internaloauth.TokenRequest{
		GrantType:    request.FormValue("grant_type"),
		Code:         request.FormValue("code"),
		RedirectURI:  request.FormValue("redirect_uri"),
		ClientID:     request.FormValue("client_id"),
		CodeVerifier: request.FormValue("code_verifier"),
		ClientSecret: request.FormValue("client_secret"),
	}
	if ok, detail := internaloauth.TokenRequestHasRequiredFields(payload); !ok {
		writeOAuthError(writer, http.StatusBadRequest, "invalid_request", detail)
		return internaloauth.TokenRequest{}, false
	}
	return payload, true
}

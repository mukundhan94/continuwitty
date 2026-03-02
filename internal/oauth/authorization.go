package oauth

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"engram/internal/config"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

const (
	defaultOAuthAuthorizePath = "/oauth/authorize"
	defaultCodeChallenge      = "S256"
	minAuthorizationCodeTTL   = 30
)

// AuthorizationRequest captures OAuth authorization endpoint query and caller context.
type AuthorizationRequest struct {
	RequestPathWithQuery string
	Issuer               string
	ResponseType         string
	ClientID             string
	RedirectURI          string
	State                string
	Scope                string
	CodeChallenge        string
	CodeChallengeMethod  string
	Resource             string
	SessionUser          *SessionUser
}

// AuthorizationResult captures authorize endpoint output for either redirects or oauth errors.
type AuthorizationResult struct {
	RedirectURL      string
	ErrorCode        string
	ErrorDescription string
	StatusCode       int
}

// AuthorizationService handles OAuth authorization-code issuance behavior.
type AuthorizationService struct {
	db   repository.Queryer
	deps authorizationDeps
}

type authorizationDeps struct {
	nowUTC                       func() time.Time
	generateAuthorizationCode    func() (string, error)
	authorizationCodeHash        func(input AuthorizationCodeHashInput) string
	getOAuthClient               func(ctx context.Context, db repository.Queryer, clientID string) (*models.OAuthClientRecord, error)
	createOAuthAuthorizationCode func(
		ctx context.Context,
		db repository.Queryer,
		input repository.OAuthAuthorizationCodeCreateInput,
	) (*models.OAuthAuthorizationCodeRecord, error)
}

func defaultAuthorizationDeps() authorizationDeps {
	return authorizationDeps{
		nowUTC:                       func() time.Time { return time.Now().UTC() },
		generateAuthorizationCode:    GenerateAuthorizationCode,
		authorizationCodeHash:        AuthorizationCodeHash,
		getOAuthClient:               repository.GetOAuthClient,
		createOAuthAuthorizationCode: repository.CreateOAuthAuthorizationCode,
	}
}

// NewAuthorizationService builds an authorization service with repository defaults.
func NewAuthorizationService(db repository.Queryer) *AuthorizationService {
	return &AuthorizationService{db: db, deps: defaultAuthorizationDeps()}
}

// HandleAuthorize validates authorize request parameters and returns redirect/error output.
func (service *AuthorizationService) HandleAuthorize(
	ctx context.Context,
	settings config.Settings,
	request AuthorizationRequest,
) (AuthorizationResult, error) {
	client, validationResult, err := service.resolveOAuthClient(ctx, request)
	if err != nil {
		return AuthorizationResult{}, err
	}
	if validationResult != nil {
		return *validationResult, nil
	}
	challengeMethod, authorizeResult := validateAuthorizeRequest(*client, request)
	if authorizeResult != nil {
		return *authorizeResult, nil
	}
	if loginResult := resolveAuthorizeLoginRedirect(request); loginResult != nil {
		return *loginResult, nil
	}
	code, err := service.createAuthorizationCode(ctx, settings, request, challengeMethod)
	if err != nil {
		return AuthorizationResult{}, err
	}
	redirectURL, err := buildAuthorizeSuccessRedirectURL(request, code)
	if err != nil {
		return AuthorizationResult{}, err
	}
	return AuthorizationResult{RedirectURL: redirectURL}, nil
}

func (service *AuthorizationService) resolveOAuthClient(
	ctx context.Context,
	request AuthorizationRequest,
) (*models.OAuthClientRecord, *AuthorizationResult, error) {
	client, err := service.deps.getOAuthClient(ctx, service.db, strings.TrimSpace(request.ClientID))
	if err != nil {
		return nil, nil, err
	}
	if client == nil {
		return nil, &AuthorizationResult{
			ErrorCode:        "invalid_client",
			ErrorDescription: "Unknown client_id.",
			StatusCode:       401,
		}, nil
	}
	if !RedirectURIAllowed(
		RedirectURIAllowedInput{
			RedirectURI: request.RedirectURI,
			AllowedURIs: client.RedirectURIs,
		},
	) {
		return nil, &AuthorizationResult{
			ErrorCode:        "invalid_request",
			ErrorDescription: "redirect_uri is not registered for this client.",
			StatusCode:       400,
		}, nil
	}
	return client, nil, nil
}

func validateAuthorizeRequest(
	client models.OAuthClientRecord,
	request AuthorizationRequest,
) (string, *AuthorizationResult) {
	if strings.TrimSpace(request.ResponseType) != "code" {
		return "", redirectAuthorizeError(
			request,
			"unsupported_response_type",
			"Only response_type=code is supported.",
		)
	}
	if !ClientSupportsAuthorizationCode(client) {
		return "", redirectAuthorizeError(
			request,
			"unauthorized_client",
			"Client is not allowed to use authorization_code.",
		)
	}
	challengeMethod := resolveCodeChallengeMethod(request.CodeChallengeMethod)
	if challengeMethod != defaultCodeChallenge {
		return "", redirectAuthorizeError(
			request,
			"invalid_request",
			"Unsupported code_challenge_method.",
		)
	}
	if strings.TrimSpace(client.TokenEndpointAuthMethod) == "none" && strings.TrimSpace(request.CodeChallenge) == "" {
		return "", redirectAuthorizeError(
			request,
			"invalid_request",
			"code_challenge is required for public clients.",
		)
	}
	return challengeMethod, nil
}

func resolveCodeChallengeMethod(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return defaultCodeChallenge
	}
	return normalized
}

func resolveAuthorizeLoginRedirect(request AuthorizationRequest) *AuthorizationResult {
	if request.SessionUser == nil {
		return &AuthorizationResult{RedirectURL: oauthLoginRedirectURL(request)}
	}
	if _, err := uuid.Parse(strings.TrimSpace(request.SessionUser.UserID)); err != nil {
		return &AuthorizationResult{RedirectURL: oauthLoginRedirectURL(request)}
	}
	return nil
}

func oauthLoginRedirectURL(request AuthorizationRequest) string {
	nextPath := strings.TrimSpace(request.RequestPathWithQuery)
	if nextPath == "" {
		nextPath = defaultOAuthAuthorizePath
	}
	return "/login?next=" + url.QueryEscape(nextPath)
}

func (service *AuthorizationService) createAuthorizationCode(
	ctx context.Context,
	settings config.Settings,
	request AuthorizationRequest,
	challengeMethod string,
) (string, error) {
	userID, _ := uuid.Parse(strings.TrimSpace(request.SessionUser.UserID))
	generatedCode, err := service.deps.generateAuthorizationCode()
	if err != nil {
		return "", err
	}
	codeHash := service.deps.authorizationCodeHash(
		AuthorizationCodeHashInput{
			Code:   generatedCode,
			Pepper: settings.MCPTokenPepper,
		},
	)
	_, err = service.deps.createOAuthAuthorizationCode(
		ctx,
		service.db,
		repository.OAuthAuthorizationCodeCreateInput{
			CodeID:              uuid.New(),
			CodeHash:            codeHash,
			ClientID:            strings.TrimSpace(request.ClientID),
			UserID:              userID,
			RedirectURI:         strings.TrimSpace(request.RedirectURI),
			CodeChallenge:       strings.TrimSpace(request.CodeChallenge),
			CodeChallengeMethod: challengeMethod,
			RequestedScope:      NormalizeScope(request.Scope),
			Resource:            optionalTrimmedValue(request.Resource),
			ExpiresAt:           service.deps.nowUTC().Add(resolveAuthorizationCodeTTL(settings)),
		},
	)
	if err != nil {
		return "", err
	}
	return generatedCode, nil
}

func resolveAuthorizationCodeTTL(settings config.Settings) time.Duration {
	ttlSeconds := settings.OAuthAuthorizationCodeTTLSeconds
	if ttlSeconds < minAuthorizationCodeTTL {
		ttlSeconds = minAuthorizationCodeTTL
	}
	return time.Duration(ttlSeconds) * time.Second
}

func buildAuthorizeSuccessRedirectURL(request AuthorizationRequest, code string) (string, error) {
	return mergeAuthorizeQueryParams(
		request.RedirectURI,
		map[string]string{
			"code": code,
			"iss":  request.Issuer,
		},
		request.State,
	)
}

func redirectAuthorizeError(
	request AuthorizationRequest,
	errorCode string,
	description string,
) *AuthorizationResult {
	redirectURL, err := mergeAuthorizeQueryParams(
		request.RedirectURI,
		map[string]string{
			"error":             errorCode,
			"error_description": description,
		},
		request.State,
	)
	if err != nil {
		return &AuthorizationResult{
			ErrorCode:        "server_error",
			ErrorDescription: "failed to build oauth redirect URL",
			StatusCode:       500,
		}
	}
	return &AuthorizationResult{RedirectURL: redirectURL}
}

func mergeAuthorizeQueryParams(
	rawRedirectURI string,
	params map[string]string,
	state string,
) (string, error) {
	redirectURI, err := url.Parse(strings.TrimSpace(rawRedirectURI))
	if err != nil {
		return "", err
	}
	query := redirectURI.Query()
	for key, value := range params {
		if strings.TrimSpace(value) == "" {
			continue
		}
		query.Set(key, value)
	}
	if strings.TrimSpace(state) != "" {
		query.Set("state", state)
	}
	redirectURI.RawQuery = query.Encode()
	return redirectURI.String(), nil
}

func optionalTrimmedValue(value string) *string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

// AuthorizationResultStatus returns response status for oauth error outputs.
func AuthorizationResultStatus(result AuthorizationResult) int {
	if result.StatusCode > 0 {
		return result.StatusCode
	}
	return 400
}

// AuthorizationResultErrorMessage returns a stable fallback description for logging.
func AuthorizationResultErrorMessage(result AuthorizationResult) string {
	if strings.TrimSpace(result.ErrorDescription) != "" {
		return result.ErrorDescription
	}
	return "oauth authorization error"
}

// AuthorizationResultCode returns a stable oauth error code fallback.
func AuthorizationResultCode(result AuthorizationResult) string {
	code := strings.TrimSpace(result.ErrorCode)
	if code != "" {
		return code
	}
	return "invalid_request"
}

// AuthorizationRequestPathWithQuery formats request path and query for login redirects.
func AuthorizationRequestPathWithQuery(path string, query string) string {
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		normalizedPath = defaultOAuthAuthorizePath
	}
	normalizedQuery := strings.TrimSpace(query)
	if normalizedQuery == "" {
		return normalizedPath
	}
	return normalizedPath + "?" + normalizedQuery
}

// AuthorizationRequestHasRequiredFields validates required authorization query parameters.
func AuthorizationRequestHasRequiredFields(request AuthorizationRequest) (bool, string) {
	if strings.TrimSpace(request.ResponseType) == "" {
		return false, "response_type is required"
	}
	if strings.TrimSpace(request.ClientID) == "" {
		return false, "client_id is required"
	}
	if strings.TrimSpace(request.RedirectURI) == "" {
		return false, "redirect_uri is required"
	}
	return true, ""
}

// ParseOAuthTokenTTL returns a positive integer string for compatibility logs.
func ParseOAuthTokenTTL(seconds int) string {
	if seconds <= 0 {
		return "0"
	}
	return strconv.Itoa(seconds)
}

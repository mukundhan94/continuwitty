package oauth

import (
	"context"
	"errors"
	"sort"
	"strings"

	"engram/internal/config"
	"engram/internal/models"
	"engram/internal/repository"
)

var (
	supportedGrantTypes              = map[string]struct{}{"authorization_code": {}}
	supportedResponseTypes           = map[string]struct{}{"code": {}}
	supportedTokenEndpointAuthMethod = map[string]struct{}{"none": {}, "client_secret_post": {}}
)

// SessionUser captures authorization context for protected registration.
type SessionUser struct {
	UserID   string
	Username string
	Role     string
}

// RegistrationRequest captures OAuth dynamic client registration metadata.
type RegistrationRequest struct {
	RedirectURIs            []string
	ClientName              string
	GrantTypes              []string
	ResponseTypes           []string
	TokenEndpointAuthMethod string
}

// RegistrationResponse captures successful OAuth dynamic registration output.
type RegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientSecret            *string  `json:"client_secret,omitempty"`
	ClientSecretExpiresAt   *int     `json:"client_secret_expires_at,omitempty"`
}

// RegistrationError captures oauth-style registration errors.
type RegistrationError struct {
	ErrorCode   string
	Description string
	StatusCode  int
}

func (err RegistrationError) Error() string {
	return err.Description
}

// RegistrationService handles OAuth dynamic client registration.
type RegistrationService struct {
	db   repository.Queryer
	deps registrationDeps
}

type registrationDeps struct {
	buildClientID     func() string
	issueClientSecret func() (string, error)
	oauthSecretHash   func(input SecretHashInput) string
	createOAuthClient func(ctx context.Context, db repository.Queryer, input repository.OAuthClientCreateInput) (*models.OAuthClientRecord, error)
}

func defaultRegistrationDeps() registrationDeps {
	return registrationDeps{
		buildClientID:     BuildOAuthClientID,
		issueClientSecret: IssueOAuthClientSecret,
		oauthSecretHash:   OAuthSecretHash,
		createOAuthClient: repository.CreateOAuthClient,
	}
}

// NewRegistrationService builds a registration service with repository defaults.
func NewRegistrationService(db repository.Queryer) *RegistrationService {
	return &RegistrationService{db: db, deps: defaultRegistrationDeps()}
}

// HandleRegister validates and persists OAuth dynamic client registration metadata.
func (service *RegistrationService) HandleRegister(
	ctx context.Context,
	settings config.Settings,
	payload RegistrationRequest,
	sessionUser *SessionUser,
) (RegistrationResponse, error) {
	if !settings.OAuthEnabled {
		return RegistrationResponse{}, RegistrationError{
			ErrorCode:   "oauth_disabled",
			Description: "OAuth endpoints disabled",
			StatusCode:  404,
		}
	}
	if err := authorizeRegistrationRequest(settings, sessionUser); err != nil {
		return RegistrationResponse{}, err
	}
	validatedPayload, err := validateRegistrationPayload(payload)
	if err != nil {
		return RegistrationResponse{}, err
	}
	createdClientID := service.deps.buildClientID()
	clientSecret, clientSecretHash, err := issueConfidentialClientSecret(
		service.deps,
		settings,
		createdClientID,
		validatedPayload.TokenEndpointAuthMethod,
	)
	if err != nil {
		return RegistrationResponse{}, err
	}
	created, err := service.deps.createOAuthClient(
		ctx,
		service.db,
		repository.OAuthClientCreateInput{
			ClientID:                createdClientID,
			ClientName:              strings.TrimSpace(payload.ClientName),
			RedirectURIs:            validatedPayload.RedirectURIs,
			GrantTypes:              validatedPayload.GrantTypes,
			ResponseTypes:           validatedPayload.ResponseTypes,
			TokenEndpointAuthMethod: validatedPayload.TokenEndpointAuthMethod,
			ClientSecretHash:        clientSecretHash,
			MetadataJSON:            payloadToMetadata(payload),
		},
	)
	if err != nil {
		return RegistrationResponse{}, err
	}
	response := RegistrationResponse{
		ClientID:                created.ClientID,
		ClientName:              created.ClientName,
		RedirectURIs:            created.RedirectURIs,
		GrantTypes:              created.GrantTypes,
		ResponseTypes:           created.ResponseTypes,
		TokenEndpointAuthMethod: created.TokenEndpointAuthMethod,
		ClientIDIssuedAt:        created.CreatedAt.Unix(),
		ClientSecret:            clientSecret,
	}
	if clientSecret != nil {
		expiresAt := 0
		response.ClientSecretExpiresAt = &expiresAt
	}
	return response, nil
}

type validatedRegistrationRequest struct {
	RedirectURIs            []string
	GrantTypes              []string
	ResponseTypes           []string
	TokenEndpointAuthMethod string
}

// DedupStringList trims and deduplicates string items while preserving order.
func DedupStringList(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func authorizeRegistrationRequest(settings config.Settings, sessionUser *SessionUser) error {
	if !settings.OAuthRequireProtectedRegistration {
		return nil
	}
	if sessionUser == nil {
		return RegistrationError{
			ErrorCode:   "invalid_client",
			Description: "Authenticated admin session required for dynamic registration.",
			StatusCode:  401,
		}
	}
	if strings.ToLower(strings.TrimSpace(sessionUser.Role)) != "admin" {
		return RegistrationError{
			ErrorCode:   "insufficient_privilege",
			Description: "Admin role is required for dynamic registration.",
			StatusCode:  403,
		}
	}
	return nil
}

func validateRegistrationPayload(payload RegistrationRequest) (validatedRegistrationRequest, error) {
	redirectURIs := DedupStringList(payload.RedirectURIs)
	if len(redirectURIs) == 0 {
		return validatedRegistrationRequest{}, RegistrationError{
			ErrorCode:   "invalid_redirect_uri",
			Description: "At least one redirect_uri is required.",
			StatusCode:  400,
		}
	}
	tokenEndpointAuthMethod := strings.TrimSpace(payload.TokenEndpointAuthMethod)
	if tokenEndpointAuthMethod == "" {
		tokenEndpointAuthMethod = "none"
	}
	if _, ok := supportedTokenEndpointAuthMethod[tokenEndpointAuthMethod]; !ok {
		return validatedRegistrationRequest{}, RegistrationError{
			ErrorCode:   "invalid_client_metadata",
			Description: "Unsupported token_endpoint_auth_method.",
			StatusCode:  400,
		}
	}
	grantTypes := normalizeMetadataList(payload.GrantTypes, supportedGrantTypes)
	if !containsValue(grantTypes, "authorization_code") {
		return validatedRegistrationRequest{}, RegistrationError{
			ErrorCode:   "invalid_client_metadata",
			Description: "authorization_code grant type is required.",
			StatusCode:  400,
		}
	}
	responseTypes := normalizeMetadataList(payload.ResponseTypes, supportedResponseTypes)
	if !containsValue(responseTypes, "code") {
		return validatedRegistrationRequest{}, RegistrationError{
			ErrorCode:   "invalid_client_metadata",
			Description: "code response type is required.",
			StatusCode:  400,
		}
	}
	return validatedRegistrationRequest{
		RedirectURIs:            redirectURIs,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		TokenEndpointAuthMethod: tokenEndpointAuthMethod,
	}, nil
}

func normalizeMetadataList(values []string, fallback map[string]struct{}) []string {
	normalized := DedupStringList(values)
	if len(normalized) > 0 {
		return normalized
	}
	fallbackValues := make([]string, 0, len(fallback))
	for value := range fallback {
		fallbackValues = append(fallbackValues, value)
	}
	sort.Strings(fallbackValues)
	return fallbackValues
}

func containsValue(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func issueConfidentialClientSecret(
	deps registrationDeps,
	settings config.Settings,
	clientID string,
	tokenEndpointAuthMethod string,
) (*string, *string, error) {
	if tokenEndpointAuthMethod != "client_secret_post" {
		return nil, nil, nil
	}
	clientSecret, err := deps.issueClientSecret()
	if err != nil {
		return nil, nil, err
	}
	clientSecretHash := deps.oauthSecretHash(SecretHashInput{
		Identifier: clientID,
		Secret:     clientSecret,
		Pepper:     settings.OAuthClientSecretPepper,
	})
	return &clientSecret, &clientSecretHash, nil
}

func payloadToMetadata(payload RegistrationRequest) map[string]any {
	return map[string]any{
		"redirect_uris":              payload.RedirectURIs,
		"client_name":                payload.ClientName,
		"grant_types":                payload.GrantTypes,
		"response_types":             payload.ResponseTypes,
		"token_endpoint_auth_method": payload.TokenEndpointAuthMethod,
	}
}

func asRegistrationError(err error) (RegistrationError, bool) {
	registrationErr := RegistrationError{}
	if errors.As(err, &registrationErr) {
		return registrationErr, true
	}
	return RegistrationError{}, false
}

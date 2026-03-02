package api

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"engram/internal/auth"
	"engram/internal/models"
)

const sessionUITestOIDCAuthorizeURL = "https://accounts.example.com/oauth/authorize?client_id=engram-web"

type sessionUITestOIDCProvider struct {
	enabled          bool
	authURL          string
	authURLError     error
	identity         *auth.OIDCIdentity
	authenticateErr  error
	authURLCalls     []sessionUITestOIDCAuthURLCall
	authenticateCall []sessionUITestOIDCAuthenticateCall
}

type sessionUITestOIDCAuthURLCall struct {
	State string
	Nonce string
}

type sessionUITestOIDCAuthenticateCall struct {
	Code  string
	Nonce string
}

type oidcCallbackFailureCase struct {
	name                string
	provider            *sessionUITestOIDCProvider
	lookupByUsername    SessionUserByUsernameLookup
	expectedStatus      int
	expectedDetail      string
	expectedRedirect    string
	expectedAuditDetail string
}

type oidcIdentityFailureSpec struct {
	name                string
	subject             string
	username            string
	email               string
	lookupByUsername    SessionUserByUsernameLookup
	expectedStatus      int
	expectedDetail      string
	expectedAuditDetail string
}

func (provider *sessionUITestOIDCProvider) Enabled() bool {
	return provider != nil && provider.enabled
}

func (provider *sessionUITestOIDCProvider) AuthCodeURL(state string, nonce string) (string, error) {
	provider.authURLCalls = append(provider.authURLCalls, sessionUITestOIDCAuthURLCall{
		State: state,
		Nonce: nonce,
	})
	if provider.authURLError != nil {
		return "", provider.authURLError
	}
	if provider.authURL == "" {
		return "https://accounts.example.com/oauth/authorize", nil
	}
	return provider.authURL, nil
}

func (provider *sessionUITestOIDCProvider) AuthenticateCode(
	_ context.Context,
	code string,
	nonce string,
) (*auth.OIDCIdentity, error) {
	provider.authenticateCall = append(provider.authenticateCall, sessionUITestOIDCAuthenticateCall{
		Code:  code,
		Nonce: nonce,
	})
	if provider.authenticateErr != nil {
		return nil, provider.authenticateErr
	}
	if provider.identity != nil {
		return provider.identity, nil
	}
	return &auth.OIDCIdentity{
		Subject:  "subject-1",
		Username: "admin",
		Email:    "admin@example.com",
	}, nil
}

func buildOIDCCallbackFailureCases() []oidcCallbackFailureCase {
	return []oidcCallbackFailureCase{
		oidcProviderVerificationFailureCase(),
		oidcIdentityFailureCase(oidcIdentityFailureSpec{
			name:                "identity without username",
			subject:             "subject-identity-missing-username",
			username:            "",
			email:               "missing@example.com",
			expectedStatus:      401,
			expectedDetail:      "oidc identity missing username",
			expectedAuditDetail: "identity_missing_username",
		}),
		oidcIdentityFailureCase(oidcIdentityFailureSpec{
			name:                "unmapped user",
			subject:             "subject-unknown-user",
			username:            "ghost-user",
			email:               "ghost@example.com",
			expectedStatus:      403,
			expectedDetail:      "oidc user is not authorized",
			expectedAuditDetail: "user_not_authorized",
		}),
		oidcIdentityFailureCase(oidcIdentityFailureSpec{
			name:                "inactive mapped user",
			subject:             "subject-inactive-user",
			username:            "admin",
			email:               "admin@example.com",
			lookupByUsername:    oidcInactiveUserLookup,
			expectedStatus:      403,
			expectedDetail:      "oidc user is not authorized",
			expectedAuditDetail: "user_not_authorized",
		}),
	}
}

func oidcProviderVerificationFailureCase() oidcCallbackFailureCase {
	return oidcCallbackFailureCase{
		name: "provider verification failure",
		provider: &sessionUITestOIDCProvider{
			enabled:         true,
			authURL:         sessionUITestOIDCAuthorizeURL,
			authenticateErr: errors.New("id token validation failed"),
		},
		expectedRedirect:    "/login",
		expectedAuditDetail: "token_exchange_or_verification_failed",
	}
}

func oidcIdentityFailureCase(spec oidcIdentityFailureSpec) oidcCallbackFailureCase {
	return oidcCallbackFailureCase{
		name: spec.name,
		provider: &sessionUITestOIDCProvider{
			enabled: true,
			authURL: sessionUITestOIDCAuthorizeURL,
			identity: &auth.OIDCIdentity{
				Subject:  spec.subject,
				Username: spec.username,
				Email:    spec.email,
			},
		},
		lookupByUsername:    spec.lookupByUsername,
		expectedStatus:      spec.expectedStatus,
		expectedDetail:      spec.expectedDetail,
		expectedAuditDetail: spec.expectedAuditDetail,
	}
}

func oidcInactiveUserLookup(_ context.Context, username string) (*models.UserAuthRecord, error) {
	if username != "admin" {
		return nil, nil
	}
	return &models.UserAuthRecord{
		Username: "admin",
		Role:     models.UserRoleAdmin,
		IsActive: false,
	}, nil
}

func runOIDCCallbackFailureCase(t *testing.T, testCase oidcCallbackFailureCase) {
	t.Helper()
	auditLogPath := filepath.Join(t.TempDir(), "oidc-callback-failure-audit.log")
	handler, manager, _ := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{
			oidcProvider:     testCase.provider,
			lookupByUsername: testCase.lookupByUsername,
			auditLogPath:     auditLogPath,
		},
	)
	pendingCookie, pendingState := startOIDCLoginFlow(t, handler, manager)
	callbackResponse := performOIDCCallbackRequest(handler, pendingCookie, pendingState.OIDCState, "auth-code-1")

	if testCase.expectedRedirect != "" {
		assertRedirect(t, callbackResponse, testCase.expectedRedirect)
	} else {
		assertJSONDetail(t, callbackResponse, testCase.expectedStatus, testCase.expectedDetail)
	}
	assertOIDCCallbackCall(t, testCase.provider, pendingState.OIDCNonce)
	assertOIDCPendingStateCleared(t, manager, callbackResponse)
	assertAuditLogContains(t, auditLogPath, "\"event_type\":\"oidc_login_failed\"")
	assertAuditLogContains(t, auditLogPath, "\"detail\":\""+testCase.expectedAuditDetail+"\"")
}

package api

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"engram/internal/audit"
	"engram/internal/auth"
)

func TestMountSessionUIRoutesOIDCCallbackAuthenticatesAndRedirects(t *testing.T) {
	oidcProvider := &sessionUITestOIDCProvider{
		enabled: true,
		authURL: sessionUITestOIDCAuthorizeURL,
		identity: &auth.OIDCIdentity{
			Subject:  "subject-2",
			Username: "admin",
			Email:    "admin@example.com",
		},
	}
	handler, manager, _ := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{oidcProvider: oidcProvider},
	)
	pendingCookie, pendingState := startOIDCLoginFlow(t, handler, manager)

	callbackResponse := performOIDCCallbackRequest(handler, pendingCookie, pendingState.OIDCState, "auth-code-1")

	assertRedirect(t, callbackResponse, "/ui/admin")
	assertOIDCCallbackCall(t, oidcProvider, pendingState.OIDCNonce)
	assertAuthenticatedOIDCSessionState(t, manager, callbackResponse)
}

func TestMountSessionUIRoutesOIDCCallbackRejectsReplayAfterPendingStateConsumed(t *testing.T) {
	auditLogPath := filepath.Join(t.TempDir(), "oidc-callback-replay-audit.log")
	oidcProvider := &sessionUITestOIDCProvider{
		enabled: true,
		authURL: sessionUITestOIDCAuthorizeURL,
		identity: &auth.OIDCIdentity{
			Subject:  "subject-2",
			Username: "admin",
			Email:    "admin@example.com",
		},
	}
	handler, manager, _ := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{
			oidcProvider: oidcProvider,
			auditLogPath: auditLogPath,
		},
	)
	pendingCookie, pendingState := startOIDCLoginFlow(t, handler, manager)

	firstCallback := performOIDCCallbackRequest(handler, pendingCookie, pendingState.OIDCState, "auth-code-1")
	assertRedirect(t, firstCallback, "/ui/admin")
	assertOIDCCallbackCall(t, oidcProvider, pendingState.OIDCNonce)

	consumedStateCookie := findResponseCookie(firstCallback, manager.CookieName())
	if consumedStateCookie == nil {
		t.Fatalf("expected callback response to include authenticated session cookie")
	}
	replayCallback := performOIDCCallbackRequest(handler, consumedStateCookie, pendingState.OIDCState, "auth-code-2")

	assertJSONDetail(t, replayCallback, http.StatusForbidden, "invalid oidc state")
	if len(oidcProvider.authenticateCall) != 1 {
		t.Fatalf("expected oidc callback exchange to remain single-use for consumed pending state")
	}
	assertAuditLogContains(t, auditLogPath, "\"event_type\":\"oidc_login_failed\"")
	assertAuditLogContains(t, auditLogPath, "\"detail\":\"state_mismatch\"")
}

func TestMountSessionUIRoutesOIDCCallbackRejectsInvalidRequest(t *testing.T) {
	testCases := []struct {
		name                string
		providerState       string
		usePendingState     bool
		authorizationCode   string
		expectedStatus      int
		expectedDetail      string
		expectedAuditDetail string
	}{
		{
			name:                "state mismatch",
			providerState:       "invalid-state",
			authorizationCode:   "auth-code-1",
			expectedStatus:      http.StatusForbidden,
			expectedDetail:      "invalid oidc state",
			expectedAuditDetail: "state_mismatch",
		},
		{
			name:                "missing authorization code",
			usePendingState:     true,
			authorizationCode:   "",
			expectedStatus:      http.StatusBadRequest,
			expectedDetail:      "missing oidc authorization code",
			expectedAuditDetail: "missing_code",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			auditLogPath := filepath.Join(t.TempDir(), "oidc-invalid-request-audit.log")
			oidcProvider := &sessionUITestOIDCProvider{
				enabled: true,
				authURL: sessionUITestOIDCAuthorizeURL,
			}
			handler, manager, _ := buildSessionUITestHandler(
				t,
				sessionUITestHandlerOptions{
					oidcProvider: oidcProvider,
					auditLogPath: auditLogPath,
				},
			)
			pendingCookie, pendingState := startOIDCLoginFlow(t, handler, manager)
			providerState := testCase.providerState
			if testCase.usePendingState {
				providerState = pendingState.OIDCState
			}

			callbackResponse := performOIDCCallbackRequest(
				handler,
				pendingCookie,
				providerState,
				testCase.authorizationCode,
			)

			assertJSONDetail(t, callbackResponse, testCase.expectedStatus, testCase.expectedDetail)
			assertOIDCCallbackNotCalled(t, oidcProvider)
			assertAuditLogContains(t, auditLogPath, "\"event_type\":\"oidc_login_failed\"")
			assertAuditLogContains(t, auditLogPath, "\"detail\":\""+testCase.expectedAuditDetail+"\"")
		})
	}
}

func TestMountSessionUIRoutesOIDCCallbackFailurePaths(t *testing.T) {
	for _, testCase := range buildOIDCCallbackFailureCases() {
		t.Run(testCase.name, func(t *testing.T) {
			runOIDCCallbackFailureCase(t, testCase)
		})
	}
}

func TestMountSessionUIRoutesOIDCCallbackFailureEmitsAuditEventToSink(t *testing.T) {
	var sinkMu sync.Mutex
	receivedAuthHeaders := []string{}
	receivedBodies := []string{}
	sink := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		payload, _ := io.ReadAll(request.Body)
		sinkMu.Lock()
		receivedAuthHeaders = append(receivedAuthHeaders, request.Header.Get("Authorization"))
		receivedBodies = append(receivedBodies, string(payload))
		sinkMu.Unlock()
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer sink.Close()

	sinkLogger := audit.NewLogger(audit.LoggerOptions{
		SinkURL:       sink.URL,
		SinkAuthToken: "oidc-sink-token",
		SinkRequired:  true,
	})
	oidcProvider := &sessionUITestOIDCProvider{
		enabled:         true,
		authURL:         sessionUITestOIDCAuthorizeURL,
		authenticateErr: errors.New("id token validation failed"),
	}
	handler, manager, _ := buildSessionUITestHandler(
		t,
		sessionUITestHandlerOptions{
			oidcProvider: oidcProvider,
			auditLogger: func(
				request *http.Request,
				eventType string,
				success bool,
				username string,
				detail string,
				metadata map[string]any,
			) {
				err := sinkLogger.LogRequestEvent(audit.RequestEvent{
					Request:   request,
					EventType: eventType,
					Success:   success,
					Username:  username,
					Detail:    detail,
					Metadata:  metadata,
				})
				if err != nil {
					t.Fatalf("expected oidc audit sink write to succeed: %v", err)
				}
			},
		},
	)
	pendingCookie, pendingState := startOIDCLoginFlow(t, handler, manager)
	callbackResponse := performOIDCCallbackRequest(handler, pendingCookie, pendingState.OIDCState, "auth-code-1")

	assertRedirect(t, callbackResponse, "/login")
	assertOIDCCallbackCall(t, oidcProvider, pendingState.OIDCNonce)
	assertOIDCPendingStateCleared(t, manager, callbackResponse)

	sinkMu.Lock()
	headers := append([]string(nil), receivedAuthHeaders...)
	bodies := append([]string(nil), receivedBodies...)
	sinkMu.Unlock()
	if len(headers) == 0 {
		t.Fatalf("expected centralized audit sink to receive oidc events")
	}
	if !containsString(headers, "Bearer oidc-sink-token") {
		t.Fatalf("expected sink auth header to be forwarded")
	}
	if !anyStringContains(bodies, "\"event_type\":\"oidc_login_failed\"") {
		t.Fatalf("expected oidc failure event in sink payloads")
	}
	if !anyStringContains(bodies, "\"detail\":\"token_exchange_or_verification_failed\"") {
		t.Fatalf("expected provider verification failure detail in sink payloads")
	}
}

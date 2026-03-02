package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionManagerEncodeDecodeRoundTrip(t *testing.T) {
	manager, err := NewSessionManager("dev-session-secret-for-tests", DefaultSessionCookieName)
	if err != nil {
		t.Fatalf("expected manager creation to succeed: %v", err)
	}

	token, err := manager.Encode(
		SessionState{
			User: &SessionUser{
				UserID:   "00000000-0000-0000-0000-000000000511",
				Username: "admin",
				Role:     "admin",
			},
			CSRFToken: "csrf-token",
		},
	)
	if err != nil {
		t.Fatalf("expected session encoding to succeed: %v", err)
	}

	state, err := manager.Decode(token)
	if err != nil {
		t.Fatalf("expected session decoding to succeed: %v", err)
	}
	assertDecodedSessionUser(
		t,
		state.User,
		SessionUser{
			UserID:   "00000000-0000-0000-0000-000000000511",
			Username: "admin",
			Role:     "admin",
		},
	)
	assertDecodedSessionMetadata(t, state, "csrf-token")
}

func TestSessionManagerDecodeRejectsTamperedToken(t *testing.T) {
	manager, err := NewSessionManager("dev-session-secret-for-tests", DefaultSessionCookieName)
	if err != nil {
		t.Fatalf("expected manager creation to succeed: %v", err)
	}
	token, err := manager.Encode(SessionState{User: &SessionUser{UserID: "u1", Username: "admin", Role: "admin"}})
	if err != nil {
		t.Fatalf("expected session encoding to succeed: %v", err)
	}

	tampered := token + "tampered"
	if _, err := manager.Decode(tampered); err == nil {
		t.Fatalf("expected tampered session token to fail decoding")
	}
}

func TestSessionManagerDecodeRequestReadsCookie(t *testing.T) {
	manager, err := NewSessionManager("dev-session-secret-for-tests", DefaultSessionCookieName)
	if err != nil {
		t.Fatalf("expected manager creation to succeed: %v", err)
	}
	token, err := manager.Encode(
		SessionState{
			User: &SessionUser{
				UserID:   "00000000-0000-0000-0000-000000000512",
				Username: "admin",
				Role:     "admin",
			},
		},
	)
	if err != nil {
		t.Fatalf("expected session encoding to succeed: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/memory/sessions", nil)
	request.AddCookie(&http.Cookie{Name: manager.CookieName(), Value: token})

	state, err := manager.DecodeRequest(request)
	if err != nil {
		t.Fatalf("expected session decode request to succeed: %v", err)
	}
	if state.User == nil || state.User.UserID != "00000000-0000-0000-0000-000000000512" {
		t.Fatalf("unexpected decoded session user: %#v", state.User)
	}
}

func TestNewSessionManagerRejectsEmptySecret(t *testing.T) {
	_, err := NewSessionManager("   ", DefaultSessionCookieName)
	if err == nil {
		t.Fatalf("expected empty secret to fail")
	}
}

func TestSessionManagerDecodeRejectsExpiredToken(t *testing.T) {
	currentTime := time.Date(2026, 2, 22, 22, 0, 0, 0, time.UTC)
	manager, err := NewSessionManagerWithOptions(
		"dev-session-secret-for-tests",
		SessionManagerOptions{
			CookieName: DefaultSessionCookieName,
			TTL:        time.Minute,
			Now: func() time.Time {
				return currentTime
			},
		},
	)
	if err != nil {
		t.Fatalf("expected manager creation to succeed: %v", err)
	}
	token, err := manager.Encode(
		SessionState{
			User: &SessionUser{
				UserID:   "00000000-0000-0000-0000-000000000513",
				Username: "admin",
				Role:     "admin",
			},
		},
	)
	if err != nil {
		t.Fatalf("expected session encoding to succeed: %v", err)
	}

	currentTime = currentTime.Add(2 * time.Minute)
	if _, err := manager.Decode(token); err == nil {
		t.Fatalf("expected expired session token to fail decoding")
	}
}

func assertDecodedSessionUser(t *testing.T, actual *SessionUser, expected SessionUser) {
	t.Helper()
	if actual == nil {
		t.Fatalf("expected decoded user")
	}
	if actual.UserID != expected.UserID {
		t.Fatalf("unexpected user id %q", actual.UserID)
	}
	if actual.Username != expected.Username {
		t.Fatalf("unexpected username %q", actual.Username)
	}
	if actual.Role != expected.Role {
		t.Fatalf("unexpected role %q", actual.Role)
	}
}

func assertDecodedSessionMetadata(t *testing.T, state SessionState, expectedCSRFToken string) {
	t.Helper()
	if state.CSRFToken != expectedCSRFToken {
		t.Fatalf("unexpected csrf token %q", state.CSRFToken)
	}
	if state.IssuedAt == 0 {
		t.Fatalf("expected issued_at to be set")
	}
}

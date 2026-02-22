package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
	if state.User == nil {
		t.Fatalf("expected decoded user")
	}
	if state.User.UserID != "00000000-0000-0000-0000-000000000511" {
		t.Fatalf("unexpected user id %q", state.User.UserID)
	}
	if state.User.Role != "admin" {
		t.Fatalf("unexpected role %q", state.User.Role)
	}
	if state.CSRFToken != "csrf-token" {
		t.Fatalf("unexpected csrf token %q", state.CSRFToken)
	}
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

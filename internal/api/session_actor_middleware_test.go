package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/auth"
	"engram/internal/models"

	"github.com/google/uuid"
)

func TestSessionActorMiddlewareInjectsActorFromSessionCookie(t *testing.T) {
	manager, err := auth.NewSessionManager("dev-session-secret-for-tests", auth.DefaultSessionCookieName)
	if err != nil {
		t.Fatalf("expected manager creation to succeed: %v", err)
	}
	sessionUserID := uuid.MustParse("00000000-0000-0000-0000-000000000611")
	token, err := manager.Encode(
		auth.SessionState{
			User: &auth.SessionUser{
				UserID:   sessionUserID.String(),
				Username: "admin",
				Role:     "admin",
			},
		},
	)
	if err != nil {
		t.Fatalf("expected token encode to succeed: %v", err)
	}

	lookupCalled := false
	middleware := SessionActorMiddleware(
		manager,
		func(_ context.Context, userID uuid.UUID) (*models.UserAuthRecord, error) {
			lookupCalled = true
			if userID != sessionUserID {
				t.Fatalf("unexpected lookup user id: %s", userID)
			}
			return &models.UserAuthRecord{
				UserID:   sessionUserID,
				Username: "admin",
				Role:     models.UserRoleAdmin,
				IsActive: true,
			}, nil
		},
	)

	handler := middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		actor, err := RequireAdminActorFromContext(request)
		if err != nil {
			t.Fatalf("expected context actor, got error: %v", err)
		}
		if actor.UserID != sessionUserID {
			t.Fatalf("unexpected actor user id: %s", actor.UserID)
		}
		if actor.Role != "admin" {
			t.Fatalf("unexpected actor role: %s", actor.Role)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/memory/sessions", nil)
	request.AddCookie(&http.Cookie{Name: manager.CookieName(), Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", response.Code)
	}
	if !lookupCalled {
		t.Fatalf("expected user lookup to run")
	}
}

func TestSessionActorMiddlewareSkipsInactiveUsers(t *testing.T) {
	manager, err := auth.NewSessionManager("dev-session-secret-for-tests", auth.DefaultSessionCookieName)
	if err != nil {
		t.Fatalf("expected manager creation to succeed: %v", err)
	}
	sessionUserID := uuid.MustParse("00000000-0000-0000-0000-000000000612")
	token, err := manager.Encode(
		auth.SessionState{
			User: &auth.SessionUser{
				UserID:   sessionUserID.String(),
				Username: "admin",
				Role:     "admin",
			},
		},
	)
	if err != nil {
		t.Fatalf("expected token encode to succeed: %v", err)
	}

	middleware := SessionActorMiddleware(
		manager,
		func(_ context.Context, _ uuid.UUID) (*models.UserAuthRecord, error) {
			return &models.UserAuthRecord{
				UserID:   sessionUserID,
				Username: "admin",
				Role:     models.UserRoleAdmin,
				IsActive: false,
			}, nil
		},
	)

	handler := middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, err := RequireAdminActorFromContext(request)
		if err == nil {
			t.Fatalf("expected no context actor for inactive user")
		}
		writer.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/memory/sessions", nil)
	request.AddCookie(&http.Cookie{Name: manager.CookieName(), Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", response.Code)
	}
}

func TestSessionActorMiddlewareIgnoresInvalidSessionCookie(t *testing.T) {
	manager, err := auth.NewSessionManager("dev-session-secret-for-tests", auth.DefaultSessionCookieName)
	if err != nil {
		t.Fatalf("expected manager creation to succeed: %v", err)
	}

	middleware := SessionActorMiddleware(
		manager,
		func(_ context.Context, _ uuid.UUID) (*models.UserAuthRecord, error) {
			t.Fatalf("lookup should not run for invalid cookie")
			return nil, nil
		},
	)

	handler := middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, err := RequireAdminActorFromContext(request)
		if err == nil {
			t.Fatalf("expected no actor for invalid cookie")
		}
		writer.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/memory/sessions", nil)
	request.AddCookie(&http.Cookie{Name: manager.CookieName(), Value: "bad-token"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", response.Code)
	}
}

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
	runSessionActorMiddlewareCase(t, sessionActorMiddlewareCase{
		sessionUserID: uuid.MustParse("00000000-0000-0000-0000-000000000611"),
		lookupFn: func(_ context.Context, userID uuid.UUID) (*models.UserAuthRecord, error) {
			assertLookupUserID(t, userID, uuid.MustParse("00000000-0000-0000-0000-000000000611"))
			return &models.UserAuthRecord{
				UserID:   uuid.MustParse("00000000-0000-0000-0000-000000000611"),
				Username: "admin",
				Role:     models.UserRoleAdmin,
				IsActive: true,
			}, nil
		},
		assertRequestFn: func(t *testing.T, request *http.Request) {
			assertAdminActorInRequestContext(
				t,
				request,
				uuid.MustParse("00000000-0000-0000-0000-000000000611"),
			)
		},
		expectLookupCalled: true,
	})
}

func TestSessionActorMiddlewareSkipsInactiveUsers(t *testing.T) {
	sessionUserID := uuid.MustParse("00000000-0000-0000-0000-000000000612")
	runSessionActorMiddlewareCase(t, sessionActorMiddlewareCase{
		sessionUserID: sessionUserID,
		lookupFn: func(_ context.Context, _ uuid.UUID) (*models.UserAuthRecord, error) {
			return &models.UserAuthRecord{
				UserID:   sessionUserID,
				Username: "admin",
				Role:     models.UserRoleAdmin,
				IsActive: false,
			}, nil
		},
		assertRequestFn: func(t *testing.T, request *http.Request) {
			_, err := RequireAdminActorFromContext(request)
			if err == nil {
				t.Fatalf("expected no context actor for inactive user")
			}
		},
		expectLookupCalled: true,
	})
}

func TestSessionActorMiddlewareIgnoresInvalidSessionCookie(t *testing.T) {
	manager := mustNewSessionManagerForTests(t)

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
	assertNoContentStatus(t, response.Code)
}

func mustNewSessionManagerForTests(t *testing.T) *auth.SessionManager {
	t.Helper()
	manager, err := auth.NewSessionManager("dev-session-secret-for-tests", auth.DefaultSessionCookieName)
	if err != nil {
		t.Fatalf("expected manager creation to succeed: %v", err)
	}
	return manager
}

func mustEncodeSessionTokenForTests(
	t *testing.T,
	manager *auth.SessionManager,
	sessionUserID uuid.UUID,
) string {
	t.Helper()
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
	return token
}

func assertLookupUserID(t *testing.T, received uuid.UUID, expected uuid.UUID) {
	t.Helper()
	if received != expected {
		t.Fatalf("unexpected lookup user id: %s", received)
	}
}

func assertAdminActorInRequestContext(t *testing.T, request *http.Request, expectedUserID uuid.UUID) {
	t.Helper()
	actor, err := RequireAdminActorFromContext(request)
	if err != nil {
		t.Fatalf("expected context actor, got error: %v", err)
	}
	if actor.UserID != expectedUserID {
		t.Fatalf("unexpected actor user id: %s", actor.UserID)
	}
	if actor.Role != "admin" {
		t.Fatalf("unexpected actor role: %s", actor.Role)
	}
}

func assertNoContentStatus(t *testing.T, statusCode int) {
	t.Helper()
	if statusCode != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", statusCode)
	}
}

func assertLookupRan(t *testing.T, lookupCalled bool) {
	t.Helper()
	if !lookupCalled {
		t.Fatalf("expected user lookup to run")
	}
}

type sessionActorMiddlewareCase struct {
	sessionUserID     uuid.UUID
	lookupFn          func(context.Context, uuid.UUID) (*models.UserAuthRecord, error)
	assertRequestFn   func(*testing.T, *http.Request)
	expectLookupCalled bool
}

func runSessionActorMiddlewareCase(t *testing.T, testCase sessionActorMiddlewareCase) {
	t.Helper()
	manager := mustNewSessionManagerForTests(t)
	token := mustEncodeSessionTokenForTests(t, manager, testCase.sessionUserID)

	lookupCalled := false
	middleware := SessionActorMiddleware(
		manager,
		func(ctx context.Context, userID uuid.UUID) (*models.UserAuthRecord, error) {
			lookupCalled = true
			return testCase.lookupFn(ctx, userID)
		},
	)

	handler := middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		testCase.assertRequestFn(t, request)
		writer.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/memory/sessions", nil)
	request.AddCookie(&http.Cookie{Name: manager.CookieName(), Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	assertNoContentStatus(t, response.Code)
	if lookupCalled != testCase.expectLookupCalled {
		t.Fatalf("expected lookupCalled=%v, got %v", testCase.expectLookupCalled, lookupCalled)
	}
}

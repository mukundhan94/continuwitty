package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/config"

	"github.com/google/uuid"
)

func assertRouteStatus(
	t *testing.T,
	router http.Handler,
	path string,
	expectedStatus int,
) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != expectedStatus {
		t.Fatalf("expected status %d for %s, got %d", expectedStatus, path, response.Code)
	}
}

func assertVersionFieldValue(
	t *testing.T,
	payload map[string]string,
	key string,
	expectedValue string,
) {
	t.Helper()
	if payload[key] != expectedValue {
		t.Fatalf("expected %s %q, got %q", key, expectedValue, payload[key])
	}
}

func assertRouteRedirect(
	t *testing.T,
	router http.Handler,
	path string,
	expectedLocation string,
) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect status 303 for %s, got %d", path, response.Code)
	}
	if response.Header().Get("Location") != expectedLocation {
		t.Fatalf("expected redirect location %q for %s, got %q", expectedLocation, path, response.Header().Get("Location"))
	}
}

func exerciseActorScopedDependencyRoute(
	t *testing.T,
	dependencies RouterDependencies,
	path string,
) int {
	t.Helper()
	router := NewRouterWithDependencies(
		config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"},
		dependencies,
	)
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request = WithAdminActor(
		request,
		AdminActor{
			UserID: uuid.MustParse("00000000-0000-0000-0000-000000000242"),
			Role:   "analyst",
		},
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response.Code
}

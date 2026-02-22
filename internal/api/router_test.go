package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/config"
)

func TestHealthz(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	router := NewRouter(settings)

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("expected status payload to be 'ok', got %q", payload["status"])
	}
}

func TestVersionEndpoint(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	router := NewRouter(settings)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload["semantic_version"] != settings.AppSemanticVersion {
		t.Fatalf("expected semantic version %q, got %q", settings.AppSemanticVersion, payload["semantic_version"])
	}
	if payload["release"] != "v"+settings.AppSemanticVersion {
		t.Fatalf("expected release %q, got %q", "v"+settings.AppSemanticVersion, payload["release"])
	}
	if payload["commit_id"] == "" {
		t.Fatalf("expected non-empty commit id")
	}
}

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/admin"
	"engram/internal/config"
	"engram/internal/models"

	"github.com/google/uuid"
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

func TestMemoryAdminRoutesNotMountedWithoutDependencies(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	router := NewRouter(settings)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/memory/sessions", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
}

func TestMemoryAdminRoutesMountedWithDependencies(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	serviceCalled := false
	actorCalled := false
	service := &fakeMemoryAdminService{
		listSessionsFn: func(_ context.Context, request admin.MemoryAdminListRequest) ([]models.AdminChatSessionRecord, error) {
			serviceCalled = true
			if request.Limit != 3 {
				t.Fatalf("expected limit 3, got %d", request.Limit)
			}
			if request.Offset != 1 {
				t.Fatalf("expected offset 1, got %d", request.Offset)
			}
			return []models.AdminChatSessionRecord{}, nil
		},
	}
	router := NewRouterWithDependencies(
		settings,
		RouterDependencies{
			MemoryAdminService: service,
			RequireAdminActor: func(_ *http.Request) (AdminActor, error) {
				actorCalled = true
				return AdminActor{
					UserID: uuid.MustParse("00000000-0000-0000-0000-000000000211"),
					Role:   "admin",
				}, nil
			},
		},
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/memory/sessions?limit=3&offset=1", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if !actorCalled {
		t.Fatalf("expected actor resolver to be called")
	}
	if !serviceCalled {
		t.Fatalf("expected list sessions service to be called")
	}
}

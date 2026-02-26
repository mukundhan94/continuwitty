package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"engram/internal/config"
	"engram/internal/mcp"

	"github.com/google/uuid"
)

func TestMCPRoutesNotMountedWithoutDependencies(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	router := NewRouterWithDependencies(settings, RouterDependencies{})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/mcp/stream", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected mcp route status 404, got %d", response.Code)
	}
}

func TestMCPRoutesMountedWithDependencies(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	router := NewRouterWithDependencies(
		settings,
		RouterDependencies{
			MCPService:       mcp.NewCompatibilityService(settings.AppSemanticVersion),
			MCPActorResolver: staticMCPActorResolver{},
		},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/mcp/stream",
		strings.NewReader(`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{}}`),
	)
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected mcp route status 200, got %d", response.Code)
	}
	body := map[string]any{}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode mcp response: %v", err)
	}
	if _, ok := body["result"]; !ok {
		t.Fatalf("expected initialize result")
	}
}

type staticMCPActorResolver struct{}

func (staticMCPActorResolver) ResolveActor(_ *http.Request) (mcp.ResolvedActor, error) {
	return mcp.ResolvedActor{
		Actor: mcp.Actor{
			UserID: uuid.MustParse("50000000-0000-0000-0000-000000000005"),
			Role:   "admin",
		},
	}, nil
}

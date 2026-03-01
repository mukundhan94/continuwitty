package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"engram/internal/chat"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
)

func TestInMemoryRequestMetricsAggregatesSamples(t *testing.T) {
	metrics := NewInMemoryRequestMetrics()
	metrics.Record(
		RequestMetricSample{
			Domain:     "api",
			Method:     http.MethodGet,
			Route:      "/api/v1/engrams",
			StatusCode: http.StatusOK,
			Duration:   50 * time.Millisecond,
		},
	)
	metrics.Record(
		RequestMetricSample{
			Domain:     "api",
			Method:     http.MethodGet,
			Route:      "/api/v1/engrams",
			StatusCode: http.StatusInternalServerError,
			Duration:   120 * time.Millisecond,
		},
	)

	snapshot := metrics.Snapshot()
	if snapshot.Totals.Requests != 2 {
		t.Fatalf("expected 2 requests, got %d", snapshot.Totals.Requests)
	}
	if snapshot.Totals.Errors != 1 {
		t.Fatalf("expected 1 error request, got %d", snapshot.Totals.Errors)
	}
	if snapshot.ByStatusClass["2xx"] != 1 {
		t.Fatalf("expected one 2xx request")
	}
	if snapshot.ByStatusClass["5xx"] != 1 {
		t.Fatalf("expected one 5xx request")
	}
	routeStats, ok := snapshot.ByRoute["GET /api/v1/engrams"]
	if !ok {
		t.Fatalf("expected route stats for engram list route")
	}
	if routeStats.Count != 2 {
		t.Fatalf("expected route count 2, got %d", routeStats.Count)
	}
	if routeStats.ErrorCount != 1 {
		t.Fatalf("expected route error count 1, got %d", routeStats.ErrorCount)
	}
	if routeStats.AverageDurationMS <= 0 {
		t.Fatalf("expected positive average duration, got %f", routeStats.AverageDurationMS)
	}
}

func TestInMemoryRequestMetricsAggregatesChatObservabilitySamples(t *testing.T) {
	metrics := NewInMemoryRequestMetrics()
	metrics.RecordProviderFailure(
		chat.ProviderFailureSample{
			Provider:  models.ChatProviderOpenAI,
			Operation: "send",
			ErrorCode: "provider_rate_limit",
		},
	)
	metrics.RecordStreamHealth(
		chat.StreamHealthSample{
			Provider:   models.ChatProviderAnthropic,
			Operation:  "stream",
			Outcome:    "completed",
			ChunkCount: 3,
			Duration:   90 * time.Millisecond,
		},
	)
	metrics.RecordStreamHealth(
		chat.StreamHealthSample{
			Provider:   models.ChatProviderAnthropic,
			Operation:  "stream",
			Outcome:    "provider_error",
			ErrorCode:  "provider_rate_limit",
			ChunkCount: 0,
			Duration:   30 * time.Millisecond,
		},
	)
	metrics.RecordLifecycleTrace(
		chat.LifecycleTraceSample{
			TraceID:   "trace-123",
			Operation: "send",
			Stage:     "provider_failure",
			Provider:  models.ChatProviderOpenAI,
			ErrorCode: "provider_rate_limit",
		},
	)

	snapshot := metrics.Snapshot()
	if snapshot.ProviderFails["send openai provider_rate_limit"] != 1 {
		t.Fatalf("expected provider failure count")
	}
	healthy, ok := snapshot.StreamHealth["stream anthropic completed"]
	if !ok {
		t.Fatalf("expected stream health entry for completed stream")
	}
	if healthy.Count != 1 || healthy.TotalChunks != 3 {
		t.Fatalf("unexpected completed stream stats: %+v", healthy)
	}
	failed, ok := snapshot.StreamHealth["stream anthropic provider_error"]
	if !ok {
		t.Fatalf("expected stream health entry for provider_error")
	}
	if failed.ErrorCount != 1 {
		t.Fatalf("expected failed stream error count 1, got %d", failed.ErrorCount)
	}
	if snapshot.Lifecycle["send provider_failure provider_rate_limit"] != 1 {
		t.Fatalf("expected lifecycle trace count")
	}
}

func TestRequestTelemetryMiddlewareRecordsRoutePattern(t *testing.T) {
	metrics := NewInMemoryRequestMetrics()
	router := chi.NewRouter()
	router.Use(newRequestTelemetryMiddleware(nil, metrics))
	router.Get("/api/v1/items/{item_id}", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusCreated)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/items/42", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	snapshot := metrics.Snapshot()
	if snapshot.ByRoute["GET /api/v1/items/{item_id}"].Count != 1 {
		t.Fatalf("expected one recorded route hit")
	}
	if snapshot.ByDomain["api"] != 1 {
		t.Fatalf("expected api domain count to be 1")
	}
	if snapshot.ByStatusClass["2xx"] != 1 {
		t.Fatalf("expected one 2xx status class entry")
	}
}

func TestRequestDomainClassification(t *testing.T) {
	testCases := []struct {
		path         string
		expectedType string
	}{
		{path: "/api/v1/engrams", expectedType: "api"},
		{path: "/api/v1/mcp/stream", expectedType: "mcp"},
		{path: "/oauth/authorize", expectedType: "oauth"},
		{path: "/.well-known/openid-configuration", expectedType: "oauth"},
		{path: "/login/oidc", expectedType: "ui"},
		{path: "/ui/admin", expectedType: "ui"},
		{path: "/random", expectedType: "other"},
	}
	for _, testCase := range testCases {
		if actual := requestDomain(testCase.path); actual != testCase.expectedType {
			t.Fatalf("expected %q for path %q, got %q", testCase.expectedType, testCase.path, actual)
		}
	}
}

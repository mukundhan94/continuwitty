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
	settings := config.Settings{
		AppSemanticVersion:      "1.2.3",
		AppCommitSHA:            "abc1234",
		ChatPromptPolicyVersion: "chat-policy-v9",
		MCPToolPolicyVersion:    "mcp-policy-v4",
		EvalSuiteVersion:        "eval-suite-v2",
	}
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
	expectedFields := map[string]string{
		"semantic_version":           settings.AppSemanticVersion,
		"release":                    "v" + settings.AppSemanticVersion,
		"commit_id":                  settings.AppCommitSHA,
		"chat_prompt_policy_version": settings.ChatPromptPolicyVersion,
		"mcp_tool_policy_version":    settings.MCPToolPolicyVersion,
		"eval_suite_version":         settings.EvalSuiteVersion,
	}
	for key, expectedValue := range expectedFields {
		assertVersionFieldValue(t, payload, key, expectedValue)
	}
}

func TestObservabilityMetricsRouteMountedWithRecorder(t *testing.T) {
	settings := config.Settings{AppSemanticVersion: "1.2.3", AppCommitSHA: "abc1234"}
	metrics := NewInMemoryRequestMetrics()
	router := NewRouterWithDependencies(
		settings,
		RouterDependencies{
			RequestMetrics: metrics,
		},
	)

	healthRequest := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthResponse := httptest.NewRecorder()
	router.ServeHTTP(healthResponse, healthRequest)
	if healthResponse.Code != http.StatusOK {
		t.Fatalf("expected health status 200, got %d", healthResponse.Code)
	}

	metricsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	metricsResponse := httptest.NewRecorder()
	router.ServeHTTP(metricsResponse, metricsRequest)
	if metricsResponse.Code != http.StatusOK {
		t.Fatalf("expected metrics status 200, got %d", metricsResponse.Code)
	}
	var payload RequestMetricsSnapshot
	if err := json.Unmarshal(metricsResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode metrics response: %v", err)
	}
	if payload.Totals.Requests < 1 {
		t.Fatalf("expected at least one recorded request")
	}
	if payload.ByRoute["GET /healthz"].Count < 1 {
		t.Fatalf("expected recorded /healthz route stats")
	}
}

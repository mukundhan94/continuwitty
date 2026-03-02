package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	internalapi "engram/internal/api"
	"engram/internal/chat"
	"engram/internal/config"
	"engram/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func TestBuildChatRouterRegistersRuntimeRoutes(t *testing.T) {
	router := buildChatRouter(config.Settings{EmbeddingDim: 256}, nil, nil)
	routes := collectRegisteredRoutes(t, router)

	requiredRoutes := []string{
		"/api/v1/chat/sessions",
		"/api/v1/chat/sessions/{session_id}",
		"/api/v1/chat/sessions/{session_id}/messages",
		"/api/v1/chat/sessions/{session_id}/messages/stream",
		"/api/v1/chat/sessions/{session_id}/lifecycle-policy",
		"/api/v1/chat/sessions/{session_id}/timeline",
		"/api/v1/chat/sessions/{session_id}/engrams",
		"/api/v1/chat/sessions/{session_id}/documents",
		"/api/v1/chat/sessions/{session_id}/save-engram",
		"/api/v1/chat/sessions/{session_id}/continue",
	}
	for _, route := range requiredRoutes {
		if _, ok := routes[route]; !ok {
			t.Fatalf("expected route %q to be registered", route)
		}
	}
}

func TestResolveChatProviderDependencyResolvesConfiguredProvider(t *testing.T) {
	resolver := resolveChatProviderDependency(config.Settings{})

	adapter, err := resolver(models.ChatProviderOpenAI)
	if err != nil {
		t.Fatalf("resolve provider: %v", err)
	}
	if adapter.Provider() != models.ChatProviderOpenAI {
		t.Fatalf("expected provider %q, got %q", models.ChatProviderOpenAI, adapter.Provider())
	}
}

func TestResolveChatProviderDependencyRejectsUnknownProvider(t *testing.T) {
	resolver := resolveChatProviderDependency(config.Settings{})

	_, err := resolver(models.ChatProvider("unsupported"))
	if err == nil {
		t.Fatalf("expected unknown provider error")
	}
}

func TestChatActorResolverFromContextReturnsActorPayload(t *testing.T) {
	actorUserID := uuid.MustParse("00000000-0000-0000-0000-000000000901")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions", nil)
	request = internalapi.WithAdminActor(
		request,
		internalapi.AdminActor{
			UserID: actorUserID,
			Role:   "admin",
		},
	)

	actorPayload, err := chatActorResolverFromContext(request)
	if err != nil {
		t.Fatalf("resolve actor payload: %v", err)
	}
	if actorPayload["user_id"] != actorUserID.String() {
		t.Fatalf("expected user_id %q, got %#v", actorUserID.String(), actorPayload["user_id"])
	}
	if actorPayload["role"] != "admin" {
		t.Fatalf("expected role admin, got %#v", actorPayload["role"])
	}
}

func TestChatActorResolverFromContextRejectsMissingActor(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions", nil)

	_, err := chatActorResolverFromContext(request)
	if err == nil {
		t.Fatalf("expected missing actor error")
	}
}

func TestMapRuntimeMessageMetadataConvertsTokenUsageAndCopiesSlices(t *testing.T) {
	provider := "openai"
	modelID := "gpt-4o-mini"
	usedEngramID := uuid.MustParse("00000000-0000-0000-0000-000000000902")
	metadata := &chat.RuntimeMessageMetadata{
		Provider:      &provider,
		ModelID:       &modelID,
		TokenUsage:    map[string]int{"input_tokens": 12},
		UsedEngramIDs: []uuid.UUID{usedEngramID},
	}

	mapped := mapRuntimeMessageMetadata(metadata)
	if mapped == nil {
		t.Fatalf("expected mapped metadata")
	}
	if mapped.TokenUsageJSON["input_tokens"] != 12 {
		t.Fatalf("expected input_tokens=12, got %#v", mapped.TokenUsageJSON["input_tokens"])
	}
	if len(mapped.UsedEngramIDs) != 1 || mapped.UsedEngramIDs[0] != usedEngramID {
		t.Fatalf("expected copied engram ids, got %#v", mapped.UsedEngramIDs)
	}

	metadata.TokenUsage["input_tokens"] = 99
	metadata.UsedEngramIDs[0] = uuid.Nil
	if mapped.TokenUsageJSON["input_tokens"] != 12 {
		t.Fatalf("expected mapped token usage to remain 12, got %#v", mapped.TokenUsageJSON["input_tokens"])
	}
	if mapped.UsedEngramIDs[0] != usedEngramID {
		t.Fatalf("expected mapped engram id to remain %s, got %s", usedEngramID, mapped.UsedEngramIDs[0])
	}
}

func TestApplyGraphNoiseSuppressionPolicyDefaultsSetsUnsetValues(t *testing.T) {
	defaulted := applyGraphNoiseSuppressionPolicyDefaults(
		chat.ChatContextRequest{},
		config.Settings{
			GraphLinkNoiseSuppressionEnabled: false,
			GraphLinkNoiseScoreThreshold:     0.73,
		},
	)
	requireOptionalBoolRuntime(t, defaulted.LinkNoiseSuppressionEnabled, false, "link_noise_suppression_enabled")
	requireOptionalFloatRuntime(t, defaulted.LinkNoiseScoreThreshold, 0.73, "link_noise_score_threshold")
}

func TestApplyGraphNoiseSuppressionPolicyDefaultsPreservesOverrides(t *testing.T) {
	overrideEnabled := true
	overrideThreshold := 0.22
	overridden := applyGraphNoiseSuppressionPolicyDefaults(
		chat.ChatContextRequest{
			LinkNoiseSuppressionEnabled: &overrideEnabled,
			LinkNoiseScoreThreshold:     &overrideThreshold,
		},
		config.Settings{
			GraphLinkNoiseSuppressionEnabled: false,
			GraphLinkNoiseScoreThreshold:     0.73,
		},
	)
	requireOptionalBoolRuntime(t, overridden.LinkNoiseSuppressionEnabled, true, "link_noise_suppression_enabled")
	requireOptionalFloatRuntime(t, overridden.LinkNoiseScoreThreshold, 0.22, "link_noise_score_threshold")
}

func TestBuildProviderFallbackStrategyDisablesCrossProviderFallbackForBedrock(t *testing.T) {
	strategy := buildProviderFallbackStrategy(
		config.Settings{
			DefaultChatProvider: string(models.ChatProviderOpenAI),
			DefaultChatModel:    "gpt-4o-mini",
		},
	)

	bedrockCandidates := strategy.Candidates(
		models.ChatSessionRecord{
			Provider: models.ChatProviderBedrock,
			ModelID:  "eu.anthropic.claude-haiku-4-5-20251001-v1:0",
		},
	)
	if len(bedrockCandidates) != 1 {
		t.Fatalf("expected bedrock candidates to remain primary-only, got %d", len(bedrockCandidates))
	}
	if bedrockCandidates[0].Provider != models.ChatProviderBedrock {
		t.Fatalf("expected bedrock primary candidate, got %q", bedrockCandidates[0].Provider)
	}

	openAICandidates := strategy.Candidates(
		models.ChatSessionRecord{
			Provider: models.ChatProviderOpenAI,
			ModelID:  "gpt-4o-mini",
		},
	)
	if len(openAICandidates) < 2 {
		t.Fatalf("expected openai candidates to include fallback providers, got %d", len(openAICandidates))
	}
}

func collectRegisteredRoutes(t *testing.T, router chi.Router) map[string]struct{} {
	t.Helper()
	paths := map[string]struct{}{}
	if err := chi.Walk(
		router,
		func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
			_ = method
			paths[route] = struct{}{}
			return nil
		},
	); err != nil {
		t.Fatalf("walk routes: %v", err)
	}
	return paths
}

func requireOptionalBoolRuntime(t *testing.T, value *bool, expected bool, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%v, got %v", field, expected, value)
	}
}

func requireOptionalFloatRuntime(t *testing.T, value *float64, expected float64, field string) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("expected %s=%v, got %v", field, expected, value)
	}
}

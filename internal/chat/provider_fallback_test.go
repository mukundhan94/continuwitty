package chat

import (
	"testing"

	"engram/internal/models"
)

func TestNewStaticProviderFallbackStrategyDisabledReturnsPrimaryOnly(t *testing.T) {
	strategy := NewStaticProviderFallbackStrategy(ProviderFallbackStrategyOptions{Enabled: false})
	session := models.ChatSessionRecord{
		Provider: models.ChatProviderOpenAI,
		ModelID:  "gpt-4o-mini",
	}

	candidates := strategy.Candidates(session)
	if len(candidates) != 1 {
		t.Fatalf("expected one candidate, got %d", len(candidates))
	}
	requireEqualAnyRuntime(t, models.ChatProviderOpenAI, candidates[0].Provider)
	requireEqualAnyRuntime(t, "gpt-4o-mini", candidates[0].ModelID)
}

func TestStaticProviderFallbackStrategyAddsOrderedUniqueProviders(t *testing.T) {
	strategy := NewStaticProviderFallbackStrategy(
		ProviderFallbackStrategyOptions{
			Enabled:                true,
			FallbackOrder:          []models.ChatProvider{models.ChatProviderOpenAI, models.ChatProviderAnthropic, models.ChatProviderBedrock, models.ChatProviderAnthropic},
			DefaultFallbackModelID: "fallback-default",
			FallbackModelByProvider: map[models.ChatProvider]string{
				models.ChatProviderAnthropic: "claude-3-5-haiku-latest",
			},
		},
	)
	session := models.ChatSessionRecord{
		Provider: models.ChatProviderOpenAI,
		ModelID:  "gpt-4o-mini",
	}

	candidates := strategy.Candidates(session)
	if len(candidates) != 3 {
		t.Fatalf("expected 3 unique candidates, got %d", len(candidates))
	}
	requireEqualAnyRuntime(t, models.ChatProviderOpenAI, candidates[0].Provider)
	requireEqualAnyRuntime(t, "gpt-4o-mini", candidates[0].ModelID)
	requireEqualAnyRuntime(t, models.ChatProviderAnthropic, candidates[1].Provider)
	requireEqualAnyRuntime(t, "claude-3-5-haiku-latest", candidates[1].ModelID)
	requireEqualAnyRuntime(t, models.ChatProviderBedrock, candidates[2].Provider)
	requireEqualAnyRuntime(t, "fallback-default", candidates[2].ModelID)
}

func TestStaticProviderFallbackStrategyUsesPrimaryModelWhenNoFallbackModelConfigured(t *testing.T) {
	strategy := NewStaticProviderFallbackStrategy(
		ProviderFallbackStrategyOptions{
			Enabled:       true,
			FallbackOrder: []models.ChatProvider{models.ChatProviderAnthropic},
		},
	)
	session := models.ChatSessionRecord{
		Provider: models.ChatProviderOpenAI,
		ModelID:  "gpt-4o-mini",
	}

	candidates := strategy.Candidates(session)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	requireEqualAnyRuntime(t, "gpt-4o-mini", candidates[1].ModelID)
}

func TestStaticProviderFallbackStrategyCanDisableCrossProviderForBedrock(t *testing.T) {
	strategy := NewStaticProviderFallbackStrategy(
		ProviderFallbackStrategyOptions{
			Enabled:                 true,
			FallbackOrder:           []models.ChatProvider{models.ChatProviderOpenAI, models.ChatProviderAnthropic},
			DefaultFallbackModelID:  "fallback-default",
			DisableCrossProviderFor: []models.ChatProvider{models.ChatProviderBedrock},
		},
	)
	session := models.ChatSessionRecord{
		Provider: models.ChatProviderBedrock,
		ModelID:  "eu.anthropic.claude-haiku-4-5-20251001-v1:0",
	}

	candidates := strategy.Candidates(session)
	if len(candidates) != 1 {
		t.Fatalf("expected bedrock session to keep primary-only fallback, got %d", len(candidates))
	}
	requireEqualAnyRuntime(t, models.ChatProviderBedrock, candidates[0].Provider)
	requireEqualAnyRuntime(t, "eu.anthropic.claude-haiku-4-5-20251001-v1:0", candidates[0].ModelID)
}

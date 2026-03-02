package providers

import (
	"testing"

	"engram/internal/config"
	"engram/internal/models"
)

func TestBuildProviderRegistryCreatesAllAdapters(t *testing.T) {
	settings := providerRegistryTestSettings()
	registry := BuildProviderRegistry(settings)
	assertRegistrySize(t, registry, 3)
	assertRegistryHasOpenAIProvider(t, registry)
	assertRegistryHasAnthropicProvider(t, registry)
	assertBedrockCredentialsMatchSettings(t, registry, settings)
}

func assertRegistrySize(t *testing.T, registry map[models.ChatProvider]ChatProviderAdapter, expected int) {
	t.Helper()
	if len(registry) != expected {
		t.Fatalf("expected %d providers, got %d", expected, len(registry))
	}
}

func assertRegistryHasOpenAIProvider(t *testing.T, registry map[models.ChatProvider]ChatProviderAdapter) {
	t.Helper()
	if _, ok := registry[models.ChatProviderOpenAI].(*OpenAIProvider); !ok {
		t.Fatalf("expected openai adapter")
	}
}

func assertRegistryHasAnthropicProvider(t *testing.T, registry map[models.ChatProvider]ChatProviderAdapter) {
	t.Helper()
	if _, ok := registry[models.ChatProviderAnthropic].(*AnthropicProvider); !ok {
		t.Fatalf("expected anthropic adapter")
	}
}

func assertBedrockCredentialsMatchSettings(
	t *testing.T,
	registry map[models.ChatProvider]ChatProviderAdapter,
	settings config.Settings,
) {
	t.Helper()
	bedrock, ok := registry[models.ChatProviderBedrock].(*BedrockProvider)
	if !ok {
		t.Fatalf("expected bedrock adapter")
	}
	if bedrock.credentials.RegionName != settings.AWSRegion {
		t.Fatalf("expected region %q, got %q", settings.AWSRegion, bedrock.credentials.RegionName)
	}
	if bedrock.credentials.AccessKeyID != settings.AWSAccessKeyID {
		t.Fatalf("expected access key")
	}
	if bedrock.credentials.SecretAccessKey != settings.AWSSecretAccessKey {
		t.Fatalf("expected secret key")
	}
	if bedrock.credentials.SessionToken != settings.AWSSessionToken {
		t.Fatalf("expected session token")
	}
}

func TestGetProviderAdapterReturnsRequestedProvider(t *testing.T) {
	settings := providerRegistryTestSettings()

	adapter, err := GetProviderAdapter(models.ChatProviderOpenAI, settings)
	requireNoError(t, err)
	assertAdapterIsOpenAIProvider(t, adapter)
}

func assertAdapterIsOpenAIProvider(t *testing.T, adapter ChatProviderAdapter) {
	t.Helper()
	if _, ok := adapter.(*OpenAIProvider); !ok {
		t.Fatalf("expected OpenAI provider adapter")
	}
}

func providerRegistryTestSettings() config.Settings {
	return config.Settings{
		OpenAIAPIKey:       "openai-key",
		OpenAIBaseURL:      "https://api.openai.com",
		AnthropicAPIKey:    "anthropic-key",
		AnthropicBaseURL:   "https://api.anthropic.com",
		AnthropicVersion:   "2023-06-01",
		AWSRegion:          "us-east-1",
		AWSAccessKeyID:     "access-key",
		AWSSecretAccessKey: "secret-key",
		AWSSessionToken:    "session-token",
	}
}

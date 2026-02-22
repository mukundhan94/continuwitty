package providers

import (
	"testing"

	"engram/internal/config"
	"engram/internal/models"
)

func TestBuildProviderRegistryCreatesAllAdapters(t *testing.T) {
	settings := config.Settings{
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

	registry := BuildProviderRegistry(settings)

	if len(registry) != 3 {
		t.Fatalf("expected 3 providers, got %d", len(registry))
	}
	if _, ok := registry[models.ChatProviderOpenAI].(*OpenAIProvider); !ok {
		t.Fatalf("expected openai adapter")
	}
	if _, ok := registry[models.ChatProviderAnthropic].(*AnthropicProvider); !ok {
		t.Fatalf("expected anthropic adapter")
	}
	bedrock, ok := registry[models.ChatProviderBedrock].(*BedrockProvider)
	if !ok {
		t.Fatalf("expected bedrock adapter")
	}
	if bedrock.credentials.RegionName != "us-east-1" {
		t.Fatalf("expected region us-east-1, got %q", bedrock.credentials.RegionName)
	}
	if bedrock.credentials.AccessKeyID != "access-key" {
		t.Fatalf("expected access key")
	}
	if bedrock.credentials.SecretAccessKey != "secret-key" {
		t.Fatalf("expected secret key")
	}
	if bedrock.credentials.SessionToken != "session-token" {
		t.Fatalf("expected session token")
	}
}

func TestGetProviderAdapterReturnsRequestedProvider(t *testing.T) {
	settings := config.Settings{
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

	adapter, err := GetProviderAdapter(models.ChatProviderOpenAI, settings)
	requireNoError(t, err)

	if _, ok := adapter.(*OpenAIProvider); !ok {
		t.Fatalf("expected OpenAI provider adapter")
	}
}

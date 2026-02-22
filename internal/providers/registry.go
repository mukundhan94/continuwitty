package providers

import (
	"fmt"

	"engram/internal/config"
	"engram/internal/models"
)

// BuildProviderRegistry constructs provider adapters from app settings.
func BuildProviderRegistry(settings config.Settings) map[models.ChatProvider]ChatProviderAdapter {
	return map[models.ChatProvider]ChatProviderAdapter{
		models.ChatProviderOpenAI: NewOpenAIProvider(
			settings.OpenAIAPIKey,
			settings.OpenAIBaseURL,
			nil,
		),
		models.ChatProviderAnthropic: NewAnthropicProvider(
			settings.AnthropicAPIKey,
			settings.AnthropicBaseURL,
			settings.AnthropicVersion,
			nil,
		),
		models.ChatProviderBedrock: NewBedrockProvider(
			AWSRuntimeCredentials{
				RegionName:      settings.AWSRegion,
				AccessKeyID:     settings.AWSAccessKeyID,
				SecretAccessKey: settings.AWSSecretAccessKey,
				SessionToken:    settings.AWSSessionToken,
			},
			nil,
		),
	}
}

// GetProviderAdapter resolves an adapter by provider enum.
func GetProviderAdapter(provider models.ChatProvider, settings config.Settings) (ChatProviderAdapter, error) {
	registry := BuildProviderRegistry(settings)
	adapter, ok := registry[provider]
	if !ok {
		return nil, fmt.Errorf("provider adapter not available: %s", provider)
	}
	return adapter, nil
}

package embeddings

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	defaultEmbeddingProvider = "local"
	defaultOpenAIModel       = "text-embedding-3-small"
	defaultOpenAIBaseURL     = "https://api.openai.com"
)

// RuntimeConfig captures embedding runtime provider configuration.
type RuntimeConfig struct {
	Provider        string
	Model           string
	FallbackToLocal bool
	OpenAIAPIKey    string
	OpenAIBaseURL   string
	Timeout         time.Duration
}

var (
	defaultServiceMu sync.RWMutex
	defaultService   = NewService(LocalDeterministicEmbeddingProvider{}, nil)
)

// ConfigureDefault configures the process-wide default embedding service.
func ConfigureDefault(config RuntimeConfig) error {
	service, err := BuildService(config)
	if err != nil {
		return err
	}
	SetDefaultService(service)
	return nil
}

// BuildService creates an embedding service from runtime config.
func BuildService(config RuntimeConfig) (*Service, error) {
	providerName := normalizedProviderName(config.Provider)
	switch providerName {
	case "local":
		return NewService(LocalDeterministicEmbeddingProvider{}, nil), nil
	case "openai":
		primary, err := buildOpenAIProvider(config)
		if err != nil {
			return nil, err
		}
		fallback := resolveFallbackProvider(config.FallbackToLocal)
		return NewService(primary, fallback), nil
	default:
		return nil, fmt.Errorf("unsupported embedding provider %q", config.Provider)
	}
}

func normalizedProviderName(provider string) string {
	trimmed := strings.ToLower(strings.TrimSpace(provider))
	if trimmed == "" {
		return defaultEmbeddingProvider
	}
	return trimmed
}

func resolveFallbackProvider(enabled bool) Provider {
	if !enabled {
		return nil
	}
	return LocalDeterministicEmbeddingProvider{}
}

func buildOpenAIProvider(config RuntimeConfig) (Provider, error) {
	if strings.TrimSpace(config.OpenAIAPIKey) == "" {
		return nil, errors.New("embedding provider openai requires OPENAI_API_KEY")
	}
	provider, err := NewOpenAIEmbeddingProvider(
		OpenAIEmbeddingProviderInput{
			APIKey:     config.OpenAIAPIKey,
			BaseURL:    withDefault(config.OpenAIBaseURL, defaultOpenAIBaseURL),
			Model:      withDefault(config.Model, defaultOpenAIModel),
			Timeout:    resolveTimeout(config.Timeout),
			ProviderID: "",
		},
	)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func withDefault(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func resolveTimeout(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}
	return 20 * time.Second
}

// SetDefaultService swaps process-wide embedding service implementation.
func SetDefaultService(service *Service) {
	if service == nil {
		service = NewService(LocalDeterministicEmbeddingProvider{}, nil)
	}
	defaultServiceMu.Lock()
	defer defaultServiceMu.Unlock()
	defaultService = service
}

// DefaultService returns the current process-wide embedding service.
func DefaultService() *Service {
	defaultServiceMu.RLock()
	defer defaultServiceMu.RUnlock()
	return defaultService
}

// EmbedText embeds one text using process-wide runtime provider configuration.
func EmbedText(text string, dim int) (Result, error) {
	return DefaultService().Embed(text, dim)
}

// EmbedTexts embeds multiple texts using process-wide runtime provider configuration.
func EmbedTexts(texts []string, dim int) ([]Result, error) {
	return DefaultService().EmbedMany(texts, dim)
}

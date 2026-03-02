package embeddings

import (
	"testing"
	"time"
)

func TestConfigureDefaultLocalProvider(t *testing.T) {
	SetDefaultService(NewService(LocalDeterministicEmbeddingProvider{}, nil))
	err := ConfigureDefault(
		RuntimeConfig{
			Provider:        "local",
			FallbackToLocal: true,
			Timeout:         20 * time.Second,
		},
	)
	if err != nil {
		t.Fatalf("expected local provider config to succeed: %v", err)
	}
	result, embedErr := EmbedText("configured-local", 8)
	if embedErr != nil {
		t.Fatalf("expected embed with local provider to succeed: %v", embedErr)
	}
	if result.ProviderID != localProviderID {
		t.Fatalf("expected local provider id %q, got %q", localProviderID, result.ProviderID)
	}
}

func TestBuildServiceRejectsUnsupportedProvider(t *testing.T) {
	_, err := BuildService(RuntimeConfig{Provider: "unsupported"})
	if err == nil {
		t.Fatalf("expected unsupported provider to fail")
	}
}

func TestBuildServiceOpenAIRequiresAPIKey(t *testing.T) {
	_, err := BuildService(
		RuntimeConfig{
			Provider:      "openai",
			Model:         "text-embedding-3-small",
			OpenAIBaseURL: "https://api.openai.com",
		},
	)
	if err == nil {
		t.Fatalf("expected openai provider without api key to fail")
	}
}

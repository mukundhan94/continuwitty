package embeddings

import "testing"

type failingProvider struct{}

func (failingProvider) ProviderID() string {
	return "failing"
}

func (failingProvider) Embed(_ string, _ int) ([]float64, error) {
	return nil, &ProviderError{Message: "provider unavailable"}
}

func (failingProvider) EmbedMany(_ []string, _ int) ([][]float64, error) {
	return nil, &ProviderError{Message: "provider unavailable"}
}

func TestEmbeddingServiceFallsBackToLocalProvider(t *testing.T) {
	service := NewService(failingProvider{}, LocalDeterministicEmbeddingProvider{})

	result, err := service.Embed("memory continuity", 12)
	if err != nil {
		t.Fatalf("expected embed with fallback to succeed: %v", err)
	}

	if len(result.Vector) != 12 {
		t.Fatalf("expected vector dimension 12, got %d", len(result.Vector))
	}
	if result.ProviderID != localProviderID {
		t.Fatalf("expected provider id %q, got %q", localProviderID, result.ProviderID)
	}
}

func TestEmbeddingServiceEmbedManyUsesProviderIDFromActiveProvider(t *testing.T) {
	service := NewService(LocalDeterministicEmbeddingProvider{}, nil)

	results, err := service.EmbedMany([]string{"alpha", "beta"}, 10)
	if err != nil {
		t.Fatalf("expected embed_many to succeed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected two embedding results, got %d", len(results))
	}

	for _, result := range results {
		if result.ProviderID != localProviderID {
			t.Fatalf("expected provider id %q, got %q", localProviderID, result.ProviderID)
		}
		if len(result.Vector) != 10 {
			t.Fatalf("expected vector dimension 10, got %d", len(result.Vector))
		}
	}
}

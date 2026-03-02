package embeddings

import (
	"crypto/sha256"
	"fmt"
)

const localProviderID = "local-deterministic-v1"

// LocalDeterministicEmbeddingProvider provides offline deterministic embeddings.
type LocalDeterministicEmbeddingProvider struct{}

func (LocalDeterministicEmbeddingProvider) ProviderID() string {
	return localProviderID
}

func (LocalDeterministicEmbeddingProvider) Embed(text string, dim int) ([]float64, error) {
	return EmbedTextLocal(text, dim)
}

func (provider LocalDeterministicEmbeddingProvider) EmbedMany(texts []string, dim int) ([][]float64, error) {
	results := make([][]float64, 0, len(texts))
	for _, text := range texts {
		vector, err := provider.Embed(text, dim)
		if err != nil {
			return nil, err
		}
		results = append(results, vector)
	}
	return results, nil
}

// EmbedTextLocal mirrors Python's deterministic local embedding algorithm.
func EmbedTextLocal(text string, dim int) ([]float64, error) {
	if dim <= 0 {
		return nil, fmt.Errorf("dim must be > 0")
	}
	if text == "" {
		text = " "
	}

	output := make([]float64, 0, dim)
	counter := 0
	for len(output) < dim {
		digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", text, counter)))
		for _, byteValue := range digest {
			value := (float64(byteValue) / 127.5) - 1.0
			output = append(output, value)
			if len(output) == dim {
				break
			}
		}
		counter += 1
	}
	return output, nil
}

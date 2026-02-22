package embeddings

import "errors"

// Result is the embedding output envelope used by callers.
type Result struct {
	Vector     []float64
	ProviderID string
}

// Provider describes a vector embedding backend.
type Provider interface {
	ProviderID() string
	Embed(text string, dim int) ([]float64, error)
	EmbedMany(texts []string, dim int) ([][]float64, error)
}

// Service embeds text with primary provider and optional fallback.
type Service struct {
	primary  Provider
	fallback Provider
}

func NewService(primary Provider, fallback Provider) *Service {
	return &Service{primary: primary, fallback: fallback}
}

func (service *Service) Embed(text string, dim int) (Result, error) {
	cleaned := cleanedText(text)
	vector, providerID, err := service.embedWithFallback(cleaned, dim)
	if err != nil {
		return Result{}, err
	}
	return Result{Vector: vector, ProviderID: providerID}, nil
}

func (service *Service) EmbedMany(texts []string, dim int) ([]Result, error) {
	cleaned := make([]string, 0, len(texts))
	for _, text := range texts {
		cleaned = append(cleaned, cleanedText(text))
	}

	vectors, providerID, err := service.embedManyWithFallback(cleaned, dim)
	if err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(vectors))
	for _, vector := range vectors {
		results = append(results, Result{Vector: vector, ProviderID: providerID})
	}
	return results, nil
}

func (service *Service) embedWithFallback(text string, dim int) ([]float64, string, error) {
	vector, err := service.primary.Embed(text, dim)
	if err == nil {
		return vector, service.primary.ProviderID(), nil
	}
	if !isProviderError(err) || service.fallback == nil {
		return nil, "", err
	}
	fallbackVector, fallbackErr := service.fallback.Embed(text, dim)
	if fallbackErr != nil {
		return nil, "", fallbackErr
	}
	return fallbackVector, service.fallback.ProviderID(), nil
}

func (service *Service) embedManyWithFallback(texts []string, dim int) ([][]float64, string, error) {
	vectors, err := service.primary.EmbedMany(texts, dim)
	if err == nil {
		return vectors, service.primary.ProviderID(), nil
	}
	if !isProviderError(err) || service.fallback == nil {
		return nil, "", err
	}
	fallbackVectors, fallbackErr := service.fallback.EmbedMany(texts, dim)
	if fallbackErr != nil {
		return nil, "", fallbackErr
	}
	return fallbackVectors, service.fallback.ProviderID(), nil
}

func cleanedText(text string) string {
	if text == "" {
		return " "
	}
	return text
}

func isProviderError(err error) bool {
	var providerErr *ProviderError
	return errors.As(err, &providerErr)
}

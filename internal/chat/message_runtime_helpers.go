package chat

import (
	"errors"
	"math"

	"engram/internal/providers"
)

// ResolveTokenUsage returns provider token usage or deterministic estimated usage when missing.
func ResolveTokenUsage(tokenUsage map[string]int, inputChars int, outputChars int) (map[string]int, bool) {
	resolved := cloneTokenUsage(tokenUsage)
	if resolved["total_tokens"] > 0 {
		return resolved, false
	}
	return estimateTokenUsage(inputChars, outputChars), true
}

// MapProviderError maps provider adapter errors into chat provider execution errors.
func MapProviderError(err error) error {
	if err == nil {
		return nil
	}
	for _, mapping := range providerErrorStatusMappings() {
		mapped, ok := mapping.tryMap(err)
		if ok {
			return mapped
		}
	}
	return err
}

type providerErrorStatusMapping interface {
	tryMap(err error) (error, bool)
}

type providerErrorMapping[T error] struct {
	statusCode int
	errorCode  func(target T) string
}

func (mapping providerErrorMapping[T]) tryMap(err error) (error, bool) {
	var target T
	if !errors.As(err, &target) {
		return nil, false
	}
	return NewChatProviderExecutionError(
		err.Error(),
		mapping.statusCode,
		mapping.errorCode(target),
	), true
}

func providerErrorStatusMappings() []providerErrorStatusMapping {
	return []providerErrorStatusMapping{
		providerErrorMapping[*providers.ProviderRateLimitError]{
			statusCode: 429,
			errorCode:  func(target *providers.ProviderRateLimitError) string { return target.Code },
		},
		providerErrorMapping[*providers.ProviderAuthError]{
			statusCode: 503,
			errorCode:  func(target *providers.ProviderAuthError) string { return target.Code },
		},
		providerErrorMapping[*providers.ProviderRequestError]{
			statusCode: 400,
			errorCode:  func(target *providers.ProviderRequestError) string { return target.Code },
		},
		providerErrorMapping[*providers.ProviderAPIError]{
			statusCode: 502,
			errorCode:  func(target *providers.ProviderAPIError) string { return target.Code },
		},
		providerErrorMapping[*providers.ProviderError]{
			statusCode: 502,
			errorCode:  func(target *providers.ProviderError) string { return target.Code },
		},
	}
}

func estimateTokenUsage(inputChars int, outputChars int) map[string]int {
	inputTokens := estimatedTokenCount(inputChars)
	outputTokens := estimatedTokenCount(outputChars)
	return map[string]int{
		"input_tokens":  inputTokens,
		"output_tokens": outputTokens,
		"total_tokens":  inputTokens + outputTokens,
	}
}

func estimatedTokenCount(chars int) int {
	return max(1, int(math.Round(float64(chars)/4.0)))
}

func cloneTokenUsage(tokenUsage map[string]int) map[string]int {
	if tokenUsage == nil {
		return map[string]int{}
	}
	cloned := make(map[string]int, len(tokenUsage))
	for key, value := range tokenUsage {
		cloned[key] = value
	}
	return cloned
}

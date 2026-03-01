package chat

import (
	"strings"

	"engram/internal/models"
)

// ProviderFallbackCandidate describes one provider/model route attempt.
type ProviderFallbackCandidate struct {
	Provider models.ChatProvider
	ModelID  string
}

// ProviderFallbackStrategy resolves ordered provider/model candidates per session.
type ProviderFallbackStrategy interface {
	Candidates(session models.ChatSessionRecord) []ProviderFallbackCandidate
}

type noopProviderFallbackStrategy struct{}

func (noopProviderFallbackStrategy) Candidates(session models.ChatSessionRecord) []ProviderFallbackCandidate {
	return []ProviderFallbackCandidate{
		{
			Provider: session.Provider,
			ModelID:  strings.TrimSpace(session.ModelID),
		},
	}
}

// ProviderFallbackStrategyOptions controls fallback candidate resolution.
type ProviderFallbackStrategyOptions struct {
	Enabled                 bool
	FallbackOrder           []models.ChatProvider
	DefaultFallbackModelID  string
	FallbackModelByProvider map[models.ChatProvider]string
	DisableCrossProviderFor []models.ChatProvider
}

type staticProviderFallbackStrategy struct {
	fallbackOrder           []models.ChatProvider
	defaultFallbackModelID  string
	fallbackModelByProvider map[models.ChatProvider]string
	disableCrossProviderFor map[models.ChatProvider]struct{}
}

// NewStaticProviderFallbackStrategy builds an ordered fallback strategy.
func NewStaticProviderFallbackStrategy(options ProviderFallbackStrategyOptions) ProviderFallbackStrategy {
	if !options.Enabled {
		return noopProviderFallbackStrategy{}
	}
	return &staticProviderFallbackStrategy{
		fallbackOrder:           append([]models.ChatProvider(nil), options.FallbackOrder...),
		defaultFallbackModelID:  strings.TrimSpace(options.DefaultFallbackModelID),
		fallbackModelByProvider: cloneFallbackModelMap(options.FallbackModelByProvider),
		disableCrossProviderFor: mapDisableCrossProvider(options.DisableCrossProviderFor),
	}
}

func (strategy *staticProviderFallbackStrategy) Candidates(
	session models.ChatSessionRecord,
) []ProviderFallbackCandidate {
	primaryProvider := session.Provider
	primaryModelID := strings.TrimSpace(session.ModelID)
	candidates := []ProviderFallbackCandidate{
		{
			Provider: primaryProvider,
			ModelID:  primaryModelID,
		},
	}
	if strategy != nil && strategy.crossProviderDisabled(primaryProvider) {
		return candidates
	}
	seen := map[models.ChatProvider]struct{}{
		primaryProvider: {},
	}
	for _, provider := range strategy.fallbackOrder {
		if _, exists := seen[provider]; exists {
			continue
		}
		seen[provider] = struct{}{}
		candidates = append(
			candidates,
			ProviderFallbackCandidate{
				Provider: provider,
				ModelID:  strategy.resolveFallbackModelID(provider, primaryModelID),
			},
		)
	}
	return candidates
}

func (strategy *staticProviderFallbackStrategy) crossProviderDisabled(provider models.ChatProvider) bool {
	if strategy == nil || len(strategy.disableCrossProviderFor) == 0 {
		return false
	}
	_, disabled := strategy.disableCrossProviderFor[provider]
	return disabled
}

func (strategy *staticProviderFallbackStrategy) resolveFallbackModelID(
	provider models.ChatProvider,
	primaryModelID string,
) string {
	if strategy == nil {
		return primaryModelID
	}
	if modelID, ok := strategy.fallbackModelByProvider[provider]; ok {
		trimmed := strings.TrimSpace(modelID)
		if trimmed != "" {
			return trimmed
		}
	}
	if strategy.defaultFallbackModelID != "" {
		return strategy.defaultFallbackModelID
	}
	return primaryModelID
}

func cloneFallbackModelMap(
	source map[models.ChatProvider]string,
) map[models.ChatProvider]string {
	if source == nil {
		return map[models.ChatProvider]string{}
	}
	cloned := make(map[models.ChatProvider]string, len(source))
	for provider, modelID := range source {
		cloned[provider] = modelID
	}
	return cloned
}

func mapDisableCrossProvider(providers []models.ChatProvider) map[models.ChatProvider]struct{} {
	if len(providers) == 0 {
		return map[models.ChatProvider]struct{}{}
	}
	mapped := make(map[models.ChatProvider]struct{}, len(providers))
	for _, provider := range providers {
		mapped[provider] = struct{}{}
	}
	return mapped
}

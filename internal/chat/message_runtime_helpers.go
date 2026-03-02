package chat

import (
	"errors"
	"math"
	"strings"

	"engram/internal/models"
	"engram/internal/providers"
)

const defaultChatHistoryLimit = 40

// HistoryAsProviderMessages normalizes persisted chat messages into provider messages.
func HistoryAsProviderMessages(
	messages []models.ChatMessageRecord,
	historyLimit int,
) []providers.ProviderMessage {
	normalized := make([]providers.ProviderMessage, 0, len(messages))
	for _, message := range messages {
		if !isProviderHistoryRole(message.Role) {
			continue
		}
		normalized = append(normalized, providers.ProviderMessage{
			Role:    message.Role,
			Content: message.ContentText,
		})
	}
	limit := normalizeHistoryLimit(historyLimit)
	if len(normalized) <= limit {
		return normalized
	}
	return normalized[len(normalized)-limit:]
}

// BuildSystemPrompt combines base prompt and retrieval context for provider requests.
func BuildSystemPrompt(basePrompt string, contextMarkdown string) string {
	sections := make([]string, 0, 2)
	trimmedBasePrompt := strings.TrimSpace(basePrompt)
	if trimmedBasePrompt != "" {
		sections = append(sections, trimmedBasePrompt)
	}
	if contextMarkdown != "" {
		sections = append(
			sections,
			"Use the retrieved engram context below when relevant. "+
				"If you cite evidence, prefer source URLs from the context.\n\n"+contextMarkdown,
		)
	}
	return strings.Join(sections, "\n\n")
}

// PreviewText normalizes whitespace and truncates to a stable preview size.
func PreviewText(value string, maxChars int) string {
	return truncatePreviewText(normalizePreviewSpaces(value), maxChars)
}

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

func isProviderHistoryRole(role string) bool {
	return role == "user" || role == "assistant"
}

func normalizeHistoryLimit(historyLimit int) int {
	if historyLimit <= 0 {
		return defaultChatHistoryLimit
	}
	return historyLimit
}

func normalizePreviewSpaces(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func truncatePreviewText(value string, maxChars int) string {
	if maxChars <= 0 {
		return value
	}
	if len(value) <= maxChars {
		return value
	}
	if maxChars <= 3 {
		return value[:maxChars]
	}
	return strings.TrimSpace(value[:maxChars-3]) + "..."
}

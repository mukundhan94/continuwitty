package chat

import "strings"

const (
	defaultChatContextTokenBudget = 1200
	minChatContextTokenBudget     = 200
	maxChatContextTokenBudget     = 8000
	estimatedCharsPerToken        = 4
)

type budgetedContextResult struct {
	markdown      string
	tokenEstimate int
	truncated     bool
}

func normalizeContextTokenBudgetOption(request ChatContextRequest) ChatContextRequest {
	if request.ContextTokenBudget == nil {
		request.ContextTokenBudget = intPointer(defaultChatContextTokenBudget)
	}
	bounded := clamp(*request.ContextTokenBudget, minChatContextTokenBudget, maxChatContextTokenBudget)
	request.ContextTokenBudget = intPointer(bounded)
	return request
}

func buildBudgetedContextMarkdown(
	sections []string,
	tokenBudget int,
) budgetedContextResult {
	filteredSections := contextSectionsWithoutEmptyValues(sections)
	if len(filteredSections) == 0 {
		return budgetedContextResult{markdown: "", tokenEstimate: 0, truncated: false}
	}
	markdown, truncated := joinContextSectionsWithinCharBudget(filteredSections, maxCharsForTokenBudget(tokenBudget))
	return budgetedContextResult{
		markdown:      markdown,
		tokenEstimate: estimateTokenCount(markdown),
		truncated:     truncated,
	}
}

func contextSectionsWithoutEmptyValues(sections []string) []string {
	filtered := make([]string, 0, len(sections))
	for _, section := range sections {
		trimmed := strings.TrimSpace(section)
		if trimmed == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	return filtered
}

func joinContextSectionsWithinCharBudget(sections []string, maxChars int) (string, bool) {
	if maxChars <= 0 {
		return "", len(sections) > 0
	}
	selected := make([]string, 0, len(sections))
	usedChars := 0
	for _, section := range sections {
		sectionLength := len(section)
		separatorLength := 0
		if len(selected) > 0 {
			separatorLength = 2
		}
		nextLength := usedChars + separatorLength + sectionLength
		if nextLength <= maxChars {
			selected = append(selected, section)
			usedChars = nextLength
			continue
		}
		if len(selected) == 0 {
			selected = append(selected, trimTextToCharacterBudget(section, maxChars))
		}
		return strings.Join(selected, "\n\n"), true
	}
	return strings.Join(selected, "\n\n"), false
}

func trimTextToCharacterBudget(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}
	if maxChars <= 3 {
		return text[:maxChars]
	}
	return strings.TrimSpace(text[:maxChars-3]) + "..."
}

func maxCharsForTokenBudget(tokenBudget int) int {
	return tokenBudget * estimatedCharsPerToken
}

func estimateTokenCount(text string) int {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0
	}
	return (len(trimmed) + estimatedCharsPerToken - 1) / estimatedCharsPerToken
}

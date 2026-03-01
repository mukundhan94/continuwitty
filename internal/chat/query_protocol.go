package chat

import "strings"

const (
	cwQueryPrefix = "cw>"
	cwModeAuto    = "auto"
)

// CWQueryPlan captures normalized cw> protocol directives applied to a message.
type CWQueryPlan struct {
	Mode              string `json:"mode"`
	Project           string `json:"project,omitempty"`
	CitationsRequired bool   `json:"citations_required"`
}

// ParseCWQueryProtocol extracts an optional cw> plan from the first line and returns normalized user query content.
func ParseCWQueryProtocol(contentText string) (string, *CWQueryPlan) {
	lines := strings.Split(contentText, "\n")
	if len(lines) == 0 {
		return contentText, nil
	}
	firstLine := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(strings.ToLower(firstLine), cwQueryPrefix) {
		return contentText, nil
	}

	directive := strings.TrimSpace(firstLine[len(cwQueryPrefix):])
	plan, inlineQuery := parseCWDirective(directive)
	bodyLines := lines[1:]
	if inlineQuery != "" {
		bodyLines = append([]string{inlineQuery}, bodyLines...)
	}
	return strings.TrimSpace(strings.Join(bodyLines, "\n")), plan
}

func parseCWDirective(directive string) (*CWQueryPlan, string) {
	plan := &CWQueryPlan{Mode: cwModeAuto}
	tokens := strings.Fields(directive)
	if len(tokens) == 0 {
		return plan, ""
	}

	inlineQueryTokens := make([]string, 0)
	startIndex := 0
	if !strings.Contains(tokens[0], "=") {
		plan.Mode = normalizeCWMode(tokens[0])
		startIndex = 1
	}

	for _, token := range tokens[startIndex:] {
		key, value, hasPair := strings.Cut(token, "=")
		if !hasPair {
			inlineQueryTokens = append(inlineQueryTokens, token)
			continue
		}
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		normalizedValue := strings.TrimSpace(value)
		if normalizedKey == "" || normalizedValue == "" {
			continue
		}
		switch normalizedKey {
		case "mode":
			plan.Mode = normalizeCWMode(normalizedValue)
		case "project":
			plan.Project = normalizedValue
		case "citations":
			plan.CitationsRequired = strings.EqualFold(normalizedValue, "required")
		}
	}

	return plan, strings.TrimSpace(strings.Join(inlineQueryTokens, " "))
}

func normalizeCWMode(rawMode string) string {
	mode := strings.ToLower(strings.TrimSpace(rawMode))
	if mode == "" {
		return cwModeAuto
	}
	return mode
}

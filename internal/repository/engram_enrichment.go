package repository

import (
	"regexp"
	"sort"
	"strings"

	"engram/internal/models"
)

const (
	defaultEngramAbstract      = "Conversation snapshot."
	maxEnrichmentAbstractChars = 320
	maxEnrichmentKeywords      = 12
	maxEnrichmentTags          = 8
)

var (
	enrichmentTokenPattern    = regexp.MustCompile(`[a-z0-9][a-z0-9_-]{2,}`)
	enrichmentParagraphSplit  = regexp.MustCompile(`\n\s*\n`)
	enrichmentMarkdownCleaner = strings.NewReplacer("`", " ", "*", " ", "_", " ", ">", " ", "#", " ", "-", " ")
	enrichmentStopwords       = map[string]struct{}{
		"about": {}, "after": {}, "again": {}, "also": {}, "been": {}, "before": {}, "being": {}, "between": {}, "could": {},
		"does": {}, "doing": {}, "done": {}, "from": {}, "have": {}, "into": {}, "just": {}, "like": {}, "many": {},
		"more": {}, "most": {}, "much": {}, "need": {}, "only": {}, "other": {}, "over": {}, "same": {}, "some": {},
		"such": {}, "than": {}, "that": {}, "their": {}, "them": {}, "there": {}, "these": {}, "they": {}, "this": {},
		"those": {}, "through": {}, "under": {}, "using": {}, "very": {}, "what": {}, "when": {}, "where": {}, "which": {},
		"while": {}, "with": {}, "would": {}, "your": {},
	}
	enrichmentTagKeywords = map[string]map[string]struct{}{
		"incident": {"incident": {}, "outage": {}, "triage": {}, "mitigation": {}, "latency": {}, "sev": {}, "rollback": {}, "blast": {}, "impact": {}},
		"support":  {"support": {}, "customer": {}, "ticket": {}, "handoff": {}, "escalation": {}},
		"product":  {"product": {}, "feature": {}, "roadmap": {}, "market": {}, "pricing": {}, "growth": {}},
		"research": {"research": {}, "analysis": {}, "experiment": {}, "hypothesis": {}, "benchmark": {}},
		"security": {"security": {}, "vulnerability": {}, "threat": {}, "auth": {}, "oidc": {}, "compliance": {}},
		"mcp":      {"mcp": {}, "jsonrpc": {}, "sse": {}, "tool": {}, "tools": {}, "interop": {}},
		"chat":     {"chat": {}, "session": {}, "conversation": {}, "assistant": {}, "prompt": {}},
		"ops":      {"ops": {}, "runbook": {}, "oncall": {}, "deploy": {}, "deployment": {}, "monitoring": {}, "infra": {}},
		"docs":     {"docs": {}, "documentation": {}, "readme": {}, "playbook": {}, "guide": {}},
		"release":  {"release": {}, "version": {}, "hotfix": {}, "changelog": {}, "rollback": {}},
	}
)

type rankedKeyword struct {
	token enrichmentToken
	count int
	index int
}

type enrichmentText string
type enrichmentToken string

func (text enrichmentText) String() string {
	return string(text)
}

func enrichEngramPayloadIfMissing(
	payload models.MemoryEngramCreate,
	enrichmentOrigin string,
) (models.MemoryEngramCreate, map[string]any) {
	report := defaultEnrichmentReport(enrichmentOrigin)
	resolved := payload

	abstractMissing := strings.TrimSpace(payload.Abstract) == ""
	keywordsMissing := len(nonEmptyTrimmedStrings(payload.Keywords)) == 0
	tagsMissing := len(nonEmptyTrimmedStrings(payload.Tags)) == 0

	if abstractMissing {
		derived, source := deriveAbstractFromMarkdownOrRetrieval(
			enrichmentText(payload.DetailedSummaryMarkdown),
			payload.RetrievalText,
		)
		resolved.Abstract = derived
		report["abstract_derived"] = true
		report["abstract_source"] = source
	}

	sourceText := strings.Join(
		[]string{
			payload.Title,
			resolved.Abstract,
			payload.DetailedSummaryMarkdown,
			stringPointerValue(payload.RetrievalText),
		},
		"\n",
	)

	if keywordsMissing {
		keywords := extractKeywords(enrichmentText(sourceText), maxEnrichmentKeywords)
		resolved.Keywords = keywords
		report["keywords_derived"] = true
		report["auto_keywords"] = append([]string(nil), keywords...)
	}

	if tagsMissing {
		seed := resolved.Keywords
		if !keywordsMissing {
			seed = nonEmptyTrimmedStrings(payload.Keywords)
		}
		tags := mapKeywordsToTags(seed, maxEnrichmentTags)
		resolved.Tags = tags
		report["tags_derived"] = true
		report["auto_tags"] = append([]string(nil), tags...)
	}

	report["enrichment_applied"] = report["abstract_derived"] == true ||
		report["tags_derived"] == true ||
		report["keywords_derived"] == true

	resolved.Tags = normalizeEngramStringSlice(resolved.Tags)
	resolved.Keywords = normalizeEngramStringSlice(resolved.Keywords)
	return resolved, report
}

func deriveAbstractFromMarkdownOrRetrieval(markdown enrichmentText, retrievalText *string) (string, string) {
	trimmedMarkdown := strings.TrimSpace(markdown.String())
	if trimmedMarkdown != "" {
		if assistantSection := firstAssistantSection(enrichmentText(trimmedMarkdown)); assistantSection != "" {
			return truncateWithEllipsis(assistantSection, maxEnrichmentAbstractChars), "assistant_section"
		}
		if paragraph := firstSignalParagraph(enrichmentText(trimmedMarkdown)); paragraph != "" {
			return truncateWithEllipsis(paragraph, maxEnrichmentAbstractChars), "first_signal_paragraph"
		}
	}
	retrieval := normalizeEnrichmentSpaces(enrichmentText(stringPointerValue(retrievalText)))
	if retrieval != "" {
		return truncateWithEllipsis(retrieval, maxEnrichmentAbstractChars), "retrieval_text"
	}
	return defaultEngramAbstract, "fallback"
}

func firstAssistantSection(markdown enrichmentText) enrichmentText {
	lines := strings.Split(markdown.String(), "\n")
	capturing := false
	buffer := make([]string, 0, len(lines))
	for _, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "## ") {
			if strings.HasPrefix(lower, "## assistant") {
				capturing = true
				buffer = buffer[:0]
				continue
			}
			if capturing {
				break
			}
		}
		if capturing {
			buffer = append(buffer, rawLine)
		}
	}
	return stripMarkdownNoise(enrichmentText(strings.Join(buffer, "\n")))
}

func firstSignalParagraph(markdown enrichmentText) enrichmentText {
	paragraphs := enrichmentParagraphSplit.Split(strings.TrimSpace(markdown.String()), -1)
	for _, paragraph := range paragraphs {
		cleaned := stripMarkdownNoise(enrichmentText(paragraph))
		if len(cleaned) >= 36 && len(strings.Fields(cleaned.String())) >= 7 {
			return cleaned
		}
	}
	return enrichmentText("")
}

func extractKeywords(sourceText enrichmentText, maxKeywords int) []string {
	lowered := strings.ToLower(sourceText.String())
	matches := enrichmentTokenPattern.FindAllString(lowered, -1)
	if len(matches) == 0 {
		return []string{}
	}

	counts, firstIndex := collectKeywordStats(toEnrichmentTokens(matches))
	if len(counts) == 0 {
		return []string{}
	}

	ranked := rankKeywordCounts(counts, firstIndex)
	limit := clampKeywordLimit(maxKeywords, len(ranked))
	keywords := make([]string, 0, limit)
	for _, item := range ranked {
		if len(keywords) >= limit {
			break
		}
		keywords = append(keywords, item.token.String())
	}
	return keywords
}

func toEnrichmentTokens(values []string) []enrichmentToken {
	tokens := make([]enrichmentToken, 0, len(values))
	for _, value := range values {
		tokens = append(tokens, enrichmentToken(value))
	}
	return tokens
}

func collectKeywordStats(matches []enrichmentToken) (map[enrichmentToken]int, map[enrichmentToken]int) {
	counts := map[enrichmentToken]int{}
	firstIndex := map[enrichmentToken]int{}
	for _, token := range matches {
		if !isKeywordCandidate(token) {
			continue
		}
		counts[token]++
		if _, seen := firstIndex[token]; !seen {
			firstIndex[token] = len(firstIndex)
		}
	}
	return counts, firstIndex
}

func rankKeywordCounts(
	counts map[enrichmentToken]int,
	firstIndex map[enrichmentToken]int,
) []rankedKeyword {
	ranked := make([]rankedKeyword, 0, len(counts))
	for token, count := range counts {
		ranked = append(ranked, rankedKeyword{
			token: token,
			count: count,
			index: firstIndex[token],
		})
	}
	sort.Slice(ranked, func(left, right int) bool {
		if ranked[left].count != ranked[right].count {
			return ranked[left].count > ranked[right].count
		}
		if ranked[left].index != ranked[right].index {
			return ranked[left].index < ranked[right].index
		}
		return ranked[left].token < ranked[right].token
	})
	return ranked
}

func clampKeywordLimit(maxKeywords int, available int) int {
	limit := maxKeywords
	if limit < 1 {
		limit = 1
	}
	if available < limit {
		return available
	}
	return limit
}

func mapKeywordsToTags(keywords []string, maxTags int) []string {
	normalizedKeywords := make(map[string]struct{}, len(keywords))
	for _, keyword := range nonEmptyTrimmedStrings(keywords) {
		normalizedKeywords[strings.ToLower(keyword)] = struct{}{}
	}

	tags := make([]string, 0, len(enrichmentTagKeywords))
	for tag, mappedKeywords := range enrichmentTagKeywords {
		if hasKeywordOverlap(normalizedKeywords, mappedKeywords) {
			tags = append(tags, tag)
		}
	}
	sort.Strings(tags)
	if len(tags) == 0 {
		tags = []string{"conversation", "snapshot"}
	}

	limit := maxTags
	if limit < 1 {
		limit = 1
	}
	if len(tags) <= limit {
		return tags
	}
	return tags[:limit]
}

func hasKeywordOverlap(
	keywords map[string]struct{},
	candidates map[string]struct{},
) bool {
	for keyword := range keywords {
		if _, exists := candidates[keyword]; exists {
			return true
		}
	}
	return false
}

func isKeywordCandidate(token enrichmentToken) bool {
	if _, stopword := enrichmentStopwords[token.String()]; stopword {
		return false
	}
	return token.hasLetter()
}

func (token enrichmentToken) String() string {
	return string(token)
}

func (token enrichmentToken) hasLetter() bool {
	for _, char := range token {
		if char >= 'a' && char <= 'z' {
			return true
		}
	}
	return false
}

func nonEmptyTrimmedStrings(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		item := strings.TrimSpace(value)
		if item == "" {
			continue
		}
		trimmed = append(trimmed, item)
	}
	return trimmed
}

func stripMarkdownNoise(value enrichmentText) enrichmentText {
	return normalizeEnrichmentSpaces(enrichmentText(enrichmentMarkdownCleaner.Replace(value.String())))
}

func normalizeEnrichmentSpaces(value enrichmentText) enrichmentText {
	return enrichmentText(strings.Join(strings.Fields(strings.TrimSpace(value.String())), " "))
}

func truncateWithEllipsis(value enrichmentText, maxChars int) string {
	if maxChars <= 0 {
		return ""
	}
	if len(value) <= maxChars {
		return value.String()
	}
	if maxChars <= 3 {
		return value.String()[:maxChars]
	}
	return strings.TrimSpace(value.String()[:maxChars-3]) + "..."
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

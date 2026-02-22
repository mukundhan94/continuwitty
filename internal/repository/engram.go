package repository

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"engram/internal/models"

	"github.com/google/uuid"
)

var tokenPattern = regexp.MustCompile(`[a-z0-9]{2,}`)

var genericChatAbstracts = map[string]struct{}{
	"":                                   {},
	"snapshot from active chat session.": {},
	"snapshot from active chat session":  {},
	"chat snapshot":                      {},
	"session snapshot":                   {},
}

type rehydrationContextParts struct {
	Title           string
	CompactSummary  string
	DetailedExcerpt string
	Decisions       []map[string]any
	OpenQuestions   []string
	Citations       []models.RehydrationCitation
}

func vectorLiteral(values []float64) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%.6f", value))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func buildRetrievalText(payload models.MemoryEngramCreate) string {
	if payload.RetrievalText != nil && *payload.RetrievalText != "" {
		return *payload.RetrievalText
	}

	decisionParts := make([]string, 0, len(payload.Decisions))
	for _, decision := range payload.Decisions {
		decisionParts = append(decisionParts, decision.Decision)
	}
	claimParts := make([]string, 0, len(payload.Claims))
	for _, claim := range payload.Claims {
		claimParts = append(claimParts, claim.Claim)
	}

	combined := strings.Join(
		[]string{
			payload.Title,
			payload.Abstract,
			strings.Join(decisionParts, " "),
			strings.Join(payload.OpenQuestions, " "),
			strings.Join(claimParts, " "),
		},
		" ",
	)
	return strings.TrimSpace(combined)
}

func tokenize(text string) map[string]struct{} {
	matches := tokenPattern.FindAllString(strings.ToLower(text), -1)
	tokens := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		tokens[match] = struct{}{}
	}
	return tokens
}

func normalizeSpaces(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func truncateText(value string, maxChars int) string {
	if maxChars <= 0 || len(value) <= maxChars {
		return value
	}
	if maxChars <= 3 {
		return value[:maxChars]
	}
	return strings.TrimSpace(value[:maxChars-3]) + "..."
}

func extractDetailedExcerpt(markdown string, maxChars int) string {
	text := strings.TrimSpace(markdown)
	if text == "" {
		return ""
	}

	excerpt := text
	lines := strings.Split(text, "\n")
	for lineIndex, line := range lines {
		heading := strings.ToLower(strings.TrimSpace(line))
		if !strings.HasPrefix(heading, "## assistant") {
			continue
		}

		bodyLines := make([]string, 0)
		for innerIndex := lineIndex + 1; innerIndex < len(lines); innerIndex++ {
			trimmed := strings.TrimSpace(lines[innerIndex])
			if strings.HasPrefix(trimmed, "## ") {
				break
			}
			bodyLines = append(bodyLines, lines[innerIndex])
		}

		candidate := strings.TrimSpace(strings.Join(bodyLines, "\n"))
		if candidate != "" {
			excerpt = candidate
		}
		break
	}

	if excerpt == "" {
		return ""
	}
	return truncateText(excerpt, maxChars)
}

func resolveCompactSummary(abstract, detailedSummaryMarkdown string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = 800
	}

	abstractClean := normalizeSpaces(abstract)
	if _, isGeneric := genericChatAbstracts[strings.ToLower(abstractClean)]; !isGeneric {
		return truncateText(abstractClean, maxChars)
	}

	fallback := normalizeSpaces(extractDetailedExcerpt(detailedSummaryMarkdown, maxChars))
	if fallback != "" {
		return fallback
	}
	if abstractClean != "" {
		return truncateText(abstractClean, maxChars)
	}
	return "No summary available."
}

func lexicalOverlapScore(query string, candidateParts []string) float64 {
	queryTokens := tokenize(query)
	if len(queryTokens) == 0 {
		return 0
	}

	documentTokens := make(map[string]struct{})
	for _, part := range candidateParts {
		for token := range tokenize(part) {
			documentTokens[token] = struct{}{}
		}
	}
	if len(documentTokens) == 0 {
		return 0
	}

	overlap := 0
	for token := range queryTokens {
		if _, exists := documentTokens[token]; exists {
			overlap += 1
		}
	}
	return float64(overlap) / float64(len(queryTokens))
}

func combinedRankScore(distance, lexicalOverlap float64) float64 {
	denseScore := 1.0 / (1.0 + math.Max(distance, 0))
	return (denseScore * 0.8) + (lexicalOverlap * 0.2)
}

func packCitations(citations []models.RehydrationCitation, limit int) []models.RehydrationCitation {
	if limit <= 0 {
		return []models.RehydrationCitation{}
	}

	packed := make([]models.RehydrationCitation, 0, min(limit, len(citations)))
	seenURLs := make(map[string]struct{}, len(citations))
	for _, citation := range citations {
		urlKey := strings.ToLower(strings.TrimSpace(citation.URL))
		if _, exists := seenURLs[urlKey]; exists {
			continue
		}
		seenURLs[urlKey] = struct{}{}
		packed = append(packed, citation)
		if len(packed) >= limit {
			break
		}
	}
	return packed
}

func buildEngramQueryWhere(
	request models.EngramQueryRequest,
	actorUserID *uuid.UUID,
	queryLiteral string,
) (string, []any) {
	whereClauses := []string{"deleted_at IS NULL"}
	params := []any{queryLiteral}

	if request.ProjectID != nil && *request.ProjectID != "" {
		whereClauses = append(whereClauses, "project_id = %s")
		params = append(params, *request.ProjectID)
	}
	if actorUserID != nil {
		whereClauses = append(
			whereClauses,
			"(owner_user_id = %s OR visibility_scope = 'project' OR owner_user_id IS NULL)",
		)
		params = append(params, *actorUserID)
	}
	if len(request.Tags) > 0 {
		whereClauses = append(whereClauses, "tags && %s")
		params = append(params, request.Tags)
	}
	if len(request.Keywords) > 0 {
		whereClauses = append(whereClauses, "keywords && %s")
		params = append(params, request.Keywords)
	}
	if request.CreatedAfter != nil {
		whereClauses = append(whereClauses, "created_at >= %s")
		params = append(params, *request.CreatedAfter)
	}
	if request.CreatedBefore != nil {
		whereClauses = append(whereClauses, "created_at <= %s")
		params = append(params, *request.CreatedBefore)
	}

	return "WHERE " + strings.Join(whereClauses, " AND "), params
}

func rerankByCombinedScore(rows []map[string]any, query string, topK int) []map[string]any {
	type rankedRow struct {
		score     float64
		createdAt time.Time
		row       map[string]any
	}

	ranked := make([]rankedRow, 0, len(rows))
	for _, row := range rows {
		lexicalScore := lexicalOverlapScore(
			query,
			[]string{
				stringFromAny(row["title"]),
				stringFromAny(row["abstract"]),
				stringFromAny(row["retrieval_text"]),
				strings.Join(stringSliceFromAny(row["tags"]), " "),
				strings.Join(stringSliceFromAny(row["keywords"]), " "),
			},
		)
		ranked = append(
			ranked,
			rankedRow{
				score:     combinedRankScore(float64FromAny(row["distance"]), lexicalScore),
				createdAt: timeFromAny(row["created_at"]),
				row:       row,
			},
		)
	}

	sort.Slice(ranked, func(left, right int) bool {
		if ranked[left].score == ranked[right].score {
			return ranked[left].createdAt.After(ranked[right].createdAt)
		}
		return ranked[left].score > ranked[right].score
	})

	if topK <= 0 || topK > len(ranked) {
		topK = len(ranked)
	}
	trimmed := make([]map[string]any, 0, topK)
	for _, row := range ranked[:topK] {
		trimmed = append(trimmed, row.row)
	}
	return trimmed
}

func formatCitations(citations []models.RehydrationCitation) string {
	lines := make([]string, 0, len(citations))
	for _, citation := range citations {
		label := citation.URL
		if citation.Title != nil && *citation.Title != "" {
			label = *citation.Title
		}
		snippet := strings.TrimSpace(strings.ReplaceAll(citation.Snippet, "\n", " "))
		if len(snippet) > 140 {
			snippet = snippet[:137] + "..."
		}
		line := fmt.Sprintf("- %s (%s)", label, citation.URL)
		if snippet != "" {
			line += ": " + snippet
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return "- No citations available"
	}
	return strings.Join(lines, "\n")
}

func formatDecisions(decisions []map[string]any) string {
	lines := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		lines = append(
			lines,
			fmt.Sprintf(
				"- %s: %s",
				stringFromAny(decision["decision"]),
				stringFromAny(decision["rationale"]),
			),
		)
	}
	if len(lines) == 0 {
		return "- None"
	}
	return strings.Join(lines, "\n")
}

func formatOpenQuestions(openQuestions []string) string {
	if len(openQuestions) == 0 {
		return "- None"
	}
	lines := make([]string, 0, len(openQuestions))
	for _, question := range openQuestions {
		lines = append(lines, "- "+question)
	}
	return strings.Join(lines, "\n")
}

func buildRehydrationContextMarkdown(parts rehydrationContextParts) string {
	sections := []string{
		fmt.Sprintf("# Rehydration Context: %s", parts.Title),
		fmt.Sprintf("## Compact Summary\n%s", parts.CompactSummary),
	}
	if strings.TrimSpace(parts.DetailedExcerpt) != "" {
		sections = append(sections, fmt.Sprintf("## Detailed Notes Excerpt\n%s", parts.DetailedExcerpt))
	}

	sections = append(
		sections,
		fmt.Sprintf("## Key Decisions\n%s", formatDecisions(parts.Decisions)),
		fmt.Sprintf("## Open Questions\n%s", formatOpenQuestions(parts.OpenQuestions)),
		fmt.Sprintf("## Top Citations\n%s", formatCitations(parts.Citations)),
	)
	return strings.Join(sections, "\n\n")
}

func buildEngramJSONPayload(
	payload models.MemoryEngramCreate,
	enrichmentReport map[string]any,
	createdAt time.Time,
) map[string]any {
	decisions := make([]map[string]any, 0, len(payload.Decisions))
	for _, decision := range payload.Decisions {
		decisions = append(
			decisions,
			map[string]any{
				"decision":  decision.Decision,
				"rationale": decision.Rationale,
			},
		)
	}

	claims := make([]map[string]any, 0, len(payload.Claims))
	for _, claim := range payload.Claims {
		supportingSources := make([]map[string]any, 0, len(claim.SupportingSources))
		for _, source := range claim.SupportingSources {
			supportingSources = append(
				supportingSources,
				map[string]any{
					"url":         source.URL,
					"title":       source.Title,
					"snippet":     source.Snippet,
					"captured_at": source.CapturedAt,
				},
			)
		}
		claims = append(
			claims,
			map[string]any{
				"claim":              claim.Claim,
				"supporting_sources": supportingSources,
			},
		)
	}

	artifacts := make([]map[string]any, 0, len(payload.Artifacts))
	for _, artifact := range payload.Artifacts {
		artifacts = append(
			artifacts,
			map[string]any{
				"artifact_type": artifact.ArtifactType,
				"storage_uri":   artifact.StorageURI,
				"metadata":      artifact.Metadata,
			},
		)
	}

	visibilityScope := payload.VisibilityScope
	if visibilityScope == "" {
		visibilityScope = "private"
	}

	var threadID any
	if payload.ThreadID != nil {
		threadID = *payload.ThreadID
	}

	var sourceSessionID any
	if payload.SourceSessionID != nil {
		sourceSessionID = payload.SourceSessionID.String()
	}

	return map[string]any{
		"schema_version":            "1.0",
		"project_id":                payload.ProjectID,
		"thread_id":                 threadID,
		"title":                     payload.Title,
		"abstract":                  payload.Abstract,
		"detailed_summary_markdown": payload.DetailedSummaryMarkdown,
		"decisions":                 decisions,
		"assumptions":               payload.Assumptions,
		"open_questions":            payload.OpenQuestions,
		"claims":                    claims,
		"tags":                      payload.Tags,
		"keywords":                  payload.Keywords,
		"artifacts":                 artifacts,
		"visibility_scope":          visibilityScope,
		"source_session_id":         sourceSessionID,
		"auto_metadata":             enrichmentReport,
		"created_at":                createdAt.Format("2006-01-02T15:04:05-07:00"),
	}
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func stringSliceFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			values = append(values, stringFromAny(item))
		}
		return values
	default:
		return []string{}
	}
}

func float64FromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func timeFromAny(value any) time.Time {
	if typed, ok := value.(time.Time); ok {
		return typed
	}
	return time.Time{}
}

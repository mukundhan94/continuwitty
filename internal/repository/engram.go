package repository

import (
	"fmt"
	"strings"
	"time"

	"engram/internal/embeddings"
	"engram/internal/models"

	"github.com/google/uuid"
)

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

type textLimitInput struct {
	value    string
	maxChars int
}

type detailedExcerptInput struct {
	markdown string
	maxChars int
}

type sectionBodyInput struct {
	lines []string
	start int
}

type compactSummaryInput struct {
	abstract                string
	detailedSummaryMarkdown string
	maxChars                int
}

type citationPackInput struct {
	citations []models.RehydrationCitation
	limit     int
}

func vectorLiteral(values []float64) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%.6f", value))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// BuildLocalQueryLiteral creates a pgvector-compatible literal for query search.
func BuildLocalQueryLiteral(query string, embeddingDim int) (string, error) {
	provider := embeddings.LocalDeterministicEmbeddingProvider{}
	vector, err := provider.Embed(query, embeddingDim)
	if err != nil {
		return "", err
	}
	return vectorLiteral(vector), nil
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

func normalizeSpaces(input textLimitInput) string {
	return strings.Join(strings.Fields(strings.TrimSpace(input.value)), " ")
}

func truncateText(input textLimitInput) string {
	if input.maxChars <= 0 || len(input.value) <= input.maxChars {
		return input.value
	}
	if input.maxChars <= 3 {
		return input.value[:input.maxChars]
	}
	return strings.TrimSpace(input.value[:input.maxChars-3]) + "..."
}

func extractDetailedExcerpt(input detailedExcerptInput) string {
	text := strings.TrimSpace(input.markdown)
	if text == "" {
		return ""
	}

	excerpt := assistantExcerpt(text)
	if excerpt == "" {
		excerpt = text
	}
	return truncateText(textLimitInput{value: excerpt, maxChars: input.maxChars})
}

func assistantExcerpt(markdown string) string {
	lines := strings.Split(markdown, "\n")
	assistantStart := assistantSectionStart(lines)
	if assistantStart < 0 {
		return ""
	}
	return sectionBody(sectionBodyInput{lines: lines, start: assistantStart})
}

func assistantSectionStart(lines []string) int {
	for index, line := range lines {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "## assistant") {
			return index + 1
		}
	}
	return -1
}

func sectionBody(input sectionBodyInput) string {
	bodyLines := make([]string, 0)
	for index := input.start; index < len(input.lines); index++ {
		if strings.HasPrefix(strings.TrimSpace(input.lines[index]), "## ") {
			break
		}
		bodyLines = append(bodyLines, input.lines[index])
	}
	return strings.TrimSpace(strings.Join(bodyLines, "\n"))
}

func resolveCompactSummary(input compactSummaryInput) string {
	if input.maxChars <= 0 {
		input.maxChars = 800
	}

	abstractClean := normalizeSpaces(textLimitInput{value: input.abstract})
	if _, isGeneric := genericChatAbstracts[strings.ToLower(abstractClean)]; !isGeneric {
		return truncateText(textLimitInput{value: abstractClean, maxChars: input.maxChars})
	}

	fallback := normalizeSpaces(
		textLimitInput{
			value: extractDetailedExcerpt(
				detailedExcerptInput{
					markdown: input.detailedSummaryMarkdown,
					maxChars: input.maxChars,
				},
			),
		},
	)
	if fallback != "" {
		return fallback
	}
	if abstractClean != "" {
		return truncateText(textLimitInput{value: abstractClean, maxChars: input.maxChars})
	}
	return "No summary available."
}

func packCitations(input citationPackInput) []models.RehydrationCitation {
	if input.limit <= 0 {
		return []models.RehydrationCitation{}
	}

	packed := make([]models.RehydrationCitation, 0, min(input.limit, len(input.citations)))
	seenURLs := make(map[string]struct{}, len(input.citations))
	for _, citation := range input.citations {
		urlKey := strings.ToLower(strings.TrimSpace(citation.URL))
		if _, exists := seenURLs[urlKey]; exists {
			continue
		}
		seenURLs[urlKey] = struct{}{}
		packed = append(packed, citation)
		if len(packed) >= input.limit {
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
	builder := newEngramQueryWhereBuilder(queryLiteral)
	builder.addProjectFilter(request.ProjectID)
	builder.addActorVisibilityFilter(actorUserID)
	builder.addStringSliceOverlapFilter("tags", request.Tags)
	builder.addStringSliceOverlapFilter("keywords", request.Keywords)
	addOptionalPointerClause(builder, request.CreatedAfter, "created_at >= %s")
	addOptionalPointerClause(builder, request.CreatedBefore, "created_at <= %s")
	addOptionalPointerClause(builder, request.AccessCountMin, "COALESCE(access_count, 0) >= %s")
	addOptionalPointerClause(builder, request.FreshnessScoreMin, "COALESCE(freshness_score, 1.0) >= %s")
	addOptionalPointerClause(builder, request.LastAccessedAfter, "COALESCE(last_accessed_at, created_at) >= %s")
	addOptionalPointerClause(builder, request.LastAccessedBefore, "COALESCE(last_accessed_at, created_at) <= %s")
	addOptionalPointerClause(builder, request.FreshnessComputedAfter, "COALESCE(freshness_last_computed_at, created_at) >= %s")
	addOptionalPointerClause(builder, request.FreshnessComputedBefore, "COALESCE(freshness_last_computed_at, created_at) <= %s")
	return builder.whereClause(), builder.params
}

type engramQueryWhereBuilder struct {
	whereClauses []string
	params       []any
}

func newEngramQueryWhereBuilder(queryLiteral string) *engramQueryWhereBuilder {
	return &engramQueryWhereBuilder{
		whereClauses: []string{"deleted_at IS NULL"},
		params:       []any{queryLiteral},
	}
}

func (builder *engramQueryWhereBuilder) nextPlaceholder() string {
	return pgxPlaceholder(len(builder.params) + 1)
}

func (builder *engramQueryWhereBuilder) addClauseWithParam(clauseFormat string, param any) {
	builder.whereClauses = append(builder.whereClauses, fmt.Sprintf(clauseFormat, builder.nextPlaceholder()))
	builder.params = append(builder.params, param)
}

func (builder *engramQueryWhereBuilder) addProjectFilter(projectID *string) {
	if projectID == nil || *projectID == "" {
		return
	}
	builder.addClauseWithParam("project_id = %s", *projectID)
}

func (builder *engramQueryWhereBuilder) addActorVisibilityFilter(actorUserID *uuid.UUID) {
	if actorUserID == nil {
		return
	}
	actorPlaceholder := builder.nextPlaceholder()
	builder.whereClauses = append(
		builder.whereClauses,
		buildMembershipReadClause(
			membershipReadClauseInput{
				ownerColumn:      "owner_user_id",
				visibilityColumn: "visibility_scope",
				projectColumn:    "project_id",
				actorPlaceholder: actorPlaceholder,
				includeOwnerless: true,
			},
		),
	)
	builder.params = append(builder.params, *actorUserID)
}

func (builder *engramQueryWhereBuilder) addStringSliceOverlapFilter(column string, values []string) {
	if len(values) == 0 {
		return
	}
	builder.addClauseWithParam(fmt.Sprintf("%s && %%s", column), values)
}

func (builder *engramQueryWhereBuilder) whereClause() string {
	return "WHERE " + strings.Join(builder.whereClauses, " AND ")
}

func addOptionalPointerClause[T any](
	builder *engramQueryWhereBuilder,
	value *T,
	clauseFormat string,
) {
	if value == nil {
		return
	}
	builder.addClauseWithParam(clauseFormat, *value)
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
	return map[string]any{
		"schema_version":            "1.0",
		"project_id":                payload.ProjectID,
		"thread_id":                 threadIDValue(payload),
		"title":                     payload.Title,
		"abstract":                  payload.Abstract,
		"detailed_summary_markdown": payload.DetailedSummaryMarkdown,
		"decisions":                 decisionPayload(payload.Decisions),
		"assumptions":               payload.Assumptions,
		"open_questions":            payload.OpenQuestions,
		"claims":                    claimPayload(payload.Claims),
		"tags":                      payload.Tags,
		"keywords":                  payload.Keywords,
		"artifacts":                 artifactPayload(payload.Artifacts),
		"visibility_scope":          visibilityScopeValue(payload),
		"source_session_id":         sourceSessionIDValue(payload),
		"auto_metadata":             enrichmentReport,
		"created_at":                createdAt.Format("2006-01-02T15:04:05-07:00"),
	}
}

func decisionPayload(decisions []models.Decision) []map[string]any {
	payload := make([]map[string]any, 0, len(decisions))
	for _, decision := range decisions {
		payload = append(
			payload,
			map[string]any{
				"decision":  decision.Decision,
				"rationale": decision.Rationale,
			},
		)
	}
	return payload
}

func claimPayload(claims []models.Claim) []map[string]any {
	payload := make([]map[string]any, 0, len(claims))
	for _, claim := range claims {
		payload = append(
			payload,
			map[string]any{
				"claim":              claim.Claim,
				"supporting_sources": supportingSourcePayload(claim.SupportingSources),
			},
		)
	}
	return payload
}

func supportingSourcePayload(sources []models.SupportingSource) []map[string]any {
	payload := make([]map[string]any, 0, len(sources))
	for _, source := range sources {
		payload = append(
			payload,
			map[string]any{
				"url":         source.URL,
				"title":       source.Title,
				"snippet":     source.Snippet,
				"captured_at": source.CapturedAt,
			},
		)
	}
	return payload
}

func artifactPayload(artifacts []models.ArtifactIn) []map[string]any {
	payload := make([]map[string]any, 0, len(artifacts))
	for _, artifact := range artifacts {
		payload = append(
			payload,
			map[string]any{
				"artifact_type": artifact.ArtifactType,
				"storage_uri":   artifact.StorageURI,
				"metadata":      artifact.Metadata,
			},
		)
	}
	return payload
}

func visibilityScopeValue(payload models.MemoryEngramCreate) string {
	if payload.VisibilityScope == "" {
		return "private"
	}
	return payload.VisibilityScope
}

func threadIDValue(payload models.MemoryEngramCreate) any {
	if payload.ThreadID == nil {
		return nil
	}
	return *payload.ThreadID
}

func sourceSessionIDValue(payload models.MemoryEngramCreate) any {
	if payload.SourceSessionID == nil {
		return nil
	}
	return payload.SourceSessionID.String()
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
	number, ok := numberFromAny(value)
	if !ok {
		return 0
	}
	return number
}

func intFromAny(value any) int {
	number, ok := numberFromAny(value)
	if !ok {
		return 0
	}
	return int(number)
}

func numberFromAny(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	default:
		return 0, false
	}
}

func timeFromAny(value any) time.Time {
	if typed, ok := value.(time.Time); ok {
		return typed
	}
	return time.Time{}
}

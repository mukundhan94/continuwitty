package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

const (
	defaultChatContextMaxEngrams   = 6
	defaultChatContextRetrievalTop = 4
	defaultChatContextDocumentTop  = 4
	maxChatContextDocumentBudget   = 12
	defaultLinkRecallDepth         = 1
	maxLinkRecallDepth             = 3
	defaultLinkRecallMaxNeighbors  = 8
	maxLinkRecallMaxNeighbors      = 24
	maxLinkRecallCandidatePool     = 32
	defaultLinkNoiseScoreThreshold = 0.30

	engramSourceType   = "engram_source"
	documentSourceType = "document_chunk"
)

var errChatContextDependenciesIncomplete = errors.New("chat context dependencies incomplete")

// ChatSourceReference captures URL/document evidence used for a chat response.
type ChatSourceReference struct {
	SourceType  string     `json:"source_type"`
	EngramID    uuid.UUID  `json:"engram_id"`
	EngramTitle string     `json:"engram_title"`
	URL         string     `json:"url"`
	Title       *string    `json:"title,omitempty"`
	Snippet     string     `json:"snippet"`
	CapturedAt  time.Time  `json:"captured_at"`
	DocumentID  *uuid.UUID `json:"document_id,omitempty"`
	ChunkID     *uuid.UUID `json:"chunk_id,omitempty"`
	ChunkIndex  *int       `json:"chunk_index,omitempty"`
}

// AssembledChatContext contains rendered context plus trace IDs for persistence/debug.
type AssembledChatContext struct {
	ContextMarkdown      string                `json:"context_markdown"`
	UsedEngramIDs        []uuid.UUID           `json:"used_engram_ids"`
	UsedEngramLinkIDs    []uuid.UUID           `json:"used_engram_link_ids"`
	EngramTracePaths     []EngramTracePath     `json:"engram_trace_paths"`
	UsedDocumentChunkIDs []uuid.UUID           `json:"used_document_chunk_ids"`
	SourceReferences     []ChatSourceReference `json:"source_references"`
}

// EngramTracePath captures one compact root->target trace chain used in context recall.
type EngramTracePath struct {
	RootEngramID   uuid.UUID   `json:"root_engram_id"`
	TargetEngramID uuid.UUID   `json:"target_engram_id"`
	Depth          int         `json:"depth"`
	LinkIDs        []uuid.UUID `json:"link_ids"`
	EngramIDs      []uuid.UUID `json:"engram_ids"`
	Score          float64     `json:"score"`
}

// ChatContextRequest captures retrieval inputs used when building chat context.
type ChatContextRequest struct {
	Session                     models.ChatSessionRecord
	ActorUserID                 uuid.UUID
	UserQuery                   string
	EmbeddingDim                int
	MaxEngrams                  int
	RetrievalTopK               int
	DocumentTopK                int
	LinkRecallEnabled           *bool
	LinkRecallDepth             *int
	LinkRecallMaxNeighbors      *int
	LinkNoiseSuppressionEnabled *bool
	LinkNoiseScoreThreshold     *float64
}

// ChatContextDependencies captures retrieval operations used by context assembly.
type ChatContextDependencies struct {
	ListPinnedEngramSummaries func(ctx context.Context, sessionID uuid.UUID, actorUserID uuid.UUID) ([]models.EngramSummary, error)
	ListPinnedDocuments       func(ctx context.Context, sessionID uuid.UUID, actorUserID uuid.UUID) ([]models.PinnedDocumentRecord, error)
	QueryEngrams              func(ctx context.Context, actorUserID uuid.UUID, request models.EngramQueryRequest, embeddingDim int) ([]models.EngramQueryResult, error)
	GetRehydrationBundle      func(ctx context.Context, engramID uuid.UUID, actorUserID uuid.UUID) (*models.RehydrationBundle, error)
	TraverseEngramLinks       func(
		ctx context.Context,
		rootEngramID uuid.UUID,
		actorUserID uuid.UUID,
		maxDepth int,
		maxNeighbors int,
		includeArchived bool,
	) ([]models.EngramLinkTraversalStep, error)
	QueryDocumentChunks func(ctx context.Context, actorUserID uuid.UUID, request models.DocumentChunkQueryRequest, embeddingDim int) ([]models.DocumentChunkQueryResult, error)
}

type assembledEngramContext struct {
	bundles           []models.RehydrationBundle
	usedEngramID      []uuid.UUID
	usedEngramLinkIDs []uuid.UUID
	tracePaths        []EngramTracePath
}

type assembledDocumentContext struct {
	pinnedChunks         []models.DocumentChunkQueryResult
	selectedChunks       []models.DocumentChunkQueryResult
	usedDocumentChunkIDs []uuid.UUID
}

type rehydrationBundleCollectInput struct {
	EngramIDs   []uuid.UUID
	ActorUserID uuid.UUID
	Limit       int
}

// DefaultChatContextDependencies maps chat context dependencies to repository operations.
func DefaultChatContextDependencies(db repository.Queryer) ChatContextDependencies {
	return ChatContextDependencies{
		ListPinnedEngramSummaries: listPinnedEngramSummariesDependency(db),
		ListPinnedDocuments:       listPinnedDocumentsDependency(db),
		QueryEngrams:              queryEngramsDependency(db),
		GetRehydrationBundle:      getRehydrationBundleDependency(db),
		TraverseEngramLinks:       traverseEngramLinksDependency(db),
		QueryDocumentChunks:       queryDocumentChunksDependency(db),
	}
}

// AssembleChatContext builds merged engram/document context for chat generation.
func AssembleChatContext(
	ctx context.Context,
	request ChatContextRequest,
	dependencies ChatContextDependencies,
) (AssembledChatContext, error) {
	if err := validateChatContextDependencies(dependencies); err != nil {
		return AssembledChatContext{}, err
	}
	normalizedRequest := normalizeChatContextRequest(request)

	engramContext, err := assembleEngramContext(ctx, normalizedRequest, dependencies)
	if err != nil {
		return AssembledChatContext{}, err
	}
	documentContext, err := assembleDocumentContext(ctx, normalizedRequest, dependencies)
	if err != nil {
		return AssembledChatContext{}, err
	}
	if len(engramContext.bundles) == 0 && len(documentContext.selectedChunks) == 0 {
		return AssembledChatContext{
			ContextMarkdown:      "",
			UsedEngramIDs:        engramContext.usedEngramID,
			UsedEngramLinkIDs:    engramContext.usedEngramLinkIDs,
			EngramTracePaths:     engramContext.tracePaths,
			UsedDocumentChunkIDs: documentContext.usedDocumentChunkIDs,
			SourceReferences:     []ChatSourceReference{},
		}, nil
	}

	contextSections := buildContextSections(
		engramContext.bundles,
		documentContext.selectedChunks,
		documentContext.pinnedChunks,
	)
	sourceReferences := dedupeSourceReferences(
		append(
			collectBundleSourceReferences(engramContext.bundles),
			collectDocumentSourceReferences(documentContext.selectedChunks)...,
		),
		16,
	)
	return AssembledChatContext{
		ContextMarkdown:      strings.Join(contextSections, "\n\n"),
		UsedEngramIDs:        engramContext.usedEngramID,
		UsedEngramLinkIDs:    engramContext.usedEngramLinkIDs,
		EngramTracePaths:     engramContext.tracePaths,
		UsedDocumentChunkIDs: documentContext.usedDocumentChunkIDs,
		SourceReferences:     sourceReferences,
	}, nil
}

func assembleEngramContext(
	ctx context.Context,
	request ChatContextRequest,
	dependencies ChatContextDependencies,
) (assembledEngramContext, error) {
	pinned, err := dependencies.ListPinnedEngramSummaries(
		ctx,
		request.Session.SessionID,
		request.ActorUserID,
	)
	if err != nil {
		return assembledEngramContext{}, err
	}
	retrieved, err := dependencies.QueryEngrams(
		ctx,
		request.ActorUserID,
		buildContextEngramQueryRequest(request),
		request.EmbeddingDim,
	)
	if err != nil {
		return assembledEngramContext{}, err
	}
	selectedIDs := selectContextEngramIDs(request.MaxEngrams, pinned, retrieved)
	seedScores := buildSeedRelevanceScores(pinned, retrieved)
	linkSelection, err := selectLinkedEngramContext(
		ctx,
		request,
		dependencies,
		selectedIDs,
		seedScores,
	)
	if err != nil {
		return assembledEngramContext{}, err
	}
	rankedCandidateIDs := rankEngramContextCandidates(
		selectedIDs,
		linkSelection.linkedIDs,
		seedScores,
		linkSelection.tracePaths,
		request.MaxEngrams,
	)
	bundles, err := collectRehydrationBundles(
		ctx,
		dependencies,
		rehydrationBundleCollectInput{
			EngramIDs:   rankedCandidateIDs,
			ActorUserID: request.ActorUserID,
			Limit:       request.MaxEngrams,
		},
	)
	if err != nil {
		return assembledEngramContext{}, err
	}
	usedEngramIDs := collectBundleEngramIDs(bundles)
	filteredTracePaths := filterTracePathsForUsedEngrams(
		linkSelection.tracePaths,
		usedEngramIDs,
	)
	return assembledEngramContext{
		bundles:           bundles,
		usedEngramID:      usedEngramIDs,
		usedEngramLinkIDs: collectUsedLinkIDs(filteredTracePaths),
		tracePaths:        filteredTracePaths,
	}, nil
}

func assembleDocumentContext(
	ctx context.Context,
	request ChatContextRequest,
	dependencies ChatContextDependencies,
) (assembledDocumentContext, error) {
	pinnedDocuments, err := dependencies.ListPinnedDocuments(
		ctx,
		request.Session.SessionID,
		request.ActorUserID,
	)
	if err != nil {
		return assembledDocumentContext{}, err
	}
	pinnedDocumentIDs := collectPinnedDocumentIDs(pinnedDocuments)
	pinnedChunks, err := collectPinnedDocumentChunks(ctx, dependencies, request, pinnedDocumentIDs)
	if err != nil {
		return assembledDocumentContext{}, err
	}
	retrievedChunks, err := dependencies.QueryDocumentChunks(
		ctx,
		request.ActorUserID,
		buildContextDocumentQueryRequest(request),
		request.EmbeddingDim,
	)
	if err != nil {
		return assembledDocumentContext{}, err
	}
	selected := dedupeDocumentChunksPreserveOrder(append(pinnedChunks, retrievedChunks...))
	selected = truncateDocumentChunks(selected, chatContextChunkBudget(request.DocumentTopK, len(pinnedChunks)))
	return assembledDocumentContext{
		pinnedChunks:         pinnedChunks,
		selectedChunks:       selected,
		usedDocumentChunkIDs: collectSelectedChunkIDs(selected),
	}, nil
}

func buildContextEngramQueryRequest(request ChatContextRequest) models.EngramQueryRequest {
	return models.EngramQueryRequest{
		Query:     request.UserQuery,
		ProjectID: &request.Session.ProjectID,
		TopK:      request.RetrievalTopK,
	}
}

func buildContextDocumentQueryRequest(request ChatContextRequest) models.DocumentChunkQueryRequest {
	return models.DocumentChunkQueryRequest{
		Query:     request.UserQuery,
		ProjectID: &request.Session.ProjectID,
		TopK:      request.DocumentTopK,
	}
}

func validateChatContextDependencies(dependencies ChatContextDependencies) error {
	switch {
	case dependencies.ListPinnedEngramSummaries == nil:
		return errChatContextDependenciesIncomplete
	case dependencies.ListPinnedDocuments == nil:
		return errChatContextDependenciesIncomplete
	case dependencies.QueryEngrams == nil:
		return errChatContextDependenciesIncomplete
	case dependencies.GetRehydrationBundle == nil:
		return errChatContextDependenciesIncomplete
	case dependencies.QueryDocumentChunks == nil:
		return errChatContextDependenciesIncomplete
	default:
		return nil
	}
}

func normalizeChatContextRequest(request ChatContextRequest) ChatContextRequest {
	if request.MaxEngrams <= 0 {
		request.MaxEngrams = defaultChatContextMaxEngrams
	}
	if request.RetrievalTopK <= 0 {
		request.RetrievalTopK = defaultChatContextRetrievalTop
	}
	if request.DocumentTopK <= 0 {
		request.DocumentTopK = defaultChatContextDocumentTop
	}
	request = normalizeLinkRecallOptions(request)
	request = normalizeLinkNoiseOptions(request)
	return request
}

func normalizeLinkRecallOptions(request ChatContextRequest) ChatContextRequest {
	if request.LinkRecallEnabled == nil {
		request.LinkRecallEnabled = boolPointer(true)
	}
	if request.LinkRecallDepth == nil {
		request.LinkRecallDepth = intPointer(defaultLinkRecallDepth)
	}
	depth := clamp(*request.LinkRecallDepth, 1, maxLinkRecallDepth)
	request.LinkRecallDepth = intPointer(depth)
	if request.LinkRecallMaxNeighbors == nil {
		request.LinkRecallMaxNeighbors = intPointer(defaultLinkRecallMaxNeighbors)
	}
	maxNeighbors := clamp(*request.LinkRecallMaxNeighbors, 1, maxLinkRecallMaxNeighbors)
	request.LinkRecallMaxNeighbors = intPointer(maxNeighbors)
	return request
}

func normalizeLinkNoiseOptions(request ChatContextRequest) ChatContextRequest {
	if request.LinkNoiseSuppressionEnabled == nil {
		request.LinkNoiseSuppressionEnabled = boolPointer(true)
	}
	if request.LinkNoiseScoreThreshold == nil {
		request.LinkNoiseScoreThreshold = floatPointer(defaultLinkNoiseScoreThreshold)
	}
	threshold := clampFloat(*request.LinkNoiseScoreThreshold, 0.0, 1.0)
	request.LinkNoiseScoreThreshold = floatPointer(threshold)
	return request
}

func chatContextChunkBudget(documentTopK int, pinnedChunkCount int) int {
	budget := max(documentTopK, pinnedChunkCount)
	return min(budget, maxChatContextDocumentBudget)
}

func collectBundleEngramIDs(bundles []models.RehydrationBundle) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(bundles))
	for _, bundle := range bundles {
		ids = append(ids, bundle.EngramID)
	}
	return ids
}

func collectPinnedDocumentIDs(records []models.PinnedDocumentRecord) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.DocumentID)
	}
	return ids
}

func floatPointer(value float64) *float64 {
	copied := value
	return &copied
}

func collectPinnedDocumentChunks(
	ctx context.Context,
	dependencies ChatContextDependencies,
	request ChatContextRequest,
	pinnedDocumentIDs []uuid.UUID,
) ([]models.DocumentChunkQueryResult, error) {
	chunks := make([]models.DocumentChunkQueryResult, 0, len(pinnedDocumentIDs))
	for _, documentID := range dedupeUUIDsPreserveOrder(pinnedDocumentIDs) {
		scoped, err := dependencies.QueryDocumentChunks(
			ctx,
			request.ActorUserID,
			models.DocumentChunkQueryRequest{
				Query:       request.UserQuery,
				ProjectID:   &request.Session.ProjectID,
				DocumentIDs: []uuid.UUID{documentID},
				TopK:        1,
			},
			request.EmbeddingDim,
		)
		if err != nil {
			return nil, err
		}
		if len(scoped) == 0 {
			continue
		}
		chunks = append(chunks, scoped[0])
	}
	return chunks, nil
}

func dedupeUUIDsPreserveOrder(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	ordered := make([]uuid.UUID, 0, len(ids))
	for _, item := range ids {
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		ordered = append(ordered, item)
	}
	return ordered
}

func dedupeDocumentChunksPreserveOrder(chunks []models.DocumentChunkQueryResult) []models.DocumentChunkQueryResult {
	seen := make(map[uuid.UUID]struct{}, len(chunks))
	ordered := make([]models.DocumentChunkQueryResult, 0, len(chunks))
	for _, chunk := range chunks {
		if _, exists := seen[chunk.ChunkID]; exists {
			continue
		}
		seen[chunk.ChunkID] = struct{}{}
		ordered = append(ordered, chunk)
	}
	return ordered
}

func truncateDocumentChunks(chunks []models.DocumentChunkQueryResult, budget int) []models.DocumentChunkQueryResult {
	if budget <= 0 || len(chunks) <= budget {
		return chunks
	}
	return chunks[:budget]
}

func collectSelectedChunkIDs(chunks []models.DocumentChunkQueryResult) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(chunks))
	for _, chunk := range chunks {
		ids = append(ids, chunk.ChunkID)
	}
	return ids
}

func selectContextEngramIDs(
	maxEngrams int,
	pinned []models.EngramSummary,
	retrieved []models.EngramQueryResult,
) []uuid.UUID {
	candidateIDs := make([]uuid.UUID, 0, len(pinned)+len(retrieved))
	for _, item := range pinned {
		candidateIDs = append(candidateIDs, item.EngramID)
	}
	for _, item := range retrieved {
		candidateIDs = append(candidateIDs, item.EngramID)
	}
	ordered := dedupeUUIDsPreserveOrder(candidateIDs)
	if len(ordered) <= maxEngrams {
		return ordered
	}
	return ordered[:maxEngrams]
}

func collectRehydrationBundles(
	ctx context.Context,
	dependencies ChatContextDependencies,
	input rehydrationBundleCollectInput,
) ([]models.RehydrationBundle, error) {
	if input.Limit <= 0 {
		return []models.RehydrationBundle{}, nil
	}
	dedupedEngramIDs := dedupeUUIDsPreserveOrder(input.EngramIDs)
	bundles := make([]models.RehydrationBundle, 0, min(input.Limit, len(dedupedEngramIDs)))
	for _, engramID := range dedupedEngramIDs {
		if len(bundles) >= input.Limit {
			break
		}
		bundle, err := dependencies.GetRehydrationBundle(ctx, engramID, input.ActorUserID)
		if err != nil {
			return nil, err
		}
		if bundle == nil {
			continue
		}
		bundles = append(bundles, *bundle)
	}
	return bundles, nil
}

func formatListSection(lines []string, fallback string) string {
	if len(lines) == 0 {
		return fallback
	}
	return strings.Join(lines, "\n")
}

func truncateDetailedExcerpt(detailedSummaryMarkdown string, maxChars int) string {
	detailedExcerpt := strings.TrimSpace(detailedSummaryMarkdown)
	if detailedExcerpt == "" {
		return "- None"
	}
	if len([]rune(detailedExcerpt)) <= maxChars {
		return detailedExcerpt
	}
	if maxChars <= 3 {
		return string([]rune(detailedExcerpt)[:maxChars])
	}
	return strings.TrimSpace(string([]rune(detailedExcerpt)[:maxChars-3])) + "..."
}

func citationLine(bundle models.RehydrationBundle, index int) string {
	citation := bundle.TopCitations[index]
	return fmt.Sprintf(
		"- %s (%s): %s",
		citationTitleOrURL(citation),
		citation.URL,
		truncateTextNoEllipsis(citation.Snippet, 140),
	)
}

func citationTitleOrURL(citation models.RehydrationCitation) string {
	if citation.Title == nil || strings.TrimSpace(*citation.Title) == "" {
		return citation.URL
	}
	return *citation.Title
}

func truncateTextNoEllipsis(value string, maxChars int) string {
	runes := []rune(value)
	if maxChars <= 0 || len(runes) <= maxChars {
		return value
	}
	return string(runes[:maxChars])
}

func bundleSection(bundle models.RehydrationBundle) string {
	detailedExcerpt := truncateDetailedExcerpt(bundle.DetailedSummaryMarkdown, 1200)
	decisions := formatListSection(bundleDecisionLines(bundle.KeyDecisions), "- None")
	questions := formatListSection(bundleQuestionLines(bundle.OpenQuestions), "- None")
	citations := formatListSection(bundleCitationLines(bundle), "- None")
	return fmt.Sprintf(
		"## %s (%s)\nSummary: %s\n\nDetailed notes excerpt:\n%s\n\nKey decisions:\n%s\n\nOpen questions:\n%s\n\nTop citations:\n%s",
		bundle.Title,
		bundle.EngramID,
		bundle.CompactSummary,
		detailedExcerpt,
		decisions,
		questions,
		citations,
	)
}

func bundleDecisionLines(decisions []map[string]any) []string {
	limit := min(3, len(decisions))
	lines := make([]string, 0, limit)
	for index := range limit {
		decision := strings.TrimSpace(anyToString(decisions[index]["decision"]))
		rationale := strings.TrimSpace(anyToString(decisions[index]["rationale"]))
		lines = append(lines, fmt.Sprintf("- %s: %s", decision, rationale))
	}
	return lines
}

func bundleQuestionLines(questions []string) []string {
	limit := min(3, len(questions))
	lines := make([]string, 0, limit)
	for index := range limit {
		lines = append(lines, "- "+questions[index])
	}
	return lines
}

func bundleCitationLines(bundle models.RehydrationBundle) []string {
	limit := min(3, len(bundle.TopCitations))
	lines := make([]string, 0, limit)
	for index := range limit {
		lines = append(lines, citationLine(bundle, index))
	}
	return lines
}

func anyToString(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func collectBundleSourceReferences(bundles []models.RehydrationBundle) []ChatSourceReference {
	references := make([]ChatSourceReference, 0, min(12, len(bundles)*3))
	seen := make(map[string]struct{})
	for _, bundle := range bundles {
		limit := min(3, len(bundle.TopCitations))
		for index := range limit {
			citation := bundle.TopCitations[index]
			key := strings.ToLower(strings.TrimSpace(citation.URL))
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}

			snippet := strings.ReplaceAll(strings.TrimSpace(citation.Snippet), "\n", " ")
			snippet = truncateSnippetWithEllipsis(snippet, 240)
			references = append(
				references,
				ChatSourceReference{
					SourceType:  engramSourceType,
					EngramID:    bundle.EngramID,
					EngramTitle: bundle.Title,
					URL:         citation.URL,
					Title:       citation.Title,
					Snippet:     snippet,
					CapturedAt:  citation.CapturedAt,
				},
			)
			if len(references) >= 12 {
				return references
			}
		}
	}
	return references
}

func truncateSnippetWithEllipsis(snippet string, maxChars int) string {
	runes := []rune(snippet)
	if len(runes) <= maxChars {
		return snippet
	}
	if maxChars <= 3 {
		return string(runes[:maxChars])
	}
	return string(runes[:maxChars-3]) + "..."
}

func collectDocumentSourceReferences(chunks []models.DocumentChunkQueryResult) []ChatSourceReference {
	references := make([]ChatSourceReference, 0, min(8, len(chunks)))
	seenDocuments := make(map[uuid.UUID]struct{})
	for _, chunk := range chunks {
		if _, exists := seenDocuments[chunk.DocumentID]; exists {
			continue
		}
		seenDocuments[chunk.DocumentID] = struct{}{}

		title := resolveDocumentReferenceTitle(chunk)
		chunkIndex := chunk.ChunkIndex
		references = append(
			references,
			ChatSourceReference{
				SourceType:  documentSourceType,
				EngramID:    chunk.DocumentID,
				EngramTitle: chunk.Title,
				URL:         fmt.Sprintf("document://%s", chunk.DocumentID),
				Title:       &title,
				Snippet:     chunk.Snippet,
				CapturedAt:  chunk.CreatedAt,
				DocumentID:  uuidPtr(chunk.DocumentID),
				ChunkID:     uuidPtr(chunk.ChunkID),
				ChunkIndex:  &chunkIndex,
			},
		)
		if len(references) >= 8 {
			break
		}
	}
	return references
}

func resolveDocumentReferenceTitle(chunk models.DocumentChunkQueryResult) string {
	if chunk.SourceName != nil && strings.TrimSpace(*chunk.SourceName) != "" {
		return *chunk.SourceName
	}
	return fmt.Sprintf("%s · chunk %d", chunk.Title, chunk.ChunkIndex)
}

func dedupeSourceReferences(references []ChatSourceReference, limit int) []ChatSourceReference {
	seen := make(map[string]struct{}, len(references))
	deduped := make([]ChatSourceReference, 0, min(limit, len(references)))
	for _, reference := range references {
		key := fmt.Sprintf(
			"%s|%s",
			strings.ToLower(strings.TrimSpace(reference.SourceType)),
			strings.ToLower(strings.TrimSpace(reference.URL)),
		)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, reference)
		if len(deduped) >= limit {
			break
		}
	}
	return deduped
}

func documentChunkSection(chunk models.DocumentChunkQueryResult) string {
	title := chunk.Title
	if chunk.SourceName != nil && strings.TrimSpace(*chunk.SourceName) != "" {
		title = *chunk.SourceName
	}
	return fmt.Sprintf(
		"## Document Chunk: %s (chunk %d)\nDocument ID: %s\nSnippet: %s\nReference: document://%s#chunk=%d",
		title,
		chunk.ChunkIndex,
		chunk.DocumentID,
		chunk.Snippet,
		chunk.DocumentID,
		chunk.ChunkIndex,
	)
}

func buildContextSections(
	bundles []models.RehydrationBundle,
	selectedChunks []models.DocumentChunkQueryResult,
	pinnedChunks []models.DocumentChunkQueryResult,
) []string {
	sections := make([]string, 0)
	sections = appendEngramContextSections(sections, bundles)
	pinnedContextChunks, retrievedContextChunks := splitDocumentChunks(selectedChunks, pinnedChunks)
	sections = appendDocumentContextSections(sections, "# Pinned Document Context", pinnedContextChunks)
	sections = appendDocumentContextSections(sections, "# Document Retrieval Context", retrievedContextChunks)
	return sections
}

func appendEngramContextSections(
	sections []string,
	bundles []models.RehydrationBundle,
) []string {
	if len(bundles) == 0 {
		return sections
	}
	sections = append(sections, "# Engram Retrieval Context")
	for _, bundle := range bundles {
		sections = append(sections, bundleSection(bundle))
	}
	return sections
}

func appendDocumentContextSections(
	sections []string,
	header string,
	chunks []models.DocumentChunkQueryResult,
) []string {
	if len(chunks) == 0 {
		return sections
	}
	sections = append(sections, header)
	for _, chunk := range chunks {
		sections = append(sections, documentChunkSection(chunk))
	}
	return sections
}

func splitDocumentChunks(
	selectedChunks []models.DocumentChunkQueryResult,
	pinnedChunks []models.DocumentChunkQueryResult,
) ([]models.DocumentChunkQueryResult, []models.DocumentChunkQueryResult) {
	pinnedChunkIDs := make(map[uuid.UUID]struct{}, len(pinnedChunks))
	for _, item := range pinnedChunks {
		pinnedChunkIDs[item.ChunkID] = struct{}{}
	}
	pinnedContextChunks := make([]models.DocumentChunkQueryResult, 0)
	retrievedContextChunks := make([]models.DocumentChunkQueryResult, 0)
	for _, item := range selectedChunks {
		if _, exists := pinnedChunkIDs[item.ChunkID]; exists {
			pinnedContextChunks = append(pinnedContextChunks, item)
			continue
		}
		retrievedContextChunks = append(retrievedContextChunks, item)
	}
	return pinnedContextChunks, retrievedContextChunks
}

func uuidPtr(value uuid.UUID) *uuid.UUID {
	copy := value
	return &copy
}

func boolPointer(value bool) *bool {
	copy := value
	return &copy
}

func intPointer(value int) *int {
	copy := value
	return &copy
}

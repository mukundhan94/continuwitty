package graph

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
)

const (
	defaultSuggestionLimit         = 5
	defaultSuggestionMaxCandidates = 20
	maxSuggestionCandidates        = 50
	sourceSuggestionLimit          = 200
)

var (
	linkSuggestionTokenPattern = regexp.MustCompile(`[a-z0-9]{2,}`)
	nowUTC                     = func() time.Time { return time.Now().UTC() }
)

// LinkSuggestionInput captures suggestion controls for one source engram.
type LinkSuggestionInput struct {
	SourceEngramID  uuid.UUID
	ActorUserID     uuid.UUID
	Limit           int
	MaxCandidates   int
	MinimumScore    float64
	IncludeArchived bool
}

type suggestionBounds struct {
	limit         int
	maxCandidates int
}

type relationSignal struct {
	semantic   float64
	shared     float64
	continuity float64
}

type suggestionReasonsInput struct {
	signal        relationSignal
	sharedSources []string
}

type suggestionSourceContext struct {
	bundle          *models.RehydrationBundle
	projectID       string
	sourceSources   []models.EngramSourceRecord
	existingTargets map[uuid.UUID]struct{}
}

// LinkSuggestionService ranks candidate source->target edges for user confirmation.
type LinkSuggestionService struct {
	getRehydrationBundle func(
		ctx context.Context,
		engramID uuid.UUID,
		actorUserID uuid.UUID,
	) (*models.RehydrationBundle, error)
	queryEngrams func(
		ctx context.Context,
		request models.EngramQueryRequest,
		actorUserID uuid.UUID,
	) ([]models.EngramQueryResult, error)
	getEngramSources func(
		ctx context.Context,
		engramID uuid.UUID,
		limit int,
		actorUserID uuid.UUID,
	) ([]models.EngramSourceRecord, error)
	listEngramLinks func(
		ctx context.Context,
		input repository.EngramLinkListInput,
	) ([]models.EngramLinkRecord, error)
	now func() time.Time
}

// NewLinkSuggestionService builds repository-backed link suggestion scoring.
func NewLinkSuggestionService(db repository.Queryer, embeddingDim int) *LinkSuggestionService {
	if db == nil {
		return nil
	}
	return &LinkSuggestionService{
		getRehydrationBundle: func(
			ctx context.Context,
			engramID uuid.UUID,
			actorUserID uuid.UUID,
		) (*models.RehydrationBundle, error) {
			return repository.GetRehydrationBundle(
				ctx,
				db,
				repository.RehydrationInput{
					EngramID:    engramID,
					ActorUserID: &actorUserID,
				},
			)
		},
		queryEngrams: func(
			ctx context.Context,
			request models.EngramQueryRequest,
			actorUserID uuid.UUID,
		) ([]models.EngramQueryResult, error) {
			queryLiteral, err := repository.BuildLocalQueryLiteral(request.Query, embeddingDim)
			if err != nil {
				return nil, err
			}
			return repository.QueryEngrams(
				ctx,
				db,
				repository.QueryEngramsInput{
					Request:      request,
					QueryLiteral: queryLiteral,
					ActorUserID:  &actorUserID,
				},
			)
		},
		getEngramSources: func(
			ctx context.Context,
			engramID uuid.UUID,
			limit int,
			actorUserID uuid.UUID,
		) ([]models.EngramSourceRecord, error) {
			return repository.GetEngramSources(
				ctx,
				db,
				repository.EngramSourceListInput{
					EngramID:    engramID,
					Limit:       limit,
					ActorUserID: &actorUserID,
				},
			)
		},
		listEngramLinks: func(
			ctx context.Context,
			input repository.EngramLinkListInput,
		) ([]models.EngramLinkRecord, error) {
			return repository.ListEngramLinks(ctx, db, input)
		},
		now: nowUTC,
	}
}

// SuggestLinks returns scored candidate links for the source engram.
func (service *LinkSuggestionService) SuggestLinks(
	ctx context.Context,
	input LinkSuggestionInput,
) ([]models.EngramLinkSuggestion, error) {
	if err := validateLinkSuggestionInput(input); err != nil {
		return nil, err
	}
	if service == nil {
		return []models.EngramLinkSuggestion{}, nil
	}
	bounds := resolveSuggestionBounds(input)
	sourceContext, err := service.loadSuggestionSourceContext(ctx, input)
	if err != nil {
		return nil, err
	}
	if sourceContext == nil {
		return []models.EngramLinkSuggestion{}, nil
	}

	queryText := buildSuggestionQueryText(sourceContext.bundle)
	results, err := service.queryEngrams(
		ctx,
		models.EngramQueryRequest{
			Query:     queryText,
			TopK:      bounds.maxCandidates,
			ProjectID: &sourceContext.projectID,
			Tags:      []string{},
			Keywords:  []string{},
		},
		input.ActorUserID,
	)
	if err != nil {
		return nil, err
	}

	suggestions, err := service.rankSuggestions(
		ctx,
		rankSuggestionsInput{
			sourceEngramID:  input.SourceEngramID,
			sourceBundle:    sourceContext.bundle,
			sourceSources:   sourceContext.sourceSources,
			results:         results,
			actorUserID:     input.ActorUserID,
			existingTargets: sourceContext.existingTargets,
			minimumScore:    input.MinimumScore,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(suggestions) > bounds.limit {
		return suggestions[:bounds.limit], nil
	}
	return suggestions, nil
}

func resolveSuggestionBounds(input LinkSuggestionInput) suggestionBounds {
	limit := input.Limit
	if limit <= 0 {
		limit = defaultSuggestionLimit
	}
	if limit > maxSuggestionCandidates {
		limit = maxSuggestionCandidates
	}
	maxCandidates := input.MaxCandidates
	if maxCandidates <= 0 {
		maxCandidates = defaultSuggestionMaxCandidates
	}
	if maxCandidates < limit {
		maxCandidates = limit
	}
	if maxCandidates > maxSuggestionCandidates {
		maxCandidates = maxSuggestionCandidates
	}
	return suggestionBounds{limit: limit, maxCandidates: maxCandidates}
}

func (service *LinkSuggestionService) loadSuggestionSourceContext(
	ctx context.Context,
	input LinkSuggestionInput,
) (*suggestionSourceContext, error) {
	sourceBundle, err := service.getRehydrationBundle(ctx, input.SourceEngramID, input.ActorUserID)
	if err != nil {
		return nil, err
	}
	if sourceBundle == nil {
		return nil, nil
	}
	projectID := strings.TrimSpace(sourceBundle.ProjectID)
	if projectID == "" {
		return nil, nil
	}
	sourceSources, err := service.getEngramSources(
		ctx,
		input.SourceEngramID,
		sourceSuggestionLimit,
		input.ActorUserID,
	)
	if err != nil {
		return nil, err
	}
	existingTargets, err := service.listExistingTargets(
		ctx,
		input.SourceEngramID,
		input.ActorUserID,
		input.IncludeArchived,
	)
	if err != nil {
		return nil, err
	}
	return &suggestionSourceContext{
		bundle:          sourceBundle,
		projectID:       projectID,
		sourceSources:   sourceSources,
		existingTargets: existingTargets,
	}, nil
}

type rankSuggestionsInput struct {
	sourceEngramID  uuid.UUID
	sourceBundle    *models.RehydrationBundle
	sourceSources   []models.EngramSourceRecord
	results         []models.EngramQueryResult
	actorUserID     uuid.UUID
	existingTargets map[uuid.UUID]struct{}
	minimumScore    float64
}

func (service *LinkSuggestionService) rankSuggestions(
	ctx context.Context,
	input rankSuggestionsInput,
) ([]models.EngramLinkSuggestion, error) {
	sourceText := buildSourceText(input.sourceBundle)
	suggestions := make([]models.EngramLinkSuggestion, 0, len(input.results))
	for _, result := range input.results {
		if result.EngramID == input.sourceEngramID {
			continue
		}
		if _, exists := input.existingTargets[result.EngramID]; exists {
			continue
		}
		candidateSources, err := service.getEngramSources(
			ctx,
			result.EngramID,
			sourceSuggestionLimit,
			input.actorUserID,
		)
		if err != nil {
			return nil, err
		}
		suggestion := buildLinkSuggestion(
			buildLinkSuggestionInput{
				sourceEngramID:   input.sourceEngramID,
				projectID:        input.sourceBundle.ProjectID,
				sourceText:       sourceText,
				sourceSources:    input.sourceSources,
				candidate:        result,
				candidateSources: candidateSources,
				now:              service.now(),
			},
		)
		if suggestion.Score < input.minimumScore {
			continue
		}
		suggestions = append(suggestions, suggestion)
	}
	sort.Slice(suggestions, func(left, right int) bool {
		if suggestions[left].Score == suggestions[right].Score {
			return suggestions[left].TargetCreatedAt.After(suggestions[right].TargetCreatedAt)
		}
		return suggestions[left].Score > suggestions[right].Score
	})
	return suggestions, nil
}

func (service *LinkSuggestionService) listExistingTargets(
	ctx context.Context,
	sourceEngramID uuid.UUID,
	actorUserID uuid.UUID,
	includeArchived bool,
) (map[uuid.UUID]struct{}, error) {
	links, err := service.listEngramLinks(
		ctx,
		repository.EngramLinkListInput{
			SourceEngramID:  sourceEngramID,
			ActorUserID:     actorUserID,
			IncludeArchived: includeArchived,
			Limit:           500,
			Offset:          0,
		},
	)
	if err != nil {
		return nil, err
	}
	targets := make(map[uuid.UUID]struct{}, len(links))
	for _, link := range links {
		targets[link.TargetEngramID] = struct{}{}
	}
	return targets, nil
}

func validateLinkSuggestionInput(input LinkSuggestionInput) error {
	if input.SourceEngramID == uuid.Nil {
		return errors.New("source_engram_id is required")
	}
	if input.ActorUserID == uuid.Nil {
		return errors.New("actor_user_id is required")
	}
	if input.MinimumScore < 0 || input.MinimumScore > 1 {
		return errors.New("minimum_score must be between 0 and 1")
	}
	return nil
}

func buildSuggestionQueryText(bundle *models.RehydrationBundle) string {
	parts := []string{
		strings.TrimSpace(bundle.Title),
		strings.TrimSpace(bundle.CompactSummary),
		strings.TrimSpace(bundle.DetailedSummaryMarkdown),
	}
	queryText := strings.Join(parts, " ")
	queryText = strings.Join(strings.Fields(queryText), " ")
	if queryText == "" {
		return "memory context"
	}
	if len(queryText) > 1200 {
		return queryText[:1200]
	}
	return queryText
}

func buildSourceText(bundle *models.RehydrationBundle) string {
	return strings.TrimSpace(
		strings.Join(
			[]string{
				bundle.Title,
				bundle.CompactSummary,
				bundle.DetailedSummaryMarkdown,
			},
			" ",
		),
	)
}

type buildLinkSuggestionInput struct {
	sourceEngramID   uuid.UUID
	projectID        string
	sourceText       string
	sourceSources    []models.EngramSourceRecord
	candidate        models.EngramQueryResult
	candidateSources []models.EngramSourceRecord
	now              time.Time
}

func buildLinkSuggestion(input buildLinkSuggestionInput) models.EngramLinkSuggestion {
	semantic := semanticScore(input.candidate.Distance)
	sharedScore, sharedSources := sharedSourceScore(input.sourceSources, input.candidateSources)
	continuity := lexicalContinuityScore(input.sourceText, buildCandidateText(input.candidate))
	temporal := temporalRecencyScore(input.now, input.candidate.CreatedAt)
	signal := relationSignal{
		semantic:   semantic,
		shared:     sharedScore,
		continuity: continuity,
	}
	score := clampScore((semantic * 0.5) + (sharedScore * 0.25) + (continuity * 0.15) + (temporal * 0.1))
	weight := clampScore((semantic * 0.45) + (sharedScore * 0.2) + (continuity * 0.25) + (temporal * 0.1))
	confidence := clampScore((semantic * 0.4) + (sharedScore * 0.3) + (continuity * 0.2) + (temporal * 0.1))
	relationType := suggestRelationType(signal)
	reasons := buildSuggestionReasons(
		suggestionReasonsInput{
			signal:        signal,
			sharedSources: sharedSources,
		},
	)
	return models.EngramLinkSuggestion{
		SourceEngramID:  input.sourceEngramID,
		TargetEngramID:  input.candidate.EngramID,
		ProjectID:       input.projectID,
		TargetTitle:     input.candidate.Title,
		TargetAbstract:  input.candidate.Abstract,
		TargetCreatedAt: input.candidate.CreatedAt,
		RelationType:    relationType,
		Weight:          weight,
		TemporalWeight:  temporal,
		Confidence:      confidence,
		Score:           score,
		Origin:          models.EngramLinkOriginSuggested,
		Status:          models.EngramLinkStatusSuggested,
		Reasons:         reasons,
		EvidenceJSON: map[string]any{
			"score_components": map[string]any{
				"semantic":   semantic,
				"shared":     sharedScore,
				"continuity": continuity,
				"temporal":   temporal,
				"combined":   score,
			},
			"candidate_distance": input.candidate.Distance,
			"shared_source_urls": sharedSources,
		},
	}
}

func buildCandidateText(candidate models.EngramQueryResult) string {
	return strings.TrimSpace(strings.Join([]string{candidate.Title, candidate.Abstract}, " "))
}

func semanticScore(distance float64) float64 {
	if distance < 0 {
		return 1
	}
	return clampScore(1 / (1 + distance))
}

func sharedSourceScore(
	sourceSources []models.EngramSourceRecord,
	candidateSources []models.EngramSourceRecord,
) (float64, []string) {
	sourceURLs := buildSourceURLSet(sourceSources)
	sharedURLs := make([]string, 0)
	seen := map[string]struct{}{}
	for _, source := range candidateSources {
		key := normalizeURL(source.URL)
		if key == "" {
			continue
		}
		if _, exists := sourceURLs[key]; !exists {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		sharedURLs = append(sharedURLs, source.URL)
	}
	sort.Strings(sharedURLs)
	score := clampScore(float64(len(sharedURLs)) / 3.0)
	return score, sharedURLs
}

func buildSourceURLSet(sources []models.EngramSourceRecord) map[string]struct{} {
	seen := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		key := normalizeURL(source.URL)
		if key == "" {
			continue
		}
		seen[key] = struct{}{}
	}
	return seen
}

func normalizeURL(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func lexicalContinuityScore(sourceText, candidateText string) float64 {
	sourceTokens := tokenize(sourceText)
	candidateTokens := tokenize(candidateText)
	if len(sourceTokens) == 0 || len(candidateTokens) == 0 {
		return 0
	}
	shared := 0
	for token := range sourceTokens {
		if _, exists := candidateTokens[token]; exists {
			shared += 1
		}
	}
	return clampScore(float64(shared) / float64(len(sourceTokens)))
}

func tokenize(text string) map[string]struct{} {
	matches := linkSuggestionTokenPattern.FindAllString(strings.ToLower(text), -1)
	tokens := make(map[string]struct{}, len(matches))
	for _, token := range matches {
		tokens[token] = struct{}{}
	}
	return tokens
}

func temporalRecencyScore(now time.Time, createdAt time.Time) float64 {
	age := now.Sub(createdAt)
	if age <= 0 {
		return 1
	}
	ageDays := age.Hours() / 24
	return clampScore(1 / (1 + (ageDays / 90)))
}

func suggestRelationType(signal relationSignal) models.EngramLinkRelationType {
	switch {
	case signal.shared >= 0.66:
		return models.EngramLinkRelationDerivedFrom
	case signal.semantic >= 0.75 && signal.continuity >= 0.35:
		return models.EngramLinkRelationSupports
	default:
		return models.EngramLinkRelationRelatedTo
	}
}

func buildSuggestionReasons(input suggestionReasonsInput) []string {
	reasons := make([]string, 0, 3)
	if input.signal.semantic >= 0.5 {
		reasons = append(reasons, fmt.Sprintf("semantic_overlap=%.2f", input.signal.semantic))
	}
	if len(input.sharedSources) > 0 {
		reasons = append(reasons, fmt.Sprintf("shared_sources=%d", len(input.sharedSources)))
	}
	if input.signal.continuity >= 0.2 {
		reasons = append(reasons, fmt.Sprintf("continuity_overlap=%.2f", input.signal.continuity))
	}
	if len(reasons) == 0 {
		return []string{"low_signal_candidate"}
	}
	return reasons
}

func clampScore(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

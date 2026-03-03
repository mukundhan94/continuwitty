package main

import (
	"context"
	"strings"
	"sync"
	"time"

	"engram/internal/chat"
	"engram/internal/graph"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultScheduledLinkHygieneInterval       = 24 * time.Hour
	defaultScheduledLinkHygieneMaxAutoArchive = 12
	defaultLinkCurationSuggestionListLimit    = 500
	hygieneAutoArchiveActionStaleLowValue     = "archive_stale_low_value"
)

type linkHygieneRunTracker struct {
	mu              sync.Mutex
	lastRunBySource map[uuid.UUID]time.Time
}

func newLinkHygieneRunTracker() *linkHygieneRunTracker {
	return &linkHygieneRunTracker{
		lastRunBySource: make(map[uuid.UUID]time.Time),
	}
}

func (tracker *linkHygieneRunTracker) shouldRun(
	sourceEngramID uuid.UUID,
	now time.Time,
	interval time.Duration,
) bool {
	if tracker == nil || sourceEngramID == uuid.Nil {
		return false
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	lastRunAt, exists := tracker.lastRunBySource[sourceEngramID]
	if !exists {
		return true
	}
	return now.Sub(lastRunAt) >= interval
}

func (tracker *linkHygieneRunTracker) markRun(sourceEngramID uuid.UUID, now time.Time) {
	if tracker == nil || sourceEngramID == uuid.Nil {
		return
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	tracker.lastRunBySource[sourceEngramID] = now
}

type scheduledLinkHygieneExecutor struct {
	interval       time.Duration
	maxAutoArchive int
	tracker        *linkHygieneRunTracker
	recommend      func(
		ctx context.Context,
		input graph.LinkHygieneInput,
	) ([]models.EngramLinkHygieneRecommendation, error)
	archive func(
		ctx context.Context,
		input repository.EngramLinkArchiveInput,
	) (*models.EngramLinkRecord, error)
	resolveSourceProjectID      func(ctx context.Context, sourceEngramID uuid.UUID) (*string, error)
	listLinkCurationSuggestions func(
		ctx context.Context,
		projectID string,
	) ([]models.MemoryCurationSuggestion, error)
	createMemoryCurationSuggestion func(
		ctx context.Context,
		input repository.MemoryCurationSuggestionCreateInput,
	) (*models.MemoryCurationSuggestion, error)
}

type linkCurationPersistenceInput struct {
	projectID       string
	sourceEngramID  uuid.UUID
	recommendations []models.EngramLinkHygieneRecommendation
	suggestedAt     time.Time
	dedupedKeys     map[string]struct{}
}

type linkReinforcementRequest struct {
	ctx         context.Context
	pool        *pgxpool.Pool
	actorUserID uuid.UUID
	now         time.Time
}

func newScheduledLinkHygieneExecutor(pool *pgxpool.Pool) *scheduledLinkHygieneExecutor {
	hygieneService := graph.NewLinkHygieneService(pool)
	if hygieneService == nil {
		return nil
	}
	return &scheduledLinkHygieneExecutor{
		interval:       defaultScheduledLinkHygieneInterval,
		maxAutoArchive: defaultScheduledLinkHygieneMaxAutoArchive,
		tracker:        newLinkHygieneRunTracker(),
		recommend:      hygieneService.Recommend,
		archive: func(
			ctx context.Context,
			input repository.EngramLinkArchiveInput,
		) (*models.EngramLinkRecord, error) {
			return repository.ArchiveEngramLink(ctx, pool, input)
		},
		resolveSourceProjectID: func(ctx context.Context, sourceEngramID uuid.UUID) (*string, error) {
			record, err := repository.GetAdminEngram(ctx, pool, sourceEngramID, false)
			if err != nil || record == nil {
				return nil, err
			}
			projectID := strings.TrimSpace(record.ProjectID)
			if projectID == "" {
				return nil, nil
			}
			return &projectID, nil
		},
		listLinkCurationSuggestions: func(
			ctx context.Context,
			projectID string,
		) ([]models.MemoryCurationSuggestion, error) {
			suggestionType := models.MemoryCurationSuggestionTypeLink
			status := models.MemoryCurationSuggestionStatusSuggested
			return repository.ListMemoryCurationSuggestions(
				ctx,
				pool,
				repository.MemoryCurationSuggestionListInput{
					ProjectID:      &projectID,
					SuggestionType: &suggestionType,
					Status:         &status,
					Limit:          defaultLinkCurationSuggestionListLimit,
					Offset:         0,
				},
			)
		},
		createMemoryCurationSuggestion: func(
			ctx context.Context,
			input repository.MemoryCurationSuggestionCreateInput,
		) (*models.MemoryCurationSuggestion, error) {
			return repository.CreateMemoryCurationSuggestion(ctx, pool, input)
		},
	}
}

func (executor *scheduledLinkHygieneExecutor) execute(
	ctx context.Context,
	actorUserID uuid.UUID,
	sourceEngramIDs []uuid.UUID,
	now time.Time,
) error {
	if executor == nil {
		return nil
	}
	for _, sourceEngramID := range dedupeUUIDs(sourceEngramIDs) {
		if err := executor.executeForSource(ctx, actorUserID, sourceEngramID, now); err != nil {
			return err
		}
	}
	return nil
}

func (executor *scheduledLinkHygieneExecutor) executeForSource(
	ctx context.Context,
	actorUserID uuid.UUID,
	sourceEngramID uuid.UUID,
	now time.Time,
) error {
	if !executor.tracker.shouldRun(sourceEngramID, now, executor.interval) {
		return nil
	}
	recommendations, err := executor.recommendForSource(ctx, actorUserID, sourceEngramID)
	if err != nil {
		return err
	}
	if err := executor.archiveRecommendedLinks(ctx, actorUserID, recommendations); err != nil {
		return err
	}
	if err := executor.persistLinkCurationSuggestions(ctx, sourceEngramID, recommendations, now); err != nil {
		return err
	}
	executor.tracker.markRun(sourceEngramID, now)
	return nil
}

func (executor *scheduledLinkHygieneExecutor) recommendForSource(
	ctx context.Context,
	actorUserID uuid.UUID,
	sourceEngramID uuid.UUID,
) ([]models.EngramLinkHygieneRecommendation, error) {
	return executor.recommend(
		ctx,
		graph.LinkHygieneInput{
			SourceEngramID:  sourceEngramID,
			ActorUserID:     actorUserID,
			IncludeArchived: false,
		},
	)
}

func (executor *scheduledLinkHygieneExecutor) archiveRecommendedLinks(
	ctx context.Context,
	actorUserID uuid.UUID,
	recommendations []models.EngramLinkHygieneRecommendation,
) error {
	for _, linkID := range selectAutoArchiveLinkIDs(recommendations, executor.maxAutoArchive) {
		if _, err := executor.archive(
			ctx,
			repository.EngramLinkArchiveInput{
				LinkID:      linkID,
				ActorUserID: actorUserID,
			},
		); err != nil {
			return err
		}
	}
	return nil
}

func (executor *scheduledLinkHygieneExecutor) persistLinkCurationSuggestions(
	ctx context.Context,
	sourceEngramID uuid.UUID,
	recommendations []models.EngramLinkHygieneRecommendation,
	suggestedAt time.Time,
) error {
	if !executor.hasLinkCurationDependencies() {
		return nil
	}
	projectID, err := executor.resolveSourceProjectID(ctx, sourceEngramID)
	if err != nil {
		return err
	}
	if projectID == nil {
		return nil
	}
	dedupedKeys, err := executor.loadLinkCurationDedupKeys(ctx, *projectID)
	if err != nil {
		return err
	}
	return executor.createPendingLinkCurationSuggestions(
		ctx,
		linkCurationPersistenceInput{
			projectID:       *projectID,
			sourceEngramID:  sourceEngramID,
			recommendations: recommendations,
			suggestedAt:     suggestedAt,
			dedupedKeys:     dedupedKeys,
		},
	)
}

func (executor *scheduledLinkHygieneExecutor) hasLinkCurationDependencies() bool {
	if executor == nil {
		return false
	}
	if executor.resolveSourceProjectID == nil {
		return false
	}
	if executor.listLinkCurationSuggestions == nil {
		return false
	}
	if executor.createMemoryCurationSuggestion == nil {
		return false
	}
	return true
}

func (executor *scheduledLinkHygieneExecutor) loadLinkCurationDedupKeys(
	ctx context.Context,
	projectID string,
) (map[string]struct{}, error) {
	existing, err := executor.listLinkCurationSuggestions(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return existingLinkCurationDedupKeys(existing), nil
}

func (executor *scheduledLinkHygieneExecutor) createPendingLinkCurationSuggestions(
	ctx context.Context,
	input linkCurationPersistenceInput,
) error {
	for _, recommendation := range input.recommendations {
		createInput, dedupKey, shouldCreate := linkCurationSuggestionInputFromRecommendation(
			input.projectID,
			input.sourceEngramID,
			recommendation,
			input.suggestedAt,
		)
		if !shouldCreate {
			continue
		}
		if _, exists := input.dedupedKeys[dedupKey]; exists {
			continue
		}
		if _, err := executor.createMemoryCurationSuggestion(
			ctx,
			createInput,
		); err != nil {
			return err
		}
		input.dedupedKeys[dedupKey] = struct{}{}
	}
	return nil
}

func reinforceEngramLinksDependency(
	pool *pgxpool.Pool,
) func(ctx context.Context, actorUserID uuid.UUID, linkIDs []uuid.UUID) error {
	hygieneExecutor := newScheduledLinkHygieneExecutor(pool)
	return func(ctx context.Context, actorUserID uuid.UUID, linkIDs []uuid.UUID) error {
		request := linkReinforcementRequest{
			ctx:         ctx,
			pool:        pool,
			actorUserID: actorUserID,
			now:         time.Now().UTC(),
		}
		sourceEngramIDs, err := reinforceResolvedLinks(
			request,
			dedupeUUIDs(linkIDs),
		)
		if err != nil {
			return err
		}
		if err := hygieneExecutor.execute(
			request.ctx,
			request.actorUserID,
			sourceEngramIDs,
			request.now,
		); err != nil {
			return err
		}
		return nil
	}
}

func reinforceResolvedLinks(
	request linkReinforcementRequest,
	linkIDs []uuid.UUID,
) ([]uuid.UUID, error) {
	sourceEngramIDs := make([]uuid.UUID, 0, len(linkIDs))
	for _, linkID := range linkIDs {
		sourceEngramID, err := reinforceSingleEngramLink(request, linkID)
		if err != nil {
			return nil, err
		}
		if sourceEngramID != uuid.Nil {
			sourceEngramIDs = append(sourceEngramIDs, sourceEngramID)
		}
	}
	return sourceEngramIDs, nil
}

func reinforceSingleEngramLink(
	request linkReinforcementRequest,
	linkID uuid.UUID,
) (uuid.UUID, error) {
	link, err := repository.GetEngramLink(
		request.ctx,
		request.pool,
		repository.EngramLinkGetInput{
			LinkID:          linkID,
			ActorUserID:     request.actorUserID,
			IncludeArchived: true,
		},
	)
	if err != nil {
		return uuid.Nil, err
	}
	if link == nil {
		return uuid.Nil, nil
	}
	if !linkIsReinforceable(*link) {
		return link.SourceEngramID, nil
	}
	if err := updateReinforcedEngramLink(request, linkID, *link); err != nil {
		return uuid.Nil, err
	}
	return link.SourceEngramID, nil
}

func linkIsReinforceable(link models.EngramLinkRecord) bool {
	return link.Status != models.EngramLinkStatusArchived &&
		link.Status != models.EngramLinkStatusRejected
}

func updateReinforcedEngramLink(
	request linkReinforcementRequest,
	linkID uuid.UUID,
	link models.EngramLinkRecord,
) error {
	temporalWeight := chat.ReinforcedLinkTemporalWeight(request.now, link)
	status := statusForReinforcedLink(link)
	_, err := repository.UpdateEngramLink(
		request.ctx,
		request.pool,
		repository.EngramLinkUpdateInput{
			LinkID:           linkID,
			ActorUserID:      request.actorUserID,
			TemporalWeight:   &temporalWeight,
			Status:           status,
			LastReinforcedAt: &request.now,
		},
	)
	return err
}

func selectAutoArchiveLinkIDs(
	recommendations []models.EngramLinkHygieneRecommendation,
	maxActions int,
) []uuid.UUID {
	limit := normalizeAutoArchiveLimit(maxActions)
	selected := make([]uuid.UUID, 0, min(limit, len(recommendations)))
	seen := make(map[uuid.UUID]struct{}, limit)
	for _, recommendation := range recommendations {
		if !shouldAutoArchiveRecommendation(recommendation) || len(recommendation.LinkIDs) == 0 {
			continue
		}
		linkID := recommendation.LinkIDs[0]
		if linkID == uuid.Nil {
			continue
		}
		if _, exists := seen[linkID]; exists {
			continue
		}
		seen[linkID] = struct{}{}
		selected = append(selected, linkID)
		if len(selected) >= limit {
			break
		}
	}
	return selected
}

func shouldAutoArchiveRecommendation(
	recommendation models.EngramLinkHygieneRecommendation,
) bool {
	return strings.EqualFold(
		strings.TrimSpace(recommendation.SuggestedAction),
		hygieneAutoArchiveActionStaleLowValue,
	)
}

func normalizeAutoArchiveLimit(value int) int {
	if value <= 0 {
		return defaultScheduledLinkHygieneMaxAutoArchive
	}
	return value
}

func statusForReinforcedLink(
	link models.EngramLinkRecord,
) *models.EngramLinkStatus {
	if link.Status == models.EngramLinkStatusSuggested {
		active := models.EngramLinkStatusActive
		return &active
	}
	return nil
}

func dedupeUUIDs(values []uuid.UUID) []uuid.UUID {
	if len(values) <= 1 {
		return append([]uuid.UUID(nil), values...)
	}
	seen := make(map[uuid.UUID]struct{}, len(values))
	deduped := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	return deduped
}

func existingLinkCurationDedupKeys(
	suggestions []models.MemoryCurationSuggestion,
) map[string]struct{} {
	keys := make(map[string]struct{}, len(suggestions))
	for _, suggestion := range suggestions {
		linkID, _ := payloadStringValue(suggestion.PayloadJSON, "link_id")
		action, _ := payloadStringValue(suggestion.PayloadJSON, "suggested_action")
		targetID, _ := payloadStringValue(suggestion.PayloadJSON, "target_engram_id")
		if linkID == "" || action == "" {
			continue
		}
		keys[linkCurationDedupKey(linkID, targetID, action)] = struct{}{}
	}
	return keys
}

func linkCurationSuggestionInputFromRecommendation(
	projectID string,
	sourceEngramID uuid.UUID,
	recommendation models.EngramLinkHygieneRecommendation,
	suggestedAt time.Time,
) (repository.MemoryCurationSuggestionCreateInput, string, bool) {
	if shouldAutoArchiveRecommendation(recommendation) {
		return repository.MemoryCurationSuggestionCreateInput{}, "", false
	}
	linkID := firstNonNilUUID(recommendation.LinkIDs)
	if linkID == uuid.Nil {
		return repository.MemoryCurationSuggestionCreateInput{}, "", false
	}
	dedupKey := linkCurationRecommendationDedupKey(recommendation, linkID)
	return repository.MemoryCurationSuggestionCreateInput{
		ProjectID:      projectID,
		SuggestionType: models.MemoryCurationSuggestionTypeLink,
		Reason:         linkCurationReason(recommendation),
		Recommendation: linkCurationRecommendationText(recommendation),
		PayloadJSON: linkCurationPayload(
			sourceEngramID,
			recommendation,
			linkID,
		),
		ConfidenceScore: linkCurationConfidenceScore(recommendation),
		SuggestedAt:     suggestedAt,
	}, dedupKey, true
}

func linkCurationRecommendationDedupKey(
	recommendation models.EngramLinkHygieneRecommendation,
	linkID uuid.UUID,
) string {
	return linkCurationDedupKey(
		linkID.String(),
		recommendation.TargetEngramID.String(),
		recommendation.SuggestedAction,
	)
}

func linkCurationDedupKey(
	linkID string,
	targetEngramID string,
	suggestedAction string,
) string {
	return strings.Join(
		[]string{
			strings.TrimSpace(strings.ToLower(linkID)),
			strings.TrimSpace(strings.ToLower(targetEngramID)),
			strings.TrimSpace(strings.ToLower(suggestedAction)),
		},
		"|",
	)
}

func firstNonNilUUID(values []uuid.UUID) uuid.UUID {
	for _, value := range values {
		if value != uuid.Nil {
			return value
		}
	}
	return uuid.Nil
}

func linkCurationReason(recommendation models.EngramLinkHygieneRecommendation) string {
	detail := strings.TrimSpace(recommendation.Detail)
	if detail != "" {
		return detail
	}
	return "Graph hygiene recommendation identified a link quality issue."
}

func linkCurationRecommendationText(recommendation models.EngramLinkHygieneRecommendation) string {
	action := strings.TrimSpace(recommendation.SuggestedAction)
	if action == "" {
		return "Review link hygiene recommendation."
	}
	return "Review and apply link hygiene action: " + action + "."
}

func linkCurationPayload(
	sourceEngramID uuid.UUID,
	recommendation models.EngramLinkHygieneRecommendation,
	linkID uuid.UUID,
) map[string]any {
	return map[string]any{
		"source_engram_id": sourceEngramID.String(),
		"target_engram_id": recommendation.TargetEngramID.String(),
		"link_id":          linkID.String(),
		"suggested_action": recommendation.SuggestedAction,
		"category":         recommendation.Category,
		"severity":         recommendation.Severity,
		"score":            recommendation.Score,
		"detail":           recommendation.Detail,
	}
}

func linkCurationConfidenceScore(recommendation models.EngramLinkHygieneRecommendation) float64 {
	severity := strings.TrimSpace(strings.ToLower(recommendation.Severity))
	switch severity {
	case "high":
		return 0.9
	case "medium":
		return 0.75
	default:
		return 0.6
	}
}

func payloadStringValue(payload map[string]any, key string) (string, bool) {
	if payload == nil {
		return "", false
	}
	rawValue, exists := payload[key]
	if !exists {
		return "", false
	}
	value, ok := rawValue.(string)
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

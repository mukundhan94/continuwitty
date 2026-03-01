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

func reinforceEngramLinksDependency(
	pool *pgxpool.Pool,
) func(ctx context.Context, actorUserID uuid.UUID, linkIDs []uuid.UUID) error {
	hygieneExecutor := newScheduledLinkHygieneExecutor(pool)
	return func(ctx context.Context, actorUserID uuid.UUID, linkIDs []uuid.UUID) error {
		now := time.Now().UTC()
		sourceEngramIDs, err := reinforceResolvedLinks(
			ctx,
			pool,
			actorUserID,
			dedupeUUIDs(linkIDs),
			now,
		)
		if err != nil {
			return err
		}
		if err := hygieneExecutor.execute(ctx, actorUserID, sourceEngramIDs, now); err != nil {
			return err
		}
		return nil
	}
}

func reinforceResolvedLinks(
	ctx context.Context,
	pool *pgxpool.Pool,
	actorUserID uuid.UUID,
	linkIDs []uuid.UUID,
	now time.Time,
) ([]uuid.UUID, error) {
	sourceEngramIDs := make([]uuid.UUID, 0, len(linkIDs))
	for _, linkID := range linkIDs {
		sourceEngramID, err := reinforceSingleEngramLink(ctx, pool, actorUserID, linkID, now)
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
	ctx context.Context,
	pool *pgxpool.Pool,
	actorUserID uuid.UUID,
	linkID uuid.UUID,
	now time.Time,
) (uuid.UUID, error) {
	link, err := repository.GetEngramLink(
		ctx,
		pool,
		repository.EngramLinkGetInput{
			LinkID:          linkID,
			ActorUserID:     actorUserID,
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
	if err := updateReinforcedEngramLink(ctx, pool, actorUserID, linkID, *link, now); err != nil {
		return uuid.Nil, err
	}
	return link.SourceEngramID, nil
}

func linkIsReinforceable(link models.EngramLinkRecord) bool {
	return link.Status != models.EngramLinkStatusArchived &&
		link.Status != models.EngramLinkStatusRejected
}

func updateReinforcedEngramLink(
	ctx context.Context,
	pool *pgxpool.Pool,
	actorUserID uuid.UUID,
	linkID uuid.UUID,
	link models.EngramLinkRecord,
	now time.Time,
) error {
	temporalWeight := chat.ReinforcedLinkTemporalWeight(now, link)
	status := statusForReinforcedLink(link)
	_, err := repository.UpdateEngramLink(
		ctx,
		pool,
		repository.EngramLinkUpdateInput{
			LinkID:           linkID,
			ActorUserID:      actorUserID,
			TemporalWeight:   &temporalWeight,
			Status:           status,
			LastReinforcedAt: &now,
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

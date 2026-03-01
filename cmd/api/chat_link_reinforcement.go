package main

import (
	"context"
	"time"

	"engram/internal/chat"
	"engram/internal/models"
	"engram/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func reinforceEngramLinksDependency(
	pool *pgxpool.Pool,
) func(ctx context.Context, actorUserID uuid.UUID, linkIDs []uuid.UUID) error {
	return func(ctx context.Context, actorUserID uuid.UUID, linkIDs []uuid.UUID) error {
		now := time.Now().UTC()
		for _, linkID := range dedupeUUIDs(linkIDs) {
			link, err := repository.GetEngramLink(
				ctx,
				pool,
				repository.EngramLinkGetInput{
					LinkID:          linkID,
					ActorUserID:     actorUserID,
					IncludeArchived: true,
				},
			)
			if err != nil || link == nil {
				if err != nil {
					return err
				}
				continue
			}
			if link.Status == models.EngramLinkStatusArchived || link.Status == models.EngramLinkStatusRejected {
				continue
			}
			temporalWeight := chat.ReinforcedLinkTemporalWeight(now, *link)
			status := statusForReinforcedLink(*link)
			updated, err := repository.UpdateEngramLink(
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
			if err != nil {
				return err
			}
			if updated == nil {
				continue
			}
		}
		return nil
	}
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

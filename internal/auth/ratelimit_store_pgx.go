package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGXRateLimitStore persists distributed limiter state in the rate_limit_state table.
type PGXRateLimitStore struct {
	pool *pgxpool.Pool
}

type persistedRateLimitState struct {
	Key       RateLimitStateKey
	State     rateLimitWindowState
	UpdatedAt time.Time
}

// NewPGXRateLimitStore creates a DB-backed distributed limiter store.
func NewPGXRateLimitStore(pool *pgxpool.Pool) *PGXRateLimitStore {
	return &PGXRateLimitStore{pool: pool}
}

// MutateState acquires a row lock, invokes mutator, and optionally saves updated state.
func (store *PGXRateLimitStore) MutateState(
	ctx context.Context,
	key RateLimitStateKey,
	now time.Time,
	mutator RateLimitStateMutator,
) error {
	if err := store.validate(true, mutator); err != nil {
		return err
	}
	nowUTC := now.UTC()
	return store.withTransaction(ctx, func(tx pgx.Tx) error {
		if err := ensureRateLimitStateRow(ctx, tx, key, nowUTC); err != nil {
			return err
		}
		state, err := loadLockedRateLimitState(ctx, tx, key)
		if err != nil {
			return err
		}
		next, persist, err := mutator(state)
		if err != nil {
			return err
		}
		if !persist {
			return nil
		}
		return saveRateLimitState(
			ctx,
			tx,
			persistedRateLimitState{
				Key:       key,
				State:     next,
				UpdatedAt: nowUTC,
			},
		)
	})
}

// DeleteState removes one persisted limiter key.
func (store *PGXRateLimitStore) DeleteState(ctx context.Context, key RateLimitStateKey) error {
	return store.deleteByQuery(
		ctx,
		`DELETE FROM rate_limit_state
		WHERE namespace = $1 AND rate_key = $2`,
		key.Namespace,
		key.RateKey,
	)
}

// ClearNamespace removes all persisted limiter entries for one namespace.
func (store *PGXRateLimitStore) ClearNamespace(ctx context.Context, namespace string) error {
	return store.deleteByQuery(
		ctx,
		`DELETE FROM rate_limit_state
		WHERE namespace = $1`,
		namespace,
	)
}

func (store *PGXRateLimitStore) deleteByQuery(ctx context.Context, query string, args ...any) error {
	if err := store.validate(false, nil); err != nil {
		return err
	}
	_, err := store.pool.Exec(ctx, query, args...)
	return err
}

func (store *PGXRateLimitStore) validate(requireMutator bool, mutator RateLimitStateMutator) error {
	if store == nil || store.pool == nil {
		return errors.New("rate limit store pool is not configured")
	}
	if requireMutator && mutator == nil {
		return errors.New("rate limit mutator is required")
	}
	return nil
}

func (store *PGXRateLimitStore) withTransaction(ctx context.Context, operation func(tx pgx.Tx) error) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	if err := operation(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func ensureRateLimitStateRow(ctx context.Context, tx pgx.Tx, key RateLimitStateKey, now time.Time) error {
	_, err := tx.Exec(
		ctx,
		`INSERT INTO rate_limit_state (
			namespace,
			rate_key,
			event_count,
			window_started_at,
			blocked_until,
			updated_at
		) VALUES ($1, $2, 0, $3, NULL, $4)
		ON CONFLICT (namespace, rate_key) DO NOTHING`,
		key.Namespace,
		key.RateKey,
		now,
		now,
	)
	return err
}

func loadLockedRateLimitState(ctx context.Context, tx pgx.Tx, key RateLimitStateKey) (rateLimitWindowState, error) {
	var (
		state        rateLimitWindowState
		blockedUntil *time.Time
	)
	err := tx.QueryRow(
		ctx,
		`SELECT event_count, window_started_at, blocked_until
		FROM rate_limit_state
		WHERE namespace = $1 AND rate_key = $2
		FOR UPDATE`,
		key.Namespace,
		key.RateKey,
	).Scan(&state.EventCount, &state.WindowStartedAt, &blockedUntil)
	if err != nil {
		return rateLimitWindowState{}, err
	}
	state.WindowStartedAt = state.WindowStartedAt.UTC()
	if blockedUntil != nil {
		state.BlockedUntil = blockedUntil.UTC()
	}
	return state, nil
}

func saveRateLimitState(ctx context.Context, tx pgx.Tx, persisted persistedRateLimitState) error {
	var blockedUntil *time.Time
	if !persisted.State.BlockedUntil.IsZero() {
		value := persisted.State.BlockedUntil.UTC()
		blockedUntil = &value
	}
	_, err := tx.Exec(
		ctx,
		`UPDATE rate_limit_state
		SET
			event_count = $1,
			window_started_at = $2,
			blocked_until = $3,
			updated_at = $4
		WHERE namespace = $5 AND rate_key = $6`,
		persisted.State.EventCount,
		persisted.State.WindowStartedAt.UTC(),
		blockedUntil,
		persisted.UpdatedAt.UTC(),
		persisted.Key.Namespace,
		persisted.Key.RateKey,
	)
	return err
}

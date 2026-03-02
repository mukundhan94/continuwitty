package auth

import (
	"context"
	"strings"
	"time"
)

const (
	// DefaultLoginAttemptNamespace stores login lockout state in rate_limit_state.
	DefaultLoginAttemptNamespace = "login_attempts"
)

// RateLimitStateKey identifies one row in rate_limit_state.
type RateLimitStateKey struct {
	Namespace string
	RateKey   string
}

type rateLimitWindowState struct {
	EventCount      int
	WindowStartedAt time.Time
	BlockedUntil    time.Time
}

// RateLimitStateMutator updates distributed limiter state while holding a DB row lock.
type RateLimitStateMutator func(state rateLimitWindowState) (next rateLimitWindowState, persist bool, err error)

// DistributedRateLimitStore persists limiter state so lockouts survive process restarts.
type DistributedRateLimitStore interface {
	MutateState(ctx context.Context, key RateLimitStateKey, now time.Time, mutator RateLimitStateMutator) error
	DeleteState(ctx context.Context, key RateLimitStateKey) error
	ClearNamespace(ctx context.Context, namespace string) error
}

func normalizeDistributedWindowState(
	state rateLimitWindowState,
	now time.Time,
	window time.Duration,
	resetCountOnBlockExpiry bool,
) rateLimitWindowState {
	if state.WindowStartedAt.IsZero() || now.Sub(state.WindowStartedAt) >= window {
		return rateLimitWindowState{
			EventCount:      0,
			WindowStartedAt: now,
		}
	}
	if !state.BlockedUntil.IsZero() && !now.Before(state.BlockedUntil) {
		if resetCountOnBlockExpiry {
			return rateLimitWindowState{
				EventCount:      0,
				WindowStartedAt: now,
			}
		}
		return rateLimitWindowState{
			EventCount:      state.EventCount,
			WindowStartedAt: state.WindowStartedAt,
		}
	}
	return state
}

func sameRateLimitWindowState(left, right rateLimitWindowState) bool {
	return left.EventCount == right.EventCount &&
		left.WindowStartedAt.Equal(right.WindowStartedAt) &&
		left.BlockedUntil.Equal(right.BlockedUntil)
}

func retryAfterSeconds(blockedUntil time.Time, now time.Time) int {
	seconds := int(blockedUntil.Sub(now).Seconds())
	if seconds < 1 {
		return 1
	}
	return seconds
}

func maxInt(value int, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}

func normalizeNamespace(value string, fallback string) string {
	namespace := strings.TrimSpace(value)
	if namespace != "" {
		return namespace
	}
	return fallback
}

func distributedOrFallback(
	distributed func() (bool, int, error),
	fallback func() (bool, int),
) (bool, int) {
	allowed, retryAfter, err := distributed()
	if err == nil {
		return allowed, retryAfter
	}
	return fallback()
}

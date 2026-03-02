package auth

import (
	"context"
	"sync"
	"time"
)

type requestWindowState struct {
	eventCount      int
	windowStartedAt time.Time
	blockedUntil    time.Time
}

// RequestRateLimiter tracks request thresholds within a rolling window.
type RequestRateLimiter struct {
	maxRequests   int
	window        time.Duration
	blockDuration time.Duration
	namespace     string
	now           func() time.Time
	store         DistributedRateLimitStore

	mutex sync.Mutex
	state map[string]requestWindowState
}

// RequestRateLimiterOptions configures RequestRateLimiter construction.
type RequestRateLimiterOptions struct {
	MaxRequests   int
	WindowSeconds int
	BlockSeconds  int
	Namespace     string
	Now           func() time.Time
	Store         DistributedRateLimitStore
}

// NewRequestRateLimiter creates a rate limiter using the system clock.
func NewRequestRateLimiter(
	maxRequests int,
	windowSeconds int,
	blockSeconds int,
	namespace string,
) *RequestRateLimiter {
	return NewRequestRateLimiterWithClock(
		maxRequests,
		windowSeconds,
		blockSeconds,
		namespace,
		time.Now,
	)
}

// NewRequestRateLimiterWithClock creates a rate limiter with a caller-provided clock.
func NewRequestRateLimiterWithClock(
	maxRequests int,
	windowSeconds int,
	blockSeconds int,
	namespace string,
	now func() time.Time,
) *RequestRateLimiter {
	return NewRequestRateLimiterWithOptions(RequestRateLimiterOptions{
		MaxRequests:   maxRequests,
		WindowSeconds: windowSeconds,
		BlockSeconds:  blockSeconds,
		Namespace:     namespace,
		Now:           now,
	})
}

// NewRequestRateLimiterWithOptions creates a limiter with optional distributed store support.
func NewRequestRateLimiterWithOptions(options RequestRateLimiterOptions) *RequestRateLimiter {
	if options.Now == nil {
		options.Now = time.Now
	}
	return &RequestRateLimiter{
		maxRequests:   maxInt(options.MaxRequests, 1),
		window:        time.Duration(maxInt(options.WindowSeconds, 1)) * time.Second,
		blockDuration: time.Duration(maxInt(options.BlockSeconds, 0)) * time.Second,
		namespace:     normalizeNamespace(options.Namespace, "request_rate_limiter"),
		now:           options.Now,
		store:         options.Store,
		state:         map[string]requestWindowState{},
	}
}

// SetDistributedStore enables distributed limiter state persistence for this namespace.
func (limiter *RequestRateLimiter) SetDistributedStore(store DistributedRateLimitStore) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	limiter.store = store
}

// Reset clears in-memory limiter state.
func (limiter *RequestRateLimiter) Reset() {
	limiter.mutex.Lock()
	limiter.state = map[string]requestWindowState{}
	limiter.mutex.Unlock()
	if limiter.store != nil {
		_ = limiter.store.ClearNamespace(context.Background(), limiter.namespace)
	}
}

func (limiter *RequestRateLimiter) rateLimitStateKey(key string) RateLimitStateKey {
	return RateLimitStateKey{
		Namespace: limiter.namespace,
		RateKey:   key,
	}
}

func (limiter *RequestRateLimiter) consumeDistributed(key string, now time.Time) (bool, int, error) {
	allowed := true
	retryAfter := 0
	err := limiter.store.MutateState(
		context.Background(),
		limiter.rateLimitStateKey(key),
		now,
		func(state rateLimitWindowState) (rateLimitWindowState, bool, error) {
			normalized := normalizeDistributedWindowState(state, now, limiter.window, true)
			if !normalized.BlockedUntil.IsZero() && now.Before(normalized.BlockedUntil) {
				allowed = false
				retryAfter = retryAfterSeconds(normalized.BlockedUntil, now)
				return normalized, false, nil
			}

			requestCount := normalized.EventCount + 1
			blockedUntil := normalized.BlockedUntil
			if requestCount > limiter.maxRequests {
				blockedUntil = limiter.blockedUntilFor(normalized.WindowStartedAt, now)
				allowed = false
				retryAfter = retryAfterSeconds(blockedUntil, now)
			}

			next := rateLimitWindowState{
				EventCount:      requestCount,
				WindowStartedAt: normalized.WindowStartedAt,
				BlockedUntil:    blockedUntil,
			}
			return next, true, nil
		},
	)
	if err != nil {
		return false, 0, err
	}
	return allowed, retryAfter, nil
}

func (limiter *RequestRateLimiter) consumeLocal(key string, now time.Time) (bool, int) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	state := limiter.normalizeState(key, now)
	if !state.blockedUntil.IsZero() && now.Before(state.blockedUntil) {
		limiter.state[key] = state
		return false, retryAfterSeconds(state.blockedUntil, now)
	}
	state.eventCount++
	if state.eventCount <= limiter.maxRequests {
		limiter.state[key] = state
		return true, 0
	}
	state.blockedUntil = limiter.blockedUntilFor(state.windowStartedAt, now)
	limiter.state[key] = state
	return false, retryAfterSeconds(state.blockedUntil, now)
}

// Consume records an event and reports if request is allowed with retry-after on block.
func (limiter *RequestRateLimiter) Consume(key string) (bool, int) {
	now := limiter.now().UTC()
	if limiter.store != nil {
		return distributedOrFallback(
			func() (bool, int, error) {
				return limiter.consumeDistributed(key, now)
			},
			func() (bool, int) {
				return limiter.consumeLocal(key, now)
			},
		)
	}
	return limiter.consumeLocal(key, now)
}

func (limiter *RequestRateLimiter) normalizeState(key string, now time.Time) requestWindowState {
	state, exists := limiter.state[key]
	if !exists || state.windowStartedAt.IsZero() {
		return requestWindowState{windowStartedAt: now}
	}
	if now.Sub(state.windowStartedAt) >= limiter.window {
		return requestWindowState{windowStartedAt: now}
	}
	if !state.blockedUntil.IsZero() && !now.Before(state.blockedUntil) {
		return requestWindowState{windowStartedAt: now}
	}
	return state
}

func (limiter *RequestRateLimiter) blockedUntilFor(windowStartedAt time.Time, now time.Time) time.Time {
	if limiter.blockDuration > 0 {
		return now.Add(limiter.blockDuration)
	}
	return windowStartedAt.Add(limiter.window)
}

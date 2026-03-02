package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fixedClock struct {
	current time.Time
}

func (clock *fixedClock) now() time.Time {
	return clock.current
}

type inMemoryDistributedStore struct {
	mutex sync.Mutex
	state map[RateLimitStateKey]rateLimitWindowState

	mutateErr error
	deleteErr error
	clearErr  error
}

func newInMemoryDistributedStore() *inMemoryDistributedStore {
	return &inMemoryDistributedStore{
		state: map[RateLimitStateKey]rateLimitWindowState{},
	}
}

func (store *inMemoryDistributedStore) MutateState(
	_ context.Context,
	key RateLimitStateKey,
	now time.Time,
	mutator RateLimitStateMutator,
) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if store.mutateErr != nil {
		return store.mutateErr
	}
	state, exists := store.state[key]
	if !exists || state.WindowStartedAt.IsZero() {
		state = rateLimitWindowState{
			WindowStartedAt: now.UTC(),
		}
	}
	next, persist, err := mutator(state)
	if err != nil {
		return err
	}
	if persist {
		store.state[key] = normalizeTestRateLimitState(next)
	}
	return nil
}

func (store *inMemoryDistributedStore) DeleteState(_ context.Context, key RateLimitStateKey) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if store.deleteErr != nil {
		return store.deleteErr
	}
	delete(store.state, key)
	return nil
}

func (store *inMemoryDistributedStore) ClearNamespace(_ context.Context, namespace string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if store.clearErr != nil {
		return store.clearErr
	}
	for key := range store.state {
		if key.Namespace == namespace {
			delete(store.state, key)
		}
	}
	return nil
}

func normalizeTestRateLimitState(state rateLimitWindowState) rateLimitWindowState {
	state.WindowStartedAt = state.WindowStartedAt.UTC()
	if !state.BlockedUntil.IsZero() {
		state.BlockedUntil = state.BlockedUntil.UTC()
	}
	return state
}

func TestRegisterFailureLocksAfterMaxAttempts(t *testing.T) {
	clock := &fixedClock{current: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	guard := NewLoginAttemptGuardWithClock(2, 60, 30, clock.now)
	key := "admin@example"

	guard.RegisterFailure(key)
	allowed, seconds := guard.Check(key)
	if !allowed || seconds != 0 {
		t.Fatalf("expected first failure to remain allowed, got allowed=%v seconds=%d", allowed, seconds)
	}

	clock.current = clock.current.Add(time.Second)
	guard.RegisterFailure(key)
	allowed, seconds = guard.Check(key)
	if allowed {
		t.Fatalf("expected key to be locked after threshold failures")
	}
	if seconds < 1 || seconds > 30 {
		t.Fatalf("expected retry seconds between 1 and 30, got %d", seconds)
	}
}

func TestCheckUnlocksAfterLockoutExpiry(t *testing.T) {
	clock := &fixedClock{current: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	guard := NewLoginAttemptGuardWithClock(1, 60, 10, clock.now)
	key := "admin@example"

	guard.RegisterFailure(key)
	if allowed, _ := guard.Check(key); allowed {
		t.Fatalf("expected key to be locked immediately")
	}

	clock.current = clock.current.Add(11 * time.Second)
	if allowed, seconds := guard.Check(key); !allowed || seconds != 0 {
		t.Fatalf("expected lockout to expire, got allowed=%v seconds=%d", allowed, seconds)
	}
}

func TestFailureWindowDropsStaleAttempts(t *testing.T) {
	clock := &fixedClock{current: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	guard := NewLoginAttemptGuardWithClock(2, 5, 20, clock.now)
	key := "admin@example"

	guard.RegisterFailure(key)
	clock.current = clock.current.Add(6 * time.Second)
	guard.RegisterFailure(key)
	if allowed, seconds := guard.Check(key); !allowed || seconds != 0 {
		t.Fatalf("expected stale failure to be dropped, got allowed=%v seconds=%d", allowed, seconds)
	}

	guard.RegisterFailure(key)
	allowed, seconds := guard.Check(key)
	if allowed {
		t.Fatalf("expected guard to lock after consecutive failures inside window")
	}
	if seconds < 1 || seconds > 20 {
		t.Fatalf("expected retry seconds between 1 and 20, got %d", seconds)
	}
}

func TestRegisterSuccessClearsPriorFailures(t *testing.T) {
	clock := &fixedClock{current: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	guard := NewLoginAttemptGuardWithClock(2, 60, 30, clock.now)
	key := "admin@example"

	guard.RegisterFailure(key)
	guard.RegisterSuccess(key)
	if allowed, seconds := guard.Check(key); !allowed || seconds != 0 {
		t.Fatalf("expected prior failures to be cleared, got allowed=%v seconds=%d", allowed, seconds)
	}
}

func TestLoginAttemptGuardDistributedStateSharedAcrossInstances(t *testing.T) {
	clock := &fixedClock{current: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	store := newInMemoryDistributedStore()
	guardPrimary := NewLoginAttemptGuardWithClock(2, 60, 30, clock.now)
	guardPrimary.SetDistributedStore(DefaultLoginAttemptNamespace, store)
	guardSecondary := NewLoginAttemptGuardWithClock(2, 60, 30, clock.now)
	guardSecondary.SetDistributedStore(DefaultLoginAttemptNamespace, store)
	key := "admin@example"

	guardPrimary.RegisterFailure(key)
	clock.current = clock.current.Add(time.Second)
	guardSecondary.RegisterFailure(key)

	allowed, seconds := guardPrimary.Check(key)
	if allowed {
		t.Fatalf("expected distributed lockout state to block on second instance")
	}
	if seconds < 1 || seconds > 30 {
		t.Fatalf("expected retry seconds between 1 and 30, got %d", seconds)
	}

	guardSecondary.RegisterSuccess(key)
	allowed, seconds = guardPrimary.Check(key)
	if !allowed || seconds != 0 {
		t.Fatalf("expected distributed success reset, got allowed=%v seconds=%d", allowed, seconds)
	}
}

func TestLoginAttemptGuardFallsBackToLocalStateWhenDistributedStoreFails(t *testing.T) {
	clock := &fixedClock{current: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	store := newInMemoryDistributedStore()
	store.mutateErr = errors.New("database unavailable")
	store.deleteErr = errors.New("database unavailable")
	guard := NewLoginAttemptGuardWithClock(2, 60, 30, clock.now)
	guard.SetDistributedStore(DefaultLoginAttemptNamespace, store)
	key := "admin@example"

	guard.RegisterFailure(key)
	guard.RegisterFailure(key)
	allowed, seconds := guard.Check(key)
	if allowed {
		t.Fatalf("expected local fallback lockout when distributed store fails")
	}
	if seconds < 1 || seconds > 30 {
		t.Fatalf("expected retry seconds between 1 and 30, got %d", seconds)
	}

	guard.RegisterSuccess(key)
	allowed, seconds = guard.Check(key)
	if !allowed || seconds != 0 {
		t.Fatalf("expected local fallback state clear, got allowed=%v seconds=%d", allowed, seconds)
	}
}

type requestRateLimitCase struct {
	name          string
	maxRequests   int
	windowSeconds int
	blockSeconds  int
	waitSeconds   int
	maxRetryAfter int
}

func assertConsumeAllowed(t *testing.T, limiter *RequestRateLimiter, key string, contextMessage string) {
	t.Helper()
	allowed, seconds := limiter.Consume(key)
	if !allowed || seconds != 0 {
		t.Fatalf("%s: expected request to pass, got allowed=%v seconds=%d", contextMessage, allowed, seconds)
	}
}

type consumeBlockExpectation struct {
	maxRetryAfter int
	context       string
}

func assertConsumeBlocked(
	t *testing.T,
	limiter *RequestRateLimiter,
	key string,
	expectation consumeBlockExpectation,
) {
	t.Helper()
	allowed, retryAfter := limiter.Consume(key)
	if allowed {
		t.Fatalf("%s: expected request to be blocked", expectation.context)
	}
	if retryAfter < 1 || retryAfter > expectation.maxRetryAfter {
		t.Fatalf(
			"%s: expected retry_after between 1 and %d, got %d",
			expectation.context,
			expectation.maxRetryAfter,
			retryAfter,
		)
	}
}

func runRequestRateLimiterThresholdCase(t *testing.T, testCase requestRateLimitCase) {
	t.Helper()
	clock := &fixedClock{current: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	limiter := NewRequestRateLimiterWithClock(
		testCase.maxRequests,
		testCase.windowSeconds,
		testCase.blockSeconds,
		"test-mcp",
		clock.now,
	)
	key := "127.0.0.1:session"

	assertConsumeAllowed(t, limiter, key, "first request")
	if testCase.maxRequests > 1 {
		assertConsumeAllowed(t, limiter, key, "second request")
	}
	assertConsumeBlocked(t, limiter, key, consumeBlockExpectation{
		maxRetryAfter: testCase.maxRetryAfter,
		context:       "threshold request",
	})

	clock.current = clock.current.Add(time.Duration(testCase.waitSeconds) * time.Second)
	assertConsumeAllowed(t, limiter, key, "post-wait request")
}

func TestRequestRateLimiterThresholdBehavior(t *testing.T) {
	testCases := []requestRateLimitCase{
		{
			name:          "fixed block duration",
			maxRequests:   2,
			windowSeconds: 60,
			blockSeconds:  15,
			waitSeconds:   16,
			maxRetryAfter: 15,
		},
		{
			name:          "window based block duration",
			maxRequests:   1,
			windowSeconds: 10,
			blockSeconds:  0,
			waitSeconds:   11,
			maxRetryAfter: 10,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			runRequestRateLimiterThresholdCase(t, testCase)
		})
	}
}

func TestRequestRateLimiterDistributedStateSharedAcrossInstances(t *testing.T) {
	clock := &fixedClock{current: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	store := newInMemoryDistributedStore()
	limiterPrimary := NewRequestRateLimiterWithClock(1, 60, 15, "test-mcp", clock.now)
	limiterPrimary.SetDistributedStore(store)
	limiterSecondary := NewRequestRateLimiterWithClock(1, 60, 15, "test-mcp", clock.now)
	limiterSecondary.SetDistributedStore(store)
	key := "127.0.0.1:session"

	assertConsumeAllowed(t, limiterPrimary, key, "distributed first request")
	assertConsumeBlocked(t, limiterSecondary, key, consumeBlockExpectation{
		maxRetryAfter: 15,
		context:       "distributed threshold request",
	})
}

func TestRequestRateLimiterFallsBackToLocalStateWhenDistributedStoreFails(t *testing.T) {
	clock := &fixedClock{current: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	store := newInMemoryDistributedStore()
	store.mutateErr = errors.New("database unavailable")
	limiter := NewRequestRateLimiterWithClock(1, 60, 10, "test-mcp", clock.now)
	limiter.SetDistributedStore(store)
	key := "127.0.0.1:session"

	assertConsumeAllowed(t, limiter, key, "fallback first request")
	assertConsumeBlocked(t, limiter, key, consumeBlockExpectation{
		maxRetryAfter: 10,
		context:       "fallback threshold request",
	})
}

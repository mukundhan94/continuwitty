package auth

import (
	"sync"
	"time"
)

type loginWindowState struct {
	failures     []time.Time
	lockoutUntil time.Time
}

// LoginAttemptGuard tracks repeated login failures and temporary lockout windows.
type LoginAttemptGuard struct {
	maxAttempts int
	window      time.Duration
	lockout     time.Duration
	now         func() time.Time

	mutex sync.Mutex
	state map[string]loginWindowState
}

// NewLoginAttemptGuard creates a guard that uses the system clock.
func NewLoginAttemptGuard(maxAttempts int, windowSeconds int, lockoutSeconds int) *LoginAttemptGuard {
	return NewLoginAttemptGuardWithClock(maxAttempts, windowSeconds, lockoutSeconds, time.Now)
}

// NewLoginAttemptGuardWithClock creates a guard using a caller-provided clock.
func NewLoginAttemptGuardWithClock(
	maxAttempts int,
	windowSeconds int,
	lockoutSeconds int,
	now func() time.Time,
) *LoginAttemptGuard {
	if now == nil {
		now = time.Now
	}
	return &LoginAttemptGuard{
		maxAttempts: maxInt(maxAttempts, 1),
		window:      time.Duration(maxInt(windowSeconds, 1)) * time.Second,
		lockout:     time.Duration(maxInt(lockoutSeconds, 1)) * time.Second,
		now:         now,
		state:       map[string]loginWindowState{},
	}
}

// Reset clears all tracked state for the guard.
func (guard *LoginAttemptGuard) Reset() {
	guard.mutex.Lock()
	defer guard.mutex.Unlock()
	guard.state = map[string]loginWindowState{}
}

// Check returns whether the key is currently allowed, and any retry-after seconds if blocked.
func (guard *LoginAttemptGuard) Check(key string) (bool, int) {
	guard.mutex.Lock()
	defer guard.mutex.Unlock()
	now := guard.now().UTC()
	state := guard.state[key]
	state.failures = activeFailures(state.failures, now, guard.window)
	if !state.lockoutUntil.IsZero() {
		if now.Before(state.lockoutUntil) {
			guard.state[key] = state
			return false, retryAfterSeconds(state.lockoutUntil, now)
		}
		state.lockoutUntil = time.Time{}
	}
	if len(state.failures) == 0 {
		delete(guard.state, key)
		return true, 0
	}
	guard.state[key] = state
	return true, 0
}

// RegisterSuccess clears failures for the key.
func (guard *LoginAttemptGuard) RegisterSuccess(key string) {
	guard.mutex.Lock()
	defer guard.mutex.Unlock()
	delete(guard.state, key)
}

// RegisterFailure records a failed attempt and applies lockout when threshold is reached.
func (guard *LoginAttemptGuard) RegisterFailure(key string) {
	guard.mutex.Lock()
	defer guard.mutex.Unlock()
	now := guard.now().UTC()
	state := guard.state[key]
	state.failures = activeFailures(state.failures, now, guard.window)
	state.failures = append(state.failures, now)
	if len(state.failures) >= guard.maxAttempts {
		state.lockoutUntil = now.Add(guard.lockout)
	}
	guard.state[key] = state
}

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

	mutex sync.Mutex
	state map[string]requestWindowState
}

// NewRequestRateLimiter creates a rate limiter using the system clock.
func NewRequestRateLimiter(
	maxRequests int,
	windowSeconds int,
	blockSeconds int,
	namespace string,
) *RequestRateLimiter {
	return NewRequestRateLimiterWithClock(maxRequests, windowSeconds, blockSeconds, namespace, time.Now)
}

// NewRequestRateLimiterWithClock creates a rate limiter with a caller-provided clock.
func NewRequestRateLimiterWithClock(
	maxRequests int,
	windowSeconds int,
	blockSeconds int,
	namespace string,
	now func() time.Time,
) *RequestRateLimiter {
	if now == nil {
		now = time.Now
	}
	return &RequestRateLimiter{
		maxRequests:   maxInt(maxRequests, 1),
		window:        time.Duration(maxInt(windowSeconds, 1)) * time.Second,
		blockDuration: time.Duration(maxInt(blockSeconds, 0)) * time.Second,
		namespace:     namespace,
		now:           now,
		state:         map[string]requestWindowState{},
	}
}

// Reset clears in-memory limiter state.
func (limiter *RequestRateLimiter) Reset() {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	limiter.state = map[string]requestWindowState{}
}

// Consume records an event and reports if request is allowed with retry-after on block.
func (limiter *RequestRateLimiter) Consume(key string) (bool, int) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	now := limiter.now().UTC()
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

func activeFailures(failures []time.Time, now time.Time, window time.Duration) []time.Time {
	cutoff := now.Add(-window)
	active := make([]time.Time, 0, len(failures))
	for _, attempt := range failures {
		if attempt.UTC().Before(cutoff) {
			continue
		}
		active = append(active, attempt.UTC())
	}
	return active
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

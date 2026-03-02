package auth

import (
	"context"
	"strings"
	"sync"
	"time"
)

type loginWindowState struct {
	failures     []time.Time
	lockoutUntil time.Time
}

type loginAttemptKey string

// LoginAttemptGuard tracks repeated login failures and temporary lockout windows.
type LoginAttemptGuard struct {
	maxAttempts int
	window      time.Duration
	lockout     time.Duration
	namespace   string
	now         func() time.Time
	store       DistributedRateLimitStore

	mutex sync.Mutex
	state map[loginAttemptKey]loginWindowState
}

// LoginAttemptGuardOptions configures LoginAttemptGuard construction.
type LoginAttemptGuardOptions struct {
	MaxAttempts    int
	WindowSeconds  int
	LockoutSeconds int
	Namespace      string
	Now            func() time.Time
	Store          DistributedRateLimitStore
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
	return NewLoginAttemptGuardWithOptions(LoginAttemptGuardOptions{
		MaxAttempts:    maxAttempts,
		WindowSeconds:  windowSeconds,
		LockoutSeconds: lockoutSeconds,
		Namespace:      DefaultLoginAttemptNamespace,
		Now:            now,
	})
}

// NewLoginAttemptGuardWithOptions creates a guard with optional distributed store support.
func NewLoginAttemptGuardWithOptions(options LoginAttemptGuardOptions) *LoginAttemptGuard {
	if options.Now == nil {
		options.Now = time.Now
	}
	return &LoginAttemptGuard{
		maxAttempts: maxInt(options.MaxAttempts, 1),
		window:      time.Duration(maxInt(options.WindowSeconds, 1)) * time.Second,
		lockout:     time.Duration(maxInt(options.LockoutSeconds, 1)) * time.Second,
		namespace:   normalizeNamespace(options.Namespace, DefaultLoginAttemptNamespace),
		now:         options.Now,
		store:       options.Store,
		state:       map[loginAttemptKey]loginWindowState{},
	}
}

// SetDistributedStore enables distributed limiter state persistence.
func (guard *LoginAttemptGuard) SetDistributedStore(namespace string, store DistributedRateLimitStore) {
	guard.mutex.Lock()
	defer guard.mutex.Unlock()
	guard.namespace = normalizeNamespace(namespace, DefaultLoginAttemptNamespace)
	guard.store = store
}

// Reset clears all tracked state for the guard.
func (guard *LoginAttemptGuard) Reset() {
	guard.mutex.Lock()
	guard.state = map[loginAttemptKey]loginWindowState{}
	guard.mutex.Unlock()
	if guard.store != nil {
		_ = guard.store.ClearNamespace(context.Background(), guard.namespace)
	}
}

func (guard *LoginAttemptGuard) rateLimitStateKey(key loginAttemptKey) RateLimitStateKey {
	return RateLimitStateKey{
		Namespace: guard.namespace,
		RateKey:   string(key),
	}
}

func (guard *LoginAttemptGuard) checkDistributed(key loginAttemptKey, now time.Time) (bool, int, error) {
	allowed := true
	retryAfter := 0
	err := guard.store.MutateState(
		context.Background(),
		guard.rateLimitStateKey(key),
		now,
		func(state rateLimitWindowState) (rateLimitWindowState, bool, error) {
			normalized := normalizeDistributedWindowState(state, now, guard.window, false)
			if !normalized.BlockedUntil.IsZero() && now.Before(normalized.BlockedUntil) {
				allowed = false
				retryAfter = retryAfterSeconds(normalized.BlockedUntil, now)
			}
			return normalized, !sameRateLimitWindowState(normalized, state), nil
		},
	)
	if err != nil {
		return false, 0, err
	}
	return allowed, retryAfter, nil
}

func (guard *LoginAttemptGuard) checkLocal(key loginAttemptKey, now time.Time) (bool, int) {
	guard.mutex.Lock()
	defer guard.mutex.Unlock()
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

// Check returns whether the key is currently allowed, and any retry-after seconds if blocked.
func (guard *LoginAttemptGuard) Check(key string) (bool, int) {
	now := guard.now().UTC()
	normalizedKey := normalizeLoginAttemptKey(key)
	if guard.store != nil {
		return distributedOrFallback(
			func() (bool, int, error) {
				return guard.checkDistributed(normalizedKey, now)
			},
			func() (bool, int) {
				return guard.checkLocal(normalizedKey, now)
			},
		)
	}
	return guard.checkLocal(normalizedKey, now)
}

// RegisterSuccess clears failures for the key.
func (guard *LoginAttemptGuard) RegisterSuccess(key string) {
	normalizedKey := normalizeLoginAttemptKey(key)
	guard.mutex.Lock()
	delete(guard.state, normalizedKey)
	guard.mutex.Unlock()
	if guard.store != nil {
		_ = guard.store.DeleteState(context.Background(), guard.rateLimitStateKey(normalizedKey))
	}
}

func (guard *LoginAttemptGuard) registerFailureDistributed(key loginAttemptKey, now time.Time) error {
	return guard.store.MutateState(
		context.Background(),
		guard.rateLimitStateKey(key),
		now,
		func(state rateLimitWindowState) (rateLimitWindowState, bool, error) {
			normalized := normalizeDistributedWindowState(state, now, guard.window, false)
			next := normalized
			next.EventCount = normalized.EventCount + 1
			if next.EventCount >= guard.maxAttempts {
				next.BlockedUntil = now.Add(guard.lockout)
			}
			return next, true, nil
		},
	)
}

func (guard *LoginAttemptGuard) registerFailureLocal(key loginAttemptKey, now time.Time) {
	guard.mutex.Lock()
	defer guard.mutex.Unlock()
	state := guard.state[key]
	state.failures = activeFailures(state.failures, now, guard.window)
	state.failures = append(state.failures, now)
	if len(state.failures) >= guard.maxAttempts {
		state.lockoutUntil = now.Add(guard.lockout)
	}
	guard.state[key] = state
}

// RegisterFailure records a failed attempt and applies lockout when threshold is reached.
func (guard *LoginAttemptGuard) RegisterFailure(key string) {
	now := guard.now().UTC()
	normalizedKey := normalizeLoginAttemptKey(key)
	if guard.store != nil {
		if err := guard.registerFailureDistributed(normalizedKey, now); err == nil {
			return
		}
	}
	guard.registerFailureLocal(normalizedKey, now)
}

func normalizeLoginAttemptKey(key string) loginAttemptKey {
	return loginAttemptKey(strings.TrimSpace(key))
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

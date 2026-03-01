package chat

import (
	"strings"
	"sync"
	"time"

	"engram/internal/models"
)

const (
	defaultCircuitFailureThreshold = 3
	defaultCircuitCooldown         = 30 * time.Second
)

var transientProviderErrorCodes = map[string]struct{}{
	"provider_rate_limit":   {},
	"provider_api_error":    {},
	"provider_error":        {},
	"provider_circuit_open": {},
}

// ProviderCircuitPolicy controls whether a provider call is currently allowed.
type ProviderCircuitPolicy interface {
	Allow(provider models.ChatProvider) bool
	RecordResult(provider models.ChatProvider, errorCode string)
}

type noopProviderCircuitPolicy struct{}

func (noopProviderCircuitPolicy) Allow(models.ChatProvider) bool {
	return true
}

func (noopProviderCircuitPolicy) RecordResult(models.ChatProvider, string) {}

// ProviderCircuitPolicyOptions controls circuit breaker thresholds.
type ProviderCircuitPolicyOptions struct {
	Enabled          bool
	FailureThreshold int
	Cooldown         time.Duration
	NowUTC           func() time.Time
}

type circuitProviderState struct {
	consecutiveFailures int
	openUntil           time.Time
}

// SimpleProviderCircuitPolicy is a local in-process circuit breaker policy.
type SimpleProviderCircuitPolicy struct {
	enabled          bool
	failureThreshold int
	cooldown         time.Duration
	nowUTC           func() time.Time
	mutex            sync.Mutex
	byProvider       map[models.ChatProvider]circuitProviderState
}

// NewSimpleProviderCircuitPolicy builds a simple provider circuit policy.
func NewSimpleProviderCircuitPolicy(options ProviderCircuitPolicyOptions) ProviderCircuitPolicy {
	if !options.Enabled {
		return noopProviderCircuitPolicy{}
	}
	threshold := options.FailureThreshold
	if threshold < 1 {
		threshold = defaultCircuitFailureThreshold
	}
	cooldown := options.Cooldown
	if cooldown <= 0 {
		cooldown = defaultCircuitCooldown
	}
	nowUTC := options.NowUTC
	if nowUTC == nil {
		nowUTC = func() time.Time { return time.Now().UTC() }
	}
	return &SimpleProviderCircuitPolicy{
		enabled:          true,
		failureThreshold: threshold,
		cooldown:         cooldown,
		nowUTC:           nowUTC,
		byProvider:       make(map[models.ChatProvider]circuitProviderState),
	}
}

// Allow reports whether the provider call can proceed.
func (policy *SimpleProviderCircuitPolicy) Allow(provider models.ChatProvider) bool {
	if policy == nil || !policy.enabled {
		return true
	}
	policy.mutex.Lock()
	defer policy.mutex.Unlock()

	state := policy.byProvider[provider]
	now := policy.nowUTC()
	if state.openUntil.IsZero() || !state.openUntil.After(now) {
		if !state.openUntil.IsZero() && !state.openUntil.After(now) {
			state.openUntil = time.Time{}
			state.consecutiveFailures = 0
			policy.byProvider[provider] = state
		}
		return true
	}
	return false
}

// RecordResult updates provider circuit state from the latest call outcome.
func (policy *SimpleProviderCircuitPolicy) RecordResult(provider models.ChatProvider, errorCode string) {
	if policy == nil || !policy.enabled {
		return
	}
	policy.mutex.Lock()
	defer policy.mutex.Unlock()

	state := policy.byProvider[provider]
	normalizedCode := normalizeProviderErrorCode(errorCode)
	if normalizedCode == "" {
		state.consecutiveFailures = 0
		state.openUntil = time.Time{}
		policy.byProvider[provider] = state
		return
	}
	if !isTransientProviderErrorCode(normalizedCode) {
		// Non-transient failures should not trip or retain the breaker.
		state.consecutiveFailures = 0
		state.openUntil = time.Time{}
		policy.byProvider[provider] = state
		return
	}
	state.consecutiveFailures++
	if state.consecutiveFailures >= policy.failureThreshold {
		state.openUntil = policy.nowUTC().Add(policy.cooldown)
	}
	policy.byProvider[provider] = state
}

func normalizeProviderErrorCode(errorCode string) string {
	return strings.TrimSpace(errorCode)
}

func isTransientProviderErrorCode(errorCode string) bool {
	_, ok := transientProviderErrorCodes[errorCode]
	return ok
}

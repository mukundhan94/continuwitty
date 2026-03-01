package chat

import (
	"testing"
	"time"

	"engram/internal/models"
)

func TestSimpleProviderCircuitPolicyDisabledAlwaysAllows(t *testing.T) {
	policy := NewSimpleProviderCircuitPolicy(ProviderCircuitPolicyOptions{Enabled: false})
	provider := models.ChatProviderOpenAI

	if !policy.Allow(provider) {
		t.Fatalf("expected disabled policy to allow provider")
	}
	policy.RecordResult(provider, "provider_rate_limit")
	if !policy.Allow(provider) {
		t.Fatalf("expected disabled policy to continue allowing provider")
	}
}

func TestSimpleProviderCircuitPolicyOpensAfterTransientThreshold(t *testing.T) {
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	clock := now
	policy := NewSimpleProviderCircuitPolicy(
		ProviderCircuitPolicyOptions{
			Enabled:          true,
			FailureThreshold: 2,
			Cooldown:         30 * time.Second,
			NowUTC:           func() time.Time { return clock },
		},
	)
	provider := models.ChatProviderOpenAI

	if !policy.Allow(provider) {
		t.Fatalf("expected provider allowed initially")
	}
	policy.RecordResult(provider, "provider_rate_limit")
	if !policy.Allow(provider) {
		t.Fatalf("expected provider allowed before threshold reached")
	}
	policy.RecordResult(provider, "provider_api_error")
	if policy.Allow(provider) {
		t.Fatalf("expected provider blocked after threshold reached")
	}

	clock = clock.Add(31 * time.Second)
	if !policy.Allow(provider) {
		t.Fatalf("expected provider allowed after cooldown expires")
	}
}

func TestSimpleProviderCircuitPolicyDoesNotOpenForNonTransientErrors(t *testing.T) {
	policy := NewSimpleProviderCircuitPolicy(
		ProviderCircuitPolicyOptions{
			Enabled:          true,
			FailureThreshold: 1,
			Cooldown:         time.Minute,
		},
	)
	provider := models.ChatProviderOpenAI

	policy.RecordResult(provider, "provider_request_error")
	if !policy.Allow(provider) {
		t.Fatalf("expected non-transient error not to open breaker")
	}
}

func TestSimpleProviderCircuitPolicyResetsOnSuccess(t *testing.T) {
	policy := NewSimpleProviderCircuitPolicy(
		ProviderCircuitPolicyOptions{
			Enabled:          true,
			FailureThreshold: 1,
			Cooldown:         time.Minute,
		},
	)
	provider := models.ChatProviderOpenAI

	policy.RecordResult(provider, "provider_rate_limit")
	if policy.Allow(provider) {
		t.Fatalf("expected circuit to open")
	}
	policy.RecordResult(provider, "")
	if !policy.Allow(provider) {
		t.Fatalf("expected success to close circuit")
	}
}

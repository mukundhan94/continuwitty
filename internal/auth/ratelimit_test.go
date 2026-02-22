package auth

import (
	"testing"
	"time"
)

type fixedClock struct {
	current time.Time
}

func (clock *fixedClock) now() time.Time {
	return clock.current
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

func TestRequestRateLimiterThresholdBehavior(t *testing.T) {
	testCases := []struct {
		name          string
		maxRequests   int
		windowSeconds int
		blockSeconds  int
		waitSeconds   int
		maxRetryAfter int
	}{
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
			clock := &fixedClock{current: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
			limiter := NewRequestRateLimiterWithClock(
				testCase.maxRequests,
				testCase.windowSeconds,
				testCase.blockSeconds,
				"test-mcp",
				clock.now,
			)
			key := "127.0.0.1:session"

			if allowed, seconds := limiter.Consume(key); !allowed || seconds != 0 {
				t.Fatalf("expected first request to pass, got allowed=%v seconds=%d", allowed, seconds)
			}
			if testCase.maxRequests > 1 {
				if allowed, seconds := limiter.Consume(key); !allowed || seconds != 0 {
					t.Fatalf("expected second request to pass, got allowed=%v seconds=%d", allowed, seconds)
				}
			}

			allowed, retryAfter := limiter.Consume(key)
			if allowed {
				t.Fatalf("expected request to be blocked at threshold")
			}
			if retryAfter < 1 || retryAfter > testCase.maxRetryAfter {
				t.Fatalf(
					"expected retry_after between 1 and %d, got %d",
					testCase.maxRetryAfter,
					retryAfter,
				)
			}

			clock.current = clock.current.Add(time.Duration(testCase.waitSeconds) * time.Second)
			if allowed, seconds := limiter.Consume(key); !allowed || seconds != 0 {
				t.Fatalf("expected request to pass after wait, got allowed=%v seconds=%d", allowed, seconds)
			}
		})
	}
}

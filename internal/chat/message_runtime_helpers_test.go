package chat

import (
	"errors"
	"testing"

	"engram/internal/providers"
)

func TestResolveTokenUsagePrefersProviderTotal(t *testing.T) {
	resolved, estimated := ResolveTokenUsage(
		map[string]int{"input_tokens": 3, "output_tokens": 2, "total_tokens": 5},
		120,
		44,
	)

	requireEqualIntRuntimeHelper(t, 5, resolved["total_tokens"])
	requireEqualIntRuntimeHelper(t, 3, resolved["input_tokens"])
	requireEqualIntRuntimeHelper(t, 2, resolved["output_tokens"])
	if estimated {
		t.Fatalf("expected provider token usage to be used without estimation")
	}
}

func TestResolveTokenUsageEstimatesWhenTotalMissing(t *testing.T) {
	resolved, estimated := ResolveTokenUsage(
		map[string]int{"input_tokens": 3},
		120,
		44,
	)

	if !estimated {
		t.Fatalf("expected estimation when provider total is missing")
	}
	if resolved["total_tokens"] <= 0 {
		t.Fatalf("expected total_tokens > 0, got %d", resolved["total_tokens"])
	}
	if resolved["input_tokens"] <= 0 {
		t.Fatalf("expected input_tokens > 0, got %d", resolved["input_tokens"])
	}
	if resolved["output_tokens"] <= 0 {
		t.Fatalf("expected output_tokens > 0, got %d", resolved["output_tokens"])
	}
}

func TestMapProviderErrorMapsProviderExceptions(t *testing.T) {
	testCases := []struct {
		name           string
		providerErr    error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "request",
			providerErr:    providers.NewProviderRequestError("bad request"),
			expectedStatus: 400,
			expectedCode:   "provider_request_error",
		},
		{
			name:           "rate_limit",
			providerErr:    providers.NewProviderRateLimitError("too many requests"),
			expectedStatus: 429,
			expectedCode:   "provider_rate_limit",
		},
		{
			name:           "auth",
			providerErr:    providers.NewProviderAuthError("missing credentials"),
			expectedStatus: 503,
			expectedCode:   "provider_auth_error",
		},
		{
			name:           "api",
			providerErr:    providers.NewProviderAPIError("provider downstream error"),
			expectedStatus: 502,
			expectedCode:   "provider_api_error",
		},
		{
			name:           "base",
			providerErr:    &providers.ProviderError{Message: "provider generic error", Code: "provider_error"},
			expectedStatus: 502,
			expectedCode:   "provider_error",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mapped := MapProviderError(testCase.providerErr)
			providerErr, ok := mapped.(*ChatProviderExecutionError)
			if !ok {
				t.Fatalf("expected ChatProviderExecutionError, got %T", mapped)
			}
			requireEqualIntRuntimeHelper(t, testCase.expectedStatus, providerErr.StatusCode())
			requireEqualStringRuntimeHelper(t, testCase.expectedCode, providerErr.ErrorCode())
		})
	}
}

func TestMapProviderErrorReturnsOriginalForUnknownError(t *testing.T) {
	original := errors.New("unexpected")
	mapped := MapProviderError(original)
	if mapped != original {
		t.Fatalf("expected unknown error to be returned unchanged")
	}
}

func requireEqualIntRuntimeHelper(t *testing.T, expected int, actual int) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %d, got %d", expected, actual)
	}
}

func requireEqualStringRuntimeHelper(t *testing.T, expected string, actual string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}

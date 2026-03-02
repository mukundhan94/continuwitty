package chat

import "testing"

func TestNewChatServiceErrorDefaultsStatusCodeToBadRequest(t *testing.T) {
	err := NewChatServiceError("invalid payload", 0)
	requireEqualStringChatError(t, "invalid payload", err.Detail())
	requireEqualIntChatError(t, 400, err.StatusCode())
	requireEqualStringChatError(t, "invalid payload", err.Error())
}

func TestNewChatSessionNotFoundErrorDefaultsDetail(t *testing.T) {
	err := NewChatSessionNotFoundError("")
	requireEqualStringChatError(t, "Chat session not found", err.Detail())
	requireEqualIntChatError(t, 404, err.StatusCode())
}

func TestNewChatValidationErrorUsesBadRequestStatus(t *testing.T) {
	err := NewChatValidationError("message cannot be empty")
	requireEqualStringChatError(t, "message cannot be empty", err.Detail())
	requireEqualIntChatError(t, 400, err.StatusCode())
}

func TestNewChatProviderExecutionErrorDefaultsAndOverrides(t *testing.T) {
	defaultErr := NewChatProviderExecutionError("provider timed out", 0, "")
	requireEqualStringChatError(t, "provider timed out", defaultErr.Detail())
	requireEqualIntChatError(t, 502, defaultErr.StatusCode())
	requireEqualStringChatError(t, "provider_error", defaultErr.ErrorCode())

	overrideErr := NewChatProviderExecutionError("upstream rejected request", 503, "upstream_unavailable")
	requireEqualStringChatError(t, "upstream rejected request", overrideErr.Detail())
	requireEqualIntChatError(t, 503, overrideErr.StatusCode())
	requireEqualStringChatError(t, "upstream_unavailable", overrideErr.ErrorCode())
}

func requireEqualStringChatError(t *testing.T, expected string, actual string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}

func requireEqualIntChatError(t *testing.T, expected int, actual int) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %d, got %d", expected, actual)
	}
}

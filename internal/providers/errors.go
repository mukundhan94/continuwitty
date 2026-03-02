package providers

import "fmt"

// ProviderError is a typed base error for provider adapter failures.
type ProviderError struct {
	Message string
	Code    string
}

func (err *ProviderError) Error() string {
	if err == nil {
		return ""
	}
	return err.Message
}

func newProviderError(message string, code string, fallback string) ProviderError {
	normalized := message
	if normalized == "" {
		normalized = fallback
	}
	return ProviderError{
		Message: normalized,
		Code:    code,
	}
}

// ProviderAuthError indicates provider authentication failures.
type ProviderAuthError struct {
	ProviderError
}

func NewProviderAuthError(message string) error {
	return &ProviderAuthError{
		ProviderError: newProviderError(message, "provider_auth_error", "provider authentication failed"),
	}
}

// ProviderRateLimitError indicates provider throttling or quota failures.
type ProviderRateLimitError struct {
	ProviderError
}

func NewProviderRateLimitError(message string) error {
	return &ProviderRateLimitError{
		ProviderError: newProviderError(message, "provider_rate_limit", "provider rate limited"),
	}
}

// ProviderAPIError indicates upstream provider failures.
type ProviderAPIError struct {
	ProviderError
}

func NewProviderAPIError(message string) error {
	return &ProviderAPIError{
		ProviderError: newProviderError(message, "provider_api_error", "provider api error"),
	}
}

// ProviderRequestError indicates invalid request parameters.
type ProviderRequestError struct {
	ProviderError
}

func NewProviderRequestError(message string) error {
	return &ProviderRequestError{
		ProviderError: newProviderError(message, "provider_request_error", "provider request is invalid"),
	}
}

func providerStatusCodeError(providerName string, statusCode int) error {
	switch statusCode {
	case 401, 403:
		return NewProviderAuthError(fmt.Sprintf("%s rejected credentials", providerName))
	case 429:
		return NewProviderRateLimitError(fmt.Sprintf("%s rate limited request", providerName))
	default:
		if statusCode >= 400 {
			return NewProviderAPIError(fmt.Sprintf("%s request failed with status %d", providerName, statusCode))
		}
		return nil
	}
}

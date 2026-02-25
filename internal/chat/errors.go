package chat

import "strings"

const (
	defaultChatServiceStatusCode     = 400
	defaultProviderErrorStatusCode   = 502
	defaultProviderErrorCode         = "provider_error"
	defaultSessionNotFoundDetailText = "Chat session not found"
)

// ChatServiceError captures detail and HTTP mapping for chat service failures.
type ChatServiceError struct {
	detail     string
	statusCode int
}

// NewChatServiceError creates a chat service error with status-code defaults.
func NewChatServiceError(detail string, statusCode int) *ChatServiceError {
	resolvedStatusCode := statusCode
	if resolvedStatusCode <= 0 {
		resolvedStatusCode = defaultChatServiceStatusCode
	}
	return &ChatServiceError{detail: detail, statusCode: resolvedStatusCode}
}

func (err *ChatServiceError) Error() string {
	return err.detail
}

// Detail returns response-ready detail text.
func (err *ChatServiceError) Detail() string {
	return err.detail
}

// StatusCode returns HTTP status mapping for the error.
func (err *ChatServiceError) StatusCode() int {
	return err.statusCode
}

// ChatProviderExecutionError captures provider-specific failure metadata.
type ChatProviderExecutionError struct {
	*ChatServiceError
	errorCode string
}

// NewChatSessionNotFoundError returns a standardized session-not-found error.
func NewChatSessionNotFoundError(detail string) *ChatServiceError {
	resolvedDetail := strings.TrimSpace(detail)
	if resolvedDetail == "" {
		resolvedDetail = defaultSessionNotFoundDetailText
	}
	return NewChatServiceError(resolvedDetail, 404)
}

// NewChatValidationError returns a standardized bad-request validation error.
func NewChatValidationError(detail string) *ChatServiceError {
	return NewChatServiceError(detail, 400)
}

// NewChatProviderExecutionError returns provider failure with default status/code values.
func NewChatProviderExecutionError(
	detail string,
	statusCode int,
	errorCode string,
) *ChatProviderExecutionError {
	resolvedErrorCode := strings.TrimSpace(errorCode)
	if resolvedErrorCode == "" {
		resolvedErrorCode = defaultProviderErrorCode
	}
	resolvedStatusCode := statusCode
	if resolvedStatusCode <= 0 {
		resolvedStatusCode = defaultProviderErrorStatusCode
	}
	return &ChatProviderExecutionError{
		ChatServiceError: NewChatServiceError(detail, resolvedStatusCode),
		errorCode:        resolvedErrorCode,
	}
}

// ErrorCode returns provider error code label used in wire responses.
func (err *ChatProviderExecutionError) ErrorCode() string {
	return err.errorCode
}

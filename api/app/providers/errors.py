from __future__ import annotations


class ProviderError(RuntimeError):
    def __init__(self, message: str, code: str = "provider_error") -> None:
        super().__init__(message)
        self.code = code


class ProviderAuthError(ProviderError):
    def __init__(self, message: str = "provider authentication failed") -> None:
        super().__init__(message, code="provider_auth_error")


class ProviderRateLimitError(ProviderError):
    def __init__(self, message: str = "provider rate limited") -> None:
        super().__init__(message, code="provider_rate_limit")


class ProviderAPIError(ProviderError):
    def __init__(self, message: str = "provider api error") -> None:
        super().__init__(message, code="provider_api_error")

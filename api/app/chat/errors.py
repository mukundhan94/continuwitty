from __future__ import annotations


class ChatServiceError(RuntimeError):
    def __init__(self, detail: str, status_code: int = 400) -> None:
        super().__init__(detail)
        self.detail = detail
        self.status_code = status_code


class ChatSessionNotFoundError(ChatServiceError):
    def __init__(self, detail: str = "Chat session not found") -> None:
        super().__init__(detail=detail, status_code=404)


class ChatValidationError(ChatServiceError):
    def __init__(self, detail: str) -> None:
        super().__init__(detail=detail, status_code=400)


class ChatProviderExecutionError(ChatServiceError):
    def __init__(
        self, detail: str, status_code: int = 502, error_code: str = "provider_error"
    ) -> None:
        super().__init__(detail=detail, status_code=status_code)
        self.error_code = error_code

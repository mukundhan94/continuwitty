class EmbeddingProviderError(RuntimeError):
    """Base error for embedding provider failures."""


class EmbeddingProviderAuthError(EmbeddingProviderError):
    """Raised when embedding credentials are missing or invalid."""


class EmbeddingProviderRequestError(EmbeddingProviderError):
    """Raised for malformed embedding requests."""


class EmbeddingProviderAPIError(EmbeddingProviderError):
    """Raised for upstream embedding provider failures."""

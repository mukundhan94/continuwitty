from __future__ import annotations

from functools import lru_cache

from app.config import Settings, get_settings

from .base import EmbeddingProvider, EmbeddingResult
from .errors import EmbeddingProviderError
from .local_provider import LocalDeterministicEmbeddingProvider
from .openai_provider import OpenAIEmbeddingProvider


class EmbeddingService:
    """Embeds text with a primary provider and deterministic local fallback."""

    def __init__(
        self,
        *,
        primary: EmbeddingProvider,
        fallback: EmbeddingProvider | None,
    ) -> None:
        self._primary = primary
        self._fallback = fallback

    @property
    def primary_provider_id(self) -> str:
        return self._primary.provider_id

    def _with_fallback(self, fn):  # noqa: ANN001, ANN201
        try:
            return fn(self._primary)
        except EmbeddingProviderError:
            if self._fallback is None:
                raise
            return fn(self._fallback)

    def embed(self, text: str, *, dim: int) -> EmbeddingResult:
        text = text if text.strip() else " "

        def _run(provider: EmbeddingProvider) -> EmbeddingResult:
            return EmbeddingResult(
                vector=provider.embed(text, dim=dim), provider_id=provider.provider_id
            )

        return self._with_fallback(_run)

    def embed_many(self, texts: list[str], *, dim: int) -> list[EmbeddingResult]:
        cleaned = [item if item.strip() else " " for item in texts]

        def _run(provider: EmbeddingProvider) -> list[EmbeddingResult]:
            vectors = provider.embed_many(cleaned, dim=dim)
            return [
                EmbeddingResult(vector=vector, provider_id=provider.provider_id)
                for vector in vectors
            ]

        return self._with_fallback(_run)


def _build_primary_provider(settings: Settings) -> EmbeddingProvider:
    provider_name = settings.embedding_provider.strip().lower()
    if provider_name == "openai":
        return OpenAIEmbeddingProvider(
            api_key=settings.openai_api_key,
            base_url=settings.openai_base_url,
            model_id=settings.embedding_model,
            timeout_seconds=settings.embedding_timeout_seconds,
        )
    return LocalDeterministicEmbeddingProvider()


def _build_fallback_provider(
    settings: Settings, primary: EmbeddingProvider
) -> EmbeddingProvider | None:
    if not settings.embedding_fallback_to_local:
        return None
    if isinstance(primary, LocalDeterministicEmbeddingProvider):
        return None
    return LocalDeterministicEmbeddingProvider()


def build_embedding_service(settings: Settings | None = None) -> EmbeddingService:
    cfg = settings or get_settings()
    primary = _build_primary_provider(cfg)
    fallback = _build_fallback_provider(cfg, primary)
    return EmbeddingService(primary=primary, fallback=fallback)


@lru_cache
def get_embedding_service() -> EmbeddingService:
    return build_embedding_service()


def embed_text(text: str, *, dim: int) -> EmbeddingResult:
    return get_embedding_service().embed(text, dim=dim)


def embed_many(texts: list[str], *, dim: int) -> list[EmbeddingResult]:
    return get_embedding_service().embed_many(texts, dim=dim)

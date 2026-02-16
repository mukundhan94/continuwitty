from __future__ import annotations

from functools import lru_cache
from time import perf_counter

from app.config import Settings, get_settings
from app.observability import record_embedding_call

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

    def embed(self, text: str, *, dim: int) -> EmbeddingResult:
        text = text if text.strip() else " "
        text_chars = len(text)

        def _run(provider: EmbeddingProvider, *, used_fallback: bool) -> EmbeddingResult:
            started_at = perf_counter()
            vector = provider.embed(text, dim=dim)
            duration_ms = (perf_counter() - started_at) * 1000.0
            record_embedding_call(
                operation="embed",
                provider_id=provider.provider_id,
                duration_ms=duration_ms,
                item_count=1,
                text_chars=text_chars,
                dim=dim,
                used_fallback=used_fallback,
            )
            return EmbeddingResult(vector=vector, provider_id=provider.provider_id)

        try:
            return _run(self._primary, used_fallback=False)
        except EmbeddingProviderError:
            if self._fallback is None:
                raise
            return _run(self._fallback, used_fallback=True)

    def embed_many(self, texts: list[str], *, dim: int) -> list[EmbeddingResult]:
        cleaned = [item if item.strip() else " " for item in texts]
        text_chars = sum(len(item) for item in cleaned)
        item_count = len(cleaned)

        def _run(provider: EmbeddingProvider, *, used_fallback: bool) -> list[EmbeddingResult]:
            started_at = perf_counter()
            vectors = provider.embed_many(cleaned, dim=dim)
            duration_ms = (perf_counter() - started_at) * 1000.0
            record_embedding_call(
                operation="embed_many",
                provider_id=provider.provider_id,
                duration_ms=duration_ms,
                item_count=item_count,
                text_chars=text_chars,
                dim=dim,
                used_fallback=used_fallback,
            )
            return [
                EmbeddingResult(vector=vector, provider_id=provider.provider_id)
                for vector in vectors
            ]

        try:
            return _run(self._primary, used_fallback=False)
        except EmbeddingProviderError:
            if self._fallback is None:
                raise
            return _run(self._fallback, used_fallback=True)


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

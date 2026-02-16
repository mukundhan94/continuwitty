from __future__ import annotations

from app.embeddings.errors import EmbeddingProviderError
from app.embeddings.service import EmbeddingService
from app.observability import ChatDebugCollector, bind_chat_debug_collector


class _FakeProvider:
    def __init__(self, provider_id: str) -> None:
        self.provider_id = provider_id

    def embed(self, text: str, *, dim: int) -> list[float]:
        _ = text
        return [0.1] * dim

    def embed_many(self, texts: list[str], *, dim: int) -> list[list[float]]:
        return [[0.2] * dim for _ in texts]


class _FailingProvider:
    provider_id = "failing"

    def embed(self, text: str, *, dim: int) -> list[float]:
        _ = text, dim
        raise EmbeddingProviderError("primary unavailable")

    def embed_many(self, texts: list[str], *, dim: int) -> list[list[float]]:
        _ = texts, dim
        raise EmbeddingProviderError("primary unavailable")


def test_embed_records_observability_metrics() -> None:
    service = EmbeddingService(primary=_FakeProvider("primary"), fallback=None)
    collector = ChatDebugCollector()

    with bind_chat_debug_collector(collector):
        result = service.embed("hello", dim=4)

    assert result.provider_id == "primary"
    assert len(collector.embedding_calls) == 1
    call = collector.embedding_calls[0]
    assert call.operation == "embed"
    assert call.provider_id == "primary"
    assert call.item_count == 1
    assert call.text_chars == 5
    assert call.used_fallback is False


def test_embed_uses_fallback_and_marks_metric() -> None:
    service = EmbeddingService(primary=_FailingProvider(), fallback=_FakeProvider("fallback"))
    collector = ChatDebugCollector()

    with bind_chat_debug_collector(collector):
        result = service.embed("fallback request", dim=8)

    assert result.provider_id == "fallback"
    assert len(collector.embedding_calls) == 1
    call = collector.embedding_calls[0]
    assert call.provider_id == "fallback"
    assert call.used_fallback is True


def test_embed_many_records_batch_metric() -> None:
    service = EmbeddingService(primary=_FakeProvider("primary"), fallback=None)
    collector = ChatDebugCollector()

    with bind_chat_debug_collector(collector):
        result = service.embed_many(["a", "bb", "ccc"], dim=3)

    assert len(result) == 3
    assert len(collector.embedding_calls) == 1
    call = collector.embedding_calls[0]
    assert call.operation == "embed_many"
    assert call.item_count == 3
    assert call.text_chars == 6

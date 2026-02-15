from __future__ import annotations

from app.embeddings.errors import EmbeddingProviderAPIError
from app.embeddings.local_provider import LocalDeterministicEmbeddingProvider
from app.embeddings.service import EmbeddingService


class _FailingProvider:
    provider_id = "failing"

    def embed(self, text: str, *, dim: int) -> list[float]:  # noqa: ARG002
        raise EmbeddingProviderAPIError("provider unavailable")

    def embed_many(self, texts: list[str], *, dim: int) -> list[list[float]]:  # noqa: ARG002
        raise EmbeddingProviderAPIError("provider unavailable")


def test_embedding_service_falls_back_to_local_provider() -> None:
    service = EmbeddingService(
        primary=_FailingProvider(),
        fallback=LocalDeterministicEmbeddingProvider(),
    )

    result = service.embed("memory continuity", dim=12)

    assert len(result.vector) == 12
    assert result.provider_id == "local-deterministic-v1"


def test_embedding_service_embed_many_uses_provider_id_from_active_provider() -> None:
    service = EmbeddingService(
        primary=LocalDeterministicEmbeddingProvider(),
        fallback=None,
    )

    results = service.embed_many(["alpha", "beta"], dim=10)

    assert len(results) == 2
    assert {item.provider_id for item in results} == {"local-deterministic-v1"}
    assert all(len(item.vector) == 10 for item in results)

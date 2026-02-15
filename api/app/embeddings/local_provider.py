from __future__ import annotations

from app.embedding import embed_text_local


class LocalDeterministicEmbeddingProvider:
    """Deterministic local embedding provider for offline/local-first operation."""

    provider_id = "local-deterministic-v1"

    def embed(self, text: str, *, dim: int) -> list[float]:
        return embed_text_local(text, dim)

    def embed_many(self, texts: list[str], *, dim: int) -> list[list[float]]:
        return [self.embed(text, dim=dim) for text in texts]

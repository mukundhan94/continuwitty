from __future__ import annotations

from dataclasses import dataclass
from typing import Protocol


class EmbeddingProvider(Protocol):
    """Provider contract used by ingestion/retrieval code.

    Providers must return vectors with deterministic ordering for batch inputs.
    """

    provider_id: str

    def embed(self, text: str, *, dim: int) -> list[float]: ...

    def embed_many(self, texts: list[str], *, dim: int) -> list[list[float]]: ...


@dataclass(frozen=True)
class EmbeddingResult:
    """Embedding output plus provider provenance used for persistence metadata."""

    vector: list[float]
    provider_id: str

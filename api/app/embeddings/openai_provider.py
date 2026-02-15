from __future__ import annotations

from typing import Any

import httpx

from .errors import (
    EmbeddingProviderAPIError,
    EmbeddingProviderAuthError,
    EmbeddingProviderRequestError,
)


class OpenAIEmbeddingProvider:
    """OpenAI embedding provider.

    The provider requests a caller-specified `dimensions` value so vectors fit the
    local pgvector column dimensions used by this repository.
    """

    def __init__(
        self,
        *,
        api_key: str | None,
        base_url: str,
        model_id: str,
        timeout_seconds: float,
        client: httpx.Client | None = None,
    ) -> None:
        self._api_key = api_key
        self._base_url = base_url.rstrip("/")
        self._model_id = model_id
        self._timeout_seconds = timeout_seconds
        self._client = client

    @property
    def provider_id(self) -> str:
        return f"openai/{self._model_id}"

    def _auth_headers(self) -> dict[str, str]:
        if not self._api_key:
            raise EmbeddingProviderAuthError("OPENAI_API_KEY is not configured")
        return {
            "Authorization": f"Bearer {self._api_key}",
            "Content-Type": "application/json",
        }

    def _http_client(self) -> httpx.Client:
        if self._client is not None:
            return self._client
        return httpx.Client(base_url=self._base_url, timeout=self._timeout_seconds)

    @staticmethod
    def _coerce_dim(vector: list[float], dim: int) -> list[float]:
        # Some gateways may ignore requested dimensions; coerce safely so writes
        # still fit the configured pgvector dimension.
        if len(vector) == dim:
            return vector
        if len(vector) > dim:
            return vector[:dim]
        return vector + ([0.0] * (dim - len(vector)))

    @staticmethod
    def _extract_vectors(body: dict[str, Any]) -> list[list[float]]:
        rows = body.get("data")
        if not isinstance(rows, list):
            raise EmbeddingProviderAPIError("OpenAI embeddings response missing data array")

        ordered_rows: list[dict[str, Any]] = []
        for row in rows:
            if not isinstance(row, dict):
                continue
            ordered_rows.append(row)

        ordered_rows.sort(key=lambda item: int(item.get("index", 0)))

        vectors: list[list[float]] = []
        for row in ordered_rows:
            embedding = row.get("embedding")
            if not isinstance(embedding, list):
                raise EmbeddingProviderAPIError(
                    "OpenAI embeddings response missing embedding values"
                )
            vectors.append([float(value) for value in embedding])
        return vectors

    def embed_many(self, texts: list[str], *, dim: int) -> list[list[float]]:
        if not texts:
            return []

        payload = {
            "model": self._model_id,
            "input": texts,
            "dimensions": dim,
        }
        response = self._http_client().post(
            f"{self._base_url}/v1/embeddings",
            json=payload,
            headers=self._auth_headers(),
        )

        if response.status_code in (401, 403):
            raise EmbeddingProviderAuthError("OpenAI rejected embedding credentials")
        if response.status_code == 400:
            raise EmbeddingProviderRequestError("OpenAI rejected embedding request payload")
        if response.status_code >= 400:
            raise EmbeddingProviderAPIError(
                f"OpenAI embeddings request failed with status {response.status_code}"
            )

        vectors = self._extract_vectors(response.json())
        if len(vectors) != len(texts):
            raise EmbeddingProviderAPIError(
                "OpenAI embeddings response size does not match request batch"
            )

        return [self._coerce_dim(vector, dim) for vector in vectors]

    def embed(self, text: str, *, dim: int) -> list[float]:
        vectors = self.embed_many([text], dim=dim)
        if not vectors:
            raise EmbeddingProviderAPIError("OpenAI embeddings returned no vectors")
        return vectors[0]

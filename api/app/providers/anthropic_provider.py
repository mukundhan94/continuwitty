from __future__ import annotations

from typing import Any

import httpx

from app.models import ChatProvider

from .base import ProviderGenerateRequest, ProviderGenerateResult
from .errors import ProviderAPIError, ProviderAuthError, ProviderRateLimitError


class AnthropicProvider:
    provider = ChatProvider.anthropic

    def __init__(
        self,
        api_key: str | None,
        base_url: str = "https://api.anthropic.com",
        anthropic_version: str = "2023-06-01",
        timeout_seconds: float = 30.0,
        client: httpx.Client | None = None,
    ) -> None:
        self._api_key = api_key
        self._base_url = base_url.rstrip("/")
        self._anthropic_version = anthropic_version
        self._timeout = timeout_seconds
        self._client = client

    def _auth_headers(self) -> dict[str, str]:
        if not self._api_key:
            raise ProviderAuthError("ANTHROPIC_API_KEY is not configured")
        return {
            "x-api-key": self._api_key,
            "anthropic-version": self._anthropic_version,
            "Content-Type": "application/json",
        }

    def _http_client(self) -> httpx.Client:
        if self._client is not None:
            return self._client
        return httpx.Client(base_url=self._base_url, timeout=self._timeout)

    def _request_payload(self, request: ProviderGenerateRequest) -> dict[str, Any]:
        messages = [{"role": item.role, "content": item.content} for item in request.messages]
        payload: dict[str, Any] = {
            "model": request.model_id,
            "messages": messages,
            "temperature": request.temperature,
            "max_tokens": request.max_tokens,
        }
        if request.system_prompt:
            payload["system"] = request.system_prompt
        return payload

    def _extract_text(self, body: dict[str, Any]) -> str:
        content = body.get("content", [])
        if isinstance(content, list):
            parts: list[str] = []
            for item in content:
                if isinstance(item, dict) and item.get("type") == "text":
                    text_value = item.get("text")
                    if isinstance(text_value, str):
                        parts.append(text_value)
            return "".join(parts)
        return ""

    def generate(self, request: ProviderGenerateRequest) -> ProviderGenerateResult:
        response = self._http_client().post(
            f"{self._base_url}/v1/messages",
            json=self._request_payload(request),
            headers=self._auth_headers(),
        )
        if response.status_code in (401, 403):
            raise ProviderAuthError("Anthropic rejected credentials")
        if response.status_code == 429:
            raise ProviderRateLimitError("Anthropic rate limited request")
        if response.status_code >= 400:
            raise ProviderAPIError(f"Anthropic request failed with status {response.status_code}")

        body = response.json()
        usage = body.get("usage", {})
        input_tokens = int(usage.get("input_tokens", 0))
        output_tokens = int(usage.get("output_tokens", 0))
        return ProviderGenerateResult(
            provider=self.provider,
            model_id=request.model_id,
            text=self._extract_text(body),
            token_usage={
                "input_tokens": input_tokens,
                "output_tokens": output_tokens,
                "total_tokens": input_tokens + output_tokens,
            },
        )

    def stream_generate(self, request: ProviderGenerateRequest):
        result = self.generate(request)
        yield result.text

    def healthcheck(self) -> dict[str, str]:
        if not self._api_key:
            raise ProviderAuthError("ANTHROPIC_API_KEY is not configured")
        return {"provider": self.provider.value, "status": "configured"}

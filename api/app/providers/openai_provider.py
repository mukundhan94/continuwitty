from __future__ import annotations

from typing import Any

import httpx

from app.models import ChatProvider

from .base import ProviderGenerateRequest, ProviderGenerateResult
from .errors import ProviderAPIError, ProviderAuthError, ProviderRateLimitError


class OpenAIProvider:
    provider = ChatProvider.openai

    def __init__(
        self,
        api_key: str | None,
        base_url: str = "https://api.openai.com",
        timeout_seconds: float = 30.0,
        client: httpx.Client | None = None,
    ) -> None:
        self._api_key = api_key
        self._base_url = base_url.rstrip("/")
        self._timeout = timeout_seconds
        self._client = client

    def _auth_headers(self) -> dict[str, str]:
        if not self._api_key:
            raise ProviderAuthError("OPENAI_API_KEY is not configured")
        return {
            "Authorization": f"Bearer {self._api_key}",
            "Content-Type": "application/json",
        }

    def _http_client(self) -> httpx.Client:
        if self._client is not None:
            return self._client
        return httpx.Client(base_url=self._base_url, timeout=self._timeout)

    def _request_payload(self, request: ProviderGenerateRequest) -> dict[str, Any]:
        messages: list[dict[str, str]] = []
        if request.system_prompt:
            messages.append({"role": "system", "content": request.system_prompt})
        messages.extend({"role": item.role, "content": item.content} for item in request.messages)
        return {
            "model": request.model_id,
            "messages": messages,
            "temperature": request.temperature,
            "max_tokens": request.max_tokens,
        }

    def _extract_text(self, body: dict[str, Any]) -> str:
        choices = body.get("choices", [])
        if not choices:
            return ""
        message = choices[0].get("message", {})
        content = message.get("content", "")
        if isinstance(content, str):
            return content
        if isinstance(content, list):
            parts: list[str] = []
            for item in content:
                if isinstance(item, dict):
                    text_value = item.get("text")
                    if isinstance(text_value, str):
                        parts.append(text_value)
            return "".join(parts)
        return ""

    def generate(self, request: ProviderGenerateRequest) -> ProviderGenerateResult:
        response = self._http_client().post(
            f"{self._base_url}/v1/chat/completions",
            json=self._request_payload(request),
            headers=self._auth_headers(),
        )
        if response.status_code in (401, 403):
            raise ProviderAuthError("OpenAI rejected credentials")
        if response.status_code == 429:
            raise ProviderRateLimitError("OpenAI rate limited request")
        if response.status_code >= 400:
            raise ProviderAPIError(f"OpenAI request failed with status {response.status_code}")

        body = response.json()
        usage = body.get("usage", {})
        return ProviderGenerateResult(
            provider=self.provider,
            model_id=request.model_id,
            text=self._extract_text(body),
            token_usage={
                "input_tokens": int(usage.get("prompt_tokens", 0)),
                "output_tokens": int(usage.get("completion_tokens", 0)),
                "total_tokens": int(usage.get("total_tokens", 0)),
            },
        )

    def stream_generate(self, request: ProviderGenerateRequest):
        result = self.generate(request)
        yield result.text

    def healthcheck(self) -> dict[str, str]:
        if not self._api_key:
            raise ProviderAuthError("OPENAI_API_KEY is not configured")
        return {"provider": self.provider.value, "status": "configured"}

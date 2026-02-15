from __future__ import annotations

import json
from typing import Any

from app.models import ChatProvider

from .base import ProviderGenerateRequest, ProviderGenerateResult
from .errors import (
    ProviderAPIError,
    ProviderAuthError,
    ProviderRateLimitError,
    ProviderRequestError,
)

try:
    import boto3
except Exception:  # pragma: no cover
    boto3 = None

try:
    from botocore.exceptions import ClientError, NoCredentialsError, PartialCredentialsError
except Exception:  # pragma: no cover
    ClientError = None  # type: ignore[assignment]
    NoCredentialsError = None  # type: ignore[assignment]
    PartialCredentialsError = None  # type: ignore[assignment]


class BedrockProvider:
    provider = ChatProvider.bedrock

    def __init__(
        self,
        region_name: str,
        access_key_id: str | None = None,
        secret_access_key: str | None = None,
        session_token: str | None = None,
        client: Any | None = None,
    ) -> None:
        self._region_name = region_name
        self._access_key_id = access_key_id
        self._secret_access_key = secret_access_key
        self._session_token = session_token
        self._client = client

    def _runtime_client(self):
        if self._client is not None:
            return self._client
        if boto3 is None:
            raise ProviderAPIError("boto3 is not available for Bedrock provider")
        client_kwargs: dict[str, Any] = {"region_name": self._region_name}
        if self._access_key_id:
            client_kwargs["aws_access_key_id"] = self._access_key_id
        if self._secret_access_key:
            client_kwargs["aws_secret_access_key"] = self._secret_access_key
        if self._session_token:
            client_kwargs["aws_session_token"] = self._session_token
        return boto3.client("bedrock-runtime", **client_kwargs)

    def _request_body(self, request: ProviderGenerateRequest) -> dict[str, Any]:
        return {
            "anthropic_version": "bedrock-2023-05-31",
            "max_tokens": request.max_tokens,
            "temperature": request.temperature,
            "system": request.system_prompt,
            "messages": [
                {
                    "role": item.role,
                    "content": [{"type": "text", "text": item.content}],
                }
                for item in request.messages
            ],
        }

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

    @staticmethod
    def _client_error_code(exc: Exception) -> str:
        if ClientError is not None and isinstance(exc, ClientError):
            return str(exc.response.get("Error", {}).get("Code") or "")
        return ""

    @staticmethod
    def _client_error_message(exc: Exception) -> str:
        if ClientError is not None and isinstance(exc, ClientError):
            return str(exc.response.get("Error", {}).get("Message") or "")
        return str(exc)

    def _raise_invocation_error(self, exc: Exception) -> None:
        if NoCredentialsError is not None and isinstance(exc, NoCredentialsError):
            raise ProviderAuthError(
                "Bedrock credentials not found in environment or AWS profile"
            ) from exc
        if PartialCredentialsError is not None and isinstance(exc, PartialCredentialsError):
            raise ProviderAuthError("Bedrock credentials are incomplete") from exc

        code = self._client_error_code(exc)
        message = self._client_error_message(exc)
        if code in {"ThrottlingException", "TooManyRequestsException"}:
            raise ProviderRateLimitError(f"Bedrock {code}: {message}") from exc
        if code == "ValidationException":
            raise ProviderRequestError(f"Bedrock {code}: {message}") from exc
        if code in {
            "UnrecognizedClientException",
            "InvalidSignatureException",
            "ExpiredTokenException",
            "IncompleteSignatureException",
        }:
            raise ProviderAuthError(f"Bedrock {code}: {message}") from exc
        if code:
            raise ProviderAPIError(f"Bedrock {code}: {message}") from exc
        raise ProviderAPIError(f"Bedrock invocation failed: {exc}") from exc

    def generate(self, request: ProviderGenerateRequest) -> ProviderGenerateResult:
        try:
            response = self._runtime_client().invoke_model(
                modelId=request.model_id,
                contentType="application/json",
                accept="application/json",
                body=json.dumps(self._request_body(request)),
            )
        except Exception as exc:
            self._raise_invocation_error(exc)
            raise

        try:
            body = json.loads(response["body"].read())
        except Exception as exc:
            raise ProviderAPIError("Failed to parse Bedrock response") from exc

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
        if not self._region_name:
            raise ProviderAuthError("AWS_REGION is not configured")
        return {"provider": self.provider.value, "status": "configured"}

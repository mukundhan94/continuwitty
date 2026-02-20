from __future__ import annotations

import json
from dataclasses import dataclass
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

_AWS_AUTH_ERROR_CODES = {
    "UnrecognizedClientException",
    "InvalidSignatureException",
    "ExpiredTokenException",
    "IncompleteSignatureException",
}
_AWS_RATE_LIMIT_ERROR_CODES = {"ThrottlingException", "TooManyRequestsException"}
_AWS_REQUEST_ERROR_CODES = {"ValidationException"}


@dataclass(frozen=True)
class AwsCredentials:
    region_name: str
    access_key_id: str | None = None
    secret_access_key: str | None = None
    session_token: str | None = None

    def runtime_client_kwargs(self) -> dict[str, Any]:
        client_kwargs: dict[str, Any] = {"region_name": self.region_name}
        if self.access_key_id:
            client_kwargs["aws_access_key_id"] = self.access_key_id
        if self.secret_access_key:
            client_kwargs["aws_secret_access_key"] = self.secret_access_key
        if self.session_token:
            client_kwargs["aws_session_token"] = self.session_token
        return client_kwargs


class BedrockProvider:
    provider = ChatProvider.bedrock

    def __init__(
        self,
        credentials: AwsCredentials,
        client: Any | None = None,
    ) -> None:
        self._credentials = credentials
        self._client = client

    def _runtime_client(self):
        if self._client is not None:
            return self._client
        if boto3 is None:
            raise ProviderAPIError("boto3 is not available for Bedrock provider")
        return boto3.client("bedrock-runtime", **self._credentials.runtime_client_kwargs())

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

    @staticmethod
    def _extract_text_part(item: Any) -> str:
        if not isinstance(item, dict):
            return ""
        if item.get("type") != "text":
            return ""
        text_value = item.get("text")
        if isinstance(text_value, str):
            return text_value
        return ""

    def _extract_text(self, body: dict[str, Any]) -> str:
        content = body.get("content", [])
        if not isinstance(content, list):
            return ""
        parts = [self._extract_text_part(item) for item in content]
        if parts:
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

    @staticmethod
    def _is_missing_credentials_error(exc: Exception) -> bool:
        return NoCredentialsError is not None and isinstance(exc, NoCredentialsError)

    @staticmethod
    def _is_partial_credentials_error(exc: Exception) -> bool:
        return PartialCredentialsError is not None and isinstance(exc, PartialCredentialsError)

    @staticmethod
    def _invocation_error_from_code(*, code: str, message: str) -> ProviderAPIError | None:
        if not code:
            return None
        if code in _AWS_RATE_LIMIT_ERROR_CODES:
            return ProviderRateLimitError(f"Bedrock {code}: {message}")
        if code in _AWS_REQUEST_ERROR_CODES:
            return ProviderRequestError(f"Bedrock {code}: {message}")
        if code in _AWS_AUTH_ERROR_CODES:
            return ProviderAuthError(f"Bedrock {code}: {message}")
        return ProviderAPIError(f"Bedrock {code}: {message}")

    def _raise_invocation_error(self, exc: Exception) -> None:
        if self._is_missing_credentials_error(exc):
            raise ProviderAuthError(
                "Bedrock credentials not found in environment or AWS profile"
            ) from exc
        if self._is_partial_credentials_error(exc):
            raise ProviderAuthError("Bedrock credentials are incomplete") from exc

        code = self._client_error_code(exc)
        message = self._client_error_message(exc)
        mapped_error = self._invocation_error_from_code(code=code, message=message)
        if mapped_error is not None:
            raise mapped_error from exc
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
        if not self._credentials.region_name:
            raise ProviderAuthError("AWS_REGION is not configured")
        return {"provider": self.provider.value, "status": "configured"}

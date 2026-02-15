from __future__ import annotations

import json

import pytest
from botocore.exceptions import ClientError, NoCredentialsError

from app.models import ChatProvider
from app.providers.anthropic_provider import AnthropicProvider
from app.providers.base import ProviderGenerateRequest, ProviderMessage
from app.providers.bedrock_provider import BedrockProvider
from app.providers.errors import ProviderAuthError, ProviderRequestError
from app.providers.openai_provider import OpenAIProvider


class _DummyResponse:
    def __init__(self, status_code: int, payload: dict):
        self.status_code = status_code
        self._payload = payload

    def json(self):
        return self._payload


class _DummyHttpClient:
    def __init__(self, response: _DummyResponse):
        self._response = response
        self.requests = []

    def post(self, url, json=None, headers=None):  # noqa: ANN001
        self.requests.append({"url": url, "json": json, "headers": headers})
        return self._response


class _DummyBody:
    def __init__(self, payload: dict):
        self._bytes = json.dumps(payload).encode("utf-8")

    def read(self):
        return self._bytes


class _DummyBedrockClient:
    def __init__(self, payload: dict):
        self.payload = payload
        self.calls = []

    def invoke_model(self, **kwargs):
        self.calls.append(kwargs)
        return {"body": _DummyBody(self.payload)}


class _FailingBedrockClient:
    def __init__(self, error: Exception):
        self.error = error

    def invoke_model(self, **kwargs):  # noqa: ANN003, ARG002
        raise self.error


def _request() -> ProviderGenerateRequest:
    return ProviderGenerateRequest(
        model_id="test-model",
        messages=[ProviderMessage(role="user", content="Hello")],
        system_prompt="You are helpful.",
        max_tokens=128,
    )


def test_openai_provider_generate_normalizes_response() -> None:
    response = _DummyResponse(
        200,
        {
            "choices": [{"message": {"content": "OpenAI reply"}}],
            "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
        },
    )
    client = _DummyHttpClient(response)
    provider = OpenAIProvider(api_key="k", client=client)

    result = provider.generate(_request())

    assert result.provider == ChatProvider.openai
    assert result.text == "OpenAI reply"
    assert result.token_usage["total_tokens"] == 15
    assert client.requests


def test_anthropic_provider_generate_normalizes_response() -> None:
    response = _DummyResponse(
        200,
        {
            "content": [{"type": "text", "text": "Anthropic reply"}],
            "usage": {"input_tokens": 11, "output_tokens": 7},
        },
    )
    client = _DummyHttpClient(response)
    provider = AnthropicProvider(api_key="k", client=client)

    result = provider.generate(_request())

    assert result.provider == ChatProvider.anthropic
    assert result.text == "Anthropic reply"
    assert result.token_usage["input_tokens"] == 11
    assert result.token_usage["total_tokens"] == 18


def test_bedrock_provider_generate_normalizes_response() -> None:
    client = _DummyBedrockClient(
        {
            "content": [{"type": "text", "text": "Bedrock reply"}],
            "usage": {"input_tokens": 13, "output_tokens": 8},
        }
    )
    provider = BedrockProvider(region_name="us-east-1", client=client)

    result = provider.generate(_request())

    assert result.provider == ChatProvider.bedrock
    assert result.text == "Bedrock reply"
    assert result.token_usage["total_tokens"] == 21
    assert client.calls


def test_openai_healthcheck_requires_api_key() -> None:
    provider = OpenAIProvider(api_key=None)
    with pytest.raises(ProviderAuthError):
        provider.healthcheck()


def test_anthropic_healthcheck_requires_api_key() -> None:
    provider = AnthropicProvider(api_key=None)
    with pytest.raises(ProviderAuthError):
        provider.healthcheck()


def test_bedrock_healthcheck_requires_region() -> None:
    provider = BedrockProvider(region_name="")
    with pytest.raises(ProviderAuthError):
        provider.healthcheck()


def test_bedrock_provider_surfaces_validation_error_details() -> None:
    err = ClientError(
        error_response={
            "Error": {
                "Code": "ValidationException",
                "Message": "Invocation of model ID test-model requires inference profile.",
            }
        },
        operation_name="InvokeModel",
    )
    provider = BedrockProvider(region_name="us-east-1", client=_FailingBedrockClient(err))

    with pytest.raises(ProviderRequestError) as exc_info:
        provider.generate(_request())

    assert "ValidationException" in str(exc_info.value)
    assert "inference profile" in str(exc_info.value)


def test_bedrock_provider_reports_missing_credentials() -> None:
    provider = BedrockProvider(
        region_name="us-east-1",
        client=_FailingBedrockClient(NoCredentialsError()),
    )

    with pytest.raises(ProviderAuthError) as exc_info:
        provider.generate(_request())

    assert "credentials not found" in str(exc_info.value)

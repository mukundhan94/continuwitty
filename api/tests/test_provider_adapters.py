from __future__ import annotations

import json
from dataclasses import dataclass

import pytest
from botocore.exceptions import ClientError, NoCredentialsError

from app.models import ChatProvider
from app.providers.anthropic_provider import AnthropicProvider
from app.providers.base import ProviderGenerateRequest, ProviderMessage
from app.providers.bedrock_provider import AwsCredentials, BedrockProvider
from app.providers.errors import ProviderAuthError, ProviderRateLimitError, ProviderRequestError
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


@dataclass(frozen=True)
class _TextProviderCase:
    provider_name: str
    payload: dict
    expected_provider: ChatProvider
    expected_text: str
    expected_usage: dict[str, int]


def _request() -> ProviderGenerateRequest:
    return ProviderGenerateRequest(
        model_id="test-model",
        messages=[ProviderMessage(role="user", content="Hello")],
        system_prompt="You are helpful.",
        max_tokens=128,
    )


@pytest.mark.parametrize(
    "case",
    [
        _TextProviderCase(
            provider_name="openai",
            payload={
                "choices": [{"message": {"content": "OpenAI reply"}}],
                "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
            },
            expected_provider=ChatProvider.openai,
            expected_text="OpenAI reply",
            expected_usage={"total_tokens": 15},
        ),
        _TextProviderCase(
            provider_name="anthropic",
            payload={
                "content": [{"type": "text", "text": "Anthropic reply"}],
                "usage": {"input_tokens": 11, "output_tokens": 7},
            },
            expected_provider=ChatProvider.anthropic,
            expected_text="Anthropic reply",
            expected_usage={"input_tokens": 11, "total_tokens": 18},
        ),
    ],
)
def test_text_providers_generate_normalize_response(
    case: _TextProviderCase,
) -> None:
    response = _DummyResponse(200, case.payload)
    client = _DummyHttpClient(response)
    if case.provider_name == "openai":
        provider = OpenAIProvider(api_key="k", client=client)
    else:
        provider = AnthropicProvider(api_key="k", client=client)

    result = provider.generate(_request())

    assert result.provider == case.expected_provider
    assert result.text == case.expected_text
    for key, value in case.expected_usage.items():
        assert result.token_usage[key] == value
    assert client.requests


def test_bedrock_provider_generate_normalizes_response() -> None:
    client = _DummyBedrockClient(
        {
            "content": [{"type": "text", "text": "Bedrock reply"}],
            "usage": {"input_tokens": 13, "output_tokens": 8},
        }
    )
    provider = BedrockProvider(
        credentials=AwsCredentials(region_name="us-east-1"),
        client=client,
    )

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
    provider = BedrockProvider(credentials=AwsCredentials(region_name=""))
    with pytest.raises(ProviderAuthError):
        provider.healthcheck()


def test_bedrock_provider_reports_missing_credentials() -> None:
    provider = BedrockProvider(
        credentials=AwsCredentials(region_name="us-east-1"),
        client=_FailingBedrockClient(NoCredentialsError()),
    )

    with pytest.raises(ProviderAuthError) as exc_info:
        provider.generate(_request())

    assert "credentials not found" in str(exc_info.value)


@pytest.mark.parametrize(
    ("error_code", "message", "expected_error", "expected_details"),
    [
        (
            "ValidationException",
            "Invocation of model ID test-model requires inference profile.",
            ProviderRequestError,
            ["ValidationException", "inference profile"],
        ),
        (
            "ThrottlingException",
            "Rate exceeded",
            ProviderRateLimitError,
            ["ThrottlingException"],
        ),
        (
            "ExpiredTokenException",
            "The security token included in the request is expired",
            ProviderAuthError,
            ["ExpiredTokenException"],
        ),
    ],
)
def test_bedrock_provider_maps_client_errors(
    error_code: str,
    message: str,
    expected_error: type[Exception],
    expected_details: list[str],
) -> None:
    err = ClientError(
        error_response={"Error": {"Code": error_code, "Message": message}},
        operation_name="InvokeModel",
    )
    provider = BedrockProvider(
        credentials=AwsCredentials(region_name="us-east-1"),
        client=_FailingBedrockClient(err),
    )

    with pytest.raises(expected_error) as exc_info:
        provider.generate(_request())

    details = str(exc_info.value)
    for expected in expected_details:
        assert expected in details


def test_bedrock_provider_extract_text_ignores_non_text_content() -> None:
    client = _DummyBedrockClient(
        {
            "content": [{"type": "tool_use", "name": "x"}, {"type": "text", "text": 7}],
            "usage": {"input_tokens": 1, "output_tokens": 1},
        }
    )
    provider = BedrockProvider(
        credentials=AwsCredentials(region_name="us-east-1"),
        client=client,
    )

    result = provider.generate(_request())

    assert result.text == ""

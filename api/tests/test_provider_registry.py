from __future__ import annotations

from types import SimpleNamespace

from app.models import ChatProvider
from app.providers import AnthropicProvider, BedrockProvider, OpenAIProvider
from app.providers.registry import build_provider_registry, get_provider_adapter


def test_build_provider_registry_creates_all_adapters() -> None:
    settings = SimpleNamespace(
        openai_api_key="openai-key",
        openai_base_url="https://api.openai.com",
        anthropic_api_key="anthropic-key",
        anthropic_base_url="https://api.anthropic.com",
        anthropic_version="2023-06-01",
        aws_region="us-east-1",
        aws_access_key_id="access-key",
        aws_secret_access_key="secret-key",
        aws_session_token="session-token",
    )

    registry = build_provider_registry(settings=settings)

    assert set(registry.keys()) == {
        ChatProvider.openai,
        ChatProvider.anthropic,
        ChatProvider.bedrock,
    }
    assert isinstance(registry[ChatProvider.openai], OpenAIProvider)
    assert isinstance(registry[ChatProvider.anthropic], AnthropicProvider)
    assert isinstance(registry[ChatProvider.bedrock], BedrockProvider)
    bedrock = registry[ChatProvider.bedrock]
    assert bedrock._credentials.region_name == "us-east-1"
    assert bedrock._credentials.access_key_id == "access-key"
    assert bedrock._credentials.secret_access_key == "secret-key"
    assert bedrock._credentials.session_token == "session-token"


def test_get_provider_adapter_returns_requested_provider() -> None:
    settings = SimpleNamespace(
        openai_api_key="openai-key",
        openai_base_url="https://api.openai.com",
        anthropic_api_key="anthropic-key",
        anthropic_base_url="https://api.anthropic.com",
        anthropic_version="2023-06-01",
        aws_region="us-east-1",
        aws_access_key_id="access-key",
        aws_secret_access_key="secret-key",
        aws_session_token="session-token",
    )

    adapter = get_provider_adapter(ChatProvider.openai, settings=settings)

    assert isinstance(adapter, OpenAIProvider)

from __future__ import annotations

from app.config import Settings, get_settings
from app.models import ChatProvider

from .anthropic_provider import AnthropicProvider
from .base import ChatProviderAdapter
from .bedrock_provider import AwsCredentials, BedrockProvider
from .openai_provider import OpenAIProvider


def build_provider_registry(
    settings: Settings | None = None,
) -> dict[ChatProvider, ChatProviderAdapter]:
    cfg = settings or get_settings()
    return {
        ChatProvider.openai: OpenAIProvider(
            api_key=cfg.openai_api_key,
            base_url=cfg.openai_base_url,
        ),
        ChatProvider.anthropic: AnthropicProvider(
            api_key=cfg.anthropic_api_key,
            base_url=cfg.anthropic_base_url,
            anthropic_version=cfg.anthropic_version,
        ),
        ChatProvider.bedrock: BedrockProvider(
            credentials=AwsCredentials(
                region_name=cfg.aws_region,
                access_key_id=cfg.aws_access_key_id,
                secret_access_key=cfg.aws_secret_access_key,
                session_token=cfg.aws_session_token,
            ),
        ),
    }


def get_provider_adapter(
    provider: ChatProvider,
    settings: Settings | None = None,
) -> ChatProviderAdapter:
    registry = build_provider_registry(settings=settings)
    adapter = registry.get(provider)
    if not adapter:
        raise ValueError(f"provider adapter not available: {provider}")
    return adapter

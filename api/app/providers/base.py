from __future__ import annotations

from dataclasses import dataclass, field
from typing import Protocol

from app.models import ChatProvider


@dataclass(frozen=True)
class ProviderMessage:
    role: str
    content: str


@dataclass(frozen=True)
class ProviderGenerateRequest:
    model_id: str
    messages: list[ProviderMessage]
    system_prompt: str = ""
    temperature: float = 0.2
    max_tokens: int = 800


@dataclass
class ProviderGenerateResult:
    provider: ChatProvider
    model_id: str
    text: str
    token_usage: dict[str, int] = field(default_factory=dict)


class ChatProviderAdapter(Protocol):
    provider: ChatProvider

    def generate(self, request: ProviderGenerateRequest) -> ProviderGenerateResult: ...

    def stream_generate(self, request: ProviderGenerateRequest): ...

    def healthcheck(self) -> dict[str, str]: ...

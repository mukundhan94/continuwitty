from __future__ import annotations

import json
import logging
from contextlib import contextmanager
from contextvars import ContextVar
from dataclasses import dataclass, field
from typing import Any

try:  # pragma: no cover - optional runtime dependency
    from langfuse import Langfuse
except Exception:  # pragma: no cover
    Langfuse = None

logger = logging.getLogger(__name__)

_CURRENT_CHAT_DEBUG_COLLECTOR: ContextVar[ChatDebugCollector | None] = ContextVar(
    "CURRENT_CHAT_DEBUG_COLLECTOR",
    default=None,
)


@dataclass(frozen=True)
class EmbeddingCallRecord:
    operation: str
    provider_id: str
    duration_ms: float
    item_count: int
    text_chars: int
    dim: int
    used_fallback: bool = False


@dataclass
class ChatDebugCollector:
    embedding_calls: list[EmbeddingCallRecord] = field(default_factory=list)

    def record_embedding_call(
        self,
        *,
        operation: str,
        provider_id: str,
        duration_ms: float,
        item_count: int,
        text_chars: int,
        dim: int,
        used_fallback: bool,
    ) -> None:
        self.embedding_calls.append(
            EmbeddingCallRecord(
                operation=operation,
                provider_id=provider_id,
                duration_ms=duration_ms,
                item_count=item_count,
                text_chars=text_chars,
                dim=dim,
                used_fallback=used_fallback,
            )
        )


@contextmanager
def bind_chat_debug_collector(collector: ChatDebugCollector):
    token = _CURRENT_CHAT_DEBUG_COLLECTOR.set(collector)
    try:
        yield
    finally:
        _CURRENT_CHAT_DEBUG_COLLECTOR.reset(token)


def record_embedding_call(
    *,
    operation: str,
    provider_id: str,
    duration_ms: float,
    item_count: int,
    text_chars: int,
    dim: int,
    used_fallback: bool,
) -> None:
    collector = _CURRENT_CHAT_DEBUG_COLLECTOR.get()
    if collector is None:
        return
    collector.record_embedding_call(
        operation=operation,
        provider_id=provider_id,
        duration_ms=duration_ms,
        item_count=item_count,
        text_chars=text_chars,
        dim=dim,
        used_fallback=used_fallback,
    )


class ChatDebugTelemetryPublisher:
    """Emits debug traces to local logs and optional Langfuse.

    This publisher is intentionally best-effort: failures must never interrupt
    chat generation flow.
    """

    def __init__(
        self,
        *,
        console_enabled: bool,
        langfuse_enabled: bool,
        langfuse_public_key: str | None,
        langfuse_secret_key: str | None,
        langfuse_host: str,
    ) -> None:
        self._console_enabled = console_enabled
        self._langfuse_enabled_requested = langfuse_enabled
        self._langfuse_client = None
        if not langfuse_enabled:
            logger.info("[engram-chat-debug] Langfuse tracing disabled by config")
            return
        if Langfuse is None:
            logger.warning(
                "[engram-chat-debug] Langfuse tracing requested but dependency is missing; "
                "install with `uv add langfuse` in /api"
            )
            return
        if not langfuse_public_key or not langfuse_secret_key:
            logger.warning(
                "[engram-chat-debug] Langfuse tracing requested but public/secret keys are missing"
            )
            return

        try:
            self._langfuse_client = Langfuse(
                public_key=langfuse_public_key,
                secret_key=langfuse_secret_key,
                host=langfuse_host,
            )
            if not self._supports_observation_api(self._langfuse_client):
                logger.warning(
                    "[engram-chat-debug] Installed Langfuse client is incompatible; "
                    "requires start_as_current_observation API"
                )
                self._langfuse_client = None
                return
            logger.info(
                "[engram-chat-debug] Langfuse tracing enabled",
                extra={"langfuse_host": langfuse_host},
            )
        except Exception:
            logger.exception("[engram-chat-debug] Langfuse client initialization failed")
            self._langfuse_client = None

    def publish_chat_trace(self, *, trace_payload: dict[str, Any]) -> None:
        if self._console_enabled:
            try:
                print(
                    "[engram-chat-debug]",
                    json.dumps(trace_payload, default=str, ensure_ascii=True),
                )
            except Exception:
                logger.exception("[engram-chat-debug] Failed to write debug payload to console")

        if self._langfuse_client is None:
            if self._langfuse_enabled_requested:
                logger.warning(
                    "[engram-chat-debug] Langfuse publish skipped: client unavailable",
                    extra={"session_id": str(trace_payload.get("session_id", ""))},
                )
            return

        try:
            self._publish_with_observation_api(trace_payload=trace_payload)
            self._langfuse_client.flush()
            llm_calls = trace_payload.get("llm_calls", [])
            logger.info(
                "[engram-chat-debug] Langfuse trace published",
                extra={
                    "session_id": str(trace_payload.get("session_id", "")),
                    "provider": str(trace_payload.get("provider", "")),
                    "model_id": str(trace_payload.get("model_id", "")),
                    "llm_call_count": len(llm_calls) if isinstance(llm_calls, list) else 0,
                },
            )
        except Exception:
            # Langfuse is optional; keep chat flow resilient if remote telemetry fails.
            logger.exception(
                "[engram-chat-debug] Langfuse trace publish failed",
                extra={"session_id": str(trace_payload.get("session_id", ""))},
            )
            return

    @staticmethod
    def _supports_observation_api(client: Any) -> bool:
        return callable(getattr(client, "start_as_current_observation", None))

    def _publish_with_observation_api(self, *, trace_payload: dict[str, Any]) -> None:
        llm_calls = trace_payload.get("llm_calls", [])
        primary_call = llm_calls[0] if isinstance(llm_calls, list) and llm_calls else {}
        if not isinstance(primary_call, dict):
            primary_call = {}

        generation_model = str(primary_call.get("model_id") or trace_payload.get("model_id") or "")
        usage_details = primary_call.get("token_usage", {})
        if not isinstance(usage_details, dict):
            usage_details = {}

        with self._langfuse_client.start_as_current_observation(
            name="chat.send_message",
            as_type="generation",
            model=generation_model or None,
            input={
                "request_input_text": trace_payload.get("request_input_text", ""),
                "system_prompt_preview": trace_payload.get("provider_system_prompt_preview", ""),
                "provider_messages": trace_payload.get("provider_messages", []),
            },
            output=trace_payload.get("response_output_text", ""),
            usage_details=usage_details,
            metadata=trace_payload,
        ):
            if callable(getattr(self._langfuse_client, "update_current_trace", None)):
                self._langfuse_client.update_current_trace(
                    name="chat.send_message",
                    user_id=str(trace_payload.get("actor_user_id", "")),
                    session_id=str(trace_payload.get("session_id", "")),
                    input=trace_payload.get("request_input_text", ""),
                    output=trace_payload.get("response_output_text", ""),
                    metadata={
                        "provider": trace_payload.get("provider", ""),
                        "model_id": trace_payload.get("model_id", ""),
                        "used_engram_count": trace_payload.get("used_engram_count", 0),
                        "source_reference_count": trace_payload.get("source_reference_count", 0),
                    },
                )

            if not callable(getattr(self._langfuse_client, "create_event", None)):
                return
            if not isinstance(llm_calls, list):
                return
            for index, item in enumerate(llm_calls, start=1):
                if not isinstance(item, dict):
                    continue
                self._langfuse_client.create_event(
                    name=f"provider.generate.{index}",
                    input={
                        "system_prompt_preview": trace_payload.get(
                            "provider_system_prompt_preview", ""
                        ),
                        "messages": trace_payload.get("provider_messages", []),
                    },
                    output=trace_payload.get("response_output_text", ""),
                    metadata={
                        "provider": item.get("provider"),
                        "model_id": item.get("model_id"),
                        "duration_ms": item.get("duration_ms"),
                        "call_type": item.get("call_type", "generate"),
                        "token_usage": item.get("token_usage", {}),
                    },
                )

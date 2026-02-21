from __future__ import annotations

from collections.abc import Callable, Generator
from dataclasses import dataclass
from time import perf_counter
from typing import Any
from uuid import UUID

from app.chat_repository import (
    ChatMessageCreateRepositoryRequest,
    MessageMetadata,
    create_chat_message,
    list_chat_messages,
)
from app.models import (
    ChatDebugEmbeddingCall,
    ChatDebugLLMCall,
    ChatDebugProviderMessage,
    ChatDebugTrace,
    ChatMessageCreateRequest,
    ChatMessageRecord,
    ChatSessionRecord,
)
from app.observability import (
    ChatDebugCollector,
    ChatDebugTelemetryPublisher,
    bind_chat_debug_collector,
)
from app.providers.base import ProviderGenerateRequest, ProviderGenerateResult, ProviderMessage
from app.providers.errors import (
    ProviderAPIError,
    ProviderAuthError,
    ProviderError,
    ProviderRateLimitError,
    ProviderRequestError,
)

from .context import AssembledChatContext, ChatContextRequest, assemble_chat_context
from .errors import ChatProviderExecutionError, ChatSessionNotFoundError, ChatValidationError
from .session_lifecycle import LifecycleMaintenanceResult

_PROVIDER_ERROR_STATUS_MAP: list[tuple[type[ProviderError], int]] = [
    (ProviderRateLimitError, 429),
    (ProviderAuthError, 503),
    (ProviderRequestError, 400),
    (ProviderAPIError, 502),
    (ProviderError, 502),
]


@dataclass(frozen=True)
class PreparedGeneration:
    session: ChatSessionRecord
    user_message: ChatMessageRecord
    context: AssembledChatContext
    provider_request: ProviderGenerateRequest
    debug_collector: ChatDebugCollector
    prepare_duration_ms: float
    context_duration_ms: float
    history_load_duration_ms: float


@dataclass(frozen=True)
class DebugBuildContext:
    actor_user_id: UUID
    prepared: PreparedGeneration
    result: ProviderGenerateResult
    llm_call_duration_ms: float
    persistence_duration_ms: float
    total_duration_ms: float
    call_type: str


@dataclass(frozen=True)
class ChatMessageRuntimeDependencies:
    embedding_dim: int
    chat_debug_enabled: bool
    chat_debug_include_raw_text: bool
    debug_publisher: ChatDebugTelemetryPublisher
    get_session: Callable[..., ChatSessionRecord]
    run_session_lifecycle_maintenance: Callable[..., LifecycleMaintenanceResult]


def _history_as_provider_messages(
    messages: list[ChatMessageRecord],
    history_limit: int = 40,
) -> list[ProviderMessage]:
    normalized = [item for item in messages if item.role in {"user", "assistant"}]
    trimmed = normalized[-history_limit:]
    return [ProviderMessage(role=item.role, content=item.content_text) for item in trimmed]


def _build_system_prompt(base_prompt: str, context_markdown: str) -> str:
    sections: list[str] = []
    if base_prompt.strip():
        sections.append(base_prompt.strip())
    if context_markdown:
        sections.append(
            "Use the retrieved engram context below when relevant. "
            "If you cite evidence, prefer source URLs from the context.\n\n"
            f"{context_markdown}"
        )
    return "\n\n".join(sections)


def _normalize_spaces(value: str) -> str:
    return " ".join(value.strip().split())


def _truncate_text(value: str, max_chars: int) -> str:
    if len(value) <= max_chars:
        return value
    return value[: max_chars - 3].rstrip() + "..."


def _duration_ms(started_at: float) -> float:
    return (perf_counter() - started_at) * 1000.0


def _preview_text(value: str, max_chars: int = 360) -> str:
    normalized = _normalize_spaces(value)
    return _truncate_text(normalized, max_chars=max_chars)


def _estimate_token_usage(*, input_chars: int, output_chars: int) -> dict[str, int]:
    input_tokens = max(1, round(input_chars / 4))
    output_tokens = max(1, round(output_chars / 4))
    return {
        "input_tokens": input_tokens,
        "output_tokens": output_tokens,
        "total_tokens": input_tokens + output_tokens,
    }


def _build_provider_message_debug(
    messages: list[ProviderMessage],
) -> list[ChatDebugProviderMessage]:
    return [
        ChatDebugProviderMessage(
            role=message.role,
            content_preview=_preview_text(message.content),
            char_count=len(message.content),
        )
        for message in messages
    ]


def _build_embedding_call_debug(embedding_calls: list[Any]) -> list[ChatDebugEmbeddingCall]:
    return [
        ChatDebugEmbeddingCall(
            operation=item.operation,
            provider_id=item.provider_id,
            duration_ms=round(item.duration_ms, 2),
            item_count=item.item_count,
            text_chars=item.text_chars,
            dim=item.dim,
            used_fallback=item.used_fallback,
        )
        for item in embedding_calls
    ]


def resolve_token_usage(
    *,
    token_usage: dict[str, int] | None,
    input_chars: int,
    output_chars: int,
) -> tuple[dict[str, int], bool]:
    resolved = token_usage or {}
    if int(resolved.get("total_tokens", 0)) <= 0:
        return _estimate_token_usage(input_chars=input_chars, output_chars=output_chars), True
    return resolved, False


def raise_provider_error(exc: Exception) -> None:
    for exception_type, status_code in _PROVIDER_ERROR_STATUS_MAP:
        if isinstance(exc, exception_type):
            raise ChatProviderExecutionError(
                detail=str(exc),
                status_code=status_code,
                error_code=exc.code,
            ) from exc
    raise exc


class ChatMessageRuntime:
    def __init__(
        self,
        *,
        dependencies: ChatMessageRuntimeDependencies,
    ) -> None:
        self._embedding_dim = dependencies.embedding_dim
        self._chat_debug_enabled = dependencies.chat_debug_enabled
        self._chat_debug_include_raw_text = dependencies.chat_debug_include_raw_text
        self._debug_publisher = dependencies.debug_publisher
        self._get_session = dependencies.get_session
        self._run_session_lifecycle_maintenance = (
            dependencies.run_session_lifecycle_maintenance
        )

    def prepare_generation(
        self,
        *,
        actor_user_id: UUID,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
    ) -> PreparedGeneration:
        prepare_started_at = perf_counter()
        if not payload.content_text.strip():
            raise ChatValidationError("Message content cannot be empty")

        session = self._get_session(actor_user_id=actor_user_id, session_id=session_id)
        user_message = create_chat_message(
            request=ChatMessageCreateRepositoryRequest(
                session_id=session.session_id,
                actor_user_id=actor_user_id,
                role="user",
                content_text=payload.content_text,
            ),
        )
        if not user_message:
            raise ChatSessionNotFoundError()

        debug_collector = ChatDebugCollector()
        context_started_at = perf_counter()
        with bind_chat_debug_collector(debug_collector):
            context = assemble_chat_context(
                request=ChatContextRequest(
                    session=session,
                    actor_user_id=actor_user_id,
                    user_query=payload.content_text,
                    embedding_dim=self._embedding_dim,
                ),
            )
        context_duration_ms = _duration_ms(context_started_at)

        history_started_at = perf_counter()
        history = list_chat_messages(
            session_id=session.session_id,
            actor_user_id=actor_user_id,
            limit=200,
            offset=0,
        )
        history_load_duration_ms = _duration_ms(history_started_at)
        provider_request = ProviderGenerateRequest(
            model_id=session.model_id,
            messages=_history_as_provider_messages(history),
            system_prompt=_build_system_prompt(session.system_prompt, context.context_markdown),
        )
        return PreparedGeneration(
            session=session,
            user_message=user_message,
            context=context,
            provider_request=provider_request,
            debug_collector=debug_collector,
            prepare_duration_ms=_duration_ms(prepare_started_at),
            context_duration_ms=context_duration_ms,
            history_load_duration_ms=history_load_duration_ms,
        )

    def persist_assistant_reply(
        self,
        *,
        actor_user_id: UUID,
        prepared: PreparedGeneration,
        result: ProviderGenerateResult,
    ) -> ChatMessageRecord:
        assistant_message = create_chat_message(
            request=ChatMessageCreateRepositoryRequest(
                session_id=prepared.session.session_id,
                actor_user_id=actor_user_id,
                role="assistant",
                content_text=result.text,
                metadata=MessageMetadata(
                    provider=prepared.session.provider.value,
                    model_id=prepared.session.model_id,
                    token_usage_json=result.token_usage,
                    used_engram_ids=prepared.context.used_engram_ids,
                ),
            ),
        )
        if not assistant_message:
            raise ChatSessionNotFoundError()
        return assistant_message

    def build_debug_trace(
        self,
        *,
        context: DebugBuildContext,
    ) -> ChatDebugTrace | None:
        if not self._chat_debug_enabled:
            return None

        response_output_text = context.result.text if self._chat_debug_include_raw_text else ""
        provider_message_debug = _build_provider_message_debug(
            context.prepared.provider_request.messages
        )
        embedding_calls = _build_embedding_call_debug(
            context.prepared.debug_collector.embedding_calls
        )

        input_chars = sum(
            len(item.content) for item in context.prepared.provider_request.messages
        ) + len(context.prepared.provider_request.system_prompt)
        output_chars = len(context.result.text)
        token_usage, token_usage_is_estimated = resolve_token_usage(
            token_usage=context.result.token_usage,
            input_chars=input_chars,
            output_chars=output_chars,
        )

        llm_call = ChatDebugLLMCall(
            provider=context.prepared.session.provider.value,
            model_id=context.prepared.session.model_id,
            call_type=context.call_type,
            duration_ms=round(context.llm_call_duration_ms, 2),
            input_chars=input_chars,
            output_chars=output_chars,
            token_usage=token_usage,
            token_usage_is_estimated=token_usage_is_estimated,
        )

        debug_trace = ChatDebugTrace(
            total_duration_ms=round(context.total_duration_ms, 2),
            prepare_duration_ms=round(context.prepared.prepare_duration_ms, 2),
            context_duration_ms=round(context.prepared.context_duration_ms, 2),
            history_load_duration_ms=round(context.prepared.history_load_duration_ms, 2),
            llm_call_duration_ms=round(context.llm_call_duration_ms, 2),
            persistence_duration_ms=round(context.persistence_duration_ms, 2),
            used_engram_count=len(context.prepared.context.used_engram_ids),
            used_document_chunk_count=len(context.prepared.context.used_document_chunk_ids),
            source_reference_count=len(context.prepared.context.source_references),
            provider=context.prepared.session.provider.value,
            model_id=context.prepared.session.model_id,
            request_input_text=context.prepared.user_message.content_text,
            response_output_text=response_output_text,
            provider_system_prompt_preview=_preview_text(
                context.prepared.provider_request.system_prompt
            ),
            provider_messages=provider_message_debug,
            embedding_calls=embedding_calls,
            llm_calls=[llm_call],
        )

        self._debug_publisher.publish_chat_trace(
            trace_payload={
                "actor_user_id": str(context.actor_user_id),
                "session_id": str(context.prepared.session.session_id),
                **debug_trace.model_dump(mode="json"),
            }
        )
        return debug_trace

    @staticmethod
    def build_stream_meta_payload(*, prepared: PreparedGeneration) -> dict[str, Any]:
        return {
            "session_id": prepared.session.session_id,
            "message_id": prepared.user_message.message_id,
            "used_engram_ids": prepared.context.used_engram_ids,
            "used_document_chunk_ids": prepared.context.used_document_chunk_ids,
            "source_references": prepared.context.source_references,
        }

    def yield_stream_chunks(
        self,
        *,
        adapter: Any,
        prepared: PreparedGeneration,
    ) -> Generator[tuple[str, dict[str, Any]], None, tuple[str, float] | None]:
        chunks: list[str] = []
        llm_started_at = perf_counter()
        try:
            for part in adapter.stream_generate(prepared.provider_request):
                chunk_text = str(part)
                if not chunk_text:
                    continue
                chunks.append(chunk_text)
                yield ("chunk", {"text": chunk_text})
        except Exception as exc:
            try:
                raise_provider_error(exc)
            except ChatProviderExecutionError as provider_error:
                yield (
                    "error",
                    {
                        "detail": provider_error.detail,
                        "status_code": provider_error.status_code,
                        "error_code": provider_error.error_code,
                    },
                )
                return None
            raise
        return "".join(chunks), _duration_ms(llm_started_at)

    def persist_stream_completion(
        self,
        *,
        actor_user_id: UUID,
        prepared: PreparedGeneration,
        result: ProviderGenerateResult,
    ) -> tuple[ChatMessageRecord, float]:
        persistence_started_at = perf_counter()
        assistant_message = self.persist_assistant_reply(
            actor_user_id=actor_user_id,
            prepared=prepared,
            result=result,
        )
        self._run_session_lifecycle_maintenance(
            actor_user_id=actor_user_id,
            session=prepared.session,
        )
        return assistant_message, _duration_ms(persistence_started_at)

    @staticmethod
    def build_stream_done_payload(
        *,
        prepared: PreparedGeneration,
        assistant_message: ChatMessageRecord,
        full_text: str,
        debug_trace: ChatDebugTrace | None,
    ) -> dict[str, Any]:
        return {
            "session_id": prepared.session.session_id,
            "message_id": prepared.user_message.message_id,
            "reply_message_id": assistant_message.message_id,
            "assistant_text": full_text,
            "used_engram_ids": prepared.context.used_engram_ids,
            "used_document_chunk_ids": prepared.context.used_document_chunk_ids,
            "source_references": prepared.context.source_references,
            "debug_trace": debug_trace.model_dump(mode="json") if debug_trace else None,
        }

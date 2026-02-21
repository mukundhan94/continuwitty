from __future__ import annotations

from collections.abc import Generator, Iterator
from time import perf_counter
from typing import Any
from uuid import UUID

from app.config import get_settings
from app.models import ChatDebugTrace, ChatMessageCreateRequest, ChatMessageRecord, ChatSendResponse
from app.observability import ChatDebugTelemetryPublisher
from app.providers.base import ProviderGenerateResult
from app.providers.registry import get_provider_adapter

from .errors import ChatServiceError
from .message_runtime import (
    ChatMessageRuntime,
    PreparedGeneration,
    raise_provider_error,
    resolve_token_usage,
)
from .message_runtime import (
    DebugBuildContext as _DebugBuildContext,
)
from .session_operations import ChatSessionOperationsMixin


def _duration_ms(started_at: float) -> float:
    return (perf_counter() - started_at) * 1000.0


def _resolve_token_usage(
    *,
    token_usage: dict[str, int] | None,
    input_chars: int,
    output_chars: int,
) -> tuple[dict[str, int], bool]:
    return resolve_token_usage(
        token_usage=token_usage,
        input_chars=input_chars,
        output_chars=output_chars,
    )


class ChatService(ChatSessionOperationsMixin):
    def __init__(self, embedding_dim: int) -> None:
        self._embedding_dim = embedding_dim
        settings = get_settings()
        self._chat_debug_enabled = settings.chat_debug_enabled
        self._chat_debug_include_raw_text = settings.chat_debug_include_raw_text
        self._debug_publisher = ChatDebugTelemetryPublisher(
            console_enabled=settings.chat_debug_log_console,
            langfuse_enabled=settings.langfuse_enabled,
            langfuse_public_key=settings.langfuse_public_key,
            langfuse_secret_key=settings.langfuse_secret_key,
            langfuse_host=settings.langfuse_host,
        )
        self._runtime = ChatMessageRuntime(
            embedding_dim=self._embedding_dim,
            chat_debug_enabled=self._chat_debug_enabled,
            chat_debug_include_raw_text=self._chat_debug_include_raw_text,
            debug_publisher=self._debug_publisher,
            get_session=self.get_session,
            run_session_lifecycle_maintenance=self._run_session_lifecycle_maintenance,
        )

    def _prepare_generation(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
    ) -> PreparedGeneration:
        return self._runtime.prepare_generation(
            actor_user_id=actor_user_id,
            session_id=session_id,
            payload=payload,
        )

    @staticmethod
    def _raise_provider_error(exc: Exception) -> None:
        raise_provider_error(exc)

    def _persist_assistant_reply(
        self,
        *,
        actor_user_id: UUID,
        prepared: PreparedGeneration,
        result: ProviderGenerateResult,
    ) -> ChatMessageRecord:
        return self._runtime.persist_assistant_reply(
            actor_user_id=actor_user_id,
            prepared=prepared,
            result=result,
        )

    def _build_debug_trace(
        self,
        *,
        context: _DebugBuildContext,
    ) -> ChatDebugTrace | None:
        return self._runtime.build_debug_trace(context=context)

    def send_message(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
    ) -> ChatSendResponse:
        call_started_at = perf_counter()
        prepared = self._prepare_generation(
            actor_user_id=actor_user_id,
            session_id=session_id,
            payload=payload,
        )
        adapter = get_provider_adapter(prepared.session.provider)
        llm_started_at = perf_counter()
        try:
            result = adapter.generate(prepared.provider_request)
        except Exception as exc:  # pragma: no cover - mapped below
            self._raise_provider_error(exc)
            raise
        llm_call_duration_ms = _duration_ms(llm_started_at)

        persistence_started_at = perf_counter()
        assistant_message = self._persist_assistant_reply(
            actor_user_id=actor_user_id,
            prepared=prepared,
            result=result,
        )
        self._run_session_lifecycle_maintenance(
            actor_user_id=actor_user_id,
            session=prepared.session,
        )
        persistence_duration_ms = _duration_ms(persistence_started_at)
        debug_trace = self._build_debug_trace(
            context=_DebugBuildContext(
                actor_user_id=actor_user_id,
                prepared=prepared,
                result=result,
                llm_call_duration_ms=llm_call_duration_ms,
                persistence_duration_ms=persistence_duration_ms,
                total_duration_ms=_duration_ms(call_started_at),
                call_type="generate",
            ),
        )
        return ChatSendResponse(
            session_id=prepared.session.session_id,
            message_id=prepared.user_message.message_id,
            reply_message_id=assistant_message.message_id,
            assistant_text=result.text,
            used_engram_ids=prepared.context.used_engram_ids,
            used_document_chunk_ids=prepared.context.used_document_chunk_ids,
            source_references=prepared.context.source_references,
            debug_trace=debug_trace,
        )

    @staticmethod
    def _build_stream_meta_payload(*, prepared: PreparedGeneration) -> dict[str, Any]:
        return ChatMessageRuntime.build_stream_meta_payload(prepared=prepared)

    def _yield_stream_chunks(
        self,
        *,
        adapter: Any,
        prepared: PreparedGeneration,
    ) -> Generator[tuple[str, dict[str, Any]], None, tuple[str, float] | None]:
        return (
            yield from self._runtime.yield_stream_chunks(
                adapter=adapter,
                prepared=prepared,
            )
        )

    def _persist_stream_completion(
        self,
        *,
        actor_user_id: UUID,
        prepared: PreparedGeneration,
        result: ProviderGenerateResult,
    ) -> tuple[ChatMessageRecord, float]:
        persistence_started_at = perf_counter()
        assistant_message = self._persist_assistant_reply(
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
    def _build_stream_done_payload(
        *,
        prepared: PreparedGeneration,
        assistant_message: ChatMessageRecord,
        full_text: str,
        debug_trace: ChatDebugTrace | None,
    ) -> dict[str, Any]:
        return ChatMessageRuntime.build_stream_done_payload(
            prepared=prepared,
            assistant_message=assistant_message,
            full_text=full_text,
            debug_trace=debug_trace,
        )

    def stream_message_events(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
    ) -> Iterator[tuple[str, dict[str, Any]]]:
        call_started_at = perf_counter()
        prepared = self._prepare_generation(
            actor_user_id=actor_user_id,
            session_id=session_id,
            payload=payload,
        )
        adapter = get_provider_adapter(prepared.session.provider)

        yield ("meta", self._build_stream_meta_payload(prepared=prepared))
        streamed = yield from self._yield_stream_chunks(adapter=adapter, prepared=prepared)
        if streamed is None:
            return
        full_text, llm_call_duration_ms = streamed
        result = ProviderGenerateResult(
            provider=prepared.session.provider,
            model_id=prepared.session.model_id,
            text=full_text,
            token_usage={},
        )
        try:
            assistant_message, persistence_duration_ms = self._persist_stream_completion(
                actor_user_id=actor_user_id,
                prepared=prepared,
                result=result,
            )
        except ChatServiceError as exc:
            yield (
                "error",
                {
                    "detail": exc.detail,
                    "status_code": exc.status_code,
                    "error_code": "persistence_error",
                },
            )
            return

        debug_trace = self._build_debug_trace(
            context=_DebugBuildContext(
                actor_user_id=actor_user_id,
                prepared=prepared,
                result=result,
                llm_call_duration_ms=llm_call_duration_ms,
                persistence_duration_ms=persistence_duration_ms,
                total_duration_ms=_duration_ms(call_started_at),
                call_type="stream_generate",
            ),
        )

        yield (
            "done",
            self._build_stream_done_payload(
                prepared=prepared,
                assistant_message=assistant_message,
                full_text=full_text,
                debug_trace=debug_trace,
            ),
        )

from __future__ import annotations

from collections.abc import Iterator
from dataclasses import dataclass
from time import perf_counter
from typing import Any
from uuid import UUID

from app.chat_repository import (
    create_chat_message,
    create_chat_session,
    get_chat_session,
    list_chat_messages,
    list_chat_sessions,
    list_pinned_documents,
    list_pinned_engram_summaries,
    list_pinned_engrams,
    pin_document_to_session,
    pin_engram_to_session,
    unpin_document_from_session,
    unpin_engram_from_session,
    update_chat_session,
)
from app.config import get_settings
from app.models import (
    ChatDebugEmbeddingCall,
    ChatDebugLLMCall,
    ChatDebugProviderMessage,
    ChatDebugTrace,
    ChatMessageCreateRequest,
    ChatMessageRecord,
    ChatSendResponse,
    ChatSessionCreateRequest,
    ChatSessionRecord,
    ChatSessionUpdateRequest,
    ContinueSessionRequest,
    ContinueSessionResponse,
    EngramSummary,
    MemoryEngramCreate,
    PinDocumentRequest,
    PinEngramRequest,
    PinnedDocumentRecord,
    SaveSessionAsEngramRequest,
    SaveSessionAsEngramResponse,
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
from app.providers.registry import get_provider_adapter
from app.repository import create_engram

from .context import AssembledChatContext, assemble_chat_context
from .errors import (
    ChatProviderExecutionError,
    ChatServiceError,
    ChatSessionNotFoundError,
    ChatValidationError,
)

_GENERIC_SNAPSHOT_ABSTRACTS = {
    "",
    "snapshot from active chat session.",
    "snapshot from active chat session",
    "chat snapshot",
    "session snapshot",
}


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


def _transcript_markdown(session: ChatSessionRecord, messages: list[ChatMessageRecord]) -> str:
    lines = [
        f"# Chat Session Snapshot: {session.title}",
        "",
        f"- Session ID: {session.session_id}",
        f"- Provider/Model: {session.provider.value}/{session.model_id}",
        "",
    ]
    for message in messages:
        lines.extend(
            [
                f"## {message.role.upper()} ({message.created_at.isoformat()})",
                message.content_text.strip() or "(empty)",
                "",
            ]
        )
    return "\n".join(lines).strip()


def _retrieval_text_from_messages(messages: list[ChatMessageRecord], tail_count: int = 8) -> str:
    tail = messages[-tail_count:]
    return " ".join(item.content_text.strip() for item in tail if item.content_text.strip())


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


def _is_generic_snapshot_abstract(value: str) -> bool:
    return _normalize_spaces(value).lower() in _GENERIC_SNAPSHOT_ABSTRACTS


def _derive_chat_snapshot_abstract(messages: list[ChatMessageRecord], max_chars: int = 320) -> str:
    for role in ("assistant", "user"):
        for message in reversed(messages):
            if message.role != role:
                continue
            normalized = _normalize_spaces(message.content_text)
            if not normalized:
                continue
            return _truncate_text(normalized, max_chars=max_chars)
    return ""


class ChatService:
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

    def create_session(
        self, actor_user_id: UUID, payload: ChatSessionCreateRequest
    ) -> ChatSessionRecord:
        return create_chat_session(owner_user_id=actor_user_id, payload=payload)

    def list_sessions(
        self,
        actor_user_id: UUID,
        project_id: str | None,
        limit: int,
        offset: int,
    ) -> list[ChatSessionRecord]:
        return list_chat_sessions(
            actor_user_id=actor_user_id,
            project_id=project_id,
            limit=limit,
            offset=offset,
        )

    def get_session(self, actor_user_id: UUID, session_id: UUID) -> ChatSessionRecord:
        session = get_chat_session(session_id=session_id, actor_user_id=actor_user_id)
        if not session:
            raise ChatSessionNotFoundError()
        return session

    def update_session(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: ChatSessionUpdateRequest,
    ) -> ChatSessionRecord:
        updated = update_chat_session(
            session_id=session_id,
            actor_user_id=actor_user_id,
            payload=payload,
        )
        if not updated:
            raise ChatSessionNotFoundError()
        return updated

    def list_messages(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        limit: int,
        offset: int,
    ) -> list[ChatMessageRecord]:
        self.get_session(actor_user_id=actor_user_id, session_id=session_id)
        return list_chat_messages(
            session_id=session_id,
            actor_user_id=actor_user_id,
            limit=limit,
            offset=offset,
        )

    def list_pinned_engrams(
        self,
        actor_user_id: UUID,
        session_id: UUID,
    ) -> list[EngramSummary]:
        self.get_session(actor_user_id=actor_user_id, session_id=session_id)
        return list_pinned_engram_summaries(
            session_id=session_id,
            actor_user_id=actor_user_id,
        )

    def list_pinned_documents(
        self,
        actor_user_id: UUID,
        session_id: UUID,
    ) -> list[PinnedDocumentRecord]:
        self.get_session(actor_user_id=actor_user_id, session_id=session_id)
        return list_pinned_documents(
            session_id=session_id,
            actor_user_id=actor_user_id,
        )

    def pin_engram(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: PinEngramRequest,
    ):
        pinned = pin_engram_to_session(
            session_id=session_id,
            engram_id=payload.engram_id,
            actor_user_id=actor_user_id,
        )
        if not pinned:
            raise ChatValidationError("Session or engram is not accessible for pinning")
        return pinned

    def pin_document(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: PinDocumentRequest,
    ) -> PinnedDocumentRecord:
        pinned = pin_document_to_session(
            session_id=session_id,
            document_id=payload.document_id,
            actor_user_id=actor_user_id,
        )
        if not pinned:
            raise ChatValidationError("Session or document is not accessible for pinning")
        return pinned

    def unpin_engram(self, actor_user_id: UUID, session_id: UUID, engram_id: UUID) -> None:
        removed = unpin_engram_from_session(
            session_id=session_id,
            engram_id=engram_id,
            actor_user_id=actor_user_id,
        )
        if not removed:
            raise ChatSessionNotFoundError("Pinned engram not found for session")

    def unpin_document(self, actor_user_id: UUID, session_id: UUID, document_id: UUID) -> None:
        removed = unpin_document_from_session(
            session_id=session_id,
            document_id=document_id,
            actor_user_id=actor_user_id,
        )
        if not removed:
            raise ChatSessionNotFoundError("Pinned document not found for session")

    def _prepare_generation(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
    ) -> PreparedGeneration:
        prepare_started_at = perf_counter()
        if not payload.content_text.strip():
            raise ChatValidationError("Message content cannot be empty")

        session = self.get_session(actor_user_id=actor_user_id, session_id=session_id)
        user_message = create_chat_message(
            session_id=session.session_id,
            actor_user_id=actor_user_id,
            role="user",
            content_text=payload.content_text,
        )
        if not user_message:
            raise ChatSessionNotFoundError()

        debug_collector = ChatDebugCollector()
        context_started_at = perf_counter()
        with bind_chat_debug_collector(debug_collector):
            context = assemble_chat_context(
                session=session,
                actor_user_id=actor_user_id,
                user_query=payload.content_text,
                embedding_dim=self._embedding_dim,
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

    @staticmethod
    def _raise_provider_error(exc: Exception) -> None:
        if isinstance(exc, ProviderRateLimitError):
            raise ChatProviderExecutionError(
                detail=str(exc),
                status_code=429,
                error_code=exc.code,
            ) from exc
        if isinstance(exc, ProviderAuthError):
            raise ChatProviderExecutionError(
                detail=str(exc),
                status_code=503,
                error_code=exc.code,
            ) from exc
        if isinstance(exc, ProviderRequestError):
            raise ChatProviderExecutionError(
                detail=str(exc),
                status_code=400,
                error_code=exc.code,
            ) from exc
        if isinstance(exc, ProviderAPIError):
            raise ChatProviderExecutionError(
                detail=str(exc),
                status_code=502,
                error_code=exc.code,
            ) from exc
        if isinstance(exc, ProviderError):
            raise ChatProviderExecutionError(
                detail=str(exc),
                status_code=502,
                error_code=exc.code,
            ) from exc
        raise

    def _persist_assistant_reply(
        self,
        *,
        actor_user_id: UUID,
        prepared: PreparedGeneration,
        result: ProviderGenerateResult,
    ) -> ChatMessageRecord:
        assistant_message = create_chat_message(
            session_id=prepared.session.session_id,
            actor_user_id=actor_user_id,
            role="assistant",
            content_text=result.text,
            provider=prepared.session.provider.value,
            model_id=prepared.session.model_id,
            token_usage_json=result.token_usage,
            used_engram_ids=prepared.context.used_engram_ids,
        )
        if not assistant_message:
            raise ChatSessionNotFoundError()
        return assistant_message

    def _build_debug_trace(
        self,
        *,
        actor_user_id: UUID,
        prepared: PreparedGeneration,
        result: ProviderGenerateResult,
        llm_call_duration_ms: float,
        persistence_duration_ms: float,
        total_duration_ms: float,
        call_type: str,
    ) -> ChatDebugTrace | None:
        if not self._chat_debug_enabled:
            return None

        response_output_text = result.text if self._chat_debug_include_raw_text else ""
        provider_message_debug = [
            ChatDebugProviderMessage(
                role=message.role,
                content_preview=_preview_text(message.content),
                char_count=len(message.content),
            )
            for message in prepared.provider_request.messages
        ]

        embedding_calls = [
            ChatDebugEmbeddingCall(
                operation=item.operation,
                provider_id=item.provider_id,
                duration_ms=round(item.duration_ms, 2),
                item_count=item.item_count,
                text_chars=item.text_chars,
                dim=item.dim,
                used_fallback=item.used_fallback,
            )
            for item in prepared.debug_collector.embedding_calls
        ]

        input_chars = sum(len(item.content) for item in prepared.provider_request.messages) + len(
            prepared.provider_request.system_prompt
        )
        output_chars = len(result.text)
        token_usage = result.token_usage or {}
        token_usage_is_estimated = False
        if int(token_usage.get("total_tokens", 0)) <= 0:
            token_usage = _estimate_token_usage(input_chars=input_chars, output_chars=output_chars)
            token_usage_is_estimated = True

        llm_call = ChatDebugLLMCall(
            provider=prepared.session.provider.value,
            model_id=prepared.session.model_id,
            call_type=call_type,
            duration_ms=round(llm_call_duration_ms, 2),
            input_chars=input_chars,
            output_chars=output_chars,
            token_usage=token_usage,
            token_usage_is_estimated=token_usage_is_estimated,
        )

        debug_trace = ChatDebugTrace(
            total_duration_ms=round(total_duration_ms, 2),
            prepare_duration_ms=round(prepared.prepare_duration_ms, 2),
            context_duration_ms=round(prepared.context_duration_ms, 2),
            history_load_duration_ms=round(prepared.history_load_duration_ms, 2),
            llm_call_duration_ms=round(llm_call_duration_ms, 2),
            persistence_duration_ms=round(persistence_duration_ms, 2),
            used_engram_count=len(prepared.context.used_engram_ids),
            used_document_chunk_count=len(prepared.context.used_document_chunk_ids),
            source_reference_count=len(prepared.context.source_references),
            provider=prepared.session.provider.value,
            model_id=prepared.session.model_id,
            request_input_text=prepared.user_message.content_text,
            response_output_text=response_output_text,
            provider_system_prompt_preview=_preview_text(prepared.provider_request.system_prompt),
            provider_messages=provider_message_debug,
            embedding_calls=embedding_calls,
            llm_calls=[llm_call],
        )

        self._debug_publisher.publish_chat_trace(
            trace_payload={
                "actor_user_id": str(actor_user_id),
                "session_id": str(prepared.session.session_id),
                **debug_trace.model_dump(mode="json"),
            }
        )
        return debug_trace

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
        persistence_duration_ms = _duration_ms(persistence_started_at)
        debug_trace = self._build_debug_trace(
            actor_user_id=actor_user_id,
            prepared=prepared,
            result=result,
            llm_call_duration_ms=llm_call_duration_ms,
            persistence_duration_ms=persistence_duration_ms,
            total_duration_ms=_duration_ms(call_started_at),
            call_type="generate",
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

        yield (
            "meta",
            {
                "session_id": prepared.session.session_id,
                "message_id": prepared.user_message.message_id,
                "used_engram_ids": prepared.context.used_engram_ids,
                "used_document_chunk_ids": prepared.context.used_document_chunk_ids,
                "source_references": prepared.context.source_references,
            },
        )

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
                self._raise_provider_error(exc)
            except ChatProviderExecutionError as provider_error:
                yield (
                    "error",
                    {
                        "detail": provider_error.detail,
                        "status_code": provider_error.status_code,
                        "error_code": provider_error.error_code,
                    },
                )
                return
            raise
        llm_call_duration_ms = _duration_ms(llm_started_at)

        full_text = "".join(chunks)
        result = ProviderGenerateResult(
            provider=prepared.session.provider,
            model_id=prepared.session.model_id,
            text=full_text,
            token_usage={},
        )
        try:
            persistence_started_at = perf_counter()
            assistant_message = self._persist_assistant_reply(
                actor_user_id=actor_user_id,
                prepared=prepared,
                result=result,
            )
            persistence_duration_ms = _duration_ms(persistence_started_at)
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
            actor_user_id=actor_user_id,
            prepared=prepared,
            result=result,
            llm_call_duration_ms=llm_call_duration_ms,
            persistence_duration_ms=persistence_duration_ms,
            total_duration_ms=_duration_ms(call_started_at),
            call_type="stream_generate",
        )

        yield (
            "done",
            {
                "session_id": prepared.session.session_id,
                "message_id": prepared.user_message.message_id,
                "reply_message_id": assistant_message.message_id,
                "assistant_text": full_text,
                "used_engram_ids": prepared.context.used_engram_ids,
                "used_document_chunk_ids": prepared.context.used_document_chunk_ids,
                "source_references": prepared.context.source_references,
                "debug_trace": debug_trace.model_dump(mode="json") if debug_trace else None,
            },
        )

    def save_session_as_engram(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: SaveSessionAsEngramRequest,
    ) -> SaveSessionAsEngramResponse:
        session = self.get_session(actor_user_id=actor_user_id, session_id=session_id)
        messages = list_chat_messages(
            session_id=session.session_id,
            actor_user_id=actor_user_id,
            limit=500,
            offset=0,
        )
        if not messages:
            raise ChatValidationError("Cannot save an empty chat session as engram")

        abstract = payload.abstract.strip()
        if _is_generic_snapshot_abstract(abstract):
            derived_abstract = _derive_chat_snapshot_abstract(messages)
            if derived_abstract:
                abstract = derived_abstract

        created = create_engram(
            payload=MemoryEngramCreate(
                project_id=session.project_id,
                thread_id=f"chat-session:{session.session_id}",
                title=payload.title,
                abstract=abstract,
                detailed_summary_markdown=_transcript_markdown(session, messages),
                tags=payload.tags,
                keywords=payload.keywords,
                visibility_scope=payload.visibility_scope.value,
                source_session_id=session.session_id,
                retrieval_text=_retrieval_text_from_messages(messages),
            ),
            embedding_dim=self._embedding_dim,
            owner_user_id=actor_user_id,
        )
        return SaveSessionAsEngramResponse(
            engram_id=created.engram_id,
            session_id=session.session_id,
            created_at=created.created_at,
        )

    def continue_session(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: ContinueSessionRequest,
    ) -> ContinueSessionResponse:
        session = self.get_session(actor_user_id=actor_user_id, session_id=session_id)
        continued = create_chat_session(
            owner_user_id=actor_user_id,
            payload=ChatSessionCreateRequest(
                project_id=session.project_id,
                title=payload.title or f"{session.title} (continued)",
                provider=session.provider,
                model_id=session.model_id,
                system_prompt=session.system_prompt,
                visibility_scope=session.visibility_scope,
                autosave_enabled=session.autosave_enabled,
            ),
        )

        carried_ids: list[UUID] = []
        for pinned in list_pinned_engrams(
            session_id=session.session_id, actor_user_id=actor_user_id
        ):
            copied = pin_engram_to_session(
                session_id=continued.session_id,
                engram_id=pinned.engram_id,
                actor_user_id=actor_user_id,
            )
            if copied:
                carried_ids.append(copied.engram_id)

        for pinned_document in list_pinned_documents(
            session_id=session.session_id, actor_user_id=actor_user_id
        ):
            pin_document_to_session(
                session_id=continued.session_id,
                document_id=pinned_document.document_id,
                actor_user_id=actor_user_id,
            )

        return ContinueSessionResponse(
            session=continued,
            carried_engram_ids=carried_ids,
        )

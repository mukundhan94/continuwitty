from __future__ import annotations

from collections.abc import Iterator
from dataclasses import dataclass
from typing import Any
from uuid import UUID

from app.chat_repository import (
    create_chat_message,
    create_chat_session,
    get_chat_session,
    list_chat_messages,
    list_chat_sessions,
    list_pinned_engrams,
    pin_engram_to_session,
    unpin_engram_from_session,
    update_chat_session,
)
from app.models import (
    ChatMessageCreateRequest,
    ChatMessageRecord,
    ChatSendResponse,
    ChatSessionCreateRequest,
    ChatSessionRecord,
    ChatSessionUpdateRequest,
    ContinueSessionRequest,
    ContinueSessionResponse,
    MemoryEngramCreate,
    PinEngramRequest,
    SaveSessionAsEngramRequest,
    SaveSessionAsEngramResponse,
)
from app.providers.base import ProviderGenerateRequest, ProviderGenerateResult, ProviderMessage
from app.providers.errors import (
    ProviderAPIError,
    ProviderAuthError,
    ProviderError,
    ProviderRateLimitError,
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


@dataclass(frozen=True)
class PreparedGeneration:
    session: ChatSessionRecord
    user_message: ChatMessageRecord
    context: AssembledChatContext
    provider_request: ProviderGenerateRequest


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


class ChatService:
    def __init__(self, embedding_dim: int) -> None:
        self._embedding_dim = embedding_dim

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

    def unpin_engram(self, actor_user_id: UUID, session_id: UUID, engram_id: UUID) -> None:
        removed = unpin_engram_from_session(
            session_id=session_id,
            engram_id=engram_id,
            actor_user_id=actor_user_id,
        )
        if not removed:
            raise ChatSessionNotFoundError("Pinned engram not found for session")

    def _prepare_generation(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
    ) -> PreparedGeneration:
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

        context = assemble_chat_context(
            session=session,
            actor_user_id=actor_user_id,
            user_query=payload.content_text,
            embedding_dim=self._embedding_dim,
        )
        history = list_chat_messages(
            session_id=session.session_id,
            actor_user_id=actor_user_id,
            limit=200,
            offset=0,
        )
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

    def send_message(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
    ) -> ChatSendResponse:
        prepared = self._prepare_generation(
            actor_user_id=actor_user_id,
            session_id=session_id,
            payload=payload,
        )
        adapter = get_provider_adapter(prepared.session.provider)
        try:
            result = adapter.generate(prepared.provider_request)
        except Exception as exc:  # pragma: no cover - mapped below
            self._raise_provider_error(exc)
            raise

        assistant_message = self._persist_assistant_reply(
            actor_user_id=actor_user_id,
            prepared=prepared,
            result=result,
        )
        return ChatSendResponse(
            session_id=prepared.session.session_id,
            message_id=prepared.user_message.message_id,
            reply_message_id=assistant_message.message_id,
            assistant_text=result.text,
            used_engram_ids=prepared.context.used_engram_ids,
            source_references=prepared.context.source_references,
        )

    def stream_message_events(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: ChatMessageCreateRequest,
    ) -> Iterator[tuple[str, dict[str, Any]]]:
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
                "source_references": prepared.context.source_references,
            },
        )

        chunks: list[str] = []
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

        full_text = "".join(chunks)
        result = ProviderGenerateResult(
            provider=prepared.session.provider,
            model_id=prepared.session.model_id,
            text=full_text,
            token_usage={},
        )
        try:
            assistant_message = self._persist_assistant_reply(
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

        yield (
            "done",
            {
                "session_id": prepared.session.session_id,
                "message_id": prepared.user_message.message_id,
                "reply_message_id": assistant_message.message_id,
                "assistant_text": full_text,
                "used_engram_ids": prepared.context.used_engram_ids,
                "source_references": prepared.context.source_references,
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

        created = create_engram(
            payload=MemoryEngramCreate(
                project_id=session.project_id,
                thread_id=f"chat-session:{session.session_id}",
                title=payload.title,
                abstract=payload.abstract,
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

        return ContinueSessionResponse(
            session=continued,
            carried_engram_ids=carried_ids,
        )

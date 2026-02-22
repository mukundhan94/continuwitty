from __future__ import annotations

from datetime import UTC, datetime
from uuid import UUID, uuid4

import pytest

from app.chat import service as chat_service_module
from app.chat.context import AssembledChatContext
from app.chat.errors import ChatProviderExecutionError
from app.chat.service import ChatService, PreparedGeneration
from app.models import (
    ChatMessageCreateRequest,
    ChatMessageRecord,
    ChatProvider,
    ChatSessionRecord,
    VisibilityScope,
)
from app.observability import ChatDebugCollector
from app.providers.base import ProviderGenerateRequest, ProviderGenerateResult, ProviderMessage
from app.providers.errors import (
    ProviderAPIError,
    ProviderAuthError,
    ProviderError,
    ProviderRateLimitError,
    ProviderRequestError,
)


def _session(owner_user_id: UUID) -> ChatSessionRecord:
    now = datetime.now(UTC)
    return ChatSessionRecord(
        session_id=uuid4(),
        owner_user_id=owner_user_id,
        project_id="project-chat",
        title="Session",
        provider=ChatProvider.openai,
        model_id="gpt-4o-mini",
        system_prompt="be helpful",
        visibility_scope=VisibilityScope.private,
        autosave_enabled=False,
        created_at=now,
        updated_at=now,
    )


def _message(
    *,
    message_id: UUID,
    session_id: UUID,
    role: str,
    content_text: str,
) -> ChatMessageRecord:
    return ChatMessageRecord(
        message_id=message_id,
        session_id=session_id,
        role=role,
        content_text=content_text,
        provider=None,
        model_id=None,
        token_usage_json={},
        used_engram_ids=[],
        created_at=datetime.now(UTC),
    )


def _context_with_source(referenced_engram_id: UUID) -> AssembledChatContext:
    return AssembledChatContext(
        context_markdown="ctx",
        used_engram_ids=[referenced_engram_id],
        used_document_chunk_ids=[],
        source_references=[
            {
                "engram_id": referenced_engram_id,
                "engram_title": "Referenced",
                "url": "https://example.com/source",
                "title": "Source",
                "snippet": "Snippet",
                "captured_at": datetime.now(UTC),
            }
        ],
    )


class _FixedReplyAdapter:
    def generate(self, request):  # noqa: ANN001
        return ProviderGenerateResult(
            provider=ChatProvider.openai,
            model_id=request.model_id,
            text="Proceed with option A.",
            token_usage={"input_tokens": 9, "output_tokens": 4, "total_tokens": 13},
        )


def _prepared_generation(
    *,
    session: ChatSessionRecord,
    user_message: ChatMessageRecord,
    context: AssembledChatContext,
) -> PreparedGeneration:
    return PreparedGeneration(
        session=session,
        user_message=user_message,
        context=context,
        provider_request=ProviderGenerateRequest(
            model_id=session.model_id,
            messages=[ProviderMessage(role="user", content=user_message.content_text)],
            system_prompt="system context",
        ),
        debug_collector=ChatDebugCollector(),
        prepare_duration_ms=1.5,
        context_duration_ms=0.8,
        history_load_duration_ms=0.5,
    )


def test_send_message_returns_used_engram_ids_and_sources(monkeypatch) -> None:
    actor_id = uuid4()
    session = _session(actor_id)
    user_message_id = uuid4()
    reply_message_id = uuid4()
    referenced_engram_id = uuid4()

    service = ChatService(embedding_dim=256)
    user_message = _message(
        message_id=user_message_id,
        session_id=session.session_id,
        role="user",
        content_text="How should we proceed?",
    )
    assistant_message = _message(
        message_id=reply_message_id,
        session_id=session.session_id,
        role="assistant",
        content_text="Proceed with option A.",
    )

    monkeypatch.setattr(
        "app.chat.session_operations.get_chat_session", lambda session_id, actor_user_id: session
    )
    monkeypatch.setattr(
        "app.chat.message_runtime.assemble_chat_context",
        lambda **kwargs: _context_with_source(referenced_engram_id),
    )
    monkeypatch.setattr("app.chat.message_runtime.list_chat_messages", lambda **kwargs: [user_message])

    def _fake_create_chat_message(*, request):
        if request.role == "user":
            return user_message
        assert request.metadata and request.metadata.used_engram_ids == [referenced_engram_id]
        return assistant_message

    monkeypatch.setattr("app.chat.message_runtime.create_chat_message", _fake_create_chat_message)
    monkeypatch.setattr("app.chat.service.get_provider_adapter", lambda provider: _FixedReplyAdapter())

    response = service.send_message(
        actor_user_id=actor_id,
        session_id=session.session_id,
        payload=ChatMessageCreateRequest(content_text="How should we proceed?"),
    )

    assert (
        response.session_id,
        response.message_id,
        response.reply_message_id,
        response.assistant_text,
        response.used_engram_ids,
        response.used_document_chunk_ids,
    ) == (session.session_id, user_message_id, reply_message_id, "Proceed with option A.", [referenced_engram_id], [])
    assert response.source_references[0].url == "https://example.com/source"
    assert response.debug_trace is not None
    assert response.debug_trace.llm_calls[0].provider == "openai"
    assert response.debug_trace.llm_calls[0].token_usage["total_tokens"] == 13
    assert response.debug_trace.request_input_text == "How should we proceed?"


def test_resolve_token_usage_prefers_provider_total() -> None:
    provided = {"input_tokens": 3, "output_tokens": 2, "total_tokens": 5}

    resolved, estimated = chat_service_module._resolve_token_usage(
        token_usage=provided,
        input_chars=100,
        output_chars=50,
    )

    assert resolved == provided
    assert estimated is False


def test_resolve_token_usage_estimates_when_total_missing() -> None:
    resolved, estimated = chat_service_module._resolve_token_usage(
        token_usage={"input_tokens": 3},
        input_chars=120,
        output_chars=44,
    )

    assert estimated is True
    assert resolved["total_tokens"] > 0
    assert resolved["input_tokens"] > 0
    assert resolved["output_tokens"] > 0


def test_stream_message_events_emits_meta_chunks_and_done(monkeypatch) -> None:
    actor_id = uuid4()
    session = _session(actor_id)
    user_message = _message(
        message_id=uuid4(),
        session_id=session.session_id,
        role="user",
        content_text="Stream this response",
    )
    assistant_message = _message(
        message_id=uuid4(),
        session_id=session.session_id,
        role="assistant",
        content_text="part-1 part-2",
    )
    prepared = _prepared_generation(
        session=session,
        user_message=user_message,
        context=AssembledChatContext(
            context_markdown="ctx",
            used_engram_ids=[],
            used_document_chunk_ids=[],
            source_references=[],
        ),
    )
    service = ChatService(embedding_dim=256)

    monkeypatch.setattr(service, "_prepare_generation", lambda **kwargs: prepared)

    class _StreamingAdapter:
        def stream_generate(self, request):  # noqa: ANN001
            _ = request
            yield "part-1 "
            yield ""
            yield "part-2"

    monkeypatch.setattr("app.chat.service.get_provider_adapter", lambda provider: _StreamingAdapter())
    monkeypatch.setattr(service, "_persist_assistant_reply", lambda **kwargs: assistant_message)
    monkeypatch.setattr(service, "_run_session_lifecycle_maintenance", lambda **kwargs: None)
    monkeypatch.setattr(service, "_build_debug_trace", lambda **kwargs: None)

    events = list(
        service.stream_message_events(
            actor_user_id=actor_id,
            session_id=session.session_id,
            payload=ChatMessageCreateRequest(content_text="Stream this response"),
        )
    )

    kinds = [event_type for event_type, _payload in events]
    assert kinds == ["meta", "chunk", "chunk", "done"]
    done_payload = events[-1][1]
    assert done_payload["assistant_text"] == "part-1 part-2"
    assert done_payload["reply_message_id"] == assistant_message.message_id
    assert done_payload["debug_trace"] is None


@pytest.mark.parametrize(
    ("provider_error", "expected_status", "expected_code"),
    [
        (ProviderRequestError("bad request to provider"), 400, "provider_request_error"),
        (ProviderRateLimitError("too many requests"), 429, "provider_rate_limit"),
        (ProviderAuthError("missing credentials"), 503, "provider_auth_error"),
        (ProviderAPIError("provider downstream error"), 502, "provider_api_error"),
        (ProviderError("provider generic error"), 502, "provider_error"),
    ],
)
def test_raise_provider_error_maps_provider_exceptions(
    provider_error: ProviderError,
    expected_status: int,
    expected_code: str,
) -> None:
    with pytest.raises(ChatProviderExecutionError) as exc_info:
        ChatService._raise_provider_error(provider_error)

    assert exc_info.value.status_code == expected_status
    assert exc_info.value.error_code == expected_code


def test_raise_provider_error_reraises_unknown_exception() -> None:
    with pytest.raises(RuntimeError, match="unexpected"):
        ChatService._raise_provider_error(RuntimeError("unexpected"))

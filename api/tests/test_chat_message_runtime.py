from __future__ import annotations

from datetime import UTC, datetime
from uuid import uuid4

import pytest

from app.chat.context import AssembledChatContext
from app.chat.errors import ChatProviderExecutionError
from app.chat.message_runtime import (
    ChatMessageRuntime,
    ChatMessageRuntimeDependencies,
    PreparedGeneration,
    raise_provider_error,
    resolve_token_usage,
)
from app.chat.session_lifecycle import LifecycleMaintenanceResult
from app.models import (
    ChatMessageRecord,
    ChatProvider,
    ChatSessionRecord,
    VisibilityScope,
)
from app.observability import ChatDebugCollector, ChatDebugTelemetryPublisher
from app.providers.base import ProviderGenerateRequest, ProviderMessage
from app.providers.errors import (
    ProviderAPIError,
    ProviderAuthError,
    ProviderError,
    ProviderRateLimitError,
    ProviderRequestError,
)


def _runtime_for_session(session: ChatSessionRecord) -> ChatMessageRuntime:
    return ChatMessageRuntime(
        dependencies=ChatMessageRuntimeDependencies(
            embedding_dim=256,
            chat_debug_enabled=False,
            chat_debug_include_raw_text=False,
            debug_publisher=ChatDebugTelemetryPublisher(
                console_enabled=False,
                langfuse_enabled=False,
                langfuse_public_key=None,
                langfuse_secret_key=None,
                langfuse_host="https://example.com",
            ),
            get_session=lambda **kwargs: session,
            run_session_lifecycle_maintenance=lambda **kwargs: LifecycleMaintenanceResult(
                snapshot_engram_id=None,
                pruned_engram_ids=[],
                skipped_reason="autosave_disabled",
            ),
        ),
    )


def test_resolve_token_usage_estimates_when_total_missing() -> None:
    resolved, estimated = resolve_token_usage(
        token_usage={"input_tokens": 3},
        input_chars=120,
        output_chars=44,
    )

    assert estimated is True
    assert resolved["total_tokens"] > 0
    assert resolved["input_tokens"] > 0
    assert resolved["output_tokens"] > 0


@pytest.mark.parametrize(
    ("provider_error", "expected_status", "expected_code"),
    [
        (ProviderRequestError("bad request"), 400, "provider_request_error"),
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
        raise_provider_error(provider_error)

    assert exc_info.value.status_code == expected_status
    assert exc_info.value.error_code == expected_code


def test_build_stream_meta_payload_includes_context_references() -> None:
    now = datetime.now(UTC)
    actor_user_id = uuid4()
    session = ChatSessionRecord(
        session_id=uuid4(),
        owner_user_id=actor_user_id,
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
    user_message_id = uuid4()
    prepared = PreparedGeneration(
        session=session,
        user_message=ChatMessageRecord(
            message_id=user_message_id,
            session_id=session.session_id,
            role="user",
            content_text="Q",
            provider=None,
            model_id=None,
            token_usage_json={},
            used_engram_ids=[],
            created_at=now,
        ),
        context=AssembledChatContext(
            context_markdown="ctx",
            used_engram_ids=[uuid4()],
            used_document_chunk_ids=[uuid4()],
            source_references=[],
        ),
        provider_request=ProviderGenerateRequest(
            model_id=session.model_id,
            messages=[ProviderMessage(role="user", content="Q")],
            system_prompt="sys",
        ),
        debug_collector=ChatDebugCollector(),
        prepare_duration_ms=1.0,
        context_duration_ms=1.0,
        history_load_duration_ms=1.0,
    )
    runtime = _runtime_for_session(session)

    payload = runtime.build_stream_meta_payload(prepared=prepared)

    assert payload["session_id"] == session.session_id
    assert payload["message_id"] == user_message_id
    assert payload["used_engram_ids"] == prepared.context.used_engram_ids
    assert payload["used_document_chunk_ids"] == prepared.context.used_document_chunk_ids

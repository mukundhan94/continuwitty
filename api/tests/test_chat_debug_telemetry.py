from __future__ import annotations

import logging
from typing import Any

from app.observability import chat_debug
from app.observability.chat_debug import ChatDebugTelemetryPublisher


def _trace_payload() -> dict[str, Any]:
    return {
        "actor_user_id": "user-1",
        "session_id": "session-1",
        "provider": "openai",
        "model_id": "gpt-4o-mini",
        "request_input_text": "hello",
        "response_output_text": "world",
        "provider_system_prompt_preview": "be concise",
        "provider_messages": [{"role": "user", "content_preview": "hello", "char_count": 5}],
        "llm_calls": [
            {
                "provider": "openai",
                "model_id": "gpt-4o-mini",
                "duration_ms": 10.0,
                "call_type": "generate",
                "token_usage": {"input_tokens": 2, "output_tokens": 1, "total_tokens": 3},
            }
        ],
    }


class _FakeObservation:
    def __init__(self) -> None:
        self.closed = False

    def __enter__(self) -> _FakeObservation:
        return self

    def __exit__(self, exc_type, exc, tb) -> None:  # noqa: ANN001
        _ = exc_type, exc, tb
        self.closed = True


class _FakeLangfuseV3:
    instances: list[_FakeLangfuseV3] = []

    def __init__(self, *, public_key: str, secret_key: str, host: str) -> None:
        self.public_key = public_key
        self.secret_key = secret_key
        self.host = host
        self.observation_calls: list[dict[str, Any]] = []
        self.event_calls: list[dict[str, Any]] = []
        self.update_trace_calls: list[dict[str, Any]] = []
        self.flush_calls = 0
        self.__class__.instances.append(self)

    def start_as_current_observation(self, **kwargs: Any) -> _FakeObservation:
        self.observation_calls.append(kwargs)
        return _FakeObservation()

    def update_current_trace(self, **kwargs: Any) -> None:
        self.update_trace_calls.append(kwargs)

    def create_event(self, **kwargs: Any) -> None:
        self.event_calls.append(kwargs)

    def flush(self) -> None:
        self.flush_calls += 1


class _FailingLangfuseV3(_FakeLangfuseV3):
    def start_as_current_observation(self, **kwargs: Any) -> _FakeObservation:
        _ = kwargs
        raise RuntimeError("observation publish failed")


class _UnsupportedLangfuseClient:
    instances: list[_UnsupportedLangfuseClient] = []

    def __init__(self, *, public_key: str, secret_key: str, host: str) -> None:
        self.public_key = public_key
        self.secret_key = secret_key
        self.host = host
        self.__class__.instances.append(self)


def test_langfuse_missing_credentials_logs_and_skips(caplog) -> None:
    # Force dependency-present branch so this test targets missing-key behavior.
    original_langfuse = chat_debug.Langfuse
    chat_debug.Langfuse = _FakeLangfuseV3
    try:
        with caplog.at_level(logging.WARNING):
            publisher = ChatDebugTelemetryPublisher(
                console_enabled=False,
                langfuse_enabled=True,
                langfuse_public_key=None,
                langfuse_secret_key=None,
                langfuse_host="https://cloud.langfuse.com",
            )

        assert "public/secret keys are missing" in caplog.text

        caplog.clear()
        with caplog.at_level(logging.WARNING):
            publisher.publish_chat_trace(trace_payload=_trace_payload())
        assert "Langfuse publish skipped: client unavailable" in caplog.text
    finally:
        chat_debug.Langfuse = original_langfuse


def test_langfuse_v3_publish_logs_success(monkeypatch, caplog) -> None:
    _FakeLangfuseV3.instances.clear()
    monkeypatch.setattr(chat_debug, "Langfuse", _FakeLangfuseV3)

    with caplog.at_level(logging.INFO):
        publisher = ChatDebugTelemetryPublisher(
            console_enabled=False,
            langfuse_enabled=True,
            langfuse_public_key="pk-test",
            langfuse_secret_key="sk-test",
            langfuse_host="https://cloud.langfuse.com",
        )
        publisher.publish_chat_trace(trace_payload=_trace_payload())

    assert "Langfuse tracing enabled" in caplog.text
    assert "Langfuse trace published" in caplog.text
    assert len(_FakeLangfuseV3.instances) == 1
    client = _FakeLangfuseV3.instances[0]
    assert client.flush_calls == 1
    assert len(client.observation_calls) == 1
    assert client.observation_calls[0]["name"] == "chat.send_message"
    assert client.observation_calls[0]["as_type"] == "generation"
    assert len(client.update_trace_calls) == 1
    assert len(client.event_calls) == 1
    assert client.event_calls[0]["name"] == "provider.generate.1"


def test_langfuse_v3_publish_logs_failures(monkeypatch, caplog) -> None:
    monkeypatch.setattr(chat_debug, "Langfuse", _FailingLangfuseV3)

    publisher = ChatDebugTelemetryPublisher(
        console_enabled=False,
        langfuse_enabled=True,
        langfuse_public_key="pk-test",
        langfuse_secret_key="sk-test",
        langfuse_host="https://cloud.langfuse.com",
    )

    caplog.clear()
    with caplog.at_level(logging.ERROR):
        publisher.publish_chat_trace(trace_payload=_trace_payload())

    assert "Langfuse trace publish failed" in caplog.text


def test_langfuse_incompatible_client_logs_and_skips(monkeypatch, caplog) -> None:
    _UnsupportedLangfuseClient.instances.clear()
    monkeypatch.setattr(chat_debug, "Langfuse", _UnsupportedLangfuseClient)

    with caplog.at_level(logging.WARNING):
        publisher = ChatDebugTelemetryPublisher(
            console_enabled=False,
            langfuse_enabled=True,
            langfuse_public_key="pk-test",
            langfuse_secret_key="sk-test",
            langfuse_host="https://cloud.langfuse.com",
        )

    assert "is incompatible" in caplog.text
    assert len(_UnsupportedLangfuseClient.instances) == 1

    caplog.clear()
    with caplog.at_level(logging.WARNING):
        publisher.publish_chat_trace(trace_payload=_trace_payload())
    assert "Langfuse publish skipped: client unavailable" in caplog.text

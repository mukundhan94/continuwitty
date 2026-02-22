from __future__ import annotations

import json
from datetime import UTC, datetime
from pathlib import Path

from starlette.requests import Request

from app.audit import log_audit_event


def _build_request(*, method: str, path: str, client_host: str | None) -> Request:
    scope = {
        "type": "http",
        "asgi": {"version": "3.0"},
        "http_version": "1.1",
        "method": method,
        "scheme": "http",
        "path": path,
        "raw_path": path.encode("utf-8"),
        "query_string": b"",
        "headers": [],
        "client": (client_host, 8000) if client_host else None,
        "server": ("testserver", 80),
    }
    return Request(scope)


def _read_single_event(path: Path) -> dict[str, object]:
    rows = path.read_text(encoding="utf-8").splitlines()
    assert len(rows) == 1
    return json.loads(rows[0])


def test_log_audit_event_writes_all_expected_fields(tmp_path: Path, monkeypatch) -> None:
    log_path = tmp_path / "logs" / "audit_events.jsonl"
    monkeypatch.setenv("AUDIT_LOG_PATH", str(log_path))

    log_audit_event(
        request=_build_request(method="POST", path="/ui/login", client_host="10.0.0.12"),
        event_type="login",
        success=True,
        username="admin",
        detail="authenticated",
        metadata={"attempt": 1},
    )

    payload = _read_single_event(log_path)
    assert payload["event_type"] == "login"
    assert payload["success"] is True
    assert payload["ip"] == "10.0.0.12"
    assert payload["method"] == "POST"
    assert payload["path"] == "/ui/login"
    assert payload["username"] == "admin"
    assert payload["detail"] == "authenticated"
    assert payload["metadata"] == {"attempt": 1}
    timestamp = datetime.fromisoformat(str(payload["timestamp"]))
    assert timestamp.tzinfo == UTC


def test_log_audit_event_omits_optional_fields_when_missing(tmp_path: Path, monkeypatch) -> None:
    log_path = tmp_path / "logs" / "audit_events.jsonl"
    monkeypatch.setenv("AUDIT_LOG_PATH", str(log_path))

    log_audit_event(
        request=_build_request(method="GET", path="/healthz", client_host=None),
        event_type="healthcheck",
        success=False,
    )

    payload = _read_single_event(log_path)
    assert payload["event_type"] == "healthcheck"
    assert payload["success"] is False
    assert payload["ip"] == "unknown"
    assert payload["method"] == "GET"
    assert payload["path"] == "/healthz"
    assert "username" not in payload
    assert "detail" not in payload
    assert "metadata" not in payload


def test_log_audit_event_truncates_oversized_payload(tmp_path: Path, monkeypatch) -> None:
    log_path = tmp_path / "logs" / "audit_events.jsonl"
    monkeypatch.setenv("AUDIT_LOG_PATH", str(log_path))
    monkeypatch.setenv("AUDIT_LOG_MAX_EVENT_BYTES", "1024")

    log_audit_event(
        request=_build_request(method="POST", path="/api/v1/mcp/stream", client_host="10.0.0.12"),
        event_type="mcp_transport_rate_limited",
        success=False,
        detail="x" * 5000,
        metadata={"payload": "y" * 5000},
    )

    payload = _read_single_event(log_path)
    assert payload["event_type"] == "mcp_transport_rate_limited"
    assert payload["metadata"] == {"truncated": True}
    assert "truncated" in str(payload["detail"])


def test_log_audit_event_writes_stdout_when_enabled(tmp_path: Path, monkeypatch, capsys) -> None:
    log_path = tmp_path / "logs" / "audit_events.jsonl"
    monkeypatch.setenv("AUDIT_LOG_PATH", str(log_path))
    monkeypatch.setenv("AUDIT_LOG_STDOUT_ENABLED", "true")

    log_audit_event(
        request=_build_request(method="GET", path="/healthz", client_host="127.0.0.1"),
        event_type="healthcheck",
        success=True,
    )

    captured = capsys.readouterr()
    assert "healthcheck" in captured.out

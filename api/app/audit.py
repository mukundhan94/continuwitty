from __future__ import annotations

import json
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from fastapi import Request

from .config import get_settings, is_production_env

_MAX_TEXT_FIELD_LENGTH = 1_024
_MAX_METADATA_ITEMS = 50
_MAX_METADATA_DEPTH = 4


def _client_ip(request: Request) -> str:
    if request.client and request.client.host:
        return request.client.host
    return "unknown"


def _sanitize_text(value: str | None, *, max_length: int = _MAX_TEXT_FIELD_LENGTH) -> str | None:
    if value is None:
        return None
    normalized = " ".join(str(value).split())
    if len(normalized) <= max_length:
        return normalized
    return f"{normalized[: max_length - 3]}..."


def _sanitize_metadata(value: Any, *, depth: int = 0) -> Any:
    if depth >= _MAX_METADATA_DEPTH:
        return "<truncated-depth>"
    if isinstance(value, dict):
        return _sanitize_metadata_dict(value, depth=depth)
    if isinstance(value, list):
        return _sanitize_metadata_list(value, depth=depth)
    return _sanitize_metadata_scalar(value)


def _sanitize_metadata_dict(value: dict[Any, Any], *, depth: int) -> dict[str, Any]:
    sanitized: dict[str, Any] = {}
    for index, (key, item) in enumerate(value.items()):
        if index >= _MAX_METADATA_ITEMS:
            sanitized["truncated"] = True
            break
        sanitized[str(key)] = _sanitize_metadata(item, depth=depth + 1)
    return sanitized


def _sanitize_metadata_list(value: list[Any], *, depth: int) -> list[Any]:
    if len(value) > _MAX_METADATA_ITEMS:
        items = value[:_MAX_METADATA_ITEMS]
        return [_sanitize_metadata(item, depth=depth + 1) for item in items] + ["<truncated-list>"]
    return [_sanitize_metadata(item, depth=depth + 1) for item in value]


def _sanitize_metadata_scalar(value: Any) -> Any:
    if isinstance(value, str):
        return _sanitize_text(value)
    if value is None or isinstance(value, bool | int | float):
        return value
    return _sanitize_text(str(value))


def _truncate_serialized_payload(
    *,
    payload: dict[str, Any],
    max_bytes: int,
) -> str:
    serialized = json.dumps(payload, sort_keys=True)
    if len(serialized.encode("utf-8")) <= max_bytes:
        return serialized
    reduced_payload = dict(payload)
    reduced_payload["detail"] = _sanitize_text(
        "audit payload exceeded max size and was truncated",
        max_length=256,
    )
    reduced_payload["metadata"] = {"truncated": True}
    return json.dumps(reduced_payload, sort_keys=True)


def log_audit_event(
    *,
    request: Request,
    event_type: str,
    success: bool,
    username: str | None = None,
    detail: str | None = None,
    metadata: dict[str, Any] | None = None,
) -> None:
    settings = get_settings()
    path = Path(settings.audit_log_path)
    path.parent.mkdir(parents=True, exist_ok=True)

    payload: dict[str, Any] = {
        "timestamp": datetime.now(UTC).isoformat(),
        "event_type": event_type,
        "success": success,
        "ip": _client_ip(request),
        "method": request.method,
        "path": request.url.path,
    }
    if username:
        payload["username"] = _sanitize_text(username)
    if detail:
        payload["detail"] = _sanitize_text(detail)
    if metadata:
        payload["metadata"] = _sanitize_metadata(metadata)

    serialized = _truncate_serialized_payload(
        payload=payload,
        max_bytes=max(1_024, settings.audit_log_max_event_bytes),
    )
    with path.open("a", encoding="utf-8") as handle:
        handle.write(serialized + "\n")

    if settings.audit_log_stdout_enabled or is_production_env(settings):
        print(serialized)

from __future__ import annotations

import json
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from fastapi import Request

from .config import get_settings


def _client_ip(request: Request) -> str:
    if request.client and request.client.host:
        return request.client.host
    return "unknown"


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
        payload["username"] = username
    if detail:
        payload["detail"] = detail
    if metadata:
        payload["metadata"] = metadata

    with path.open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(payload, sort_keys=True) + "\n")

from __future__ import annotations

from dataclasses import dataclass
from typing import Any
from uuid import UUID

from fastapi import HTTPException, Request

from ..config import Settings
from ..mcp_tokens import (
    McpTokenAuthContext,
    build_auth_context,
    get_mcp_token_by_id,
    parse_plaintext_token,
    token_is_active,
    touch_mcp_token_last_used,
    verify_token_secret,
)
from ..user_repository import get_user_auth_record_by_id


@dataclass(frozen=True)
class McpResolvedActor:
    actor: dict[str, Any]
    token_auth: McpTokenAuthContext | None = None


def _unauthorized() -> HTTPException:
    # Keep failures intentionally generic so token format/validity details are not leaked.
    return HTTPException(status_code=401, detail="Authentication required")


def _parse_bearer_token(request: Request) -> str | None:
    authorization = request.headers.get("authorization")
    if not authorization:
        return None
    auth_value = authorization.strip()
    if not auth_value.lower().startswith("bearer "):
        raise _unauthorized()
    token = auth_value[7:].strip()
    if not token:
        raise _unauthorized()
    return token


def resolve_mcp_actor(
    *,
    request: Request,
    settings: Settings,
    require_session_actor: Any,
) -> McpResolvedActor:
    maybe_token = _parse_bearer_token(request)
    if not maybe_token:
        actor = require_session_actor(request)
        return McpResolvedActor(actor=actor, token_auth=None)

    try:
        token_id, token_secret = parse_plaintext_token(maybe_token)
    except ValueError as exc:
        raise _unauthorized() from exc

    record = get_mcp_token_by_id(token_id=token_id)
    if not record or not token_is_active(record):
        raise _unauthorized()

    if not verify_token_secret(
        token_id=record.token_id,
        token_secret=token_secret,
        pepper=settings.mcp_token_pepper,
        expected_hash=record.token_secret_hash,
    ):
        raise _unauthorized()

    user = get_user_auth_record_by_id(UUID(str(record.owner_user_id)))
    if not user or not user.get("is_active"):
        raise _unauthorized()

    touch_mcp_token_last_used(token_id=record.token_id)
    actor = {
        "user_id": str(user["user_id"]),
        "username": user["username"],
        "role": user["role"],
    }
    return McpResolvedActor(actor=actor, token_auth=build_auth_context(record=record))

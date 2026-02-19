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
from ..oauth.service import issuer_url_for_request
from ..user_repository import get_user_auth_record_by_id


@dataclass(frozen=True)
class McpResolvedActor:
    actor: dict[str, Any]
    token_auth: McpTokenAuthContext | None = None


def _unauthorized(*, request: Request, settings: Settings) -> HTTPException:
    # Keep failures intentionally generic so token format/validity details are not leaked.
    headers: dict[str, str] = {}
    if settings.oauth_enabled:
        metadata_url = (
            f"{issuer_url_for_request(request=request, settings=settings)}"
            "/.well-known/oauth-protected-resource"
        )
        headers["WWW-Authenticate"] = (
            f'Bearer realm="engram-mcp", resource_metadata="{metadata_url}", scope="mcp:read"'
        )
    return HTTPException(
        status_code=401,
        detail="Authentication required",
        headers=headers,
    )


def _parse_bearer_token(*, request: Request, settings: Settings) -> str | None:
    authorization = request.headers.get("authorization")
    if not authorization:
        return None
    auth_value = authorization.strip()
    if not auth_value.lower().startswith("bearer "):
        raise _unauthorized(request=request, settings=settings)
    token = auth_value[7:].strip()
    if not token:
        raise _unauthorized(request=request, settings=settings)
    return token


def resolve_mcp_actor(
    *,
    request: Request,
    settings: Settings,
    require_session_actor: Any,
) -> McpResolvedActor:
    maybe_token = _parse_bearer_token(request=request, settings=settings)
    if not maybe_token:
        actor = require_session_actor(request)
        return McpResolvedActor(actor=actor, token_auth=None)

    try:
        token_id, token_secret = parse_plaintext_token(maybe_token)
    except ValueError as exc:
        raise _unauthorized(request=request, settings=settings) from exc

    record = get_mcp_token_by_id(token_id=token_id)
    if not record or not token_is_active(record):
        raise _unauthorized(request=request, settings=settings)

    if not verify_token_secret(
        token_id=record.token_id,
        token_secret=token_secret,
        pepper=settings.mcp_token_pepper,
        expected_hash=record.token_secret_hash,
    ):
        raise _unauthorized(request=request, settings=settings)

    user = get_user_auth_record_by_id(UUID(str(record.owner_user_id)))
    if not user or not user.get("is_active"):
        raise _unauthorized(request=request, settings=settings)

    touch_mcp_token_last_used(token_id=record.token_id)
    actor = {
        "user_id": str(user["user_id"]),
        "username": user["username"],
        "role": user["role"],
    }
    return McpResolvedActor(actor=actor, token_auth=build_auth_context(record=record))

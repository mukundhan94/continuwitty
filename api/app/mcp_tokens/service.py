from __future__ import annotations

import hashlib
import hmac
import secrets
from datetime import UTC, datetime, timedelta
from typing import Any
from uuid import UUID, uuid4

from app.models import (
    McpTokenCreateRequest,
    McpTokenCreateResponse,
    McpTokenScope,
    McpTokenSummary,
)

from .models import TOKEN_PREFIX, McpTokenAuthContext, McpTokenRecord
from .repository import create_mcp_token, list_mcp_tokens, revoke_mcp_token


def normalize_string_list(values: list[str] | None) -> list[str]:
    """Normalize user-provided list fields (tools/project IDs).

    We trim whitespace, drop blanks, and keep insertion order while deduplicating.
    This keeps token policy matching deterministic across UI/API inputs.
    """
    if not values:
        return []
    result: list[str] = []
    seen: set[str] = set()
    for raw in values:
        value = (raw or "").strip()
        if not value:
            continue
        if value in seen:
            continue
        seen.add(value)
        result.append(value)
    return result


def token_hash(*, token_id: UUID, token_secret: str, pepper: str) -> str:
    payload = f"{token_id.hex}:{token_secret}".encode()
    return hmac.new(pepper.encode(), payload, hashlib.sha256).hexdigest()


def token_secret_hint(token_secret: str) -> str:
    if len(token_secret) <= 10:
        return token_secret
    return f"{token_secret[:6]}...{token_secret[-4:]}"


def build_plaintext_token(*, token_id: UUID, token_secret: str) -> str:
    return f"{TOKEN_PREFIX}_{token_id.hex}_{token_secret}"


def parse_plaintext_token(raw: str) -> tuple[UUID, str]:
    text = (raw or "").strip()
    prefix = f"{TOKEN_PREFIX}_"
    if not text.startswith(prefix):
        raise ValueError("Token prefix mismatch")

    payload = text[len(prefix) :]
    parts = payload.split("_", 1)
    if len(parts) != 2:
        raise ValueError("Token payload format mismatch")

    token_id_hex, token_secret = parts
    if len(token_id_hex) != 32:
        raise ValueError("Token identifier length mismatch")

    try:
        token_id = UUID(hex=token_id_hex)
    except ValueError as exc:
        raise ValueError("Token identifier invalid") from exc

    if not token_secret:
        raise ValueError("Token secret missing")
    return token_id, token_secret


def issue_new_token(
    *,
    token_id: UUID,
    expires_in_days: int,
    pepper: str,
) -> tuple[str, str, str, datetime]:
    """Return plaintext token + persisted hash/hint metadata."""
    expires_at = datetime.now(UTC) + timedelta(days=expires_in_days)
    return issue_token_with_expiry(
        token_id=token_id,
        expires_at=expires_at,
        pepper=pepper,
    )


def issue_token_with_expiry(
    *,
    token_id: UUID,
    expires_at: datetime,
    pepper: str,
) -> tuple[str, str, str, datetime]:
    """Return plaintext token + persisted hash/hint metadata for a fixed expiry."""
    token_secret = secrets.token_urlsafe(32)
    hashed = token_hash(token_id=token_id, token_secret=token_secret, pepper=pepper)
    hint = token_secret_hint(token_secret)
    plaintext = build_plaintext_token(token_id=token_id, token_secret=token_secret)
    return plaintext, hashed, hint, expires_at


def verify_token_secret(
    *, token_id: UUID, token_secret: str, pepper: str, expected_hash: str
) -> bool:
    candidate_hash = token_hash(token_id=token_id, token_secret=token_secret, pepper=pepper)
    return hmac.compare_digest(candidate_hash, expected_hash)


def token_is_active(record: McpTokenRecord, *, now: datetime | None = None) -> bool:
    current = now or datetime.now(UTC)
    if record.revoked_at is not None:
        return False
    return record.expires_at > current


def build_auth_context(*, record: McpTokenRecord) -> McpTokenAuthContext:
    return McpTokenAuthContext(
        token_id=record.token_id,
        owner_user_id=record.owner_user_id,
        scope=record.scope,
        allowed_tools=set(normalize_string_list(record.allowed_tools)),
        allowed_project_ids=set(normalize_string_list(record.allowed_project_ids)),
        expires_at=record.expires_at,
    )


def token_summary_dict(*, record: McpTokenRecord, now: datetime | None = None) -> dict[str, Any]:
    return {
        "token_id": record.token_id,
        "name": record.name,
        "scope": record.scope,
        "allowed_tools": normalize_string_list(record.allowed_tools),
        "allowed_project_ids": normalize_string_list(record.allowed_project_ids),
        "token_secret_hint": record.token_secret_hint,
        "expires_at": record.expires_at,
        "last_used_at": record.last_used_at,
        "revoked_at": record.revoked_at,
        "created_at": record.created_at,
        "is_active": token_is_active(record, now=now),
    }


def create_token_for_owner(
    *,
    owner_user_id: UUID,
    payload: McpTokenCreateRequest,
    pepper: str,
) -> McpTokenCreateResponse:
    normalized_tools = normalize_string_list(payload.allowed_tools)
    normalized_projects = normalize_string_list(payload.allowed_project_ids)

    token_id = uuid4()
    plaintext_token, token_hash_value, token_hint, expires_at = issue_new_token(
        token_id=token_id,
        expires_in_days=payload.expires_in_days,
        pepper=pepper,
    )
    created = create_mcp_token(
        token_id=token_id,
        owner_user_id=owner_user_id,
        name=payload.name.strip(),
        scope=payload.scope.value,
        allowed_tools=normalized_tools,
        allowed_project_ids=normalized_projects,
        token_secret_hash=token_hash_value,
        token_secret_hint=token_hint,
        expires_at=expires_at,
    )
    return McpTokenCreateResponse(
        token_id=created.token_id,
        name=created.name,
        scope=McpTokenScope(created.scope),
        allowed_tools=created.allowed_tools,
        allowed_project_ids=created.allowed_project_ids,
        token_secret_hint=created.token_secret_hint,
        token=plaintext_token,
        expires_at=created.expires_at,
        created_at=created.created_at,
    )


def list_token_summaries(
    *,
    owner_user_id: UUID,
    limit: int = 200,
    offset: int = 0,
    now: datetime | None = None,
) -> list[McpTokenSummary]:
    records = list_mcp_tokens(owner_user_id=owner_user_id, limit=limit, offset=offset)
    current = now or datetime.now(UTC)
    return [McpTokenSummary(**token_summary_dict(record=item, now=current)) for item in records]


def revoke_token_for_owner(
    *,
    token_id: UUID,
    owner_user_id: UUID,
    now: datetime | None = None,
) -> McpTokenSummary | None:
    revoked = revoke_mcp_token(token_id=token_id, owner_user_id=owner_user_id)
    if not revoked:
        return None
    current = now or datetime.now(UTC)
    return McpTokenSummary(**token_summary_dict(record=revoked, now=current))

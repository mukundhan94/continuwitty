from __future__ import annotations

from datetime import UTC, datetime, timedelta
from uuid import uuid4

from app.mcp_tokens.models import TOKEN_PREFIX, McpTokenRecord
from app.mcp_tokens.service import (
    issue_new_token,
    parse_plaintext_token,
    token_is_active,
    verify_token_secret,
)


def _record(*, expires_at: datetime, revoked_at: datetime | None = None) -> McpTokenRecord:
    return McpTokenRecord(
        token_id=uuid4(),
        owner_user_id=uuid4(),
        name="token",
        scope="read",
        allowed_tools=[],
        allowed_project_ids=[],
        token_secret_hash="hash",
        token_secret_hint="hint",
        expires_at=expires_at,
        last_used_at=None,
        revoked_at=revoked_at,
        created_at=datetime.now(UTC),
    )


def test_issue_token_parse_and_verify_round_trip() -> None:
    token_id = uuid4()
    plaintext, token_hash_value, _hint, _expires_at = issue_new_token(
        token_id=token_id,
        expires_in_days=90,
        pepper="pepper-value",
    )
    assert plaintext.startswith(f"{TOKEN_PREFIX}_")

    parsed_id, parsed_secret = parse_plaintext_token(plaintext)
    assert parsed_id == token_id
    assert verify_token_secret(
        token_id=parsed_id,
        token_secret=parsed_secret,
        pepper="pepper-value",
        expected_hash=token_hash_value,
    )


def test_token_is_active_handles_expiry_and_revocation() -> None:
    active = _record(expires_at=datetime.now(UTC) + timedelta(days=1))
    assert token_is_active(active)

    expired = _record(expires_at=datetime.now(UTC) - timedelta(seconds=1))
    assert not token_is_active(expired)

    revoked = _record(
        expires_at=datetime.now(UTC) + timedelta(days=1),
        revoked_at=datetime.now(UTC) - timedelta(minutes=1),
    )
    assert not token_is_active(revoked)

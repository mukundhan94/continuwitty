from __future__ import annotations

from datetime import UTC, datetime, timedelta
from uuid import uuid4

from app.mcp_tokens.models import TOKEN_PREFIX, McpTokenRecord
from app.mcp_tokens.service import (
    create_token_for_owner,
    issue_new_token,
    list_token_summaries,
    parse_plaintext_token,
    revoke_token_for_owner,
    token_is_active,
    verify_token_secret,
)
from app.models import McpTokenCreateRequest, McpTokenScope


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


def test_create_token_for_owner_normalizes_lists(monkeypatch) -> None:
    owner_user_id = uuid4()
    token_id = uuid4()
    expires_at = datetime.now(UTC) + timedelta(days=30)
    captured: dict = {}

    def _fake_issue_new_token(*, token_id, expires_in_days, pepper):  # noqa: ANN001
        captured["issued"] = {
            "token_id": token_id,
            "expires_in_days": expires_in_days,
            "pepper": pepper,
        }
        return "engram_mcp_plain", "hashed-value", "secret...hint", expires_at

    def _fake_create_mcp_token(**kwargs):  # noqa: ANN001
        captured["create_args"] = kwargs
        return McpTokenRecord(
            token_id=kwargs["token_id"],
            owner_user_id=kwargs["owner_user_id"],
            name=kwargs["name"],
            scope=kwargs["scope"],
            allowed_tools=kwargs["allowed_tools"],
            allowed_project_ids=kwargs["allowed_project_ids"],
            token_secret_hash=kwargs["token_secret_hash"],
            token_secret_hint=kwargs["token_secret_hint"],
            expires_at=kwargs["expires_at"],
            last_used_at=None,
            revoked_at=None,
            created_at=datetime.now(UTC),
        )

    monkeypatch.setattr("app.mcp_tokens.service.issue_new_token", _fake_issue_new_token)
    monkeypatch.setattr("app.mcp_tokens.service.create_mcp_token", _fake_create_mcp_token)
    monkeypatch.setattr("app.mcp_tokens.service.uuid4", lambda: token_id)

    created = create_token_for_owner(
        owner_user_id=owner_user_id,
        payload=McpTokenCreateRequest(
            name="  CLI Token  ",
            scope=McpTokenScope.read,
            allowed_tools=[" engram.query ", "", "engram.query", "chat.list_sessions"],
            allowed_project_ids=["engram-vault", " engram-vault ", "phase31"],
            expires_in_days=30,
        ),
        pepper="pepper-1",
    )

    assert created.token == "engram_mcp_plain"
    assert created.scope == McpTokenScope.read
    assert created.allowed_tools == ["engram.query", "chat.list_sessions"]
    assert created.allowed_project_ids == ["engram-vault", "phase31"]
    assert captured["issued"]["pepper"] == "pepper-1"
    assert captured["create_args"]["name"] == "CLI Token"
    assert captured["create_args"]["allowed_tools"] == ["engram.query", "chat.list_sessions"]
    assert captured["create_args"]["allowed_project_ids"] == ["engram-vault", "phase31"]


def test_list_token_summaries_and_revoke_token_for_owner(monkeypatch) -> None:
    owner_user_id = uuid4()
    active_record = _record(expires_at=datetime.now(UTC) + timedelta(days=2))
    revoked_record = _record(
        expires_at=datetime.now(UTC) + timedelta(days=2),
        revoked_at=datetime.now(UTC) - timedelta(minutes=5),
    )

    monkeypatch.setattr(
        "app.mcp_tokens.service.list_mcp_tokens",
        lambda **kwargs: [active_record, revoked_record],  # noqa: ARG005
    )
    monkeypatch.setattr(
        "app.mcp_tokens.service.revoke_mcp_token",
        lambda **kwargs: revoked_record,  # noqa: ARG005
    )

    listed = list_token_summaries(owner_user_id=owner_user_id)
    assert [item.is_active for item in listed] == [True, False]
    assert listed[0].scope == "read"

    revoked = revoke_token_for_owner(token_id=uuid4(), owner_user_id=owner_user_id)
    assert revoked is not None
    assert revoked.is_active is False

    monkeypatch.setattr(
        "app.mcp_tokens.service.revoke_mcp_token",
        lambda **kwargs: None,  # noqa: ARG005
    )
    missing = revoke_token_for_owner(token_id=uuid4(), owner_user_id=owner_user_id)
    assert missing is None

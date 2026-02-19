from __future__ import annotations

import base64
import hashlib

from app.oauth.service import (
    normalize_scope,
    token_scope_from_oauth_scope,
    validate_pkce,
)


def _s256_challenge(verifier: str) -> str:
    digest = hashlib.sha256(verifier.encode()).digest()
    return base64.urlsafe_b64encode(digest).decode().rstrip("=")


def test_normalize_scope_defaults_to_mcp_read() -> None:
    assert normalize_scope("") == "mcp:read"
    assert normalize_scope(None) == "mcp:read"


def test_normalize_scope_aliases_and_deduplicates() -> None:
    assert normalize_scope("read write write custom") == "mcp:read mcp:write custom"


def test_token_scope_maps_write_scope() -> None:
    assert token_scope_from_oauth_scope("mcp:write") == "write"
    assert token_scope_from_oauth_scope("mcp:read") == "read"


def test_validate_pkce_s256_and_plain() -> None:
    verifier = "pkce-verifier-example"
    assert validate_pkce(
        code_verifier=verifier,
        code_challenge=_s256_challenge(verifier),
        code_challenge_method="S256",
    )
    assert validate_pkce(
        code_verifier=verifier,
        code_challenge=verifier,
        code_challenge_method="plain",
    )
    assert not validate_pkce(
        code_verifier="wrong",
        code_challenge=_s256_challenge(verifier),
        code_challenge_method="S256",
    )

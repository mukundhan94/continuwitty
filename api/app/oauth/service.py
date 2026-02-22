from __future__ import annotations

import base64
import hashlib
import hmac
import secrets
from collections.abc import Iterable
from datetime import UTC, datetime
from uuid import uuid4

from fastapi import Request

from ..config import Settings
from .models import OAuthAuthorizationCodeRecord, OAuthClientRecord

_DEFAULT_SCOPE = "mcp:read"
_SCOPE_ALIAS_MAP = {
    "read": "mcp:read",
    "write": "mcp:write",
}


def issuer_url_for_request(*, request: Request, settings: Settings) -> str:
    configured = (settings.oauth_issuer_url or "").strip()
    if configured:
        return configured.rstrip("/")
    base_url = str(request.base_url).strip()
    return base_url.rstrip("/")


def normalize_scope(scope: str | None) -> str:
    tokens = [(item or "").strip() for item in (scope or "").split(" ")]
    normalized: list[str] = []
    seen: set[str] = set()
    for token in tokens:
        if not token:
            continue
        mapped = _SCOPE_ALIAS_MAP.get(token, token)
        if mapped in seen:
            continue
        seen.add(mapped)
        normalized.append(mapped)

    if not normalized:
        return _DEFAULT_SCOPE
    return " ".join(normalized)


def token_scope_from_oauth_scope(scope: str | None) -> str:
    normalized = normalize_scope(scope)
    tokens = set(normalized.split())
    if "mcp:write" in tokens:
        return "write"
    return "read"


def oauth_secret_hash(*, identifier: str, secret: str, pepper: str) -> str:
    payload = f"{identifier}:{secret}".encode()
    return hmac.new(pepper.encode(), payload, hashlib.sha256).hexdigest()


def verify_oauth_secret(
    *,
    identifier: str,
    secret: str,
    pepper: str,
    expected_hash: str,
) -> bool:
    candidate = oauth_secret_hash(identifier=identifier, secret=secret, pepper=pepper)
    return hmac.compare_digest(candidate, expected_hash)


def build_oauth_client_id() -> str:
    return f"engram_client_{uuid4().hex}"


def issue_oauth_client_secret() -> str:
    return secrets.token_urlsafe(32)


def generate_authorization_code() -> str:
    return secrets.token_urlsafe(48)


def authorization_code_hash(*, code: str, pepper: str) -> str:
    return oauth_secret_hash(identifier="authorization_code", secret=code, pepper=pepper)


def authorization_code_is_active(
    record: OAuthAuthorizationCodeRecord,
    *,
    now: datetime | None = None,
) -> bool:
    current = now or datetime.now(UTC)
    if record.consumed_at is not None:
        return False
    return record.expires_at > current


def redirect_uri_allowed(*, redirect_uri: str, allowed_uris: Iterable[str]) -> bool:
    candidate = redirect_uri.strip()
    if not candidate:
        return False
    allowed = {item.strip() for item in allowed_uris if item and item.strip()}
    return candidate in allowed


def validate_pkce(
    *,
    code_verifier: str,
    code_challenge: str,
    code_challenge_method: str,
) -> bool:
    method = (code_challenge_method or "S256").strip()
    verifier = (code_verifier or "").strip()
    challenge = (code_challenge or "").strip()
    if not verifier or not challenge:
        return False
    if method == "S256":
        digest = hashlib.sha256(verifier.encode()).digest()
        encoded = base64.urlsafe_b64encode(digest).decode().rstrip("=")
        return hmac.compare_digest(encoded, challenge)
    return False


def client_supports_authorization_code(record: OAuthClientRecord) -> bool:
    return "authorization_code" in set(record.grant_types)

from __future__ import annotations

from typing import Any

from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field

from ..config import Settings
from .common import (
    SUPPORTED_GRANT_TYPES,
    SUPPORTED_RESPONSE_TYPES,
    SUPPORTED_TOKEN_ENDPOINT_AUTH_METHODS,
    oauth_error_response,
    require_oauth_enabled,
)
from .repository import create_oauth_client
from .service import build_oauth_client_id, issue_oauth_client_secret, oauth_secret_hash


class OAuthClientRegistrationRequest(BaseModel):
    redirect_uris: list[str] = Field(default_factory=list)
    client_name: str = Field(default="Engram MCP Client", min_length=1, max_length=200)
    grant_types: list[str] = Field(default_factory=lambda: ["authorization_code"])
    response_types: list[str] = Field(default_factory=lambda: ["code"])
    token_endpoint_auth_method: str = "none"


def _dedup_string_list(values: list[str]) -> list[str]:
    normalized: list[str] = []
    seen: set[str] = set()
    for item in values:
        value = (item or "").strip()
        if not value:
            continue
        if value in seen:
            continue
        seen.add(value)
        normalized.append(value)
    return normalized


def _sanitize_redirect_uris(values: list[str]) -> list[str]:
    return _dedup_string_list(values)


def _normalize_metadata_list(values: list[str], fallback: set[str]) -> list[str]:
    normalized = _dedup_string_list(values)
    if not normalized:
        return sorted(fallback)
    return normalized


def _handle_oauth_register(
    *,
    settings: Settings,
    payload: OAuthClientRegistrationRequest,
) -> JSONResponse:
    require_oauth_enabled(settings)
    redirect_uris = _sanitize_redirect_uris(payload.redirect_uris)
    if not redirect_uris:
        return oauth_error_response(
            error="invalid_redirect_uri",
            description="At least one redirect_uri is required.",
        )

    token_endpoint_auth_method = payload.token_endpoint_auth_method.strip() or "none"
    if token_endpoint_auth_method not in SUPPORTED_TOKEN_ENDPOINT_AUTH_METHODS:
        return oauth_error_response(
            error="invalid_client_metadata",
            description="Unsupported token_endpoint_auth_method.",
        )

    grant_types = _normalize_metadata_list(payload.grant_types, SUPPORTED_GRANT_TYPES)
    if "authorization_code" not in set(grant_types):
        return oauth_error_response(
            error="invalid_client_metadata",
            description="authorization_code grant type is required.",
        )

    response_types = _normalize_metadata_list(payload.response_types, SUPPORTED_RESPONSE_TYPES)
    if "code" not in set(response_types):
        return oauth_error_response(
            error="invalid_client_metadata",
            description="code response type is required.",
        )

    client_id = build_oauth_client_id()
    client_secret: str | None = None
    client_secret_hash: str | None = None
    if token_endpoint_auth_method == "client_secret_post":
        client_secret = issue_oauth_client_secret()
        client_secret_hash = oauth_secret_hash(
            identifier=client_id,
            secret=client_secret,
            pepper=settings.oauth_client_secret_pepper,
        )

    created = create_oauth_client(
        client_id=client_id,
        client_name=payload.client_name.strip(),
        redirect_uris=redirect_uris,
        grant_types=grant_types,
        response_types=response_types,
        token_endpoint_auth_method=token_endpoint_auth_method,
        client_secret_hash=client_secret_hash,
        metadata_json=payload.model_dump(mode="python"),
    )
    issued_at = int(created.created_at.timestamp())
    response_payload: dict[str, Any] = {
        "client_id": created.client_id,
        "client_name": created.client_name,
        "redirect_uris": created.redirect_uris,
        "grant_types": created.grant_types,
        "response_types": created.response_types,
        "token_endpoint_auth_method": created.token_endpoint_auth_method,
        "client_id_issued_at": issued_at,
    }
    if client_secret is not None:
        response_payload["client_secret"] = client_secret
        response_payload["client_secret_expires_at"] = 0
    return JSONResponse(status_code=201, content=response_payload)

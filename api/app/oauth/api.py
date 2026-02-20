from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from datetime import UTC, datetime, timedelta
from typing import Any
from urllib.parse import parse_qsl, quote, urlencode, urlsplit, urlunsplit
from uuid import UUID, uuid4

from fastapi import APIRouter, Form, HTTPException, Query, Request
from fastapi.responses import JSONResponse, RedirectResponse
from pydantic import BaseModel, Field

from ..config import Settings
from ..mcp_tokens import create_mcp_token, issue_token_with_expiry
from .models import OAuthAuthorizationCodeRecord, OAuthClientRecord
from .repository import (
    consume_oauth_authorization_code,
    create_oauth_authorization_code,
    create_oauth_client,
    get_oauth_authorization_code_by_hash,
    get_oauth_client,
)
from .service import (
    authorization_code_hash,
    authorization_code_is_active,
    build_oauth_client_id,
    client_supports_authorization_code,
    generate_authorization_code,
    issue_oauth_client_secret,
    issuer_url_for_request,
    normalize_scope,
    oauth_secret_hash,
    redirect_uri_allowed,
    token_scope_from_oauth_scope,
    validate_pkce,
    verify_oauth_secret,
)

_SUPPORTED_TOKEN_ENDPOINT_AUTH_METHODS = {"none", "client_secret_post"}
_SUPPORTED_CODE_CHALLENGE_METHODS = {"S256", "plain"}
_SUPPORTED_GRANT_TYPES = {"authorization_code"}
_SUPPORTED_RESPONSE_TYPES = {"code"}
_MCP_SCOPES = ["mcp:read", "mcp:write"]


class OAuthClientRegistrationRequest(BaseModel):
    redirect_uris: list[str] = Field(default_factory=list)
    client_name: str = Field(default="Engram MCP Client", min_length=1, max_length=200)
    grant_types: list[str] = Field(default_factory=lambda: ["authorization_code"])
    response_types: list[str] = Field(default_factory=lambda: ["code"])
    token_endpoint_auth_method: str = "none"


@dataclass(frozen=True)
class OAuthAuthorizeRequest:
    response_type: str
    client_id: str
    redirect_uri: str
    state: str | None
    scope: str | None
    code_challenge: str | None
    code_challenge_method: str
    resource: str | None


@dataclass(frozen=True)
class OAuthTokenRequest:
    grant_type: str
    code: str
    redirect_uri: str
    client_id: str
    code_verifier: str
    client_secret: str | None


def _require_oauth_enabled(settings: Settings) -> None:
    if settings.oauth_enabled:
        return
    raise HTTPException(status_code=404, detail="OAuth endpoints disabled")


def _oauth_error_response(
    *,
    error: str,
    description: str,
    status_code: int = 400,
    headers: dict[str, str] | None = None,
) -> JSONResponse:
    return JSONResponse(
        status_code=status_code,
        content={"error": error, "error_description": description},
        headers=headers,
    )


def _merge_query_params(url: str, params: dict[str, str]) -> str:
    parts = urlsplit(url)
    existing = dict(parse_qsl(parts.query, keep_blank_values=True))
    existing.update({key: value for key, value in params.items() if value != ""})
    return urlunsplit(
        (
            parts.scheme,
            parts.netloc,
            parts.path,
            urlencode(existing),
            parts.fragment,
        )
    )


def _oauth_redirect_error(
    *,
    redirect_uri: str,
    error: str,
    state: str | None,
    description: str,
) -> RedirectResponse:
    params = {"error": error, "error_description": description}
    if state is not None:
        params["state"] = state
    return RedirectResponse(url=_merge_query_params(redirect_uri, params), status_code=303)


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


def _oauth_protected_resource_metadata(
    *, issuer: str, resource_path: str | None = None
) -> dict[str, Any]:
    normalized_path = (resource_path or "").lstrip("/")
    resource = f"{issuer}/{normalized_path}" if normalized_path else f"{issuer}/api/v1/mcp/stream"
    return {
        "resource": resource,
        "authorization_servers": [issuer],
        "bearer_methods_supported": ["header"],
        "scopes_supported": _MCP_SCOPES,
    }


def _oauth_server_metadata(*, issuer: str) -> dict[str, Any]:
    return {
        "issuer": issuer,
        "authorization_endpoint": f"{issuer}/oauth/authorize",
        "token_endpoint": f"{issuer}/oauth/token",
        "registration_endpoint": f"{issuer}/oauth/register",
        "response_types_supported": sorted(_SUPPORTED_RESPONSE_TYPES),
        "grant_types_supported": sorted(_SUPPORTED_GRANT_TYPES),
        "token_endpoint_auth_methods_supported": sorted(_SUPPORTED_TOKEN_ENDPOINT_AUTH_METHODS),
        "scopes_supported": _MCP_SCOPES,
        "code_challenge_methods_supported": sorted(_SUPPORTED_CODE_CHALLENGE_METHODS),
    }


def _validate_oauth_client_and_redirect(
    *,
    client_id: str,
    redirect_uri: str,
) -> tuple[OAuthClientRecord | None, JSONResponse | None]:
    client = get_oauth_client(client_id=client_id)
    if not client:
        return (
            None,
            _oauth_error_response(
                error="invalid_client",
                description="Unknown client_id.",
                status_code=401,
            ),
        )

    if not redirect_uri_allowed(redirect_uri=redirect_uri, allowed_uris=client.redirect_uris):
        return (
            None,
            _oauth_error_response(
                error="invalid_request",
                description="redirect_uri is not registered for this client.",
                status_code=400,
            ),
        )

    return client, None


def _oauth_login_redirect(request: Request) -> RedirectResponse:
    next_path = request.url.path
    if request.url.query:
        next_path = f"{next_path}?{request.url.query}"
    return RedirectResponse(
        url=f"/login?next={quote(next_path, safe='')}",
        status_code=303,
    )


def _handle_oauth_register(
    *,
    settings: Settings,
    payload: OAuthClientRegistrationRequest,
) -> JSONResponse:
    _require_oauth_enabled(settings)
    redirect_uris = _sanitize_redirect_uris(payload.redirect_uris)
    if not redirect_uris:
        return _oauth_error_response(
            error="invalid_redirect_uri",
            description="At least one redirect_uri is required.",
        )

    token_endpoint_auth_method = payload.token_endpoint_auth_method.strip() or "none"
    if token_endpoint_auth_method not in _SUPPORTED_TOKEN_ENDPOINT_AUTH_METHODS:
        return _oauth_error_response(
            error="invalid_client_metadata",
            description="Unsupported token_endpoint_auth_method.",
        )

    grant_types = _normalize_metadata_list(payload.grant_types, _SUPPORTED_GRANT_TYPES)
    if "authorization_code" not in set(grant_types):
        return _oauth_error_response(
            error="invalid_client_metadata",
            description="authorization_code grant type is required.",
        )

    response_types = _normalize_metadata_list(payload.response_types, _SUPPORTED_RESPONSE_TYPES)
    if "code" not in set(response_types):
        return _oauth_error_response(
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


def _validate_authorize_request(
    *,
    client: OAuthClientRecord,
    payload: OAuthAuthorizeRequest,
) -> tuple[str | None, RedirectResponse | None]:
    if payload.response_type != "code":
        return None, _oauth_redirect_error(
            redirect_uri=payload.redirect_uri,
            error="unsupported_response_type",
            state=payload.state,
            description="Only response_type=code is supported.",
        )

    if not client_supports_authorization_code(client):
        return None, _oauth_redirect_error(
            redirect_uri=payload.redirect_uri,
            error="unauthorized_client",
            state=payload.state,
            description="Client is not allowed to use authorization_code.",
        )

    method = (payload.code_challenge_method or "S256").strip()
    if method not in _SUPPORTED_CODE_CHALLENGE_METHODS:
        return None, _oauth_redirect_error(
            redirect_uri=payload.redirect_uri,
            error="invalid_request",
            state=payload.state,
            description="Unsupported code_challenge_method.",
        )

    if client.token_endpoint_auth_method == "none" and not (payload.code_challenge or "").strip():
        return None, _oauth_redirect_error(
            redirect_uri=payload.redirect_uri,
            error="invalid_request",
            state=payload.state,
            description="code_challenge is required for public clients.",
        )
    return method, None


def _create_authorization_code(
    *,
    settings: Settings,
    client: OAuthClientRecord,
    payload: OAuthAuthorizeRequest,
    user_id: UUID,
) -> str:
    now = datetime.now(UTC)
    code = generate_authorization_code()
    hashed_code = authorization_code_hash(code=code, pepper=settings.mcp_token_pepper)
    create_oauth_authorization_code(
        code_id=uuid4(),
        code_hash=hashed_code,
        client_id=client.client_id,
        user_id=user_id,
        redirect_uri=payload.redirect_uri,
        code_challenge=(payload.code_challenge or "").strip(),
        code_challenge_method=(payload.code_challenge_method or "S256").strip(),
        requested_scope=normalize_scope(payload.scope),
        resource=(payload.resource or "").strip() or None,
        expires_at=now + timedelta(seconds=max(30, settings.oauth_authorization_code_ttl_seconds)),
    )
    return code


def _oauth_authorize_success_redirect(
    *,
    request: Request,
    settings: Settings,
    payload: OAuthAuthorizeRequest,
    code: str,
) -> RedirectResponse:
    redirect_params = {"code": code}
    if payload.state is not None:
        redirect_params["state"] = payload.state
    issuer = issuer_url_for_request(request=request, settings=settings)
    redirect_params["iss"] = issuer
    return RedirectResponse(
        url=_merge_query_params(payload.redirect_uri, redirect_params),
        status_code=303,
    )


def _handle_oauth_authorize(
    *,
    settings: Settings,
    resolve_session_user: Callable[[Request], dict[str, Any] | None],
    request: Request,
    payload: OAuthAuthorizeRequest,
) -> JSONResponse | RedirectResponse:
    _require_oauth_enabled(settings)
    client, client_validation_error = _validate_oauth_client_and_redirect(
        client_id=payload.client_id,
        redirect_uri=payload.redirect_uri,
    )
    if client_validation_error is not None:
        return client_validation_error
    if client is None:  # pragma: no cover
        return _oauth_error_response(
            error="invalid_client",
            description="Unknown client_id.",
            status_code=401,
        )

    method, authorize_error = _validate_authorize_request(client=client, payload=payload)
    if authorize_error is not None:
        return authorize_error

    user = resolve_session_user(request)
    if not user:
        return _oauth_login_redirect(request)

    code = _create_authorization_code(
        settings=settings,
        client=client,
        payload=OAuthAuthorizeRequest(
            response_type=payload.response_type,
            client_id=payload.client_id,
            redirect_uri=payload.redirect_uri,
            state=payload.state,
            scope=payload.scope,
            code_challenge=payload.code_challenge,
            code_challenge_method=method or "S256",
            resource=payload.resource,
        ),
        user_id=UUID(str(user["user_id"])),
    )
    return _oauth_authorize_success_redirect(
        request=request,
        settings=settings,
        payload=payload,
        code=code,
    )


def _validate_token_grant_type(*, payload: OAuthTokenRequest) -> JSONResponse | None:
    if payload.grant_type == "authorization_code":
        return None
    return _oauth_error_response(
        error="unsupported_grant_type",
        description="Only authorization_code is supported.",
        status_code=400,
    )


def _resolve_oauth_client_for_token(*, payload: OAuthTokenRequest) -> tuple[OAuthClientRecord | None, JSONResponse | None]:
    client = get_oauth_client(client_id=payload.client_id)
    if client is not None:
        return client, None
    return (
        None,
        _oauth_error_response(
            error="invalid_client",
            description="Unknown client_id.",
            status_code=401,
        ),
    )


def _validate_oauth_token_client_secret(
    *,
    settings: Settings,
    client: OAuthClientRecord,
    payload: OAuthTokenRequest,
) -> JSONResponse | None:
    if client.token_endpoint_auth_method != "client_secret_post":
        return None
    if not payload.client_secret or not client.client_secret_hash:
        return _oauth_error_response(
            error="invalid_client",
            description="client_secret is required.",
            status_code=401,
            headers={"WWW-Authenticate": 'Bearer realm="engram-oauth"'},
        )
    if verify_oauth_secret(
        identifier=client.client_id,
        secret=payload.client_secret,
        pepper=settings.oauth_client_secret_pepper,
        expected_hash=client.client_secret_hash,
    ):
        return None
    return _oauth_error_response(
        error="invalid_client",
        description="client_secret is invalid.",
        status_code=401,
        headers={"WWW-Authenticate": 'Bearer realm="engram-oauth"'},
    )


def _resolve_authorization_code_for_token(
    *,
    settings: Settings,
    payload: OAuthTokenRequest,
) -> tuple[OAuthAuthorizationCodeRecord | None, JSONResponse | None]:
    hashed_code = authorization_code_hash(code=payload.code, pepper=settings.mcp_token_pepper)
    code_record = get_oauth_authorization_code_by_hash(code_hash=hashed_code)
    if code_record is not None:
        return code_record, None
    return (
        None,
        _oauth_error_response(
            error="invalid_grant",
            description="Authorization code is invalid.",
            status_code=400,
        ),
    )


def _validate_authorization_code_exchange(
    *,
    payload: OAuthTokenRequest,
    client: OAuthClientRecord,
    code_record: OAuthAuthorizationCodeRecord,
) -> JSONResponse | None:
    if code_record.client_id != client.client_id:
        return _oauth_error_response(
            error="invalid_grant",
            description="Authorization code does not belong to this client.",
            status_code=400,
        )
    if code_record.redirect_uri != payload.redirect_uri:
        return _oauth_error_response(
            error="invalid_grant",
            description="redirect_uri mismatch.",
            status_code=400,
        )
    if not authorization_code_is_active(code_record):
        return _oauth_error_response(
            error="invalid_grant",
            description="Authorization code expired or already consumed.",
            status_code=400,
        )
    if validate_pkce(
        code_verifier=payload.code_verifier,
        code_challenge=code_record.code_challenge,
        code_challenge_method=code_record.code_challenge_method,
    ):
        return None
    return _oauth_error_response(
        error="invalid_grant",
        description="PKCE validation failed.",
        status_code=400,
    )


def _consume_authorization_code(
    *,
    code_record: OAuthAuthorizationCodeRecord,
) -> JSONResponse | None:
    consumed = consume_oauth_authorization_code(
        code_id=code_record.code_id,
        consumed_at=datetime.now(UTC),
    )
    if consumed is not None:
        return None
    return _oauth_error_response(
        error="invalid_grant",
        description="Authorization code already consumed.",
        status_code=400,
    )


def _issue_token_from_authorization_code(
    *,
    settings: Settings,
    client: OAuthClientRecord,
    code_record: OAuthAuthorizationCodeRecord,
) -> JSONResponse:
    now = datetime.now(UTC)
    expires_at = now + timedelta(seconds=max(60, settings.oauth_access_token_ttl_seconds))
    token_id = uuid4()
    access_token, token_hash_value, token_hint, resolved_expires_at = issue_token_with_expiry(
        token_id=token_id,
        expires_at=expires_at,
        pepper=settings.mcp_token_pepper,
    )
    create_mcp_token(
        token_id=token_id,
        owner_user_id=code_record.user_id,
        name=f"oauth:{client.client_id}",
        scope=token_scope_from_oauth_scope(code_record.requested_scope),
        allowed_tools=[],
        allowed_project_ids=[],
        token_secret_hash=token_hash_value,
        token_secret_hint=token_hint,
        expires_at=resolved_expires_at,
    )
    expires_in = int((resolved_expires_at - now).total_seconds())
    return JSONResponse(
        status_code=200,
        content={
            "access_token": access_token,
            "token_type": "Bearer",
            "expires_in": max(1, expires_in),
            "scope": normalize_scope(code_record.requested_scope),
        },
    )


def _handle_oauth_token(
    *,
    settings: Settings,
    payload: OAuthTokenRequest,
) -> JSONResponse:
    _require_oauth_enabled(settings)
    grant_error = _validate_token_grant_type(payload=payload)
    if grant_error is not None:
        return grant_error

    client, client_error = _resolve_oauth_client_for_token(payload=payload)
    if client_error is not None:
        return client_error
    if client is None:  # pragma: no cover
        return _oauth_error_response(
            error="invalid_client",
            description="Unknown client_id.",
            status_code=401,
        )

    client_secret_error = _validate_oauth_token_client_secret(
        settings=settings,
        client=client,
        payload=payload,
    )
    if client_secret_error is not None:
        return client_secret_error

    code_record, code_record_error = _resolve_authorization_code_for_token(
        settings=settings,
        payload=payload,
    )
    if code_record_error is not None:
        return code_record_error
    if code_record is None:  # pragma: no cover
        return _oauth_error_response(
            error="invalid_grant",
            description="Authorization code is invalid.",
            status_code=400,
        )

    exchange_error = _validate_authorization_code_exchange(
        payload=payload,
        client=client,
        code_record=code_record,
    )
    if exchange_error is not None:
        return exchange_error

    consume_error = _consume_authorization_code(code_record=code_record)
    if consume_error is not None:
        return consume_error

    return _issue_token_from_authorization_code(
        settings=settings,
        client=client,
        code_record=code_record,
    )


def create_oauth_router(
    *,
    settings: Settings,
    resolve_session_user: Callable[[Request], dict[str, Any] | None],
) -> APIRouter:
    router = APIRouter(tags=["oauth"])

    @router.get("/.well-known/oauth-authorization-server")
    def oauth_authorization_server_metadata(request: Request) -> dict[str, Any]:
        _require_oauth_enabled(settings)
        issuer = issuer_url_for_request(request=request, settings=settings)
        return _oauth_server_metadata(issuer=issuer)

    @router.get("/.well-known/openid-configuration")
    def openid_configuration(request: Request) -> dict[str, Any]:
        _require_oauth_enabled(settings)
        issuer = issuer_url_for_request(request=request, settings=settings)
        metadata = _oauth_server_metadata(issuer=issuer)
        metadata["claims_supported"] = []
        metadata["subject_types_supported"] = ["public"]
        return metadata

    @router.get("/.well-known/oauth-protected-resource")
    def oauth_protected_resource_metadata(request: Request) -> dict[str, Any]:
        _require_oauth_enabled(settings)
        issuer = issuer_url_for_request(request=request, settings=settings)
        return _oauth_protected_resource_metadata(issuer=issuer)

    @router.get("/.well-known/oauth-protected-resource/{resource_path:path}")
    def oauth_protected_resource_metadata_scoped(
        request: Request, resource_path: str
    ) -> dict[str, Any]:
        _require_oauth_enabled(settings)
        issuer = issuer_url_for_request(request=request, settings=settings)
        return _oauth_protected_resource_metadata(issuer=issuer, resource_path=resource_path)

    @router.post("/oauth/register")
    def oauth_register_client(payload: OAuthClientRegistrationRequest) -> JSONResponse:
        return _handle_oauth_register(settings=settings, payload=payload)

    @router.get("/oauth/authorize")
    def oauth_authorize(
        request: Request,
        response_type: str = Query(...),
        client_id: str = Query(...),
        redirect_uri: str = Query(...),
        state: str | None = Query(default=None),
        scope: str | None = Query(default=None),
        code_challenge: str | None = Query(default=None),
        code_challenge_method: str = Query(default="S256"),
        resource: str | None = Query(default=None),
    ):
        return _handle_oauth_authorize(
            settings=settings,
            resolve_session_user=resolve_session_user,
            request=request,
            payload=OAuthAuthorizeRequest(
                response_type=response_type,
                client_id=client_id,
                redirect_uri=redirect_uri,
                state=state,
                scope=scope,
                code_challenge=code_challenge,
                code_challenge_method=code_challenge_method,
                resource=resource,
            ),
        )

    @router.post("/oauth/token")
    def oauth_token(
        grant_type: str = Form(...),
        code: str = Form(...),
        redirect_uri: str = Form(...),
        client_id: str = Form(...),
        code_verifier: str = Form(...),
        client_secret: str | None = Form(default=None),
    ) -> JSONResponse:
        return _handle_oauth_token(
            settings=settings,
            payload=OAuthTokenRequest(
                grant_type=grant_type,
                code=code,
                redirect_uri=redirect_uri,
                client_id=client_id,
                code_verifier=code_verifier,
                client_secret=client_secret,
            ),
        )

    return router

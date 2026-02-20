from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass, replace
from datetime import UTC, datetime, timedelta
from typing import Any
from uuid import UUID, uuid4

from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse, RedirectResponse

from ..config import Settings
from ..mcp_tokens import create_mcp_token, issue_token_with_expiry
from .common import (
    SUPPORTED_CODE_CHALLENGE_METHODS as _SUPPORTED_CODE_CHALLENGE_METHODS,
)
from .common import (
    merge_query_params as _merge_query_params,
)
from .common import (
    oauth_error_response as _oauth_error_response,
)
from .common import (
    oauth_login_redirect as _oauth_login_redirect,
)
from .common import (
    oauth_redirect_error as _oauth_redirect_error,
)
from .common import (
    require_oauth_enabled as _require_oauth_enabled,
)
from .models import OAuthAuthorizationCodeRecord, OAuthClientRecord
from .repository import (
    consume_oauth_authorization_code,
    create_oauth_authorization_code,
    get_oauth_authorization_code_by_hash,
    get_oauth_client,
)
from .service import (
    authorization_code_hash,
    authorization_code_is_active,
    client_supports_authorization_code,
    generate_authorization_code,
    issuer_url_for_request,
    normalize_scope,
    redirect_uri_allowed,
    token_scope_from_oauth_scope,
    validate_pkce,
    verify_oauth_secret,
)


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


@dataclass(frozen=True)
class _OAuthTokenExchangeContext:
    client: OAuthClientRecord
    code_record: OAuthAuthorizationCodeRecord


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

    if _public_client_missing_pkce(client=client, payload=payload):
        return None, _oauth_redirect_error(
            redirect_uri=payload.redirect_uri,
            error="invalid_request",
            state=payload.state,
            description="code_challenge is required for public clients.",
        )
    return method, None


def _public_client_missing_pkce(
    *,
    client: OAuthClientRecord,
    payload: OAuthAuthorizeRequest,
) -> bool:
    if client.token_endpoint_auth_method != "none":
        return False
    return not (payload.code_challenge or "").strip()


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
        payload=replace(payload, code_challenge_method=method or "S256"),
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

    token_context, token_context_error = _resolve_oauth_token_exchange_context(
        settings=settings,
        payload=payload,
    )
    if token_context_error is not None:
        return token_context_error
    if token_context is None:  # pragma: no cover
        return _oauth_error_response(
            error="invalid_grant",
            description="Unable to resolve token exchange context.",
            status_code=400,
        )

    consume_error = _consume_authorization_code(code_record=token_context.code_record)
    if consume_error is not None:
        return consume_error

    return _issue_token_from_authorization_code(
        settings=settings,
        client=token_context.client,
        code_record=token_context.code_record,
    )


def _resolve_oauth_token_exchange_context(
    *,
    settings: Settings,
    payload: OAuthTokenRequest,
) -> tuple[_OAuthTokenExchangeContext | None, JSONResponse | None]:
    client, client_error = _resolve_oauth_client_for_token(payload=payload)
    if client_error is not None:
        return None, client_error
    if client is None:  # pragma: no cover
        return (
            None,
            _oauth_error_response(
                error="invalid_client",
                description="Unknown client_id.",
                status_code=401,
            ),
        )

    client_secret_error = _validate_oauth_token_client_secret(
        settings=settings,
        client=client,
        payload=payload,
    )
    if client_secret_error is not None:
        return None, client_secret_error

    code_record, code_record_error = _resolve_authorization_code_for_token(
        settings=settings,
        payload=payload,
    )
    if code_record_error is not None:
        return None, code_record_error
    if code_record is None:  # pragma: no cover
        return (
            None,
            _oauth_error_response(
                error="invalid_grant",
                description="Authorization code is invalid.",
                status_code=400,
            ),
        )

    exchange_error = _validate_authorization_code_exchange(
        payload=payload,
        client=client,
        code_record=code_record,
    )
    if exchange_error is not None:
        return None, exchange_error
    return _OAuthTokenExchangeContext(client=client, code_record=code_record), None


def create_oauth_router(
    *,
    settings: Settings,
    resolve_session_user: Callable[[Request], dict[str, Any] | None],
) -> APIRouter:
    from .router import create_oauth_router as create_oauth_router_impl

    return create_oauth_router_impl(
        settings=settings,
        resolve_session_user=resolve_session_user,
    )

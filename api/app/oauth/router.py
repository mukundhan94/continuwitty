from __future__ import annotations

from collections.abc import Callable
from typing import Any

from fastapi import APIRouter, Form, Query, Request
from fastapi.responses import JSONResponse

from ..config import Settings
from .api import (
    OAuthAuthorizeRequest,
    OAuthTokenRequest,
    _handle_oauth_authorize,
    _handle_oauth_token,
)
from .common import (
    oauth_protected_resource_metadata as _oauth_protected_resource_metadata,
)
from .common import (
    oauth_server_metadata as _oauth_server_metadata,
)
from .common import (
    require_oauth_enabled as _require_oauth_enabled,
)
from .registration import OAuthClientRegistrationRequest, _handle_oauth_register
from .service import issuer_url_for_request


def _register_oauth_well_known_routes(
    *,
    router: APIRouter,
    settings: Settings,
) -> None:
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


def _register_oauth_client_registration_route(
    *,
    router: APIRouter,
    settings: Settings,
    resolve_session_user: Callable[[Request], dict[str, Any] | None],
) -> None:
    @router.post("/oauth/register")
    def oauth_register_client(request: Request, payload: OAuthClientRegistrationRequest) -> JSONResponse:
        return _handle_oauth_register(
            settings=settings,
            payload=payload,
            session_user=resolve_session_user(request),
        )


def _register_oauth_authorization_routes(
    *,
    router: APIRouter,
    settings: Settings,
    resolve_session_user: Callable[[Request], dict[str, Any] | None],
) -> None:
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


def create_oauth_router(
    *,
    settings: Settings,
    resolve_session_user: Callable[[Request], dict[str, Any] | None],
) -> APIRouter:
    router = APIRouter(tags=["oauth"])
    _register_oauth_well_known_routes(router=router, settings=settings)
    _register_oauth_client_registration_route(
        router=router,
        settings=settings,
        resolve_session_user=resolve_session_user,
    )
    _register_oauth_authorization_routes(
        router=router,
        settings=settings,
        resolve_session_user=resolve_session_user,
    )
    return router

from __future__ import annotations

from typing import Any
from urllib.parse import parse_qsl, quote, urlencode, urlsplit, urlunsplit

from fastapi import HTTPException, Request
from fastapi.responses import JSONResponse, RedirectResponse

from ..config import Settings

SUPPORTED_TOKEN_ENDPOINT_AUTH_METHODS = {"none", "client_secret_post"}
SUPPORTED_CODE_CHALLENGE_METHODS = {"S256"}
SUPPORTED_GRANT_TYPES = {"authorization_code"}
SUPPORTED_RESPONSE_TYPES = {"code"}
MCP_SCOPES = ["mcp:read", "mcp:write"]


def require_oauth_enabled(settings: Settings) -> None:
    if settings.oauth_enabled:
        return
    raise HTTPException(status_code=404, detail="OAuth endpoints disabled")


def oauth_error_response(
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


def merge_query_params(url: str, params: dict[str, str]) -> str:
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


def oauth_redirect_error(
    *,
    redirect_uri: str,
    error: str,
    state: str | None,
    description: str,
) -> RedirectResponse:
    params = {"error": error, "error_description": description}
    if state is not None:
        params["state"] = state
    return RedirectResponse(url=merge_query_params(redirect_uri, params), status_code=303)


def oauth_login_redirect(request: Request) -> RedirectResponse:
    next_path = request.url.path
    if request.url.query:
        next_path = f"{next_path}?{request.url.query}"
    return RedirectResponse(
        url=f"/login?next={quote(next_path, safe='')}",
        status_code=303,
    )


def oauth_protected_resource_metadata(
    *, issuer: str, resource_path: str | None = None
) -> dict[str, Any]:
    normalized_path = (resource_path or "").lstrip("/")
    resource = f"{issuer}/{normalized_path}" if normalized_path else f"{issuer}/api/v1/mcp/stream"
    return {
        "resource": resource,
        "authorization_servers": [issuer],
        "bearer_methods_supported": ["header"],
        "scopes_supported": MCP_SCOPES,
    }


def oauth_server_metadata(*, issuer: str) -> dict[str, Any]:
    return {
        "issuer": issuer,
        "authorization_endpoint": f"{issuer}/oauth/authorize",
        "token_endpoint": f"{issuer}/oauth/token",
        "registration_endpoint": f"{issuer}/oauth/register",
        "response_types_supported": sorted(SUPPORTED_RESPONSE_TYPES),
        "grant_types_supported": sorted(SUPPORTED_GRANT_TYPES),
        "token_endpoint_auth_methods_supported": sorted(SUPPORTED_TOKEN_ENDPOINT_AUTH_METHODS),
        "scopes_supported": MCP_SCOPES,
        "code_challenge_methods_supported": sorted(SUPPORTED_CODE_CHALLENGE_METHODS),
    }

from __future__ import annotations

import json
import logging
from collections.abc import Callable
from typing import Any

from fastapi import APIRouter, Request
from fastapi.encoders import jsonable_encoder
from fastapi.responses import JSONResponse, Response, StreamingResponse

from app.models import McpJsonRpcRequest

from .auth import McpResolvedActor
from .service import McpService

logger = logging.getLogger(__name__)


def _sse_event(event_name: str, payload: dict[str, Any]) -> str:
    data = json.dumps(jsonable_encoder(payload))
    return f"event: {event_name}\ndata: {data}\n\n"


def _accept_quality(*, accept_header: str, media_type: str) -> float:
    """Return effective quality for a media type from an Accept header.

    We keep this parser intentionally small: exact media type matches are
    preferred over wildcard matches, and invalid `q` values are ignored.
    """
    best_exact = -1.0
    best_wildcard = -1.0
    for item in accept_header.split(","):
        part = item.strip().lower()
        if not part:
            continue

        token, *params = [segment.strip() for segment in part.split(";")]
        quality = 1.0
        for param in params:
            if not param.startswith("q="):
                continue
            try:
                parsed = float(param[2:])
            except ValueError:
                continue
            if 0.0 <= parsed <= 1.0:
                quality = parsed

        if token == media_type:
            best_exact = max(best_exact, quality)
        elif token == "*/*":
            best_wildcard = max(best_wildcard, quality)

    if best_exact >= 0.0:
        return best_exact
    if best_wildcard >= 0.0:
        return best_wildcard
    return -1.0


def _prefers_sse(accept_header: str) -> bool:
    """Select SSE only when the client clearly prefers it.

    VS Code and other MCP clients often send both `text/event-stream` and
    `application/json` for streamable HTTP. In tie cases we prefer JSON for
    request/response calls like `initialize`, which avoids client stalls.
    """
    normalized = (accept_header or "").lower().strip()
    sse_quality = _accept_quality(accept_header=normalized, media_type="text/event-stream")
    json_quality = _accept_quality(accept_header=normalized, media_type="application/json")
    if sse_quality <= 0.0:
        return False
    if json_quality <= 0.0:
        return True
    return sse_quality > json_quality


def create_mcp_router(
    *,
    mcp_service: McpService,
    resolve_mcp_actor: Callable[[Request], McpResolvedActor],
    enforce_transport_rate_limit: Callable[[Request], None] | None = None,
) -> APIRouter:
    router = APIRouter(prefix="/api/v1/mcp", tags=["mcp"])
    logger.info(
        "[engram-mcp] notification support enabled on POST /api/v1/mcp/stream "
        "(accepts JSON-RPC notifications such as notifications/initialized)"
    )

    @router.get("/stream")
    def mcp_stream_probe() -> JSONResponse:
        """Probe endpoint for clients that preflight MCP transport capabilities.

        Some MCP clients attempt GET/HEAD checks before sending POST-based JSON-RPC
        stream requests. Returning a stable 200 response avoids noisy 405 logs while
        keeping POST as the only RPC transport.
        """
        return JSONResponse(
            status_code=200,
            content={
                "name": "engram-mcp-stream",
                "transport": "sse-jsonrpc",
                "endpoint": "/api/v1/mcp/stream",
                "request_method": "POST",
                "response_type": "text/event-stream",
            },
        )

    @router.head("/stream")
    def mcp_stream_probe_head() -> Response:
        return Response(status_code=200)

    @router.post("/stream")
    def mcp_stream(request: Request, payload: McpJsonRpcRequest):
        if enforce_transport_rate_limit is not None:
            enforce_transport_rate_limit(request)
        actor_context = resolve_mcp_actor(request)
        if payload.id is None:
            # JSON-RPC notifications intentionally do not carry `id` and must
            # not produce response frames (for example `notifications/initialized`).
            mcp_service.handle_notification(request=payload)
            return Response(status_code=202)

        events = mcp_service.stream_call(
            actor=actor_context.actor,
            request=payload,
            token_auth=actor_context.token_auth,
        )
        wants_sse = _prefers_sse(request.headers.get("accept") or "")

        if wants_sse:

            def _stream():
                for frame in events:
                    yield _sse_event("jsonrpc", frame)

            return StreamingResponse(_stream(), media_type="text/event-stream")

        # Streamable HTTP clients may expect a plain JSON-RPC response for
        # request/response interactions (like initialize/tools/list). In this
        # mode we return the terminal frame while still using the same service.
        frames = list(events)
        for frame in reversed(frames):
            if frame.get("id") == payload.id and (
                frame.get("result") is not None or frame.get("error") is not None
            ):
                return JSONResponse(status_code=200, content=jsonable_encoder(frame))

        fallback = {
            "jsonrpc": "2.0",
            "id": payload.id,
            "error": {"code": -32603, "message": "Internal error"},
        }
        return JSONResponse(status_code=200, content=jsonable_encoder(fallback))

    return router

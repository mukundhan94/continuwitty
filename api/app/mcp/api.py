from __future__ import annotations

import json
from collections.abc import Callable
from typing import Any

from fastapi import APIRouter, Request
from fastapi.encoders import jsonable_encoder
from fastapi.responses import StreamingResponse

from app.models import McpJsonRpcRequest

from .service import McpService


def _sse_event(event_name: str, payload: dict[str, Any]) -> str:
    data = json.dumps(jsonable_encoder(payload))
    return f"event: {event_name}\ndata: {data}\n\n"


def create_mcp_router(
    *,
    mcp_service: McpService,
    require_api_actor: Callable[[Request], dict[str, Any]],
) -> APIRouter:
    router = APIRouter(prefix="/api/v1/mcp", tags=["mcp"])

    @router.post("/stream")
    def mcp_stream(request: Request, payload: McpJsonRpcRequest) -> StreamingResponse:
        actor = require_api_actor(request)
        events = mcp_service.stream_call(actor=actor, request=payload)

        def _stream():
            for frame in events:
                yield _sse_event("jsonrpc", frame)

        return StreamingResponse(_stream(), media_type="text/event-stream")

    return router

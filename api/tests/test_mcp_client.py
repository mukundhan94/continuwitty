from __future__ import annotations

import json
from urllib.parse import parse_qs

import httpx
import pytest

from app.mcp.client import (
    McpClientProtocolError,
    McpClientRpcError,
    McpJsonRpcErrorFrame,
    McpJsonRpcEventFrame,
    McpJsonRpcResultFrame,
    McpSseClient,
    parse_mcp_jsonrpc_frame,
)


def test_parse_mcp_jsonrpc_frame_variants() -> None:
    result_frame = parse_mcp_jsonrpc_frame({"jsonrpc": "2.0", "id": "abc", "result": {"ok": True}})
    assert isinstance(result_frame, McpJsonRpcResultFrame)
    assert result_frame.result["ok"] is True

    event_frame = parse_mcp_jsonrpc_frame(
        {
            "jsonrpc": "2.0",
            "method": "mcp.event",
            "params": {
                "id": "abc",
                "tool": "chat.send_message",
                "event": "chunk",
                "data": {"text": "hi"},
            },
        }
    )
    assert isinstance(event_frame, McpJsonRpcEventFrame)
    assert event_frame.params.tool == "chat.send_message"

    error_frame = parse_mcp_jsonrpc_frame(
        {"jsonrpc": "2.0", "id": "abc", "error": {"code": -32010, "message": "boom"}}
    )
    assert isinstance(error_frame, McpJsonRpcErrorFrame)
    assert error_frame.error.code == -32010


def test_mcp_sse_client_login_and_call_round_trip() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        if request.method == "GET" and request.url.path == "/login":
            return httpx.Response(
                200,
                text='<form><input name="csrf_token" value="csrf-token-1" /></form>',
            )

        if request.method == "POST" and request.url.path == "/login":
            form = parse_qs(request.content.decode("utf-8"))
            assert form["username"] == ["admin"]
            assert form["password"] == ["admin123"]
            assert form["csrf_token"] == ["csrf-token-1"]
            return httpx.Response(303, headers={"location": "/ui"})

        if request.method == "POST" and request.url.path == "/api/v1/mcp/stream":
            payload = json.loads(request.content.decode("utf-8"))
            assert payload["method"] == "user.get_profile"
            assert payload["id"] == "tool-call-1"
            sse = 'event: jsonrpc\ndata: {"jsonrpc":"2.0","id":"tool-call-1","result":{"profile":{"username":"admin"}}}\n\n'
            return httpx.Response(200, headers={"content-type": "text/event-stream"}, text=sse)

        return httpx.Response(404)

    transport = httpx.MockTransport(handler)
    http_client = httpx.Client(transport=transport)
    mcp_client = McpSseClient(base_url="http://testserver", http_client=http_client)
    mcp_client.login_with_password(username="admin", password="admin123")
    result = mcp_client.call_tool(method="user.get_profile", params={}, request_id="tool-call-1")

    assert result.final_result_frame is not None
    assert result.require_result()["profile"]["username"] == "admin"


def test_mcp_sse_client_raises_rpc_error_for_error_frame() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        if request.url.path == "/api/v1/mcp/stream":
            sse = 'event: jsonrpc\ndata: {"jsonrpc":"2.0","id":"err","error":{"code":-32004,"message":"Engram not found"}}\n\n'
            return httpx.Response(200, headers={"content-type": "text/event-stream"}, text=sse)
        return httpx.Response(404)

    mcp_client = McpSseClient(
        base_url="http://testserver",
        http_client=httpx.Client(transport=httpx.MockTransport(handler)),
    )
    result = mcp_client.call_tool(
        method="engram.rehydrate", params={"engram_id": "x"}, request_id="err"
    )

    with pytest.raises(McpClientRpcError) as exc:
        _ = result.require_result()
    assert exc.value.code == -32004


def test_mcp_sse_client_rejects_malformed_frames() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        if request.url.path == "/api/v1/mcp/stream":
            sse = "event: jsonrpc\ndata: not-json\n\n"
            return httpx.Response(200, headers={"content-type": "text/event-stream"}, text=sse)
        return httpx.Response(404)

    mcp_client = McpSseClient(
        base_url="http://testserver",
        http_client=httpx.Client(transport=httpx.MockTransport(handler)),
    )

    with pytest.raises(McpClientProtocolError):
        _ = mcp_client.call_tool(method="user.get_profile", request_id="bad")

from __future__ import annotations

import json
import re

import pytest

from app.config import get_settings
from app.models import ChatProvider
from app.providers.base import ProviderGenerateResult


def _extract_csrf_token(html: str) -> str:
    match = re.search(r'name="csrf_token" value="([^"]+)"', html)
    assert match is not None
    return match.group(1)


def _login(client) -> None:  # noqa: ANN001
    settings = get_settings()
    login_page = client.get("/login")
    csrf_token = _extract_csrf_token(login_page.text)
    response = client.post(
        "/login",
        data={
            "username": settings.ui_demo_username,
            "password": settings.ui_demo_password,
            "csrf_token": csrf_token,
        },
        follow_redirects=False,
    )
    assert response.status_code == 303
    assert response.headers["location"] == "/ui"


def _install_fake_provider(monkeypatch) -> None:
    class _FakeAdapter:
        def generate(self, request):  # noqa: ANN001
            last_user = ""
            for message in reversed(request.messages):
                if message.role == "user":
                    last_user = message.content
                    break
            return ProviderGenerateResult(
                provider=ChatProvider.openai,
                model_id=request.model_id,
                text=f"assistant:{last_user}",
                token_usage={"input_tokens": 5, "output_tokens": 3, "total_tokens": 8},
            )

        def stream_generate(self, request):  # noqa: ANN001
            _ = request
            yield "assistant:"
            yield "streamed"

    monkeypatch.setattr("app.chat.service.get_provider_adapter", lambda provider: _FakeAdapter())


def _mcp_frames(client, method: str, params: dict, request_id: str = "1") -> list[dict]:  # noqa: ANN001
    response = client.post(
        "/api/v1/mcp/stream",
        json={
            "jsonrpc": "2.0",
            "id": request_id,
            "method": method,
            "params": params,
        },
    )
    assert response.status_code == 200
    assert "text/event-stream" in response.headers.get("content-type", "")
    frames: list[dict] = []
    for line in response.text.splitlines():
        if line.startswith("data: "):
            frames.append(json.loads(line[6:]))
    assert frames
    return frames


def _final_result_frame(frames: list[dict]) -> dict:
    result_frames = [item for item in frames if "result" in item]
    assert result_frames, f"No result frame found in: {frames}"
    return result_frames[-1]


@pytest.mark.integration
def test_mcp_stream_requires_authentication(client, clean_db) -> None:
    response = client.post(
        "/api/v1/mcp/stream",
        json={"jsonrpc": "2.0", "id": "1", "method": "user.get_profile", "params": {}},
    )
    assert response.status_code == 401


@pytest.mark.integration
def test_mcp_returns_method_not_found_error(client, clean_db) -> None:
    _login(client)
    frames = _mcp_frames(client, method="unknown.tool", params={}, request_id="missing")
    error_frame = [item for item in frames if "error" in item][0]
    assert error_frame["id"] == "missing"
    assert error_frame["error"]["code"] == -32601
    assert error_frame["error"]["data"]["method"] == "unknown.tool"


@pytest.mark.integration
def test_mcp_returns_invalid_params_error(client, clean_db) -> None:
    _login(client)
    frames = _mcp_frames(client, method="chat.get_session", params={}, request_id="invalid-params")
    error_frame = [item for item in frames if "error" in item][0]
    assert error_frame["id"] == "invalid-params"
    assert error_frame["error"]["code"] == -32602
    assert error_frame["error"]["data"]["missing"] == "session_id"


@pytest.mark.integration
def test_mcp_initialize_and_tools_list_contract(client, clean_db) -> None:
    _login(client)

    init_frames = _mcp_frames(client, method="initialize", params={}, request_id="init")
    init_result = _final_result_frame(init_frames)["result"]
    assert init_result["protocolVersion"] == "2024-11-05"
    assert init_result["serverInfo"]["name"] == "engram-vault-mcp"
    assert init_result["capabilities"]["tools"]["listChanged"] is False

    tools_frames = _mcp_frames(client, method="tools/list", params={}, request_id="tools-list")
    tools = _final_result_frame(tools_frames)["result"]["tools"]
    tool_names = {item["name"] for item in tools}
    assert "chat.create_session" in tool_names
    assert "chat.send_message" in tool_names
    assert "engram.query" in tool_names
    assert "user.get_profile" in tool_names

    send_message_tool = next(item for item in tools if item["name"] == "chat.send_message")
    assert send_message_tool["inputSchema"]["type"] == "object"
    assert "session_id" in send_message_tool["inputSchema"]["required"]
    assert "content_text" in send_message_tool["inputSchema"]["required"]


@pytest.mark.integration
def test_mcp_tools_call_requires_name_param(client, clean_db) -> None:
    _login(client)
    frames = _mcp_frames(client, method="tools/call", params={}, request_id="tools-call-missing")
    error_frame = [item for item in frames if "error" in item][0]
    assert error_frame["id"] == "tools-call-missing"
    assert error_frame["error"]["code"] == -32602
    assert error_frame["error"]["data"]["missing"] == "name"


@pytest.mark.integration
def test_mcp_chat_send_message_stream_flow(client, clean_db, monkeypatch) -> None:
    _login(client)
    _install_fake_provider(monkeypatch)

    create_frames = _mcp_frames(
        client,
        method="chat.create_session",
        params={
            "project_id": "project-mcp-chat",
            "title": "MCP Session",
            "provider": "openai",
            "model_id": "gpt-4o-mini",
            "system_prompt": "You are concise.",
            "visibility_scope": "private",
            "autosave_enabled": False,
        },
        request_id="create-session",
    )
    session_id = _final_result_frame(create_frames)["result"]["session"]["session_id"]

    message_frames = _mcp_frames(
        client,
        method="chat.send_message",
        params={"session_id": session_id, "content_text": "hello over mcp", "stream": True},
        request_id="send-message",
    )
    event_frames = [item for item in message_frames if item.get("method") == "mcp.event"]
    assert any(item["params"]["event"] == "meta" for item in event_frames)
    assert any(item["params"]["event"] == "chunk" for item in event_frames)
    assert any(item["params"]["event"] == "done" for item in event_frames)
    final = _final_result_frame(message_frames)
    assert final["id"] == "send-message"
    assert final["result"]["message"]["assistant_text"] == "assistant:streamed"


@pytest.mark.integration
def test_mcp_tools_call_stream_flow(client, clean_db, monkeypatch) -> None:
    _login(client)
    _install_fake_provider(monkeypatch)

    create_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat.create_session",
            "arguments": {
                "project_id": "project-mcp-tools-call",
                "title": "MCP tools/call session",
                "provider": "openai",
                "model_id": "gpt-4o-mini",
                "system_prompt": "Be concise.",
                "visibility_scope": "private",
                "autosave_enabled": False,
            },
        },
        request_id="tools-call-create",
    )
    structured = _final_result_frame(create_frames)["result"]["structuredContent"]
    session_id = structured["session"]["session_id"]

    send_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat.send_message",
            "arguments": {
                "session_id": session_id,
                "content_text": "hello from tools call",
                "stream": True,
            },
        },
        request_id="tools-call-send",
    )

    event_frames = [item for item in send_frames if item.get("method") == "mcp.event"]
    assert event_frames
    assert all(item["params"]["id"] == "tools-call-send" for item in event_frames)
    assert all(item["params"]["tool"] == "chat.send_message" for item in event_frames)
    assert any(item["params"]["event"] == "chunk" for item in event_frames)
    assert any(item["params"]["event"] == "done" for item in event_frames)

    final = _final_result_frame(send_frames)
    assert final["result"]["tool_name"] == "chat.send_message"
    assert final["result"]["isError"] is False
    assert final["result"]["structuredContent"]["message"]["assistant_text"] == "assistant:streamed"


@pytest.mark.integration
def test_mcp_engram_and_user_tools(client, clean_db, monkeypatch) -> None:
    _login(client)
    _install_fake_provider(monkeypatch)

    create_session_frames = _mcp_frames(
        client,
        method="chat.create_session",
        params={
            "project_id": "project-mcp-tools",
            "title": "MCP Tool Session",
            "provider": "openai",
            "model_id": "gpt-4o-mini",
            "visibility_scope": "private",
            "autosave_enabled": False,
        },
        request_id="create-session-tools",
    )
    session_id = _final_result_frame(create_session_frames)["result"]["session"]["session_id"]

    create_engram_frames = _mcp_frames(
        client,
        method="engram.create",
        params={
            "project_id": "project-mcp-tools",
            "thread_id": "tool-thread",
            "title": "MCP Engram",
            "abstract": "Created via MCP",
            "detailed_summary_markdown": "Longer details.",
            "tags": ["mcp"],
            "keywords": ["tooling"],
        },
        request_id="create-engram",
    )
    engram_id = _final_result_frame(create_engram_frames)["result"]["engram"]["engram_id"]

    pin_frames = _mcp_frames(
        client,
        method="engram.pin_to_session",
        params={"session_id": session_id, "engram_id": engram_id},
        request_id="pin-engram",
    )
    assert _final_result_frame(pin_frames)["result"]["pinned"]["engram_id"] == engram_id

    query_frames = _mcp_frames(
        client,
        method="engram.query",
        params={"query": "mcp", "project_id": "project-mcp-tools", "top_k": 5},
        request_id="query-engram",
    )
    results = _final_result_frame(query_frames)["result"]["results"]
    assert any(item["engram_id"] == engram_id for item in results)

    rehydrate_frames = _mcp_frames(
        client,
        method="engram.rehydrate",
        params={"engram_id": engram_id},
        request_id="rehydrate-engram",
    )
    assert _final_result_frame(rehydrate_frames)["result"]["bundle"]["engram_id"] == engram_id

    profile_frames = _mcp_frames(client, method="user.get_profile", params={}, request_id="profile")
    assert (
        _final_result_frame(profile_frames)["result"]["profile"]["username"]
        == get_settings().ui_demo_username
    )

    projects_frames = _mcp_frames(
        client,
        method="user.list_projects",
        params={},
        request_id="projects",
    )
    project_ids = _final_result_frame(projects_frames)["result"]["project_ids"]
    assert "project-mcp-tools" in project_ids

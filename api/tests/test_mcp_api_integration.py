from __future__ import annotations

import json
import re
from uuid import UUID

import pytest
from fastapi.testclient import TestClient

from app.config import get_settings
from app.main import app
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


def _mcp_frames(
    client,
    method: str,
    params: dict,
    request_id: str = "1",
    headers: dict[str, str] | None = None,
) -> list[dict]:  # noqa: ANN001
    response = client.post(
        "/api/v1/mcp/stream",
        json={
            "jsonrpc": "2.0",
            "id": request_id,
            "method": method,
            "params": params,
        },
        headers={"Accept": "text/event-stream", **(headers or {})},
    )
    assert response.status_code == 200
    assert "text/event-stream" in response.headers.get("content-type", "")
    frames: list[dict] = []
    for line in response.text.splitlines():
        if line.startswith("data: "):
            frames.append(json.loads(line[6:]))
    assert frames
    return frames


def _mcp_json_response(
    client,
    method: str,
    params: dict,
    request_id: str = "1",
    headers: dict[str, str] | None = None,
) -> dict:  # noqa: ANN001
    response = client.post(
        "/api/v1/mcp/stream",
        json={
            "jsonrpc": "2.0",
            "id": request_id,
            "method": method,
            "params": params,
        },
        headers={"Accept": "application/json", **(headers or {})},
    )
    assert response.status_code == 200
    assert "application/json" in response.headers.get("content-type", "")
    return response.json()


def _create_mcp_token(
    client,
    *,
    name: str,
    scope: str,
    allowed_tools: list[str] | None = None,
    allowed_project_ids: list[str] | None = None,
    expires_in_days: int = 90,
) -> dict:  # noqa: ANN001
    response = client.post(
        "/api/v1/mcp/tokens",
        json={
            "name": name,
            "scope": scope,
            "allowed_tools": allowed_tools or [],
            "allowed_project_ids": allowed_project_ids or [],
            "expires_in_days": expires_in_days,
        },
    )
    assert response.status_code == 201
    return response.json()


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
def test_mcp_stream_probe_supports_get_and_head(client, clean_db) -> None:
    get_response = client.get("/api/v1/mcp/stream")
    assert get_response.status_code == 200
    payload = get_response.json()
    assert payload["name"] == "engram-mcp-stream"
    assert payload["request_method"] == "POST"
    assert payload["response_type"] == "text/event-stream"

    head_response = client.head("/api/v1/mcp/stream")
    assert head_response.status_code == 200


@pytest.mark.integration
def test_mcp_notifications_initialized_returns_accepted(client, clean_db) -> None:
    _login(client)
    response = client.post(
        "/api/v1/mcp/stream",
        json={
            "jsonrpc": "2.0",
            "method": "notifications/initialized",
        },
        headers={"Accept": "application/json"},
    )
    assert response.status_code == 202
    assert response.text == ""


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
    assert "chat_create_session" in tool_names
    assert "chat_get_lifecycle_policy" in tool_names
    assert "chat_update_lifecycle_policy" in tool_names
    assert "chat_list_timeline" in tool_names
    assert "chat_list_messages" in tool_names
    assert "chat_list_pinned_engrams" in tool_names
    assert "chat_pin_engram" in tool_names
    assert "chat_unpin_engram" in tool_names
    assert "chat_list_pinned_documents" in tool_names
    assert "chat_pin_document" in tool_names
    assert "chat_unpin_document" in tool_names
    assert "chat_list_project_documents" in tool_names
    assert "chat_send_message" in tool_names
    assert "engram_create_from_conversation" in tool_names
    assert "engram_query" in tool_names
    assert "user_get_profile" in tool_names

    send_message_tool = next(item for item in tools if item["name"] == "chat_send_message")
    assert send_message_tool["inputSchema"]["type"] == "object"
    assert "session_id" in send_message_tool["inputSchema"]["required"]
    assert "content_text" in send_message_tool["inputSchema"]["required"]


@pytest.mark.integration
def test_mcp_initialize_supports_json_response_mode(client, clean_db) -> None:
    _login(client)
    frame = _mcp_json_response(client, method="initialize", params={}, request_id="init-json")
    assert frame["id"] == "init-json"
    assert frame["result"]["protocolVersion"] == "2024-11-05"


@pytest.mark.integration
def test_mcp_initialize_prefers_json_when_accepts_json_and_sse(client, clean_db) -> None:
    _login(client)
    frame = _mcp_json_response(
        client,
        method="initialize",
        params={},
        request_id="init-mixed-accept",
        headers={"Accept": "text/event-stream, application/json"},
    )
    assert frame["id"] == "init-mixed-accept"
    assert frame["result"]["protocolVersion"] == "2024-11-05"


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
def test_mcp_tools_call_accepts_underscore_tool_names(client, clean_db, monkeypatch) -> None:
    _login(client)
    _install_fake_provider(monkeypatch)

    create_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat_create_session",
            "arguments": {
                "project_id": "project-mcp-underscore",
                "title": "MCP underscore session",
                "provider": "openai",
                "model_id": "gpt-4o-mini",
                "visibility_scope": "private",
                "autosave_enabled": False,
            },
        },
        request_id="underscore-create",
    )
    session_id = _final_result_frame(create_frames)["result"]["structuredContent"]["session"][
        "session_id"
    ]

    send_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat_send_message",
            "arguments": {
                "session_id": session_id,
                "content_text": "hello underscore tools",
                "stream": True,
            },
        },
        request_id="underscore-send",
    )
    final = _final_result_frame(send_frames)
    assert final["result"]["tool_name"] == "chat_send_message"
    assert final["result"]["structuredContent"]["message"]["assistant_text"] == "assistant:streamed"


@pytest.mark.integration
def test_mcp_tools_call_document_pinning_workflow(client, clean_db, monkeypatch) -> None:
    _login(client)
    _install_fake_provider(monkeypatch)

    create_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat.create_session",
            "arguments": {
                "project_id": "project-mcp-docs",
                "title": "MCP document pin session",
                "provider": "openai",
                "model_id": "gpt-4o-mini",
                "visibility_scope": "private",
                "autosave_enabled": False,
            },
        },
        request_id="mcp-doc-create-session",
    )
    session_id = _final_result_frame(create_frames)["result"]["structuredContent"]["session"][
        "session_id"
    ]

    doc_a = client.post(
        "/api/v1/ingestion/text",
        json={
            "project_id": "project-mcp-docs",
            "title": "MCP Runbook",
            "text": "Service degraded. Start queue drain and monitor retries.",
            "visibility_scope": "project",
        },
    )
    assert doc_a.status_code == 201
    doc_a_id = doc_a.json()["document"]["document_id"]

    doc_b = client.post(
        "/api/v1/ingestion/text",
        json={
            "project_id": "project-mcp-docs",
            "title": "Escalation Notes",
            "text": "Escalate to DB oncall after 15 minutes and notify support.",
            "visibility_scope": "project",
        },
    )
    assert doc_b.status_code == 201
    doc_b_id = doc_b.json()["document"]["document_id"]

    list_docs = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat.list_project_documents",
            "arguments": {"project_id": "project-mcp-docs"},
        },
        request_id="mcp-doc-list",
    )
    listed_documents = _final_result_frame(list_docs)["result"]["structuredContent"]["documents"]
    assert {item["document_id"] for item in listed_documents} >= {doc_a_id, doc_b_id}

    pin_a = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat.pin_document",
            "arguments": {"session_id": session_id, "document_id": doc_a_id},
        },
        request_id="mcp-doc-pin-a",
    )
    assert (
        _final_result_frame(pin_a)["result"]["structuredContent"]["pinned"]["document_id"]
        == doc_a_id
    )

    pin_b = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat.pin_document",
            "arguments": {"session_id": session_id, "document_id": doc_b_id},
        },
        request_id="mcp-doc-pin-b",
    )
    assert (
        _final_result_frame(pin_b)["result"]["structuredContent"]["pinned"]["document_id"]
        == doc_b_id
    )

    pinned_docs = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat.list_pinned_documents",
            "arguments": {"session_id": session_id},
        },
        request_id="mcp-doc-pins",
    )
    pinned_list = _final_result_frame(pinned_docs)["result"]["structuredContent"][
        "pinned_documents"
    ]
    assert {item["document_id"] for item in pinned_list} == {doc_a_id, doc_b_id}

    send = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat.send_message",
            "arguments": {
                "session_id": session_id,
                "content_text": "Summarize pinned docs",
                "stream": True,
            },
        },
        request_id="mcp-doc-send",
    )
    final_send = _final_result_frame(send)["result"]["structuredContent"]["message"]
    assert len(final_send["used_document_chunk_ids"]) >= 2

    unpin_a = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat.unpin_document",
            "arguments": {"session_id": session_id, "document_id": doc_a_id},
        },
        request_id="mcp-doc-unpin-a",
    )
    assert _final_result_frame(unpin_a)["result"]["structuredContent"]["removed"] is True


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


@pytest.mark.integration
def test_mcp_create_from_conversation_auto_enriches_metadata(client, clean_db) -> None:
    _login(client)

    create_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "engram.create_from_conversation",
            "arguments": {
                "project_id": "project-mcp-conversation",
                "conversation_markdown": (
                    "## USER\nSummarize incident handoff.\n\n"
                    "## ASSISTANT\nOutage impact reduced after rollback and queue drain; "
                    "support team prepared customer communication."
                ),
                "title": "MCP conversation seed",
                "abstract": "",
                "tags": [],
                "keywords": [],
                "visibility_scope": "project",
            },
        },
        request_id="mcp-create-conversation",
    )
    structured = _final_result_frame(create_frames)["result"]["structuredContent"]
    engram_id = structured["engram"]["engram_id"]
    report = structured["enrichment_report"]
    assert report["enrichment_applied"] is True
    assert report["abstract_derived"] is True
    assert len(report["auto_tags"]) > 0
    assert len(report["auto_keywords"]) > 0

    listed = client.get("/api/v1/engrams", params={"project_id": "project-mcp-conversation"})
    assert listed.status_code == 200
    created = next(item for item in listed.json() if item["engram_id"] == engram_id)
    assert created["abstract"].strip() != ""
    assert len(created["tags"]) > 0
    assert len(created["keywords"]) > 0


@pytest.mark.integration
def test_mcp_create_from_conversation_preserves_explicit_metadata(client, clean_db) -> None:
    _login(client)

    create_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "engram.create_from_conversation",
            "arguments": {
                "project_id": "project-mcp-conversation-explicit",
                "conversation_markdown": "## ASSISTANT\nRelease note draft.",
                "title": "Explicit metadata preserve",
                "abstract": "Manual abstract",
                "tags": ["manual-tag"],
                "keywords": ["manual-keyword"],
                "visibility_scope": "private",
            },
        },
        request_id="mcp-create-conversation-explicit",
    )
    structured = _final_result_frame(create_frames)["result"]["structuredContent"]
    engram_id = structured["engram"]["engram_id"]
    report = structured["enrichment_report"]
    assert report["enrichment_applied"] is False
    assert report["abstract_derived"] is False
    assert report["auto_tags"] == []
    assert report["auto_keywords"] == []

    listed = client.get(
        "/api/v1/engrams",
        params={"project_id": "project-mcp-conversation-explicit"},
    )
    assert listed.status_code == 200
    created = next(item for item in listed.json() if item["engram_id"] == engram_id)
    assert created["abstract"] == "Manual abstract"
    assert created["tags"] == ["manual-tag"]
    assert created["keywords"] == ["manual-keyword"]


@pytest.mark.integration
def test_mcp_chat_save_as_engram_without_session_end_to_end(client, clean_db) -> None:
    _login(client)

    save_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat_save_as_engram",
            "arguments": {
                "project_id": "project-mcp-no-session",
                "conversation_markdown": (
                    "## USER\nInvestigate payment latency spike.\n\n"
                    "## ASSISTANT\nLikely cache stampede and queue saturation; "
                    "rollback + throttling mitigated impact."
                ),
                "title": "MCP no-session snapshot",
                "abstract": "",
                "tags": [],
                "keywords": [],
                "visibility_scope": "project",
            },
        },
        request_id="mcp-save-without-session",
    )
    structured = _final_result_frame(save_frames)["result"]["structuredContent"]
    engram_id = structured["saved_engram"]["engram_id"]
    report = structured["enrichment_report"]
    assert report["enrichment_applied"] is True
    assert report["abstract_derived"] is True

    create_session_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat_create_session",
            "arguments": {
                "project_id": "project-mcp-no-session",
                "title": "Pin from conversation-only engram",
                "provider": "openai",
                "model_id": "gpt-4o-mini",
                "visibility_scope": "private",
                "autosave_enabled": False,
            },
        },
        request_id="mcp-save-no-session-create-chat",
    )
    session_id = _final_result_frame(create_session_frames)["result"]["structuredContent"][
        "session"
    ]["session_id"]

    pin_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat_pin_engram",
            "arguments": {"session_id": session_id, "engram_id": engram_id},
        },
        request_id="mcp-save-no-session-pin",
    )
    assert _final_result_frame(pin_frames)["result"]["structuredContent"]["pinned"][
        "engram_id"
    ] == (engram_id)

    pinned_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat_list_pinned_engrams",
            "arguments": {"session_id": session_id},
        },
        request_id="mcp-save-no-session-list-pins",
    )
    pinned = _final_result_frame(pinned_frames)["result"]["structuredContent"]["pinned_engrams"]
    assert any(item["engram_id"] == engram_id for item in pinned)

    query_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "engram_query",
            "arguments": {
                "query": "cache stampede",
                "project_id": "project-mcp-no-session",
                "top_k": 5,
            },
        },
        request_id="mcp-save-no-session-query",
    )
    results = _final_result_frame(query_frames)["result"]["structuredContent"]["results"]
    assert any(item["engram_id"] == engram_id for item in results)


@pytest.mark.integration
def test_mcp_lifecycle_policy_and_timeline_tools(client, clean_db, monkeypatch) -> None:
    _login(client)
    _install_fake_provider(monkeypatch)

    create_session_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat.create_session",
            "arguments": {
                "project_id": "project-mcp-lifecycle",
                "title": "Lifecycle Session",
                "provider": "openai",
                "model_id": "gpt-4o-mini",
                "visibility_scope": "private",
                "autosave_enabled": False,
            },
        },
        request_id="mcp-lifecycle-create",
    )
    session_id = _final_result_frame(create_session_frames)["result"]["structuredContent"][
        "session"
    ]["session_id"]

    update_policy = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat.update_lifecycle_policy",
            "arguments": {
                "session_id": session_id,
                "autosave_enabled": True,
                "autosave_strategy": "interval",
                "autosave_interval_minutes": 1,
                "retention_days": 7,
                "retention_max_snapshots": 20,
            },
        },
        request_id="mcp-lifecycle-policy-update",
    )
    updated_policy = _final_result_frame(update_policy)["result"]["structuredContent"][
        "lifecycle_policy"
    ]
    assert updated_policy["autosave_enabled"] is True
    assert updated_policy["autosave_strategy"] == "interval"

    _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat.send_message",
            "arguments": {
                "session_id": session_id,
                "content_text": (
                    "Create a detailed stakeholder update including impact timeline and mitigation status."
                ),
                "stream": False,
            },
        },
        request_id="mcp-lifecycle-send",
    )

    timeline_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat.list_timeline",
            "arguments": {"session_id": session_id},
        },
        request_id="mcp-lifecycle-timeline",
    )
    events = _final_result_frame(timeline_frames)["result"]["structuredContent"]["events"]
    assert any(item["event_type"] == "autosave_snapshot" for item in events)


@pytest.mark.integration
def test_mcp_bearer_read_token_can_call_read_tools_without_session(client, clean_db) -> None:
    _login(client)
    created = _create_mcp_token(client, name="read profile", scope="read")
    headers = {"Authorization": f"Bearer {created['token']}"}

    bearer_client = TestClient(app)
    frames = _mcp_frames(
        bearer_client,
        method="tools/call",
        params={"name": "user.get_profile", "arguments": {}},
        request_id="bearer-profile",
        headers=headers,
    )
    profile = _final_result_frame(frames)["result"]["structuredContent"]["profile"]
    assert profile["username"] == get_settings().ui_demo_username


@pytest.mark.integration
def test_mcp_bearer_read_token_cannot_perform_write_tools(client, clean_db) -> None:
    _login(client)
    created = _create_mcp_token(client, name="read no write", scope="read")
    headers = {"Authorization": f"Bearer {created['token']}"}

    bearer_client = TestClient(app)
    frames = _mcp_frames(
        bearer_client,
        method="tools/call",
        params={
            "name": "chat.create_session",
            "arguments": {"project_id": "project-bearer", "title": "Denied"},
        },
        request_id="bearer-denied-write",
        headers=headers,
    )
    error_frame = [item for item in frames if "error" in item][0]
    assert error_frame["error"]["code"] == -32003
    assert error_frame["error"]["data"]["required_scope"] == "write"
    assert error_frame["error"]["data"]["token_scope"] == "read"


@pytest.mark.integration
def test_mcp_bearer_write_token_can_perform_write_tools(client, clean_db) -> None:
    _login(client)
    created = _create_mcp_token(
        client,
        name="write token",
        scope="write",
        allowed_project_ids=["project-bearer-write"],
    )
    headers = {"Authorization": f"Bearer {created['token']}"}
    bearer_client = TestClient(app)

    create_session_frames = _mcp_frames(
        bearer_client,
        method="tools/call",
        params={
            "name": "chat.create_session",
            "arguments": {
                "project_id": "project-bearer-write",
                "title": "Write Session",
                "provider": "openai",
                "model_id": "gpt-4o-mini",
                "visibility_scope": "private",
                "autosave_enabled": False,
            },
        },
        request_id="bearer-write-create-session",
        headers=headers,
    )
    session_id = _final_result_frame(create_session_frames)["result"]["structuredContent"][
        "session"
    ]["session_id"]
    assert session_id

    create_engram_frames = _mcp_frames(
        bearer_client,
        method="tools/call",
        params={
            "name": "engram.create",
            "arguments": {
                "project_id": "project-bearer-write",
                "title": "Bearer Write Engram",
                "abstract": "Created by bearer token.",
                "detailed_summary_markdown": "This engram was created through token auth.",
            },
        },
        request_id="bearer-write-create-engram",
        headers=headers,
    )
    created_engram_id = _final_result_frame(create_engram_frames)["result"]["structuredContent"][
        "engram"
    ]["engram_id"]
    assert created_engram_id


@pytest.mark.integration
def test_mcp_token_allowed_tools_and_project_guards(client, clean_db) -> None:
    _login(client)
    tools_token = _create_mcp_token(
        client,
        name="query-only",
        scope="write",
        allowed_tools=["engram.query"],
    )
    tools_headers = {"Authorization": f"Bearer {tools_token['token']}"}
    bearer_client = TestClient(app)

    tools_list_frames = _mcp_frames(
        bearer_client,
        method="tools/list",
        params={},
        request_id="token-tools-list",
        headers=tools_headers,
    )
    visible_names = {
        tool["name"] for tool in _final_result_frame(tools_list_frames)["result"]["tools"]
    }
    assert visible_names == {"engram_query"}

    denied_write = _mcp_frames(
        bearer_client,
        method="tools/call",
        params={
            "name": "chat.create_session",
            "arguments": {"project_id": "engram-vault", "title": "blocked"},
        },
        request_id="token-tools-denied",
        headers=tools_headers,
    )
    denied_write_error = [item for item in denied_write if "error" in item][0]
    assert denied_write_error["error"]["code"] == -32003
    assert denied_write_error["error"]["data"]["tool"] == "chat.create_session"

    project_token = _create_mcp_token(
        client,
        name="multi-project",
        scope="read",
        allowed_project_ids=["project-a", "project-b"],
    )
    project_headers = {"Authorization": f"Bearer {project_token['token']}"}
    missing_project = _mcp_frames(
        bearer_client,
        method="tools/call",
        params={"name": "engram.query", "arguments": {"query": "incident"}},
        request_id="token-project-missing",
        headers=project_headers,
    )
    project_error = [item for item in missing_project if "error" in item][0]
    assert project_error["error"]["code"] == -32602
    assert project_error["error"]["data"]["missing"] == "project_id"

    single_project = _create_mcp_token(
        client,
        name="single-project-auto",
        scope="read",
        allowed_project_ids=["engram-vault"],
    )
    single_headers = {"Authorization": f"Bearer {single_project['token']}"}
    auto_project_frames = _mcp_frames(
        bearer_client,
        method="tools/call",
        params={"name": "engram.query", "arguments": {"query": "memory"}},
        request_id="token-project-auto",
        headers=single_headers,
    )
    assert _final_result_frame(auto_project_frames)["result"]["tool_name"] == "engram.query"


@pytest.mark.integration
def test_mcp_revoked_and_expired_tokens_fail_authentication(client, clean_db, db_conn) -> None:
    _login(client)
    created = _create_mcp_token(client, name="revoked token", scope="read")
    token_id = UUID(created["token_id"])
    headers = {"Authorization": f"Bearer {created['token']}"}
    bearer_client = TestClient(app)

    revoke_response = client.post(
        f"/api/v1/mcp/tokens/{token_id}/revoke", json={"reason": "rotate"}
    )
    assert revoke_response.status_code == 200

    revoked_response = bearer_client.post(
        "/api/v1/mcp/stream",
        headers=headers,
        json={"jsonrpc": "2.0", "id": "revoked", "method": "tools/list", "params": {}},
    )
    assert revoked_response.status_code == 401

    expired = _create_mcp_token(client, name="expired token", scope="read")
    expired_id = UUID(expired["token_id"])
    expired_headers = {"Authorization": f"Bearer {expired['token']}"}
    with db_conn.cursor() as cur:
        cur.execute(
            "UPDATE mcp_tokens SET expires_at = now() - interval '1 minute' WHERE token_id = %s",
            (expired_id,),
        )
    db_conn.commit()

    expired_response = bearer_client.post(
        "/api/v1/mcp/stream",
        headers=expired_headers,
        json={"jsonrpc": "2.0", "id": "expired", "method": "tools/list", "params": {}},
    )
    assert expired_response.status_code == 401

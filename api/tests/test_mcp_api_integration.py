from __future__ import annotations

import pytest

from tests.mcp_api_integration_helpers import (
    _final_result_frame,
    _login,
    _mcp_frames,
    _mcp_json_response,
)


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
    assert "chat_delete_session" in tool_names
    assert "chat_restore_session" in tool_names
    assert "project_list" in tool_names
    assert "project_create" in tool_names
    assert "project_get_default" in tool_names
    assert "project_set_default" in tool_names
    assert "chat_send_message" in tool_names
    assert "engram_list" in tool_names
    assert "engram_get" in tool_names
    assert "engram_update" in tool_names
    assert "engram_move_project" in tool_names
    assert "engram_delete" in tool_names
    assert "engram_restore" in tool_names
    assert "engram_collection_list" in tool_names
    assert "engram_collection_create" in tool_names
    assert "engram_collection_update" in tool_names
    assert "engram_collection_delete" in tool_names
    assert "engram_collection_add_items" in tool_names
    assert "engram_collection_remove_items" in tool_names
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
def test_mcp_tools_call_project_and_engram_round_trip_plus_unknown_tool(client, clean_db) -> None:
    _login(client)

    create_project = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "project.create",
            "arguments": {
                "project_id": "project-mcp-roundtrip",
                "name": "project-mcp-roundtrip",
            },
        },
        request_id="tools-call-project-create",
    )
    create_project_result = _final_result_frame(create_project)["result"]["structuredContent"]
    assert create_project_result["project"]["project_id"] == "project-mcp-roundtrip"

    create_engram = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "engram.create",
            "arguments": {
                "project_id": "project-mcp-roundtrip",
                "title": "roundtrip seed",
                "detailed_summary_markdown": "roundtrip body",
                "visibility_scope": "private",
            },
        },
        request_id="tools-call-engram-create",
    )
    engram_id = _final_result_frame(create_engram)["result"]["structuredContent"]["engram"][
        "engram_id"
    ]

    list_engrams = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "engram.list",
            "arguments": {"project_id": "project-mcp-roundtrip"},
        },
        request_id="tools-call-engram-list",
    )
    listed_engrams = _final_result_frame(list_engrams)["result"]["structuredContent"]["engrams"]
    assert any(item["engram_id"] == engram_id for item in listed_engrams)

    list_projects = _mcp_frames(
        client,
        method="tools/call",
        params={"name": "project.list", "arguments": {}},
        request_id="tools-call-project-list",
    )
    listed_projects = _final_result_frame(list_projects)["result"]["structuredContent"]["projects"]
    assert any(item["project_id"] == "project-mcp-roundtrip" for item in listed_projects)

    unknown_frames = _mcp_frames(
        client,
        method="tools/call",
        params={"name": "unknown.tool", "arguments": {}},
        request_id="tools-call-unknown",
    )
    error_frame = [item for item in unknown_frames if "error" in item][0]
    assert error_frame["id"] == "tools-call-unknown"
    assert error_frame["error"]["code"] == -32601
    assert error_frame["error"]["data"]["method"] == "unknown.tool"

from __future__ import annotations

import pytest

from app.config import get_settings
from tests.mcp_api_integration_helpers import (
    _assert_saved_engram_queryable,
    _create_tools_chat_session,
    _final_result_frame,
    _ingest_project_document,
    _install_fake_provider,
    _login,
    _mcp_frames,
    _pin_document_for_session,
    _pin_saved_engram_in_new_session,
    _save_as_engram_without_session,
    _tools_call_structured_content,
)


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
    project_id = "project-mcp-docs"
    session_id = _create_tools_chat_session(client, project_id, "mcp-doc-create-session")
    doc_a_id = _ingest_project_document(
        client,
        project_id,
        "MCP Runbook",
        "Service degraded. Start queue drain and monitor retries.",
    )
    doc_b_id = _ingest_project_document(
        client,
        project_id,
        "Escalation Notes",
        "Escalate to DB oncall after 15 minutes and notify support.",
    )

    listed_documents = _tools_call_structured_content(
        client,
        "mcp-doc-list",
        {
            "name": "chat.list_project_documents",
            "arguments": {"project_id": project_id},
        },
    )["documents"]
    assert {item["document_id"] for item in listed_documents} >= {doc_a_id, doc_b_id}

    _pin_document_for_session(client, session_id, doc_a_id, "mcp-doc-pin-a")
    _pin_document_for_session(client, session_id, doc_b_id, "mcp-doc-pin-b")

    pinned_list = _tools_call_structured_content(
        client,
        "mcp-doc-pins",
        {
            "name": "chat.list_pinned_documents",
            "arguments": {"session_id": session_id},
        },
    )["pinned_documents"]
    assert {item["document_id"] for item in pinned_list} == {doc_a_id, doc_b_id}

    message = _tools_call_structured_content(
        client,
        "mcp-doc-send",
        {
            "name": "chat.send_message",
            "arguments": {
                "session_id": session_id,
                "content_text": "Summarize pinned docs",
                "stream": True,
            },
        },
    )["message"]
    assert len(message["used_document_chunk_ids"]) >= 2

    unpin = _tools_call_structured_content(
        client,
        "mcp-doc-unpin-a",
        {
            "name": "chat.unpin_document",
            "arguments": {"session_id": session_id, "document_id": doc_a_id},
        },
    )
    assert unpin["removed"] is True


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
    project_id = "project-mcp-no-session"
    engram_id = _save_as_engram_without_session(client, project_id)
    _pin_saved_engram_in_new_session(client, project_id, engram_id)
    _assert_saved_engram_queryable(client, project_id, engram_id)


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
def test_mcp_engram_create_uses_default_project_when_project_id_omitted(client, clean_db) -> None:
    _login(client)

    create_project = client.post(
        "/api/v1/projects",
        json={
            "project_id": "project-mcp-default-fallback",
            "name": "project-mcp-default-fallback",
            "description": "Default project fallback test",
        },
    )
    assert create_project.status_code == 201

    set_default = client.patch(
        "/api/v1/projects/default",
        json={"project_id": "project-mcp-default-fallback"},
    )
    assert set_default.status_code == 200

    create_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "engram.create",
            "arguments": {
                "title": "Fallback project create",
                "abstract": "Project id omitted intentionally.",
                "detailed_summary_markdown": "MCP should resolve the actor default project.",
            },
        },
        request_id="mcp-default-project-create",
    )
    engram = _final_result_frame(create_frames)["result"]["structuredContent"]["engram"]
    assert engram["resolved_project_id"] == "project-mcp-default-fallback"
    assert engram["used_default_project"] is True

    save_frames = _mcp_frames(
        client,
        method="tools/call",
        params={
            "name": "chat.save_as_engram",
            "arguments": {
                "conversation_markdown": "## USER\nCapture a default-project fallback test snapshot.",
                "title": "Fallback save snapshot",
            },
        },
        request_id="mcp-default-project-save",
    )
    saved = _final_result_frame(save_frames)["result"]["structuredContent"]["saved_engram"]
    assert saved["resolved_project_id"] == "project-mcp-default-fallback"
    assert saved["used_default_project"] is True

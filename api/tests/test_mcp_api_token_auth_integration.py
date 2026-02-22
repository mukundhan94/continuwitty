from __future__ import annotations

from uuid import UUID, uuid4

import pytest
from fastapi.testclient import TestClient

from app.config import get_settings
from app.main import app
from tests.mcp_api_integration_helpers import (
    _create_mcp_token,
    _final_result_frame,
    _login,
    _mcp_frames,
)


def _create_project(client, project_id: str) -> None:  # noqa: ANN001
    created = client.post(
        "/api/v1/projects",
        json={
            "project_id": project_id,
            "name": project_id,
            "description": "token auth transfer test",
        },
    )
    assert created.status_code == 201


def _seed_project_engram(client, project_id: str) -> None:  # noqa: ANN001
    seeded = client.post(
        "/api/v1/engrams",
        json={
            "project_id": project_id,
            "title": "Token transfer seed",
            "abstract": "seed",
            "detailed_summary_markdown": "seed body",
            "visibility_scope": "project",
        },
    )
    assert seeded.status_code == 200


def _error_frame(frames: list[dict]) -> dict:
    return [item for item in frames if "error" in item][0]


def _read_token_export_bundle(  # noqa: ANN001
    client,
    bearer_client: TestClient,
    *,
    source_project_id: str,
) -> tuple[dict, dict[str, str]]:
    read_token = _create_mcp_token(
        client,
        name="transfer-read",
        scope="read",
        allowed_project_ids=[source_project_id],
    )
    read_headers = {"Authorization": f"Bearer {read_token['token']}"}
    export_frames = _mcp_frames(
        bearer_client,
        method="tools/call",
        params={
            "name": "project.export_bundle",
            "arguments": {"project_id": source_project_id},
        },
        request_id="token-transfer-export",
        headers=read_headers,
    )
    exported_bundle = _final_result_frame(export_frames)["result"]["structuredContent"]["bundle"]
    return exported_bundle, read_headers


def _assert_read_token_transfer_denials(  # noqa: ANN001
    bearer_client: TestClient,
    *,
    target_project_id: str,
    exported_bundle: dict,
    read_headers: dict[str, str],
) -> None:
    denied_export_error = _error_frame(
        _mcp_frames(
            bearer_client,
            method="tools/call",
            params={
                "name": "project.export_bundle",
                "arguments": {"project_id": target_project_id},
            },
            request_id="token-transfer-export-denied",
            headers=read_headers,
        )
    )["error"]
    assert denied_export_error["code"] == -32003
    assert denied_export_error["data"]["project_id"] == target_project_id

    denied_import_error = _error_frame(
        _mcp_frames(
            bearer_client,
            method="tools/call",
            params={
                "name": "project.import_bundle",
                "arguments": {
                    "project_id": target_project_id,
                    "bundle": exported_bundle,
                    "conflict_policy": "skip",
                },
            },
            request_id="token-transfer-import-read-denied",
            headers=read_headers,
        )
    )["error"]
    assert denied_import_error["code"] == -32003
    assert denied_import_error["data"]["required_scope"] == "write"


def _write_token_import_summary(  # noqa: ANN001
    client,
    bearer_client: TestClient,
    *,
    target_project_id: str,
    exported_bundle: dict,
) -> dict:
    write_token = _create_mcp_token(
        client,
        name="transfer-write",
        scope="write",
        allowed_project_ids=[target_project_id],
    )
    write_headers = {"Authorization": f"Bearer {write_token['token']}"}
    imported = _mcp_frames(
        bearer_client,
        method="tools/call",
        params={
            "name": "project.import_bundle",
            "arguments": {
                "project_id": target_project_id,
                "bundle": exported_bundle,
                "conflict_policy": "skip",
            },
        },
        request_id="token-transfer-import-allowed",
        headers=write_headers,
    )
    return _final_result_frame(imported)["result"]["structuredContent"]["summary"]


@pytest.mark.integration
def test_mcp_token_project_fallback_for_engram_create_paths(client, clean_db) -> None:
    _login(client)
    bearer_client = TestClient(app)

    single = _create_mcp_token(
        client,
        name="single-project-fallback",
        scope="write",
        allowed_project_ids=["project-mcp-single-fallback"],
    )
    single_headers = {"Authorization": f"Bearer {single['token']}"}

    create_frames = _mcp_frames(
        bearer_client,
        method="tools/call",
        params={
            "name": "engram.create",
            "arguments": {
                "title": "Token fallback engram",
                "abstract": "No project in args.",
                "detailed_summary_markdown": "Bearer token should inject single allowed project.",
            },
        },
        request_id="mcp-token-single-create",
        headers=single_headers,
    )
    created_engram = _final_result_frame(create_frames)["result"]["structuredContent"]["engram"]
    assert created_engram["resolved_project_id"] == "project-mcp-single-fallback"
    assert created_engram["used_default_project"] is False

    from_conversation_frames = _mcp_frames(
        bearer_client,
        method="tools/call",
        params={
            "name": "engram.create_from_conversation",
            "arguments": {
                "conversation_markdown": "## ASSISTANT\nToken fallback path works.",
                "title": "Token fallback conversation",
            },
        },
        request_id="mcp-token-single-conversation",
        headers=single_headers,
    )
    structured = _final_result_frame(from_conversation_frames)["result"]["structuredContent"]
    assert structured["engram"]["resolved_project_id"] == "project-mcp-single-fallback"
    assert structured["enrichment_report"]["resolved_project_id"] == "project-mcp-single-fallback"

    multi = _create_mcp_token(
        client,
        name="multi-project-fallback-denied",
        scope="write",
        allowed_project_ids=["project-a", "project-b"],
    )
    multi_headers = {"Authorization": f"Bearer {multi['token']}"}
    denied_frames = _mcp_frames(
        bearer_client,
        method="tools/call",
        params={
            "name": "engram.create",
            "arguments": {
                "title": "Should fail",
                "detailed_summary_markdown": "Multiple allowed projects require explicit project_id.",
            },
        },
        request_id="mcp-token-multi-denied",
        headers=multi_headers,
    )
    denied_error = [item for item in denied_frames if "error" in item][0]
    assert denied_error["error"]["code"] == -32602
    assert denied_error["error"]["data"]["missing"] == "project_id"


@pytest.mark.integration
def test_mcp_engram_management_write_requires_owner_or_admin(client, clean_db) -> None:
    _login(client)

    created = client.post(
        "/api/v1/engrams",
        json={
            "project_id": "engram-vault",
            "thread_id": "owner-admin-policy",
            "title": "Owner/Admin policy seed",
            "abstract": "admin-owned engram",
            "detailed_summary_markdown": "Only owner or admin should edit this.",
            "tags": ["policy"],
            "keywords": ["ownership"],
        },
    )
    assert created.status_code == 200
    engram_id = created.json()["engram_id"]

    viewer_username = f"viewer_{uuid4().hex[:8]}"
    create_user = client.post(
        "/api/v1/users",
        json={
            "username": viewer_username,
            "password": "viewerpass123",
            "role": "viewer",
            "is_active": True,
        },
    )
    assert create_user.status_code == 201

    viewer_client = TestClient(app)
    _login(viewer_client, viewer_username, "viewerpass123")
    denied_frames = _mcp_frames(
        viewer_client,
        method="tools/call",
        params={
            "name": "engram.update",
            "arguments": {
                "engram_id": engram_id,
                "title": "viewer attempt",
            },
        },
        request_id="mcp-owner-admin-denied",
    )
    denied_error = [item for item in denied_frames if "error" in item][0]
    assert denied_error["error"]["code"] == -32003
    assert denied_error["error"]["data"]["resource"] == "engram"


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
def test_mcp_token_authorization_for_project_transfer_tools(client, clean_db) -> None:
    _login(client)

    source_project_id = f"mcp-token-export-src-{uuid4().hex[:8]}"
    target_project_id = f"mcp-token-export-dst-{uuid4().hex[:8]}"
    _create_project(client, source_project_id)
    _create_project(client, target_project_id)
    _seed_project_engram(client, source_project_id)

    bearer_client = TestClient(app)
    exported_bundle, read_headers = _read_token_export_bundle(
        client,
        bearer_client,
        source_project_id=source_project_id,
    )
    assert exported_bundle["project"]["project_id"] == source_project_id

    _assert_read_token_transfer_denials(
        bearer_client,
        target_project_id=target_project_id,
        exported_bundle=exported_bundle,
        read_headers=read_headers,
    )
    summary = _write_token_import_summary(
        client,
        bearer_client,
        target_project_id=target_project_id,
        exported_bundle=exported_bundle,
    )
    assert summary["target_project_id"] == target_project_id
    assert summary["imported_engrams"] == 1


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

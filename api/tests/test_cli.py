from __future__ import annotations

import json
from datetime import UTC, datetime
from types import SimpleNamespace
from uuid import UUID, uuid4

from app.cli import main
from app.mcp.client import McpToolCallResult, parse_mcp_jsonrpc_frame
from app.models import EngramCreateResponse, EngramQueryResult, McpJsonRpcRequest, RehydrationBundle

MINIMAL_ENGRAM = {
    "project_id": "cli-project",
    "thread_id": "cli-thread",
    "title": "CLI created engram",
    "abstract": "Created by CLI upload command.",
    "detailed_summary_markdown": "CLI summary",
    "tags": ["cli"],
    "keywords": ["upload"],
}


def test_cli_upload_single_file(monkeypatch, tmp_path, capsys) -> None:
    payload_path = tmp_path / "engram.json"
    payload_path.write_text(json.dumps(MINIMAL_ENGRAM), encoding="utf-8")

    captured: dict[str, object] = {}

    def fake_create_engram(payload, embedding_dim, enrichment_origin="unknown"):  # noqa: ANN001
        captured["payload"] = payload
        captured["embedding_dim"] = embedding_dim
        captured["enrichment_origin"] = enrichment_origin
        return EngramCreateResponse(
            engram_id=UUID("11111111-1111-1111-1111-111111111111"),
            created_at=datetime(2026, 2, 15, tzinfo=UTC),
        )

    monkeypatch.setattr("app.cli.get_settings", lambda: SimpleNamespace(embedding_dim=16))
    monkeypatch.setattr("app.cli.create_engram", fake_create_engram)

    code = main(["upload", "--file", str(payload_path)])
    stdout = capsys.readouterr().out

    assert code == 0
    body = json.loads(stdout)
    assert body["created"] == 1
    assert body["items"][0]["engram_id"] == "11111111-1111-1111-1111-111111111111"
    assert captured["embedding_dim"] == 16
    assert captured["enrichment_origin"] == "cli.upload"


def test_cli_search_filters(monkeypatch, capsys) -> None:
    captured: dict[str, object] = {}

    def fake_query(request, embedding_dim):  # noqa: ANN001
        captured["request"] = request
        captured["embedding_dim"] = embedding_dim
        return [
            EngramQueryResult(
                engram_id=UUID("22222222-2222-2222-2222-222222222222"),
                project_id="cli-project",
                title="Result",
                abstract="Result abstract",
                created_at=datetime(2026, 2, 15, tzinfo=UTC),
                tags=["cli"],
                keywords=["search"],
                distance=0.12,
            )
        ]

    monkeypatch.setattr("app.cli.get_settings", lambda: SimpleNamespace(embedding_dim=24))
    monkeypatch.setattr("app.cli.query_engrams", fake_query)

    code = main(
        [
            "search",
            "--query",
            "find cli",
            "--project-id",
            "cli-project",
            "--tag",
            "cli",
            "--keyword",
            "search",
            "--top-k",
            "7",
            "--created-after",
            "2026-02-01T00:00:00Z",
            "--created-before",
            "2026-02-28T23:59:59Z",
        ]
    )

    stdout = capsys.readouterr().out
    assert code == 0
    body = json.loads(stdout)
    assert body[0]["engram_id"] == "22222222-2222-2222-2222-222222222222"
    assert captured["embedding_dim"] == 24

    request = captured["request"]
    assert request.query == "find cli"
    assert request.top_k == 7
    assert request.project_id == "cli-project"
    assert request.tags == ["cli"]
    assert request.keywords == ["search"]


def test_cli_rehydrate_success(monkeypatch, capsys) -> None:
    engram_id = uuid4()

    monkeypatch.setattr(
        "app.cli.get_rehydration_bundle",
        lambda _engram_id: RehydrationBundle(
            engram_id=engram_id,
            project_id="cli-project",
            title="CLI Bundle",
            compact_summary="compact",
            key_decisions=[],
            open_questions=[],
            top_citations=[],
            context_markdown="# Context",
        ),
    )

    code = main(["rehydrate", "--engram-id", str(engram_id)])
    stdout = capsys.readouterr().out

    assert code == 0
    body = json.loads(stdout)
    assert body["engram_id"] == str(engram_id)
    assert body["project_id"] == "cli-project"


def test_cli_rehydrate_not_found(monkeypatch, capsys) -> None:
    monkeypatch.setattr("app.cli.get_rehydration_bundle", lambda _engram_id: None)

    code = main(["rehydrate", "--engram-id", "33333333-3333-3333-3333-333333333333"])
    output = capsys.readouterr()

    assert code == 1
    assert "Engram not found" in output.err


def test_cli_consolidate(monkeypatch, capsys) -> None:
    monkeypatch.setattr(
        "app.cli.run_project_consolidation",
        lambda **_kwargs: {
            "project_id": "proj-a",
            "created": False,
            "reason": "dry_run",
            "source_count": 4,
        },
    )

    code = main(
        [
            "consolidate",
            "--project-id",
            "proj-a",
            "--source-limit",
            "10",
            "--min-items",
            "2",
            "--dry-run",
        ]
    )
    stdout = capsys.readouterr().out

    assert code == 0
    body = json.loads(stdout)
    assert body["project_id"] == "proj-a"
    assert body["reason"] == "dry_run"


def test_cli_mcp_call_with_bearer_token(monkeypatch, capsys) -> None:
    captured: dict[str, object] = {}

    class _FakeMcpClient:
        def __init__(self, *, base_url: str, bearer_token: str | None) -> None:
            captured["base_url"] = base_url
            captured["bearer_token"] = bearer_token

        def __enter__(self):
            return self

        def __exit__(self, _exc_type, _exc, _tb) -> None:
            return None

        def login_with_password(self, *, username: str, password: str) -> None:
            captured["login"] = (username, password)

        def call_tool(self, *, method: str, params: dict, request_id: str | int):  # noqa: ANN001
            captured["method"] = method
            captured["params"] = params
            captured["request_id"] = request_id
            return McpToolCallResult(
                request=McpJsonRpcRequest(
                    jsonrpc="2.0",
                    id=request_id,
                    method=method,
                    params=params,
                ),
                frames=[
                    parse_mcp_jsonrpc_frame(
                        {
                            "jsonrpc": "2.0",
                            "id": request_id,
                            "result": {"ok": True, "method": method},
                        }
                    )
                ],
            )

    monkeypatch.setattr("app.cli.McpSseClient", _FakeMcpClient)

    code = main(
        [
            "mcp-call",
            "--base-url",
            "http://localhost:9000",
            "--method",
            "tools/list",
            "--request-id",
            "smoke-1",
            "--params-json",
            '{"include": "all"}',
            "--bearer-token",
            "engram_mcp_deadbeef_secret",
        ]
    )
    stdout = capsys.readouterr().out

    assert code == 0
    assert captured["base_url"] == "http://localhost:9000"
    assert captured["bearer_token"] == "engram_mcp_deadbeef_secret"
    assert "login" not in captured
    assert captured["method"] == "tools/list"
    assert captured["params"] == {"include": "all"}
    assert captured["request_id"] == "smoke-1"

    body = json.loads(stdout)
    assert body["request"]["method"] == "tools/list"
    assert body["frames"][0]["result"]["ok"] is True


def test_cli_mcp_call_with_session_login(monkeypatch, capsys) -> None:
    captured: dict[str, object] = {}

    class _FakeMcpClient:
        def __init__(self, *, base_url: str, bearer_token: str | None) -> None:
            captured["base_url"] = base_url
            captured["bearer_token"] = bearer_token

        def __enter__(self):
            return self

        def __exit__(self, _exc_type, _exc, _tb) -> None:
            return None

        def login_with_password(self, *, username: str, password: str) -> None:
            captured["login"] = (username, password)

        def call_tool(self, *, method: str, params: dict, request_id: str | int):  # noqa: ANN001
            return McpToolCallResult(
                request=McpJsonRpcRequest(
                    jsonrpc="2.0",
                    id=request_id,
                    method=method,
                    params=params,
                ),
                frames=[
                    parse_mcp_jsonrpc_frame(
                        {
                            "jsonrpc": "2.0",
                            "id": request_id,
                            "result": {"default_project_id": "engram-vault"},
                        }
                    )
                ],
            )

    monkeypatch.setattr("app.cli.McpSseClient", _FakeMcpClient)

    code = main(
        [
            "mcp-call",
            "--method",
            "project.get_default",
            "--username",
            "admin",
            "--password",
            "admin123",
        ]
    )
    _ = capsys.readouterr().out

    assert code == 0
    assert captured["bearer_token"] is None
    assert captured["login"] == ("admin", "admin123")


def test_cli_mcp_call_requires_auth(monkeypatch, capsys) -> None:
    class _UnusedMcpClient:
        def __init__(self, **_kwargs):
            raise AssertionError("MCP client should not be constructed without auth")

    monkeypatch.setattr("app.cli.McpSseClient", _UnusedMcpClient)

    code = main(
        [
            "mcp-call",
            "--method",
            "tools/list",
        ]
    )
    output = capsys.readouterr()

    assert code == 2
    assert "Provide either --bearer-token or both --username and --password" in output.err

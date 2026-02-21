from __future__ import annotations

import json
import re
from typing import Any, cast

from app.config import get_settings
from app.models import ChatProvider
from app.providers.base import ProviderGenerateResult


def _extract_csrf_token(html: str) -> str:
    match = re.search(r'name="csrf_token" value="([^"]+)"', html)
    assert match is not None
    return match.group(1)


def _login(client, username: str | None = None, password: str | None = None) -> None:  # noqa: ANN001
    settings = get_settings()
    login_username = username or settings.ui_demo_username
    login_password = password or settings.ui_demo_password
    login_page = client.get("/login")
    csrf_token = _extract_csrf_token(login_page.text)
    response = client.post(
        "/login",
        data={
            "username": login_username,
            "password": login_password,
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
    **request_options: object,
) -> list[dict]:  # noqa: ANN001
    request_id = cast(str, request_options.pop("request_id", "1"))
    headers = cast(dict[str, str] | None, request_options.pop("headers", None))
    assert not request_options, f"Unexpected request options: {sorted(request_options)}"
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
    **request_options: object,
) -> dict:  # noqa: ANN001
    request_id = cast(str, request_options.pop("request_id", "1"))
    headers = cast(dict[str, str] | None, request_options.pop("headers", None))
    assert not request_options, f"Unexpected request options: {sorted(request_options)}"
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


def _create_mcp_token(client, **token_options: object) -> dict:  # noqa: ANN001
    name = cast(str, token_options.pop("name"))
    scope = cast(str, token_options.pop("scope"))
    allowed_tools = cast(list[str] | None, token_options.pop("allowed_tools", None))
    allowed_project_ids = cast(list[str] | None, token_options.pop("allowed_project_ids", None))
    expires_in_days = cast(int, token_options.pop("expires_in_days", 90))
    assert not token_options, f"Unexpected token options: {sorted(token_options)}"
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


def _tools_call_structured_content(
    client,
    request_id: str,
    tool_call: dict[str, Any],
) -> dict[str, Any]:  # noqa: ANN001
    headers = cast(dict[str, str] | None, tool_call.get("headers"))
    return cast(
        dict[str, Any],
        _final_result_frame(
            _mcp_frames(
                client,
                method="tools/call",
                params={
                    "name": cast(str, tool_call["name"]),
                    "arguments": cast(dict[str, Any], tool_call["arguments"]),
                },
                request_id=request_id,
                headers=headers,
            )
        )["result"]["structuredContent"],
    )


def _create_tools_chat_session(client, project_id: str, request_id: str) -> str:  # noqa: ANN001
    structured = _tools_call_structured_content(
        client,
        request_id,
        {
            "name": "chat.create_session",
            "arguments": {
                "project_id": project_id,
                "title": "MCP document pin session",
                "provider": "openai",
                "model_id": "gpt-4o-mini",
                "visibility_scope": "private",
                "autosave_enabled": False,
            },
        },
    )
    return cast(str, structured["session"]["session_id"])


def _ingest_project_document(client, project_id: str, title: str, text: str) -> str:  # noqa: ANN001
    response = client.post(
        "/api/v1/ingestion/text",
        json={
            "project_id": project_id,
            "title": title,
            "text": text,
            "visibility_scope": "project",
        },
    )
    assert response.status_code == 201
    return cast(str, response.json()["document"]["document_id"])


def _pin_document_for_session(client, session_id: str, document_id: str, request_id: str) -> None:  # noqa: ANN001
    structured = _tools_call_structured_content(
        client,
        request_id,
        {
            "name": "chat.pin_document",
            "arguments": {"session_id": session_id, "document_id": document_id},
        },
    )
    assert structured["pinned"]["document_id"] == document_id

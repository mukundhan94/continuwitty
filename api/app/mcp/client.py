from __future__ import annotations

import json
import re
from collections.abc import Iterable, Iterator
from dataclasses import dataclass
from typing import Any, Literal

import httpx
from pydantic import BaseModel, ValidationError

from app.models import McpJsonRpcRequest

_CSRF_INPUT_PATTERN = re.compile(r'name="csrf_token" value="([^"]+)"')


class McpJsonRpcErrorObject(BaseModel):
    code: int
    message: str
    data: dict[str, Any] | None = None


class McpJsonRpcResultFrame(BaseModel):
    jsonrpc: Literal["2.0"]
    id: str | int
    result: dict[str, Any]


class McpJsonRpcErrorFrame(BaseModel):
    jsonrpc: Literal["2.0"]
    id: str | int
    error: McpJsonRpcErrorObject


class McpJsonRpcEventParams(BaseModel):
    id: str | int
    tool: str
    event: str
    data: dict[str, Any]


class McpJsonRpcEventFrame(BaseModel):
    jsonrpc: Literal["2.0"]
    method: Literal["mcp.event"]
    params: McpJsonRpcEventParams


McpParsedFrame = McpJsonRpcResultFrame | McpJsonRpcErrorFrame | McpJsonRpcEventFrame


class McpClientError(RuntimeError):
    """Base error for MCP client transport and protocol failures."""


class McpClientAuthError(McpClientError):
    """Raised when form-based session login fails."""


class McpClientTransportError(McpClientError):
    def __init__(self, status_code: int, detail: str) -> None:
        super().__init__(f"MCP transport error ({status_code}): {detail}")
        self.status_code = status_code
        self.detail = detail


class McpClientProtocolError(McpClientError):
    """Raised when SSE/JSON-RPC frames are malformed or missing."""


class McpClientRpcError(McpClientError):
    def __init__(self, *, request_id: str | int, error: McpJsonRpcErrorObject) -> None:
        super().__init__(f"MCP RPC error ({error.code}) for request {request_id}: {error.message}")
        self.request_id = request_id
        self.code = error.code
        self.message = error.message
        self.data = error.data


def parse_mcp_jsonrpc_frame(payload: dict[str, Any]) -> McpParsedFrame:
    """Parse one JSON-RPC frame into a typed model.

    The MCP stream can emit result, error, and `mcp.event` frames. We keep one
    parser here so both CLI tooling and tests share the same protocol contract.
    """
    try:
        if payload.get("method") == "mcp.event":
            return McpJsonRpcEventFrame.model_validate(payload)
        if "error" in payload:
            return McpJsonRpcErrorFrame.model_validate(payload)
        if "result" in payload:
            return McpJsonRpcResultFrame.model_validate(payload)
    except ValidationError as exc:
        raise McpClientProtocolError(f"Invalid JSON-RPC frame: {exc}") from exc

    raise McpClientProtocolError(f"Unsupported JSON-RPC frame shape: {payload}")


def mcp_frame_to_json(frame: McpParsedFrame) -> dict[str, Any]:
    return frame.model_dump(mode="json")


@dataclass
class McpToolCallResult:
    request: McpJsonRpcRequest
    frames: list[McpParsedFrame]

    @property
    def event_frames(self) -> list[McpJsonRpcEventFrame]:
        return [item for item in self.frames if isinstance(item, McpJsonRpcEventFrame)]

    @property
    def final_result_frame(self) -> McpJsonRpcResultFrame | None:
        for item in reversed(self.frames):
            if isinstance(item, McpJsonRpcResultFrame):
                return item
        return None

    @property
    def final_error_frame(self) -> McpJsonRpcErrorFrame | None:
        for item in reversed(self.frames):
            if isinstance(item, McpJsonRpcErrorFrame):
                return item
        return None

    def require_result(self) -> dict[str, Any]:
        error_frame = self.final_error_frame
        if error_frame is not None:
            raise McpClientRpcError(request_id=error_frame.id, error=error_frame.error)

        result_frame = self.final_result_frame
        if result_frame is None:
            raise McpClientProtocolError("No result frame returned by MCP stream")
        return result_frame.result


def _extract_csrf_token(login_html: str) -> str:
    # Form-based auth is intentionally reused so this client can act exactly like
    # browser sessions in local/dev setups.
    match = _CSRF_INPUT_PATTERN.search(login_html)
    if not match:
        raise McpClientAuthError("Failed to find csrf_token on /login page")
    return match.group(1)


def _iter_sse_data_payloads(lines: Iterable[str]) -> Iterator[str]:
    """Yield assembled `data:` payloads from SSE line streams.

    Multiple `data:` lines belong to one event frame and are joined with newline.
    """
    data_lines: list[str] = []
    for line in lines:
        normalized = line.rstrip("\r")
        if normalized == "":
            if data_lines:
                yield "\n".join(data_lines)
                data_lines.clear()
            continue
        if normalized.startswith("data:"):
            data_lines.append(normalized[5:].lstrip())

    if data_lines:
        yield "\n".join(data_lines)


def _error_detail_from_response(response: httpx.Response) -> str:
    content_type = response.headers.get("content-type", "")
    raw_bytes = response.read()
    text = raw_bytes.decode("utf-8", errors="replace").strip()
    if "application/json" in content_type:
        try:
            body = json.loads(text) if text else {}
        except json.JSONDecodeError:
            return text or response.reason_phrase
        if isinstance(body, dict):
            detail = body.get("detail")
            if isinstance(detail, str) and detail:
                return detail
        return text or response.reason_phrase
    return text or response.reason_phrase


class McpSseClient:
    """Typed MCP transport client for JSON-RPC over HTTP SSE.

    Typical workflow:
    1. `login_with_password(...)` (session-cookie auth)
    2. `call_tool(...)` with JSON-RPC request metadata
    3. Inspect typed frames or call `require_result()`
    """

    def __init__(
        self,
        *,
        base_url: str,
        timeout_seconds: float = 30.0,
        http_client: httpx.Client | None = None,
    ) -> None:
        self._base_url = base_url.rstrip("/")
        self._owns_client = http_client is None
        self._client = http_client or httpx.Client(timeout=timeout_seconds, follow_redirects=False)

    def close(self) -> None:
        if self._owns_client:
            self._client.close()

    def __enter__(self) -> McpSseClient:
        return self

    def __exit__(self, _exc_type, _exc, _tb) -> None:
        self.close()

    def login_with_password(self, *, username: str, password: str) -> None:
        login_page = self._client.get(f"{self._base_url}/login")
        if login_page.status_code != 200:
            raise McpClientAuthError(
                f"Failed to fetch login page ({login_page.status_code} {login_page.reason_phrase})"
            )

        csrf_token = _extract_csrf_token(login_page.text)
        login_response = self._client.post(
            f"{self._base_url}/login",
            data={"username": username, "password": password, "csrf_token": csrf_token},
            follow_redirects=False,
        )

        if login_response.status_code not in {302, 303}:
            raise McpClientAuthError(
                f"Login failed ({login_response.status_code} {login_response.reason_phrase})"
            )
        if login_response.headers.get("location") != "/ui":
            raise McpClientAuthError("Login did not return expected redirect to /ui")

    def call_tool(
        self,
        *,
        method: str,
        params: dict[str, Any] | None = None,
        request_id: str | int = "mcp-call-1",
    ) -> McpToolCallResult:
        # Keep request generation centralized so callers do not drift from server
        # expectations (JSON-RPC version, id shape, params defaulting).
        request = McpJsonRpcRequest(
            jsonrpc="2.0",
            id=request_id,
            method=method,
            params=params or {},
        )

        frames: list[McpParsedFrame] = []
        with self._client.stream(
            "POST",
            f"{self._base_url}/api/v1/mcp/stream",
            json=request.model_dump(mode="json"),
            headers={"Accept": "text/event-stream"},
        ) as response:
            if response.status_code != 200:
                detail = _error_detail_from_response(response)
                raise McpClientTransportError(response.status_code, detail)

            # The server emits `event: jsonrpc` lines; we only parse `data:` payloads
            # here and validate each payload as a typed JSON-RPC frame.
            for payload_text in _iter_sse_data_payloads(response.iter_lines()):
                try:
                    payload = json.loads(payload_text)
                except json.JSONDecodeError as exc:
                    raise McpClientProtocolError(
                        f"Failed to decode JSON-RPC frame from SSE payload: {payload_text}"
                    ) from exc

                if not isinstance(payload, dict):
                    raise McpClientProtocolError(
                        f"Expected JSON object frame, received: {type(payload).__name__}"
                    )
                frames.append(parse_mcp_jsonrpc_frame(payload))

        if not frames:
            raise McpClientProtocolError("MCP stream returned no frames")

        return McpToolCallResult(request=request, frames=frames)

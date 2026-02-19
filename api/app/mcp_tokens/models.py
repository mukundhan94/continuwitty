from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from uuid import UUID

TOKEN_PREFIX = "engram_mcp"


@dataclass(frozen=True)
class McpTokenRecord:
    token_id: UUID
    owner_user_id: UUID
    name: str
    scope: str
    allowed_tools: list[str]
    allowed_project_ids: list[str]
    token_secret_hash: str
    token_secret_hint: str
    expires_at: datetime
    last_used_at: datetime | None
    revoked_at: datetime | None
    created_at: datetime


@dataclass(frozen=True)
class McpTokenAuthContext:
    token_id: UUID
    owner_user_id: UUID
    scope: str
    allowed_tools: set[str]
    allowed_project_ids: set[str]
    expires_at: datetime

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from typing import Any
from uuid import UUID


@dataclass(frozen=True)
class OAuthClientRecord:
    client_id: str
    client_name: str
    redirect_uris: list[str]
    grant_types: list[str]
    response_types: list[str]
    token_endpoint_auth_method: str
    client_secret_hash: str | None
    metadata_json: dict[str, Any]
    created_at: datetime


@dataclass(frozen=True)
class OAuthAuthorizationCodeRecord:
    code_id: UUID
    code_hash: str
    client_id: str
    user_id: UUID
    redirect_uri: str
    code_challenge: str
    code_challenge_method: str
    requested_scope: str
    resource: str | None
    expires_at: datetime
    consumed_at: datetime | None
    created_at: datetime

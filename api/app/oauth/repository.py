from __future__ import annotations

from datetime import datetime
from typing import Any
from uuid import UUID

from psycopg.types.json import Jsonb

from ..db import get_conn
from .models import OAuthAuthorizationCodeRecord, OAuthClientRecord


def _row_to_client(row: dict[str, Any]) -> OAuthClientRecord:
    return OAuthClientRecord(
        client_id=row["client_id"],
        client_name=row["client_name"],
        redirect_uris=row["redirect_uris"] or [],
        grant_types=row["grant_types"] or [],
        response_types=row["response_types"] or [],
        token_endpoint_auth_method=row["token_endpoint_auth_method"],
        client_secret_hash=row["client_secret_hash"],
        metadata_json=row["metadata_json"] or {},
        created_at=row["created_at"],
    )


def _row_to_auth_code(row: dict[str, Any]) -> OAuthAuthorizationCodeRecord:
    return OAuthAuthorizationCodeRecord(
        code_id=row["code_id"],
        code_hash=row["code_hash"],
        client_id=row["client_id"],
        user_id=row["user_id"],
        redirect_uri=row["redirect_uri"],
        code_challenge=row["code_challenge"],
        code_challenge_method=row["code_challenge_method"],
        requested_scope=row["requested_scope"] or "",
        resource=row["resource"],
        expires_at=row["expires_at"],
        consumed_at=row["consumed_at"],
        created_at=row["created_at"],
    )


def create_oauth_client(
    *,
    client_id: str,
    client_name: str,
    redirect_uris: list[str],
    grant_types: list[str],
    response_types: list[str],
    token_endpoint_auth_method: str,
    client_secret_hash: str | None,
    metadata_json: dict[str, Any],
) -> OAuthClientRecord:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO oauth_clients (
                client_id,
                client_name,
                redirect_uris,
                grant_types,
                response_types,
                token_endpoint_auth_method,
                client_secret_hash,
                metadata_json
            )
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
            RETURNING
                client_id,
                client_name,
                redirect_uris,
                grant_types,
                response_types,
                token_endpoint_auth_method,
                client_secret_hash,
                metadata_json,
                created_at
            """,
            (
                client_id,
                client_name,
                redirect_uris,
                grant_types,
                response_types,
                token_endpoint_auth_method,
                client_secret_hash,
                Jsonb(metadata_json or {}),
            ),
        )
        row = cur.fetchone()
    return _row_to_client(row)


def get_oauth_client(*, client_id: str) -> OAuthClientRecord | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT
                client_id,
                client_name,
                redirect_uris,
                grant_types,
                response_types,
                token_endpoint_auth_method,
                client_secret_hash,
                metadata_json,
                created_at
            FROM oauth_clients
            WHERE client_id = %s
            """,
            (client_id,),
        )
        row = cur.fetchone()
    if not row:
        return None
    return _row_to_client(row)


def create_oauth_authorization_code(
    *,
    code_id: UUID,
    code_hash: str,
    client_id: str,
    user_id: UUID,
    redirect_uri: str,
    code_challenge: str,
    code_challenge_method: str,
    requested_scope: str,
    resource: str | None,
    expires_at: datetime,
) -> OAuthAuthorizationCodeRecord:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO oauth_authorization_codes (
                code_id,
                code_hash,
                client_id,
                user_id,
                redirect_uri,
                code_challenge,
                code_challenge_method,
                requested_scope,
                resource,
                expires_at
            )
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            RETURNING
                code_id,
                code_hash,
                client_id,
                user_id,
                redirect_uri,
                code_challenge,
                code_challenge_method,
                requested_scope,
                resource,
                expires_at,
                consumed_at,
                created_at
            """,
            (
                code_id,
                code_hash,
                client_id,
                user_id,
                redirect_uri,
                code_challenge,
                code_challenge_method,
                requested_scope,
                resource,
                expires_at,
            ),
        )
        row = cur.fetchone()
    return _row_to_auth_code(row)


def get_oauth_authorization_code_by_hash(*, code_hash: str) -> OAuthAuthorizationCodeRecord | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT
                code_id,
                code_hash,
                client_id,
                user_id,
                redirect_uri,
                code_challenge,
                code_challenge_method,
                requested_scope,
                resource,
                expires_at,
                consumed_at,
                created_at
            FROM oauth_authorization_codes
            WHERE code_hash = %s
            """,
            (code_hash,),
        )
        row = cur.fetchone()
    if not row:
        return None
    return _row_to_auth_code(row)


def consume_oauth_authorization_code(
    *,
    code_id: UUID,
    consumed_at: datetime,
) -> OAuthAuthorizationCodeRecord | None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            UPDATE oauth_authorization_codes
            SET consumed_at = %s
            WHERE code_id = %s
              AND consumed_at IS NULL
            RETURNING
                code_id,
                code_hash,
                client_id,
                user_id,
                redirect_uri,
                code_challenge,
                code_challenge_method,
                requested_scope,
                resource,
                expires_at,
                consumed_at,
                created_at
            """,
            (consumed_at, code_id),
        )
        row = cur.fetchone()
    if not row:
        return None
    return _row_to_auth_code(row)

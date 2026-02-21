from __future__ import annotations

from dataclasses import dataclass
from datetime import UTC, datetime
from uuid import UUID

from .db import get_conn
from .models import EngramSummary, PinnedDocumentRecord, PinnedEngramRecord


@dataclass(frozen=True)
class _PinnedResourceConfig:
    resource_table: str
    resource_access_clause: str


@dataclass(frozen=True)
class _PinnedResourceMutationRequest:
    table: str
    id_column: str
    resource_id: UUID
    session_id: UUID
    actor_user_id: UUID


@dataclass(frozen=True)
class _PinnedResourcePublicConfig:
    table: str
    id_column: str
    record_factory: type[PinnedEngramRecord] | type[PinnedDocumentRecord]


@dataclass(frozen=True)
class _PinnedResourceListConfig:
    id_column: str
    join_sql: str
    resource_access_clause: str
    include_resource_actor_param: bool = False


_PINNED_RESOURCE_CONFIG = {
    "session_pinned_engrams": _PinnedResourceConfig(
        resource_table="engrams",
        resource_access_clause="""
                    r.deleted_at IS NULL
                    AND (r.owner_user_id = %s OR r.visibility_scope = 'project' OR r.owner_user_id IS NULL)
        """,
    ),
    "session_pinned_documents": _PinnedResourceConfig(
        resource_table="documents",
        resource_access_clause="""
                    (r.owner_user_id = %s OR r.visibility_scope = 'project')
        """,
    ),
}

_PINNED_RESOURCE_LIST_CONFIG = {
    "session_pinned_engrams": _PinnedResourceListConfig(
        id_column="engram_id",
        join_sql="",
        resource_access_clause="1=1",
    ),
    "session_pinned_documents": _PinnedResourceListConfig(
        id_column="document_id",
        join_sql="""
            JOIN documents d
              ON d.document_id = p.document_id
        """,
        resource_access_clause="(d.owner_user_id = %s OR d.visibility_scope = 'project')",
        include_resource_actor_param=True,
    ),
}

_PINNED_RESOURCE_PUBLIC_CONFIG = {
    "engram": _PinnedResourcePublicConfig(
        table="session_pinned_engrams",
        id_column="engram_id",
        record_factory=PinnedEngramRecord,
    ),
    "document": _PinnedResourcePublicConfig(
        table="session_pinned_documents",
        id_column="document_id",
        record_factory=PinnedDocumentRecord,
    ),
}


def _pin_resource_to_session(
    *,
    request: _PinnedResourceMutationRequest,
    conn,
):
    config = _PINNED_RESOURCE_CONFIG[request.table]
    with conn.cursor() as cur:
        cur.execute(
            f"""
            WITH accessible_session AS (
                SELECT s.session_id
                FROM chat_sessions s
                WHERE
                    s.session_id = %s
                    AND s.deleted_at IS NULL
                    AND (s.owner_user_id = %s OR s.visibility_scope = 'project')
            ),
            accessible_resource AS (
                SELECT r.{request.id_column}
                FROM {config.resource_table} r
                WHERE
                    r.{request.id_column} = %s
                    AND {config.resource_access_clause}
            )
            INSERT INTO {request.table} (
                session_id,
                {request.id_column},
                pinned_by_user_id,
                created_at
            )
            SELECT
                s.session_id,
                r.{request.id_column},
                %s,
                %s
            FROM accessible_session s
            CROSS JOIN accessible_resource r
            ON CONFLICT (session_id, {request.id_column}) DO UPDATE
                SET pinned_by_user_id = EXCLUDED.pinned_by_user_id
            RETURNING
                session_id,
                {request.id_column},
                pinned_by_user_id,
                created_at
            """,
            (
                request.session_id,
                request.actor_user_id,
                request.resource_id,
                request.actor_user_id,
                request.actor_user_id,
                datetime.now(UTC),
            ),
        )
        return cur.fetchone()


def _unpin_resource_from_session(
    *,
    request: _PinnedResourceMutationRequest,
    conn,
) -> bool:
    with conn.cursor() as cur:
        cur.execute(
            f"""
            DELETE FROM {request.table} p
            USING chat_sessions s
            WHERE
                p.session_id = s.session_id
                AND p.session_id = %s
                AND p.{request.id_column} = %s
                AND s.deleted_at IS NULL
                AND (s.owner_user_id = %s OR s.visibility_scope = 'project')
            RETURNING p.session_id
            """,
            (request.session_id, request.resource_id, request.actor_user_id),
        )
        row = cur.fetchone()
    return row is not None


def _pin_to_session(*, request: _PinnedResourceMutationRequest):
    with get_conn() as conn:
        return _pin_resource_to_session(request=request, conn=conn)


def _unpin_from_session(*, request: _PinnedResourceMutationRequest) -> bool:
    with get_conn() as conn:
        return _unpin_resource_from_session(request=request, conn=conn)


def _build_pinned_resource_mutation_request(
    *,
    config: _PinnedResourcePublicConfig,
    resource_id: UUID,
    session_id: UUID,
    actor_user_id: UUID,
) -> _PinnedResourceMutationRequest:
    return _PinnedResourceMutationRequest(
        table=config.table,
        id_column=config.id_column,
        resource_id=resource_id,
        session_id=session_id,
        actor_user_id=actor_user_id,
    )


def _pin_resource_by_config(
    *,
    config: _PinnedResourcePublicConfig,
    resource_id: UUID,
    session_id: UUID,
    actor_user_id: UUID,
) -> PinnedEngramRecord | PinnedDocumentRecord | None:
    row = _pin_to_session(
        request=_build_pinned_resource_mutation_request(
            config=config,
            resource_id=resource_id,
            session_id=session_id,
            actor_user_id=actor_user_id,
        )
    )
    return config.record_factory(**row) if row else None


def _list_pinned_resources(
    *,
    table: str,
    session_id: UUID,
    actor_user_id: UUID,
) -> list[dict]:
    config = _PINNED_RESOURCE_LIST_CONFIG[table]
    params: list[object] = [session_id, actor_user_id]
    if config.include_resource_actor_param:
        params.append(actor_user_id)

    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            f"""
            SELECT
                p.session_id,
                p.{config.id_column},
                p.pinned_by_user_id,
                p.created_at
            FROM {table} p
            JOIN chat_sessions s
              ON s.session_id = p.session_id
            {config.join_sql}
            WHERE
                p.session_id = %s
                AND s.deleted_at IS NULL
                AND (s.owner_user_id = %s OR s.visibility_scope = 'project')
                AND {config.resource_access_clause}
            ORDER BY p.created_at ASC
            """,
            params,
        )
        return cur.fetchall()


def pin_resource_to_session(
    resource_kind: str,
    session_id: UUID,
    resource_id: UUID,
    actor_user_id: UUID,
) -> PinnedEngramRecord | PinnedDocumentRecord | None:
    config = _PINNED_RESOURCE_PUBLIC_CONFIG[resource_kind]
    return _pin_resource_by_config(
        config=config,
        resource_id=resource_id,
        session_id=session_id,
        actor_user_id=actor_user_id,
    )


def unpin_resource_from_session(
    resource_kind: str,
    session_id: UUID,
    resource_id: UUID,
    actor_user_id: UUID,
) -> bool:
    config = _PINNED_RESOURCE_PUBLIC_CONFIG[resource_kind]
    return _unpin_from_session(
        request=_build_pinned_resource_mutation_request(
            config=config,
            resource_id=resource_id,
            session_id=session_id,
            actor_user_id=actor_user_id,
        )
    )


def list_pinned_engrams(session_id: UUID, actor_user_id: UUID) -> list[PinnedEngramRecord]:
    rows = _list_pinned_resources(
        table="session_pinned_engrams",
        session_id=session_id,
        actor_user_id=actor_user_id,
    )
    return [PinnedEngramRecord(**row) for row in rows]


def list_pinned_engram_summaries(
    session_id: UUID,
    actor_user_id: UUID,
) -> list[EngramSummary]:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            SELECT
                e.engram_id,
                e.project_id,
                e.thread_id,
                e.title,
                e.abstract,
                e.created_at,
                e.tags,
                e.keywords,
                e.owner_user_id,
                e.visibility_scope
            FROM session_pinned_engrams p
            JOIN chat_sessions s
              ON s.session_id = p.session_id
            JOIN engrams e
              ON e.engram_id = p.engram_id
            WHERE
                p.session_id = %s
                AND s.deleted_at IS NULL
                AND (s.owner_user_id = %s OR s.visibility_scope = 'project')
                AND e.deleted_at IS NULL
                AND (e.owner_user_id = %s OR e.visibility_scope = 'project' OR e.owner_user_id IS NULL)
            ORDER BY p.created_at ASC
            """,
            (session_id, actor_user_id, actor_user_id),
        )
        rows = cur.fetchall()
    return [EngramSummary(**row) for row in rows]


def list_pinned_documents(session_id: UUID, actor_user_id: UUID) -> list[PinnedDocumentRecord]:
    rows = _list_pinned_resources(
        table="session_pinned_documents",
        session_id=session_id,
        actor_user_id=actor_user_id,
    )
    return [PinnedDocumentRecord(**row) for row in rows]

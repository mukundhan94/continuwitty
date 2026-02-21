import re
from uuid import UUID

import pytest

from app.config import get_settings


def _extract_csrf_token(html: str) -> str:
    match = re.search(r'name="csrf_token" value="([^"]+)"', html)
    assert match is not None
    return match.group(1)


def _login(client) -> None:  # noqa: ANN001
    settings = get_settings()
    login_page = client.get("/login")
    csrf_token = _extract_csrf_token(login_page.text)
    response = client.post(
        "/login",
        data={
            "username": settings.ui_demo_username,
            "password": settings.ui_demo_password,
            "csrf_token": csrf_token,
        },
        follow_redirects=False,
    )
    assert response.status_code == 303


@pytest.mark.integration
def test_roundtrip_create_list_query_rehydrate(client, clean_db) -> None:
    _login(client)
    create_payload = {
        "project_id": "project-a",
        "thread_id": "run-1",
        "title": "LangGraph Memory Choice",
        "abstract": "LangGraph selected for durable checkpointing.",
        "detailed_summary_markdown": "Detailed summary text.",
        "decisions": [{"decision": "Use LangGraph", "rationale": "Durable runs"}],
        "assumptions": ["Local single tenant"],
        "open_questions": ["When to introduce reranking?"],
        "claims": [
            {
                "claim": "Hybrid retrieval improves robustness",
                "supporting_sources": [
                    {
                        "url": "https://example.com/hybrid",
                        "title": "Hybrid Search",
                        "snippet": "Dense + sparse improves retrieval stability.",
                        "captured_at": "2026-02-15T10:00:00Z",
                    }
                ],
            }
        ],
        "tags": ["memory", "rag"],
        "keywords": ["langgraph", "pgvector"],
        "artifacts": [
            {
                "artifact_type": "markdown",
                "storage_uri": "local://notes/run-1.md",
                "metadata": {"origin": "integration-test"},
            }
        ],
    }

    create_response = client.post("/api/v1/engrams", json=create_payload)
    assert create_response.status_code == 200
    engram_id = create_response.json()["engram_id"]

    list_response = client.get("/api/v1/engrams", params={"project_id": "project-a"})
    assert list_response.status_code == 200
    listed = list_response.json()
    assert len(listed) == 1
    assert listed[0]["engram_id"] == engram_id

    query_response = client.post(
        "/api/v1/engrams/query",
        json={
            "query": "What was chosen for durable runs?",
            "project_id": "project-a",
            "top_k": 5,
            "tags": ["memory"],
        },
    )
    assert query_response.status_code == 200
    queried = query_response.json()
    assert len(queried) >= 1
    assert any(item["engram_id"] == engram_id for item in queried)

    rehydrate_response = client.get(f"/api/v1/engrams/{engram_id}/rehydrate")
    assert rehydrate_response.status_code == 200
    body = rehydrate_response.json()
    assert body["engram_id"] == engram_id
    assert "LangGraph selected for durable checkpointing" in body["compact_summary"]
    assert len(body["top_citations"]) == 1
    assert "Rehydration Context" in body["context_markdown"]


@pytest.mark.integration
def test_create_engram_auto_enriches_empty_metadata(client, clean_db, db_conn) -> None:
    _login(client)
    create_payload = {
        "project_id": "project-auto-meta",
        "thread_id": "conversation-1",
        "title": "P1 outage continuity handoff",
        "detailed_summary_markdown": (
            "## USER\nSummarize incident status.\n\n"
            "## ASSISTANT\nIncident triage confirms outage blast radius in checkout. "
            "Rollback and queue drain mitigated impact while support prepared updates."
        ),
        "abstract": "",
        "tags": [],
        "keywords": [],
    }

    create_response = client.post("/api/v1/engrams", json=create_payload)
    assert create_response.status_code == 200
    engram_id = UUID(create_response.json()["engram_id"])

    list_response = client.get("/api/v1/engrams", params={"project_id": "project-auto-meta"})
    assert list_response.status_code == 200
    created = next(item for item in list_response.json() if item["engram_id"] == str(engram_id))
    assert created["abstract"].strip() != ""
    assert len(created["tags"]) > 0
    assert len(created["keywords"]) > 0

    with db_conn.cursor() as cur:
        cur.execute(
            """
            SELECT abstract, tags, keywords, engram_json
            FROM engrams
            WHERE engram_id = %s
            """,
            (engram_id,),
        )
        row = cur.fetchone()

    assert row is not None
    row_abstract, row_tags, row_keywords, row_engram_json = row
    auto_metadata = row_engram_json["auto_metadata"]
    assert auto_metadata["enrichment_applied"] is True
    assert auto_metadata["abstract_derived"] is True
    assert auto_metadata["tags_derived"] is True
    assert auto_metadata["keywords_derived"] is True
    assert row_tags == auto_metadata["auto_tags"]
    assert row_keywords == auto_metadata["auto_keywords"]
    assert row_abstract.strip() != ""


@pytest.mark.integration
def test_create_engram_preserves_explicit_metadata(client, clean_db, db_conn) -> None:
    _login(client)
    create_payload = {
        "project_id": "project-explicit-meta",
        "thread_id": "conversation-2",
        "title": "Release readiness review",
        "abstract": "Manual abstract from operator.",
        "detailed_summary_markdown": "Operator-provided summary markdown.",
        "tags": ["manual-tag"],
        "keywords": ["manual-keyword"],
    }

    create_response = client.post("/api/v1/engrams", json=create_payload)
    assert create_response.status_code == 200
    engram_id = UUID(create_response.json()["engram_id"])

    with db_conn.cursor() as cur:
        cur.execute(
            """
            SELECT abstract, tags, keywords, engram_json
            FROM engrams
            WHERE engram_id = %s
            """,
            (engram_id,),
        )
        row = cur.fetchone()

    assert row is not None
    row_abstract, row_tags, row_keywords, row_engram_json = row
    assert row_abstract == "Manual abstract from operator."
    assert row_tags == ["manual-tag"]
    assert row_keywords == ["manual-keyword"]
    auto_metadata = row_engram_json["auto_metadata"]
    assert auto_metadata["enrichment_applied"] is False
    assert auto_metadata["abstract_derived"] is False
    assert auto_metadata["tags_derived"] is False
    assert auto_metadata["keywords_derived"] is False


@pytest.mark.integration
def test_query_metadata_filters(client, clean_db) -> None:
    _login(client)
    payload_a = {
        "project_id": "project-a",
        "title": "A",
        "abstract": "Memory architecture A",
        "detailed_summary_markdown": "A",
        "tags": ["alpha"],
        "keywords": ["graph"],
    }
    payload_b = {
        "project_id": "project-b",
        "title": "B",
        "abstract": "Memory architecture B",
        "detailed_summary_markdown": "B",
        "tags": ["beta"],
        "keywords": ["vector"],
    }

    response_a = client.post("/api/v1/engrams", json=payload_a)
    response_b = client.post("/api/v1/engrams", json=payload_b)
    assert response_a.status_code == 200
    assert response_b.status_code == 200

    filtered = client.post(
        "/api/v1/engrams/query",
        json={
            "query": "architecture",
            "project_id": "project-a",
            "tags": ["alpha"],
            "keywords": ["graph"],
            "top_k": 5,
        },
    )
    assert filtered.status_code == 200
    items = filtered.json()
    assert len(items) >= 1
    assert all(item["project_id"] == "project-a" for item in items)


@pytest.mark.integration
def test_sources_endpoint_returns_provenance_records(client, clean_db) -> None:
    _login(client)
    payload = {
        "project_id": "project-sources",
        "thread_id": "run-sources",
        "title": "Source test",
        "abstract": "Source endpoint test",
        "detailed_summary_markdown": "summary",
        "claims": [
            {
                "claim": "Claim with source",
                "supporting_sources": [
                    {
                        "url": "https://example.com/source",
                        "title": "Source",
                        "snippet": "Provenance snippet",
                        "captured_at": "2026-02-15T10:00:00Z",
                    }
                ],
            }
        ],
    }
    created = client.post("/api/v1/engrams", json=payload)
    assert created.status_code == 200
    engram_id = created.json()["engram_id"]

    sources_response = client.get(f"/api/v1/engrams/{engram_id}/sources")
    assert sources_response.status_code == 200
    records = sources_response.json()
    assert len(records) == 1
    assert records[0]["engram_id"] == engram_id
    assert records[0]["url"] == "https://example.com/source"


@pytest.mark.integration
def test_sources_endpoint_returns_404_for_missing_engram(client, clean_db) -> None:
    _login(client)
    response = client.get("/api/v1/engrams/00000000-0000-0000-0000-000000000000/sources")
    assert response.status_code == 404


@pytest.mark.integration
def test_rehydrate_packs_unique_citations(client, clean_db) -> None:
    _login(client)
    payload = {
        "project_id": "project-citations",
        "title": "Citation packing",
        "abstract": "Ensure duplicate source URLs are packed once.",
        "detailed_summary_markdown": "summary",
        "claims": [
            {
                "claim": "Primary claim",
                "supporting_sources": [
                    {
                        "url": "https://example.com/source-a",
                        "title": "Source A (first)",
                        "snippet": "First snippet",
                        "captured_at": "2026-02-15T10:00:00Z",
                    },
                    {
                        "url": "https://example.com/source-a",
                        "title": "Source A (duplicate)",
                        "snippet": "Duplicate snippet",
                        "captured_at": "2026-02-15T10:01:00Z",
                    },
                    {
                        "url": "https://example.com/source-b",
                        "title": "Source B",
                        "snippet": "Second unique source",
                        "captured_at": "2026-02-15T10:02:00Z",
                    },
                ],
            }
        ],
    }
    created = client.post("/api/v1/engrams", json=payload)
    assert created.status_code == 200
    engram_id = created.json()["engram_id"]

    rehydrate = client.get(f"/api/v1/engrams/{engram_id}/rehydrate")
    assert rehydrate.status_code == 200
    body = rehydrate.json()

    assert len(body["top_citations"]) == 2
    urls = [item["url"] for item in body["top_citations"]]
    assert urls == ["https://example.com/source-b", "https://example.com/source-a"]


@pytest.mark.integration
def test_rehydrate_uses_detailed_summary_when_abstract_is_generic(client, clean_db) -> None:
    _login(client)
    payload = {
        "project_id": "project-chat-save",
        "title": "Chat snapshot",
        "abstract": "Snapshot from active chat session.",
        "detailed_summary_markdown": (
            "# Chat Session Snapshot: Incident\n\n"
            "## ASSISTANT (2026-02-15T16:00:12Z)\n"
            "Primary cause was DB CPU saturation from an unbounded query; "
            "rate limiting mitigated user impact while rollback had no effect."
        ),
    }
    created = client.post("/api/v1/engrams", json=payload)
    assert created.status_code == 200
    engram_id = created.json()["engram_id"]

    rehydrate = client.get(f"/api/v1/engrams/{engram_id}/rehydrate")
    assert rehydrate.status_code == 200
    body = rehydrate.json()

    assert "Primary cause was DB CPU saturation" in body["compact_summary"]
    assert "Detailed Notes Excerpt" in body["context_markdown"]

from __future__ import annotations

import re

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
def test_ingestion_text_and_query_flow(client, clean_db) -> None:
    _login(client)

    ingest = client.post(
        "/api/v1/ingestion/text",
        json={
            "project_id": "engram-vault",
            "title": "Incident Timeline",
            "text": " ".join(["latency spike"] * 300),
            "visibility_scope": "project",
            "chunk_size_chars": 360,
            "chunk_overlap_chars": 120,
            "metadata": {"domain": "incident"},
        },
    )
    assert ingest.status_code == 201
    doc = ingest.json()["document"]
    assert doc["project_id"] == "engram-vault"
    assert doc["chunk_count"] >= 1

    listed = client.get("/api/v1/ingestion/documents", params={"project_id": "engram-vault"})
    assert listed.status_code == 200
    assert len(listed.json()) == 1

    queried = client.post(
        "/api/v1/ingestion/query",
        json={
            "query": "Where did latency spike happen?",
            "project_id": "engram-vault",
            "top_k": 5,
        },
    )
    assert queried.status_code == 200
    chunk_results = queried.json()
    assert len(chunk_results) >= 1
    assert chunk_results[0]["document_id"] == doc["document_id"]


@pytest.mark.integration
def test_ingestion_file_and_blended_query_flow(client, clean_db) -> None:
    _login(client)

    create_engram = client.post(
        "/api/v1/engrams",
        json={
            "project_id": "engram-vault",
            "title": "Runbook baseline",
            "abstract": "Runbook summary",
            "detailed_summary_markdown": "Escalate after 10 minutes if queue depth grows.",
            "tags": ["runbook"],
        },
    )
    assert create_engram.status_code == 200

    upload = client.post(
        "/api/v1/ingestion/file",
        data={
            "project_id": "engram-vault",
            "title": "On-call Notes",
            "visibility_scope": "project",
            "chunk_size_chars": "340",
            "chunk_overlap_chars": "90",
        },
        files={
            "file": (
                "notes.md",
                "# On-call\n\nQueue depth exceeded threshold for 20 minutes.",
                "text/markdown",
            )
        },
    )
    assert upload.status_code == 201

    blended = client.post(
        "/api/v1/ingestion/query/blended",
        json={
            "query": "What should on-call do when queue depth grows?",
            "project_id": "engram-vault",
            "top_k_engrams": 3,
            "top_k_document_chunks": 3,
        },
    )
    assert blended.status_code == 200
    body = blended.json()
    assert len(body["engrams"]) >= 1
    assert len(body["document_chunks"]) >= 1

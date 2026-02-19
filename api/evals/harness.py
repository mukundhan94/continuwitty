from __future__ import annotations

import json
import os
import time
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any

import psycopg
from fastapi.testclient import TestClient

from app.config import get_settings
from app.main import app

DEFAULT_DB_URL = "postgresql://engram:engram@localhost:5432/engram_vault"


@dataclass
class EvalCaseResult:
    name: str
    passed: bool
    details: dict[str, Any]


def _db_url() -> str:
    return os.getenv("DATABASE_URL", DEFAULT_DB_URL)


def _bootstrap_clean_database() -> None:
    schema_path = Path(__file__).resolve().parents[2] / "db" / "init" / "001_schema.sql"
    schema_sql = schema_path.read_text(encoding="utf-8")

    with psycopg.connect(_db_url()) as conn:
        with conn.cursor() as cur:
            cur.execute(schema_sql)
            cur.execute("TRUNCATE TABLE engrams CASCADE")
        conn.commit()


def _create_engram(client: TestClient, payload: dict[str, Any]) -> str:
    response = client.post("/api/v1/engrams", json=payload)
    if response.status_code != 200:
        raise RuntimeError(f"create engram failed: {response.status_code} {response.text}")
    return response.json()["engram_id"]


def _login(client: TestClient) -> None:
    settings = get_settings()
    login_page = client.get("/login")
    csrf_token = login_page.text.split('name="csrf_token" value="', 1)[1].split('"', 1)[0]
    response = client.post(
        "/login",
        data={
            "username": settings.ui_demo_username,
            "password": settings.ui_demo_password,
            "csrf_token": csrf_token,
        },
        follow_redirects=False,
    )
    if response.status_code != 303:
        raise RuntimeError(f"eval login failed: {response.status_code} {response.text}")


def _fact_recall_case(client: TestClient) -> EvalCaseResult:
    project_id = "eval-fact"
    engram_id = _create_engram(
        client,
        {
            "project_id": project_id,
            "thread_id": "fact-1",
            "title": "Durability Framework Selection",
            "abstract": "LangGraph was selected for durable checkpointed runs.",
            "detailed_summary_markdown": "Selection summary",
            "tags": ["fact", "framework"],
            "keywords": ["langgraph", "durability"],
        },
    )

    query = client.post(
        "/api/v1/engrams/query",
        json={
            "query": "Which framework was selected for durable runs?",
            "project_id": project_id,
            "tags": ["fact"],
            "top_k": 5,
        },
    )
    body = query.json()
    hit_ids = {item["engram_id"] for item in body}
    passed = engram_id in hit_ids
    return EvalCaseResult(
        name="fact_recall",
        passed=passed,
        details={"expected_engram_id": engram_id, "returned_ids": sorted(hit_ids)},
    )


def _cross_engram_case(client: TestClient) -> EvalCaseResult:
    project_id = "eval-cross"
    first_id = _create_engram(
        client,
        {
            "project_id": project_id,
            "thread_id": "cross-1",
            "title": "Framework Decision",
            "abstract": "LangGraph chosen for orchestration and resume handling.",
            "detailed_summary_markdown": "Framework details",
            "tags": ["cross", "stack"],
            "keywords": ["langgraph"],
        },
    )
    second_id = _create_engram(
        client,
        {
            "project_id": project_id,
            "thread_id": "cross-2",
            "title": "Storage Decision",
            "abstract": "Postgres with pgvector chosen for metadata and embeddings.",
            "detailed_summary_markdown": "Storage details",
            "tags": ["cross", "stack"],
            "keywords": ["postgres", "pgvector"],
        },
    )

    query = client.post(
        "/api/v1/engrams/query",
        json={
            "query": "What stack combines workflow and storage choices?",
            "project_id": project_id,
            "tags": ["cross"],
            "top_k": 10,
        },
    )
    body = query.json()
    hit_ids = {item["engram_id"] for item in body}
    expected = {first_id, second_id}
    passed = expected.issubset(hit_ids)
    return EvalCaseResult(
        name="cross_engram_reasoning_proxy",
        passed=passed,
        details={"expected_ids": sorted(expected), "returned_ids": sorted(hit_ids)},
    )


def _temporal_case(client: TestClient) -> EvalCaseResult:
    project_id = "eval-temporal"
    old_id = _create_engram(
        client,
        {
            "project_id": project_id,
            "thread_id": "temp-1",
            "title": "Retrieval Policy v1",
            "abstract": "Use dense-only retrieval.",
            "detailed_summary_markdown": "Old policy",
            "tags": ["temporal"],
            "keywords": ["dense"],
        },
    )
    time.sleep(0.05)
    new_id = _create_engram(
        client,
        {
            "project_id": project_id,
            "thread_id": "temp-2",
            "title": "Retrieval Policy v2",
            "abstract": "Use hybrid dense+sparse retrieval.",
            "detailed_summary_markdown": "New policy",
            "tags": ["temporal"],
            "keywords": ["hybrid"],
        },
    )

    listing = client.get("/api/v1/engrams", params={"project_id": project_id, "limit": 10})
    rows = listing.json()
    first_id = rows[0]["engram_id"] if rows else None
    passed = first_id == new_id and any(row["engram_id"] == old_id for row in rows)
    return EvalCaseResult(
        name="temporal_correctness",
        passed=passed,
        details={"expected_latest_id": new_id, "first_returned_id": first_id, "count": len(rows)},
    )


def _abstention_case(client: TestClient) -> EvalCaseResult:
    query = client.post(
        "/api/v1/engrams/query",
        json={
            "query": "show a conclusion that was never captured",
            "project_id": "eval-empty-project",
            "top_k": 5,
        },
    )
    body = query.json()
    passed = body == []
    return EvalCaseResult(
        name="abstention_empty_result",
        passed=passed,
        details={"result_count": len(body)},
    )


def run_evaluations() -> dict[str, Any]:
    _bootstrap_clean_database()
    client = TestClient(app)
    _login(client)

    cases = [
        _fact_recall_case(client),
        _cross_engram_case(client),
        _temporal_case(client),
        _abstention_case(client),
    ]

    passed_count = sum(1 for case in cases if case.passed)
    summary = {
        "passed": passed_count == len(cases),
        "score": passed_count / len(cases),
        "passed_cases": passed_count,
        "total_cases": len(cases),
        "cases": [asdict(case) for case in cases],
    }
    return summary


def run_evaluations_json() -> str:
    return json.dumps(run_evaluations(), indent=2)

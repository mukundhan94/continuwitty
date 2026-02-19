from datetime import UTC, datetime
from uuid import uuid4

from fastapi.testclient import TestClient

from app.main import app


def test_healthz() -> None:
    client = TestClient(app)
    response = client.get("/healthz")
    assert response.status_code == 200
    assert response.json() == {"status": "ok"}


def test_list_engrams_uses_repository(monkeypatch) -> None:
    now = datetime.now(UTC)
    engram_id = uuid4()

    def fake_list_engrams(project_id: str | None, limit: int, offset: int):
        assert project_id == "proj-1"
        assert limit == 10
        assert offset == 2
        return [
            {
                "engram_id": engram_id,
                "project_id": "proj-1",
                "thread_id": "t1",
                "title": "Title",
                "abstract": "Abstract",
                "created_at": now,
                "tags": ["tag"],
                "keywords": ["kw"],
            }
        ]

    monkeypatch.setattr("app.main.list_engrams", fake_list_engrams)

    client = TestClient(app)
    response = client.get(
        "/api/v1/engrams", params={"project_id": "proj-1", "limit": 10, "offset": 2}
    )
    assert response.status_code == 200
    body = response.json()
    assert len(body) == 1
    assert body[0]["engram_id"] == str(engram_id)


def test_create_engram_uses_repository(monkeypatch) -> None:
    now = datetime.now(UTC)
    engram_id = uuid4()

    class FakeSettings:
        embedding_dim = 256

    def fake_get_settings():
        return FakeSettings()

    def fake_create_engram(payload, embedding_dim: int, enrichment_origin: str = "unknown"):
        assert embedding_dim == 256
        assert payload.project_id == "proj-1"
        assert enrichment_origin == "api.engrams.create"
        return {"engram_id": engram_id, "created_at": now}

    monkeypatch.setattr("app.main.get_settings", fake_get_settings)
    monkeypatch.setattr("app.main.create_engram", fake_create_engram)

    client = TestClient(app)
    response = client.post(
        "/api/v1/engrams",
        json={
            "project_id": "proj-1",
            "title": "t",
            "abstract": "a",
            "detailed_summary_markdown": "d",
        },
    )

    assert response.status_code == 200
    body = response.json()
    assert body["engram_id"] == str(engram_id)


def test_query_engrams_uses_repository(monkeypatch) -> None:
    now = datetime.now(UTC)
    engram_id = uuid4()

    class FakeSettings:
        embedding_dim = 256

    def fake_get_settings():
        return FakeSettings()

    def fake_query_engrams(payload, embedding_dim: int):
        assert embedding_dim == 256
        assert payload.query == "durable memory"
        return [
            {
                "engram_id": engram_id,
                "project_id": "proj-1",
                "title": "LangGraph decision",
                "abstract": "Used for checkpoints",
                "created_at": now,
                "tags": ["memory"],
                "keywords": ["langgraph"],
                "distance": 0.1,
            }
        ]

    monkeypatch.setattr("app.main.get_settings", fake_get_settings)
    monkeypatch.setattr("app.main.query_engrams", fake_query_engrams)

    client = TestClient(app)
    response = client.post(
        "/api/v1/engrams/query",
        json={"query": "durable memory", "top_k": 5},
    )

    assert response.status_code == 200
    body = response.json()
    assert len(body) == 1
    assert body[0]["engram_id"] == str(engram_id)


def test_rehydrate_404(monkeypatch) -> None:
    monkeypatch.setattr("app.main.get_rehydration_bundle", lambda _engram_id: None)

    client = TestClient(app)
    response = client.get(f"/api/v1/engrams/{uuid4()}/rehydrate")

    assert response.status_code == 404
    assert response.json()["detail"] == "Engram not found"

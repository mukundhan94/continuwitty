from datetime import UTC, datetime
from types import SimpleNamespace
from uuid import uuid4

from fastapi.testclient import TestClient

from app.main import app
from app.models import EngramCreateResponse


def _fake_authenticated_user() -> dict[str, str]:
    return {
        "user_id": "00000000-0000-0000-0000-000000000001",
        "username": "admin",
        "role": "admin",
    }


def test_healthz() -> None:
    client = TestClient(app)
    response = client.get("/healthz")
    assert response.status_code == 200
    assert response.json() == {"status": "ok"}


def test_version_endpoint() -> None:
    client = TestClient(app)
    response = client.get("/api/v1/version")
    assert response.status_code == 200
    payload = response.json()
    assert payload["semantic_version"] == app.version
    assert payload["release"] == f"v{app.version}"
    assert isinstance(payload["commit_id"], str)
    assert payload["commit_id"]


def test_list_engrams_uses_repository(monkeypatch) -> None:
    now = datetime.now(UTC)
    engram_id = uuid4()

    def fake_list_engrams(
        project_id: str | None,
        limit: int,
        offset: int,
        actor_user_id=None,  # noqa: ANN001
    ):
        assert project_id == "proj-1"
        assert limit == 10
        assert offset == 2
        assert actor_user_id is not None
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

    monkeypatch.setattr(
        "app.main._require_authenticated_api_user", lambda _request: _fake_authenticated_user()
    )
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

    def fake_create_engram(
        payload,  # noqa: ANN001
        embedding_dim: int,
        owner_user_id=None,  # noqa: ANN001
        enrichment_origin: str = "unknown",
    ):
        assert embedding_dim == 256
        assert payload.project_id == "proj-1"
        assert owner_user_id is not None
        assert enrichment_origin == "api.engrams.create"
        return EngramCreateResponse(engram_id=engram_id, created_at=now)

    monkeypatch.setattr(
        "app.main._require_authenticated_api_user", lambda _request: _fake_authenticated_user()
    )
    monkeypatch.setattr("app.main.get_settings", fake_get_settings)
    monkeypatch.setattr(
        "app.main.project_service.resolve_project_id_for_write",
        lambda actor_user_id, actor_role, project_id: SimpleNamespace(
            project_id=project_id or "proj-1",
            used_default_project=False,
        ),
    )
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

    def fake_query_engrams(payload, embedding_dim: int, actor_user_id=None):  # noqa: ANN001
        assert embedding_dim == 256
        assert payload.query == "durable memory"
        assert actor_user_id is not None
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

    monkeypatch.setattr(
        "app.main._require_authenticated_api_user", lambda _request: _fake_authenticated_user()
    )
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
    monkeypatch.setattr(
        "app.main._require_authenticated_api_user", lambda _request: _fake_authenticated_user()
    )
    monkeypatch.setattr(
        "app.main.get_rehydration_bundle", lambda _engram_id, actor_user_id=None: None
    )

    client = TestClient(app)
    response = client.get(f"/api/v1/engrams/{uuid4()}/rehydrate")

    assert response.status_code == 404
    assert response.json()["detail"] == "Engram not found"

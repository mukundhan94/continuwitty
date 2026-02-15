from fastapi.testclient import TestClient

from app.config import get_settings
from app.main import app


def test_home_redirects_to_login_when_not_authenticated() -> None:
    client = TestClient(app)
    response = client.get("/", follow_redirects=False)

    assert response.status_code == 303
    assert response.headers["location"] == "/login"


def test_dashboard_redirects_to_login_when_not_authenticated() -> None:
    client = TestClient(app)
    response = client.get("/ui", follow_redirects=False)

    assert response.status_code == 303
    assert response.headers["location"] == "/login"


def test_login_rejects_invalid_credentials() -> None:
    client = TestClient(app)
    response = client.post(
        "/login",
        data={"username": "wrong", "password": "wrong"},
        follow_redirects=False,
    )

    assert response.status_code == 401
    assert "Invalid username or password." in response.text


def test_login_and_logout_workflow() -> None:
    client = TestClient(app)
    settings = get_settings()

    login_response = client.post(
        "/login",
        data={"username": settings.ui_demo_username, "password": settings.ui_demo_password},
        follow_redirects=False,
    )
    assert login_response.status_code == 303
    assert login_response.headers["location"] == "/ui"

    dashboard = client.get("/ui")
    assert dashboard.status_code == 200
    assert "Engram Vault Test UI" in dashboard.text
    assert settings.ui_demo_username in dashboard.text

    logout_response = client.post("/logout", follow_redirects=False)
    assert logout_response.status_code == 303
    assert logout_response.headers["location"] == "/login"

    blocked_dashboard = client.get("/ui", follow_redirects=False)
    assert blocked_dashboard.status_code == 303
    assert blocked_dashboard.headers["location"] == "/login"

import re

from fastapi.testclient import TestClient

from app.config import get_settings
from app.main import app


def _extract_csrf_token(html: str) -> str:
    match = re.search(r'name="csrf_token" value="([^"]+)"', html)
    assert match is not None
    return match.group(1)


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
    login_page = client.get("/login")
    csrf_token = _extract_csrf_token(login_page.text)

    response = client.post(
        "/login",
        data={"username": "wrong", "password": "wrong", "csrf_token": csrf_token},
        follow_redirects=False,
    )

    assert response.status_code == 401
    assert "Invalid username or password." in response.text


def test_login_rejects_invalid_csrf() -> None:
    client = TestClient(app)
    _ = client.get("/login")

    response = client.post(
        "/login",
        data={"username": "admin", "password": "admin123", "csrf_token": "bad-token"},
        follow_redirects=False,
    )

    assert response.status_code == 403


def test_login_and_logout_workflow() -> None:
    client = TestClient(app)
    settings = get_settings()

    login_page = client.get("/login")
    csrf_token = _extract_csrf_token(login_page.text)

    login_response = client.post(
        "/login",
        data={
            "username": settings.ui_demo_username,
            "password": settings.ui_demo_password,
            "csrf_token": csrf_token,
        },
        follow_redirects=False,
    )
    assert login_response.status_code == 303
    assert login_response.headers["location"] == "/ui"

    dashboard = client.get("/ui")
    assert dashboard.status_code == 200
    assert "Engram Vault Test UI" in dashboard.text
    assert settings.ui_demo_username in dashboard.text

    dashboard_csrf_token = _extract_csrf_token(dashboard.text)
    logout_response = client.post(
        "/logout",
        data={"csrf_token": dashboard_csrf_token},
        follow_redirects=False,
    )
    assert logout_response.status_code == 303
    assert logout_response.headers["location"] == "/login"

    blocked_dashboard = client.get("/ui", follow_redirects=False)
    assert blocked_dashboard.status_code == 303
    assert blocked_dashboard.headers["location"] == "/login"


def test_logout_rejects_invalid_csrf() -> None:
    client = TestClient(app)
    settings = get_settings()

    login_page = client.get("/login")
    csrf_token = _extract_csrf_token(login_page.text)
    login_response = client.post(
        "/login",
        data={
            "username": settings.ui_demo_username,
            "password": settings.ui_demo_password,
            "csrf_token": csrf_token,
        },
    )
    assert login_response.status_code == 200

    bad_logout = client.post(
        "/logout",
        data={"csrf_token": "bad-token"},
        follow_redirects=False,
    )
    assert bad_logout.status_code == 403

    still_authenticated = client.get("/ui")
    assert still_authenticated.status_code == 200

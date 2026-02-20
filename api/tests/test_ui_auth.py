import re

import pytest
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


def test_login_rate_limit_after_repeated_failures() -> None:
    client = TestClient(app)

    for _ in range(5):
        login_page = client.get("/login")
        csrf_token = _extract_csrf_token(login_page.text)
        response = client.post(
            "/login",
            data={"username": "admin", "password": "wrong", "csrf_token": csrf_token},
            follow_redirects=False,
        )
        assert response.status_code == 401

    login_page = client.get("/login")
    csrf_token = _extract_csrf_token(login_page.text)
    blocked = client.post(
        "/login",
        data={"username": "admin", "password": "wrong", "csrf_token": csrf_token},
        follow_redirects=False,
    )
    assert blocked.status_code == 429
    assert "Too many login attempts" in blocked.json()["detail"]


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


@pytest.mark.parametrize(
    ("next_path", "expected_location"),
    [
        ("/ui/admin", "/ui/admin"),
        ("https://malicious.example/phish", "/ui"),
    ],
)
def test_login_redirect_path_sanitization(next_path: str, expected_location: str) -> None:
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
            "next_path": next_path,
        },
        follow_redirects=False,
    )

    assert login_response.status_code == 303
    assert login_response.headers["location"] == expected_location


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


def test_login_failure_writes_audit_log(monkeypatch, tmp_path) -> None:
    audit_path = tmp_path / "audit.log"
    monkeypatch.setenv("AUDIT_LOG_PATH", str(audit_path))

    client = TestClient(app)
    login_page = client.get("/login")
    csrf_token = _extract_csrf_token(login_page.text)
    response = client.post(
        "/login",
        data={"username": "admin", "password": "wrong", "csrf_token": csrf_token},
        follow_redirects=False,
    )

    assert response.status_code == 401
    assert audit_path.exists()
    lines = [line for line in audit_path.read_text(encoding="utf-8").splitlines() if line.strip()]
    assert any('"event_type": "login_failed"' in line for line in lines)

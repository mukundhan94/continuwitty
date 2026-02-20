from __future__ import annotations

from datetime import UTC, datetime, timedelta

import pytest

from app import login_guard


class _FrozenDateTime(datetime):
    current = datetime(2026, 1, 1, 12, 0, tzinfo=UTC)

    @classmethod
    def now(cls, tz=None):  # type: ignore[override]
        if tz is None:
            return cls.current.replace(tzinfo=None)
        return cls.current.astimezone(tz)


@pytest.fixture
def frozen_clock(monkeypatch):
    _FrozenDateTime.current = datetime(2026, 1, 1, 12, 0, tzinfo=UTC)
    monkeypatch.setattr(login_guard, "datetime", _FrozenDateTime)
    return _FrozenDateTime


def test_register_failure_locks_after_max_attempts(frozen_clock) -> None:
    guard = login_guard.LoginAttemptGuard(max_attempts=2, window_seconds=60, lockout_seconds=30)
    key = "admin@example"

    guard.register_failure(key)
    assert guard.check(key) == (True, 0)

    frozen_clock.current += timedelta(seconds=1)
    guard.register_failure(key)
    allowed, seconds = guard.check(key)
    assert allowed is False
    assert 1 <= seconds <= 30


def test_check_unlocks_after_lockout_expiry(frozen_clock) -> None:
    guard = login_guard.LoginAttemptGuard(max_attempts=1, window_seconds=60, lockout_seconds=10)
    key = "admin@example"

    guard.register_failure(key)
    assert guard.check(key)[0] is False

    frozen_clock.current += timedelta(seconds=11)
    assert guard.check(key) == (True, 0)


def test_failure_window_drops_stale_attempts(frozen_clock) -> None:
    guard = login_guard.LoginAttemptGuard(max_attempts=2, window_seconds=5, lockout_seconds=20)
    key = "admin@example"

    guard.register_failure(key)
    frozen_clock.current += timedelta(seconds=6)
    guard.register_failure(key)
    assert guard.check(key) == (True, 0)

    guard.register_failure(key)
    allowed, seconds = guard.check(key)
    assert allowed is False
    assert 1 <= seconds <= 20


def test_register_success_clears_prior_failures(frozen_clock) -> None:
    guard = login_guard.LoginAttemptGuard(max_attempts=2, window_seconds=60, lockout_seconds=30)
    key = "admin@example"

    guard.register_failure(key)
    guard.register_success(key)
    assert guard.check(key) == (True, 0)

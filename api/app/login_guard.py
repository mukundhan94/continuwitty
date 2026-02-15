from __future__ import annotations

from dataclasses import dataclass, field
from datetime import UTC, datetime, timedelta


@dataclass
class _LoginWindowState:
    failures: list[datetime] = field(default_factory=list)
    lockout_until: datetime | None = None


class LoginAttemptGuard:
    def __init__(self, max_attempts: int, window_seconds: int, lockout_seconds: int) -> None:
        self.max_attempts = max(max_attempts, 1)
        self.window = timedelta(seconds=max(window_seconds, 1))
        self.lockout = timedelta(seconds=max(lockout_seconds, 1))
        self._state: dict[str, _LoginWindowState] = {}

    def reset(self) -> None:
        self._state.clear()

    def _active_failures(self, key: str, now: datetime) -> list[datetime]:
        state = self._state.get(key)
        if not state:
            return []
        cutoff = now - self.window
        state.failures = [attempt for attempt in state.failures if attempt >= cutoff]
        return state.failures

    def check(self, key: str) -> tuple[bool, int]:
        now = datetime.now(UTC)
        state = self._state.get(key)
        if not state or not state.lockout_until:
            return True, 0
        if now >= state.lockout_until:
            state.lockout_until = None
            return True, 0
        seconds_remaining = int((state.lockout_until - now).total_seconds())
        return False, max(seconds_remaining, 1)

    def register_success(self, key: str) -> None:
        self._state.pop(key, None)

    def register_failure(self, key: str) -> None:
        now = datetime.now(UTC)
        state = self._state.setdefault(key, _LoginWindowState())
        failures = self._active_failures(key, now)
        failures.append(now)
        state.failures = failures
        if len(state.failures) >= self.max_attempts:
            state.lockout_until = now + self.lockout

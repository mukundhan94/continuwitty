from __future__ import annotations

from contextlib import suppress
from dataclasses import dataclass, field
from datetime import UTC, datetime, timedelta

from .db import get_conn


@dataclass
class _LoginWindowState:
    failures: list[datetime] = field(default_factory=list)
    lockout_until: datetime | None = None


@dataclass(frozen=True)
class _RateLimitWindowState:
    event_count: int
    window_started_at: datetime
    blocked_until: datetime | None


@dataclass(frozen=True)
class _RateLimitKey:
    namespace: str
    rate_key: str


@dataclass
class _RequestWindowState:
    event_count: int = 0
    window_started_at: datetime | None = None
    blocked_until: datetime | None = None


def _retry_after_seconds(*, blocked_until: datetime, now: datetime) -> int:
    return max(1, int((blocked_until - now).total_seconds()))


def _lock_rate_limit_state(
    *,
    cur,
    key: _RateLimitKey,
    now: datetime,
) -> _RateLimitWindowState:
    cur.execute(
        """
        INSERT INTO rate_limit_state (
            namespace,
            rate_key,
            event_count,
            window_started_at,
            blocked_until,
            updated_at
        )
        VALUES (%s, %s, 0, %s, NULL, %s)
        ON CONFLICT (namespace, rate_key) DO NOTHING
        """,
        (key.namespace, key.rate_key, now, now),
    )
    cur.execute(
        """
        SELECT event_count, window_started_at, blocked_until
        FROM rate_limit_state
        WHERE namespace = %s AND rate_key = %s
        FOR UPDATE
        """,
        (key.namespace, key.rate_key),
    )
    row = cur.fetchone()
    if row is None:  # pragma: no cover - defensive
        return _RateLimitWindowState(event_count=0, window_started_at=now, blocked_until=None)
    return _RateLimitWindowState(
        event_count=int(row["event_count"]),
        window_started_at=row["window_started_at"],
        blocked_until=row["blocked_until"],
    )


def _save_rate_limit_state(
    *,
    cur,
    key: _RateLimitKey,
    state: _RateLimitWindowState,
    now: datetime,
) -> None:
    cur.execute(
        """
        UPDATE rate_limit_state
        SET
            event_count = %s,
            window_started_at = %s,
            blocked_until = %s,
            updated_at = %s
        WHERE namespace = %s AND rate_key = %s
        """,
        (
            state.event_count,
            state.window_started_at,
            state.blocked_until,
            now,
            key.namespace,
            key.rate_key,
        ),
    )


def _delete_rate_limit_state(*, key: _RateLimitKey) -> None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute(
            """
            DELETE FROM rate_limit_state
            WHERE namespace = %s AND rate_key = %s
            """,
            (key.namespace, key.rate_key),
        )


def _clear_rate_limit_namespace(*, namespace: str) -> None:
    with get_conn() as conn, conn.cursor() as cur:
        cur.execute("DELETE FROM rate_limit_state WHERE namespace = %s", (namespace,))


def _bounded_window(*, seconds: int, minimum: int = 1) -> timedelta:
    return timedelta(seconds=max(seconds, minimum))


def _normalize_distributed_window_state(
    *,
    state: _RateLimitWindowState,
    now: datetime,
    window: timedelta,
    reset_count_on_block_expiry: bool,
) -> _RateLimitWindowState:
    if now - state.window_started_at >= window:
        return _RateLimitWindowState(event_count=0, window_started_at=now, blocked_until=None)
    if state.blocked_until and now >= state.blocked_until:
        event_count = 0 if reset_count_on_block_expiry else state.event_count
        window_started_at = now if reset_count_on_block_expiry else state.window_started_at
        return _RateLimitWindowState(
            event_count=event_count,
            window_started_at=window_started_at,
            blocked_until=None,
        )
    return state


class LoginAttemptGuard:
    def __init__(
        self,
        max_attempts: int,
        window_seconds: int,
        lockout_seconds: int,
        *,
        namespace: str = "login_attempts",
    ) -> None:
        self.max_attempts = max(max_attempts, 1)
        self.window = _bounded_window(seconds=window_seconds)
        self.lockout = _bounded_window(seconds=lockout_seconds)
        self._namespace = namespace
        self._state: dict[str, _LoginWindowState] = {}

    def reset(self) -> None:
        self._state.clear()
        with suppress(Exception):
            _clear_rate_limit_namespace(namespace=self._namespace)

    def _active_failures(self, key: str, now: datetime) -> list[datetime]:
        state = self._state.get(key)
        if not state:
            return []
        cutoff = now - self.window
        state.failures = [attempt for attempt in state.failures if attempt >= cutoff]
        return state.failures

    def check(self, key: str) -> tuple[bool, int]:
        now = datetime.now(UTC)
        rate_limit_key = _RateLimitKey(namespace=self._namespace, rate_key=key)

        try:
            with get_conn() as conn, conn.cursor() as cur:
                state = _lock_rate_limit_state(
                    cur=cur,
                    key=rate_limit_key,
                    now=now,
                )
                normalized = _normalize_distributed_window_state(
                    state=state,
                    now=now,
                    window=self.window,
                    reset_count_on_block_expiry=False,
                )
                if normalized != state:
                    _save_rate_limit_state(
                        cur=cur,
                        key=rate_limit_key,
                        state=normalized,
                        now=now,
                    )
                if normalized.blocked_until and now < normalized.blocked_until:
                    return False, _retry_after_seconds(
                        blocked_until=normalized.blocked_until,
                        now=now,
                    )
                return True, 0
        except Exception:
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
        with suppress(Exception):
            _delete_rate_limit_state(key=_RateLimitKey(namespace=self._namespace, rate_key=key))

    def register_failure(self, key: str) -> None:
        now = datetime.now(UTC)
        rate_limit_key = _RateLimitKey(namespace=self._namespace, rate_key=key)
        try:
            with get_conn() as conn, conn.cursor() as cur:
                state = _lock_rate_limit_state(
                    cur=cur,
                    key=rate_limit_key,
                    now=now,
                )
                normalized = _normalize_distributed_window_state(
                    state=state,
                    now=now,
                    window=self.window,
                    reset_count_on_block_expiry=False,
                )
                failure_count = normalized.event_count + 1
                blocked_until = normalized.blocked_until
                if failure_count >= self.max_attempts:
                    blocked_until = now + self.lockout
                _save_rate_limit_state(
                    cur=cur,
                    key=rate_limit_key,
                    state=_RateLimitWindowState(
                        event_count=failure_count,
                        window_started_at=normalized.window_started_at,
                        blocked_until=blocked_until,
                    ),
                    now=now,
                )
            return
        except Exception:
            state = self._state.setdefault(key, _LoginWindowState())
            failures = self._active_failures(key, now)
            failures.append(now)
            state.failures = failures
            if len(state.failures) >= self.max_attempts:
                state.lockout_until = now + self.lockout


class RequestRateLimiter:
    def __init__(
        self,
        *,
        max_requests: int,
        window_seconds: int,
        block_seconds: int,
        namespace: str,
    ) -> None:
        self.max_requests = max(max_requests, 1)
        self.window = _bounded_window(seconds=window_seconds)
        self.block_duration = _bounded_window(seconds=block_seconds, minimum=0)
        self._namespace = namespace
        self._state: dict[str, _RequestWindowState] = {}

    def reset(self) -> None:
        self._state.clear()
        with suppress(Exception):
            _clear_rate_limit_namespace(namespace=self._namespace)

    @staticmethod
    def _window_expired(*, state: _RequestWindowState, now: datetime, window: timedelta) -> bool:
        if state.window_started_at is None:
            return False
        return now - state.window_started_at >= window

    @staticmethod
    def _block_expired(*, state: _RequestWindowState, now: datetime) -> bool:
        if not state.blocked_until:
            return False
        return now >= state.blocked_until

    def _normalize_local_state(self, *, key: str, now: datetime) -> _RequestWindowState:
        state = self._state.setdefault(
            key,
            _RequestWindowState(event_count=0, window_started_at=now, blocked_until=None),
        )
        if state.window_started_at is None:
            state.window_started_at = now
            state.event_count = 0
            state.blocked_until = None
            return state
        if self._window_expired(state=state, now=now, window=self.window):
            state.window_started_at = now
            state.event_count = 0
            state.blocked_until = None
            return state
        if self._block_expired(state=state, now=now):
            state.window_started_at = now
            state.event_count = 0
            state.blocked_until = None
        return state

    def _new_blocked_until(self, *, window_started_at: datetime, now: datetime) -> datetime:
        if self.block_duration > timedelta(0):
            return now + self.block_duration
        return window_started_at + self.window

    def consume(self, key: str) -> tuple[bool, int]:
        now = datetime.now(UTC)
        try:
            return self._consume_distributed(key=key, now=now)
        except Exception:
            return self._consume_local(key=key, now=now)

    def _consume_distributed(self, *, key: str, now: datetime) -> tuple[bool, int]:
        rate_limit_key = _RateLimitKey(namespace=self._namespace, rate_key=key)
        with get_conn() as conn, conn.cursor() as cur:
            state = _lock_rate_limit_state(cur=cur, key=rate_limit_key, now=now)
            normalized = _normalize_distributed_window_state(
                state=state,
                now=now,
                window=self.window,
                reset_count_on_block_expiry=True,
            )
            if normalized.blocked_until and now < normalized.blocked_until:
                return False, _retry_after_seconds(blocked_until=normalized.blocked_until, now=now)

            request_count = normalized.event_count + 1
            blocked_until = normalized.blocked_until
            if request_count > self.max_requests:
                blocked_until = self._new_blocked_until(
                    window_started_at=normalized.window_started_at,
                    now=now,
                )
            next_state = _RateLimitWindowState(
                event_count=request_count,
                window_started_at=normalized.window_started_at,
                blocked_until=blocked_until,
            )
            _save_rate_limit_state(cur=cur, key=rate_limit_key, state=next_state, now=now)
            if request_count > self.max_requests:
                return False, _retry_after_seconds(blocked_until=blocked_until, now=now)
            return True, 0

    def _consume_local(self, *, key: str, now: datetime) -> tuple[bool, int]:
        state = self._normalize_local_state(key=key, now=now)
        if state.blocked_until and now < state.blocked_until:
            return False, _retry_after_seconds(blocked_until=state.blocked_until, now=now)

        state.event_count += 1
        if state.event_count > self.max_requests:
            state.blocked_until = self._new_blocked_until(
                window_started_at=state.window_started_at or now,
                now=now,
            )
            return False, _retry_after_seconds(blocked_until=state.blocked_until, now=now)
        return True, 0

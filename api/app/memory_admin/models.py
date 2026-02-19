from __future__ import annotations

from dataclasses import dataclass
from uuid import UUID


@dataclass(frozen=True)
class SessionDeleteOutcome:
    session_id: UUID
    linked_engrams_deleted: int

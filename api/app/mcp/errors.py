from __future__ import annotations

from dataclasses import dataclass
from typing import Any


@dataclass
class McpRpcError(RuntimeError):
    code: int
    message: str
    data: dict[str, Any] | None = None

    def __post_init__(self) -> None:
        super().__init__(self.message)

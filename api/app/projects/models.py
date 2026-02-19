from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class ProjectResolution:
    project_id: str
    used_default_project: bool

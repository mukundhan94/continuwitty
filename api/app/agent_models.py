from datetime import datetime
from typing import Any

from pydantic import BaseModel, Field


class AgentSourceInput(BaseModel):
    url: str
    title: str | None = None
    snippet: str
    captured_at: datetime


class AgentRunRequest(BaseModel):
    project_id: str
    thread_id: str
    objective: str
    notes: list[str] = Field(default_factory=list)
    assumptions: list[str] = Field(default_factory=list)
    tags: list[str] = Field(default_factory=list)
    keywords: list[str] = Field(default_factory=list)
    sources: list[AgentSourceInput] = Field(default_factory=list)
    auto_persist_engram: bool = True


class AgentResumeRequest(BaseModel):
    notes: list[str] = Field(default_factory=list)
    assumptions: list[str] = Field(default_factory=list)
    tags: list[str] = Field(default_factory=list)
    keywords: list[str] = Field(default_factory=list)
    sources: list[AgentSourceInput] = Field(default_factory=list)
    auto_persist_engram: bool = True


class AgentRunResponse(BaseModel):
    thread_id: str
    status: str
    engram_id: str | None = None
    state: dict[str, Any]

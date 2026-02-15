from __future__ import annotations

from datetime import UTC, datetime
from pathlib import Path
from typing import Any, TypedDict

from langgraph.checkpoint.sqlite import SqliteSaver
from langgraph.graph import END, START, StateGraph

from .config import get_settings
from .models import Claim, Decision, MemoryEngramCreate, SupportingSource
from .repository import create_engram


class AgentState(TypedDict, total=False):
    project_id: str
    thread_id: str
    objective: str
    notes: list[str]
    assumptions: list[str]
    tags: list[str]
    keywords: list[str]
    sources: list[dict[str, Any]]
    synthesis_title: str
    synthesis_abstract: str
    synthesis_markdown: str
    decisions: list[dict[str, str]]
    open_questions: list[str]
    status: str
    auto_persist_engram: bool
    engram_id: str | None


def _normalize_list(value: list[str] | None) -> list[str]:
    if not value:
        return []
    return [item.strip() for item in value if item and item.strip()]


def _collect_node(state: AgentState) -> AgentState:
    notes = _normalize_list(state.get("notes", []))
    assumptions = _normalize_list(state.get("assumptions", []))
    tags = _normalize_list(state.get("tags", []))
    keywords = _normalize_list(state.get("keywords", []))
    sources = state.get("sources", [])

    return {
        "notes": notes,
        "assumptions": assumptions,
        "tags": tags,
        "keywords": keywords,
        "sources": sources,
        "status": "collected",
    }


def _synthesize_node(state: AgentState) -> AgentState:
    objective = state.get("objective", "Untitled objective")
    notes = state.get("notes", [])
    assumptions = state.get("assumptions", [])

    if notes:
        abstract = notes[0][:240]
        bullet_notes = "\n".join(f"- {item}" for item in notes)
    else:
        abstract = f"No notes were provided for objective: {objective}"
        bullet_notes = "- No notes provided"

    decisions = []
    if notes:
        decisions.append(
            {
                "decision": "Consolidate run into memory engram",
                "rationale": "Preserve analysis context across sessions.",
            }
        )

    open_questions: list[str] = []
    if not notes:
        open_questions.append("Capture at least one research note before finalizing.")

    markdown = (
        f"## Objective\n{objective}\n\n"
        f"## Notes\n{bullet_notes}\n\n"
        f"## Assumptions\n"
        + ("\n".join(f"- {item}" for item in assumptions) if assumptions else "- None")
        + "\n\n## Generated At\n"
        + datetime.now(UTC).isoformat()
    )

    return {
        "synthesis_title": f"Run Summary: {objective[:80]}",
        "synthesis_abstract": abstract,
        "synthesis_markdown": markdown,
        "decisions": decisions,
        "open_questions": open_questions,
        "status": "synthesized",
    }


def _persist_node(state: AgentState) -> AgentState:
    if not state.get("auto_persist_engram", True):
        return {"status": "completed", "engram_id": None}

    source_objects = []
    for source in state.get("sources", []):
        source_objects.append(
            SupportingSource(
                url=source["url"],
                title=source.get("title"),
                snippet=source["snippet"],
                captured_at=source["captured_at"],
            )
        )

    claims = []
    if source_objects:
        claims.append(
            Claim(
                claim=f"Research run for objective '{state.get('objective', 'unknown')}' was captured.",
                supporting_sources=source_objects,
            )
        )

    payload = MemoryEngramCreate(
        project_id=state["project_id"],
        thread_id=state["thread_id"],
        title=state.get("synthesis_title") or f"Run Summary: {state.get('objective', 'Untitled')}",
        abstract=state.get("synthesis_abstract")
        or f"Research objective: {state.get('objective', 'Untitled')}",
        detailed_summary_markdown=state.get("synthesis_markdown", "No synthesis markdown."),
        decisions=[
            Decision(decision=item["decision"], rationale=item["rationale"])
            for item in state.get("decisions", [])
        ],
        assumptions=state.get("assumptions", []),
        open_questions=state.get("open_questions", []),
        claims=claims,
        tags=state.get("tags", []),
        keywords=state.get("keywords", []),
    )
    engram = create_engram(payload, embedding_dim=get_settings().embedding_dim)
    return {"status": "completed", "engram_id": str(engram.engram_id)}


def _build_graph():
    builder: StateGraph = StateGraph(AgentState)
    builder.add_node("collect", _collect_node)
    builder.add_node("synthesize", _synthesize_node)
    builder.add_node("persist", _persist_node)
    builder.add_edge(START, "collect")
    builder.add_edge("collect", "synthesize")
    builder.add_edge("synthesize", "persist")
    builder.add_edge("persist", END)
    return builder


class AgentWorkflowService:
    def __init__(self) -> None:
        settings = get_settings()
        checkpoint_path = Path(settings.langgraph_checkpoint_path)
        checkpoint_path.parent.mkdir(parents=True, exist_ok=True)
        self._checkpointer_cm = SqliteSaver.from_conn_string(str(checkpoint_path))
        self._checkpointer = self._checkpointer_cm.__enter__()
        self._graph = _build_graph().compile(checkpointer=self._checkpointer)

    def close(self) -> None:
        self._checkpointer_cm.__exit__(None, None, None)

    def run(self, state: AgentState) -> AgentState:
        config = {"configurable": {"thread_id": state["thread_id"]}}
        result: AgentState = self._graph.invoke(state, config=config)
        return result

    def get_state(self, thread_id: str) -> dict[str, Any] | None:
        config = {"configurable": {"thread_id": thread_id}}
        snapshot = self._graph.get_state(config)
        if not snapshot:
            return None
        return dict(snapshot.values)

    def resume(self, thread_id: str, updates: AgentState) -> AgentState:
        prior = self.get_state(thread_id) or {}
        merged: AgentState = {
            "project_id": prior.get("project_id", updates.get("project_id", "unknown-project")),
            "thread_id": thread_id,
            "objective": prior.get("objective", updates.get("objective", "Resumed objective")),
            "notes": prior.get("notes", []) + updates.get("notes", []),
            "assumptions": prior.get("assumptions", []) + updates.get("assumptions", []),
            "tags": sorted(set(prior.get("tags", []) + updates.get("tags", []))),
            "keywords": sorted(set(prior.get("keywords", []) + updates.get("keywords", []))),
            "sources": prior.get("sources", []) + updates.get("sources", []),
            "auto_persist_engram": updates.get("auto_persist_engram", True),
        }
        return self.run(merged)

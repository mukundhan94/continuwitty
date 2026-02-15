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
    snapshot_enabled: bool
    snapshot_every_n_notes: int
    snapshot_count: int
    snapshot_engram_ids: list[str]


def _normalize_list(value: list[str] | None) -> list[str]:
    if not value:
        return []
    return [item.strip() for item in value if item and item.strip()]


def _build_source_objects(state: AgentState) -> list[SupportingSource]:
    source_objects: list[SupportingSource] = []
    for source in state.get("sources", []):
        source_objects.append(
            SupportingSource(
                url=source["url"],
                title=source.get("title"),
                snippet=source["snippet"],
                captured_at=source["captured_at"],
            )
        )
    return source_objects


def _build_claims(state: AgentState, claim_text: str) -> list[Claim]:
    source_objects = _build_source_objects(state)
    if not source_objects:
        return []
    return [Claim(claim=claim_text, supporting_sources=source_objects)]


def _collect_node(state: AgentState) -> AgentState:
    notes = _normalize_list(state.get("notes", []))
    assumptions = _normalize_list(state.get("assumptions", []))
    tags = _normalize_list(state.get("tags", []))
    keywords = _normalize_list(state.get("keywords", []))
    sources = state.get("sources", [])

    snapshot_ids = [str(item) for item in state.get("snapshot_engram_ids", [])]
    snapshot_count = max(int(state.get("snapshot_count", 0)), len(snapshot_ids))
    snapshot_every = max(1, int(state.get("snapshot_every_n_notes", 3)))

    return {
        "notes": notes,
        "assumptions": assumptions,
        "tags": tags,
        "keywords": keywords,
        "sources": sources,
        "snapshot_enabled": bool(state.get("snapshot_enabled", False)),
        "snapshot_every_n_notes": snapshot_every,
        "snapshot_count": snapshot_count,
        "snapshot_engram_ids": snapshot_ids,
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


def _create_snapshot_engram(state: AgentState, snapshot_index: int) -> str:
    objective = state.get("objective", "Untitled objective")
    notes = state.get("notes", [])
    assumptions = state.get("assumptions", [])

    snapshot_markdown = (
        f"## Snapshot Index\n{snapshot_index}\n\n"
        f"## Objective\n{objective}\n\n"
        f"## Notes Count\n{len(notes)}\n\n"
        f"## Notes\n"
        + ("\n".join(f"- {item}" for item in notes) if notes else "- No notes")
        + "\n\n## Assumptions\n"
        + ("\n".join(f"- {item}" for item in assumptions) if assumptions else "- None")
        + "\n\n## Snapshot Generated At\n"
        + datetime.now(UTC).isoformat()
    )

    snapshot_payload = MemoryEngramCreate(
        project_id=state["project_id"],
        thread_id=state["thread_id"],
        title=f"Snapshot {snapshot_index}: {objective[:70]}",
        abstract=(notes[0][:220] if notes else f"Snapshot for objective: {objective}"),
        detailed_summary_markdown=snapshot_markdown,
        decisions=[
            Decision(
                decision=f"Capture periodic snapshot {snapshot_index}",
                rationale="Persist evolving context during long-running research.",
            )
        ],
        assumptions=assumptions,
        open_questions=[],
        claims=_build_claims(
            state,
            claim_text=f"Snapshot {snapshot_index} captured for objective '{objective}'.",
        ),
        tags=sorted(set(state.get("tags", []) + ["snapshot"])),
        keywords=sorted(set(state.get("keywords", []) + ["snapshot"])),
    )
    snapshot_engram = create_engram(snapshot_payload, embedding_dim=get_settings().embedding_dim)
    return str(snapshot_engram.engram_id)


def _snapshot_node(state: AgentState) -> AgentState:
    snapshot_enabled = bool(state.get("snapshot_enabled", False))
    snapshot_every = max(1, int(state.get("snapshot_every_n_notes", 3)))
    snapshot_count = max(int(state.get("snapshot_count", 0)), 0)
    snapshot_ids = [str(item) for item in state.get("snapshot_engram_ids", [])]
    note_count = len(state.get("notes", []))

    if snapshot_enabled:
        while note_count >= (snapshot_count + 1) * snapshot_every:
            snapshot_index = snapshot_count + 1
            snapshot_id = _create_snapshot_engram(state, snapshot_index)
            snapshot_ids.append(snapshot_id)
            snapshot_count = snapshot_index

    return {
        "snapshot_enabled": snapshot_enabled,
        "snapshot_every_n_notes": snapshot_every,
        "snapshot_count": snapshot_count,
        "snapshot_engram_ids": snapshot_ids,
        "status": "snapshotted",
    }


def _persist_node(state: AgentState) -> AgentState:
    snapshot_state = {
        "snapshot_count": state.get("snapshot_count", 0),
        "snapshot_engram_ids": state.get("snapshot_engram_ids", []),
    }

    if not state.get("auto_persist_engram", True):
        return {
            "status": "completed",
            "engram_id": None,
            **snapshot_state,
        }

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
        claims=_build_claims(
            state,
            claim_text=f"Research run for objective '{state.get('objective', 'unknown')}' was captured.",
        ),
        tags=state.get("tags", []),
        keywords=state.get("keywords", []),
    )
    engram = create_engram(payload, embedding_dim=get_settings().embedding_dim)
    return {
        "status": "completed",
        "engram_id": str(engram.engram_id),
        **snapshot_state,
    }


def _build_graph():
    builder: StateGraph = StateGraph(AgentState)
    builder.add_node("collect", _collect_node)
    builder.add_node("synthesize", _synthesize_node)
    builder.add_node("snapshot", _snapshot_node)
    builder.add_node("persist", _persist_node)
    builder.add_edge(START, "collect")
    builder.add_edge("collect", "synthesize")
    builder.add_edge("synthesize", "snapshot")
    builder.add_edge("snapshot", "persist")
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

    def resume(self, thread_id: str, updates: AgentState) -> AgentState | None:
        prior = self.get_state(thread_id)
        if not prior:
            return None

        auto_persist = updates.get("auto_persist_engram")
        if auto_persist is None:
            auto_persist = prior.get("auto_persist_engram", True)

        snapshot_enabled = updates.get("snapshot_enabled")
        if snapshot_enabled is None:
            snapshot_enabled = prior.get("snapshot_enabled", False)

        snapshot_every = updates.get("snapshot_every_n_notes")
        if snapshot_every is None:
            snapshot_every = prior.get("snapshot_every_n_notes", 3)

        merged: AgentState = {
            "project_id": prior.get("project_id", "unknown-project"),
            "thread_id": thread_id,
            "objective": prior.get("objective", "Resumed objective"),
            "notes": prior.get("notes", []) + updates.get("notes", []),
            "assumptions": prior.get("assumptions", []) + updates.get("assumptions", []),
            "tags": sorted(set(prior.get("tags", []) + updates.get("tags", []))),
            "keywords": sorted(set(prior.get("keywords", []) + updates.get("keywords", []))),
            "sources": prior.get("sources", []) + updates.get("sources", []),
            "auto_persist_engram": bool(auto_persist),
            "snapshot_enabled": bool(snapshot_enabled),
            "snapshot_every_n_notes": max(1, int(snapshot_every)),
            "snapshot_count": int(prior.get("snapshot_count", 0)),
            "snapshot_engram_ids": [str(item) for item in prior.get("snapshot_engram_ids", [])],
        }
        return self.run(merged)

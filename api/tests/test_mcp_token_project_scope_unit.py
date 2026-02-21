from __future__ import annotations

from types import SimpleNamespace
from unittest.mock import Mock
from uuid import UUID, uuid4

from app.mcp.token_project_scope import resolve_project_id_for_canonical_tool


def _scope_dependencies(parse_uuid_value: UUID) -> SimpleNamespace:
    return SimpleNamespace(
        chat_service=Mock(),
        memory_admin_service=Mock(),
        parse_uuid=Mock(return_value=parse_uuid_value),
        get_rehydration_bundle=Mock(),
    )


def test_resolve_project_id_for_session_scoped_tool_uses_session_project() -> None:
    session_id = uuid4()
    dependencies = _scope_dependencies(session_id)
    dependencies.memory_admin_service.get_session.return_value = SimpleNamespace(
        project_id="project-session"
    )

    resolved = resolve_project_id_for_canonical_tool(
        dependencies=dependencies,
        actor_user_id=uuid4(),
        canonical_tool="chat.list_messages",
        params={"session_id": str(session_id)},
    )

    assert resolved == "project-session"
    dependencies.parse_uuid.assert_called_once_with({"session_id": str(session_id)}, "session_id")
    dependencies.memory_admin_service.get_session.assert_called_once_with(
        session_id=session_id,
        include_deleted=True,
    )


def test_resolve_project_id_for_chat_save_as_engram_prefers_session_scope() -> None:
    actor_user_id = uuid4()
    session_id = uuid4()
    dependencies = _scope_dependencies(session_id)
    dependencies.chat_service.get_session.return_value = SimpleNamespace(project_id="project-chat")

    resolved = resolve_project_id_for_canonical_tool(
        dependencies=dependencies,
        actor_user_id=actor_user_id,
        canonical_tool="chat.save_as_engram",
        params={"session_id": str(session_id), "project_id": "ignored"},
    )

    assert resolved == "project-chat"
    dependencies.parse_uuid.assert_called_once_with(
        {"session_id": str(session_id), "project_id": "ignored"},
        "session_id",
    )
    dependencies.chat_service.get_session.assert_called_once_with(
        actor_user_id=actor_user_id,
        session_id=session_id,
    )


def test_resolve_project_id_for_optional_tool_normalizes_project_input() -> None:
    dependencies = _scope_dependencies(uuid4())

    resolved = resolve_project_id_for_canonical_tool(
        dependencies=dependencies,
        actor_user_id=uuid4(),
        canonical_tool="engram.query",
        params={"project_id": "  project-optional  "},
    )

    assert resolved == "project-optional"
    dependencies.parse_uuid.assert_not_called()


def test_resolve_project_id_for_rehydrate_uses_bundle_project() -> None:
    actor_user_id = uuid4()
    engram_id = uuid4()
    dependencies = _scope_dependencies(engram_id)
    dependencies.get_rehydration_bundle.return_value = SimpleNamespace(project_id="project-rehydrate")

    resolved = resolve_project_id_for_canonical_tool(
        dependencies=dependencies,
        actor_user_id=actor_user_id,
        canonical_tool="engram.rehydrate",
        params={"engram_id": str(engram_id)},
    )

    assert resolved == "project-rehydrate"
    dependencies.parse_uuid.assert_called_once_with({"engram_id": str(engram_id)}, "engram_id")
    dependencies.get_rehydration_bundle.assert_called_once_with(
        engram_id,
        actor_user_id=actor_user_id,
    )

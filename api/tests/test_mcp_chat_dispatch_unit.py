from __future__ import annotations

from unittest.mock import Mock
from uuid import uuid4

from app.mcp.chat_dispatch import (
    ChatSessionLifecycleDispatchContext,
    ChatSessionLifecycleDispatchDependencies,
    dispatch_chat_session_lifecycle_tool,
)


class _Dumpable:
    def __init__(self, payload: dict[str, object]) -> None:
        self._payload = payload

    def model_dump(self, *, mode: str = "json") -> dict[str, object]:
        assert mode == "json"
        return self._payload


def _lifecycle_dependencies(session_id: object) -> tuple[object, Mock, Mock]:
    memory_admin_service = Mock()
    parse_uuid = Mock(return_value=session_id)
    require_session_access = Mock()
    dependencies = ChatSessionLifecycleDispatchDependencies(
        memory_admin_service=memory_admin_service,
        parse_uuid=parse_uuid,
        require_session_access=require_session_access,
    )
    return dependencies, parse_uuid, require_session_access


def test_dispatch_chat_session_lifecycle_delete_routes_memory_admin_service() -> None:
    actor_user_id = uuid4()
    session_id = uuid4()
    dependencies, parse_uuid, require_session_access = _lifecycle_dependencies(session_id)
    dependencies.memory_admin_service.delete_session.return_value = _Dumpable({"deleted": True})
    context = ChatSessionLifecycleDispatchContext(
        actor={"role": "user"},
        actor_user_id=actor_user_id,
        method="chat.delete_session",
        params={"session_id": str(session_id), "delete_linked_engrams": True, "reason": "cleanup"},
    )

    result = dispatch_chat_session_lifecycle_tool(
        dependencies=dependencies,
        context=context,
    )

    assert result == {"result": {"deleted": True}}
    parse_uuid.assert_called_once_with(context.params, "session_id")
    require_session_access.assert_called_once_with(actor=context.actor, session_id=session_id)
    delete_kwargs = dependencies.memory_admin_service.delete_session.call_args.kwargs
    assert delete_kwargs["session_id"] == session_id
    assert delete_kwargs["actor_user_id"] == actor_user_id
    assert delete_kwargs["payload"].delete_linked_engrams is True
    assert delete_kwargs["payload"].reason == "cleanup"


def test_dispatch_chat_session_lifecycle_restore_routes_memory_admin_service() -> None:
    session_id = uuid4()
    dependencies, parse_uuid, require_session_access = _lifecycle_dependencies(session_id)
    dependencies.memory_admin_service.restore_session.return_value = _Dumpable({"restored": True})
    context = ChatSessionLifecycleDispatchContext(
        actor={"role": "user"},
        actor_user_id=uuid4(),
        method="chat.restore_session",
        params={"session_id": str(session_id)},
    )

    result = dispatch_chat_session_lifecycle_tool(
        dependencies=dependencies,
        context=context,
    )

    assert result == {"result": {"restored": True}}
    parse_uuid.assert_called_once_with(context.params, "session_id")
    require_session_access.assert_called_once_with(actor=context.actor, session_id=session_id)
    dependencies.memory_admin_service.restore_session.assert_called_once_with(session_id=session_id)


def test_dispatch_chat_session_lifecycle_returns_none_for_unknown_method() -> None:
    session_id = uuid4()
    dependencies, parse_uuid, require_session_access = _lifecycle_dependencies(session_id)
    context = ChatSessionLifecycleDispatchContext(
        actor={"role": "user"},
        actor_user_id=uuid4(),
        method="chat.unknown",
        params={},
    )

    assert (
        dispatch_chat_session_lifecycle_tool(
            dependencies=dependencies,
            context=context,
        )
        is None
    )
    parse_uuid.assert_not_called()
    require_session_access.assert_not_called()
    dependencies.memory_admin_service.delete_session.assert_not_called()
    dependencies.memory_admin_service.restore_session.assert_not_called()

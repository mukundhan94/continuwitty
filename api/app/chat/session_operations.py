from __future__ import annotations

from collections.abc import Callable
from typing import Any, cast
from uuid import UUID

from app.chat_repository import (
    count_session_messages_by_role,
    create_chat_session,
    delete_session_autosave_engrams,
    get_chat_session,
    list_chat_messages,
    list_chat_sessions,
    list_pinned_documents,
    list_pinned_engram_summaries,
    list_pinned_engrams,
    list_session_linked_engrams,
    pin_document_to_session,
    pin_engram_to_session,
    unpin_document_from_session,
    unpin_engram_from_session,
    update_chat_session,
)
from app.models import (
    ChatAutosaveStrategy,
    ChatLifecyclePolicy,
    ChatLifecyclePolicyUpdateRequest,
    ChatMessageRecord,
    ChatSessionCreateRequest,
    ChatSessionRecord,
    ChatSessionUpdateRequest,
    ChatTimelineEvent,
    ContinueSessionRequest,
    ContinueSessionResponse,
    EngramSummary,
    MemoryEngramCreate,
    PinDocumentRequest,
    PinEngramRequest,
    PinnedDocumentRecord,
    SaveSessionAsEngramRequest,
    SaveSessionAsEngramResponse,
)
from app.projects.repository import ensure_project_exists
from app.repository import create_engram

from .errors import ChatSessionNotFoundError, ChatValidationError
from .lifecycle_policy import classify_timeline_event_type, normalize_autosave_policy
from .session_lifecycle import (
    LifecycleMaintenanceResult,
    SessionLifecycleDependencies,
    derive_chat_snapshot_abstract,
    is_generic_snapshot_abstract,
    retrieval_text_from_messages,
    run_session_lifecycle_maintenance,
    transcript_markdown,
)


class ChatSessionOperationsMixin:
    _embedding_dim: int

    @staticmethod
    def _normalize_create_payload(payload: ChatSessionCreateRequest) -> ChatSessionCreateRequest:
        autosave_enabled, autosave_strategy = normalize_autosave_policy(
            autosave_enabled=payload.autosave_enabled,
            autosave_strategy=payload.autosave_strategy,
        )
        return payload.model_copy(
            update={
                "autosave_enabled": autosave_enabled,
                "autosave_strategy": autosave_strategy,
            }
        )

    @staticmethod
    def _normalize_update_payload(payload: ChatSessionUpdateRequest) -> ChatSessionUpdateRequest:
        autosave_enabled = payload.autosave_enabled
        autosave_strategy = payload.autosave_strategy

        if autosave_enabled is None and autosave_strategy is None:
            return payload

        if autosave_enabled is None and autosave_strategy is not None:
            autosave_enabled = autosave_strategy != ChatAutosaveStrategy.off
        if autosave_strategy is None and autosave_enabled is not None:
            autosave_strategy = (
                ChatAutosaveStrategy.interval if autosave_enabled else ChatAutosaveStrategy.off
            )

        assert autosave_enabled is not None
        assert autosave_strategy is not None
        normalized_enabled, normalized_strategy = normalize_autosave_policy(
            autosave_enabled=autosave_enabled,
            autosave_strategy=autosave_strategy,
        )
        return payload.model_copy(
            update={
                "autosave_enabled": normalized_enabled,
                "autosave_strategy": normalized_strategy,
            }
        )

    @staticmethod
    def _build_lifecycle_policy(session: ChatSessionRecord) -> ChatLifecyclePolicy:
        return ChatLifecyclePolicy(
            autosave_enabled=session.autosave_enabled,
            autosave_strategy=session.autosave_strategy,
            autosave_interval_minutes=session.autosave_interval_minutes,
            autosave_min_messages=session.autosave_min_messages,
            retention_days=session.retention_days,
            retention_max_snapshots=session.retention_max_snapshots,
        )

    def create_session(
        self, actor_user_id: UUID, payload: ChatSessionCreateRequest
    ) -> ChatSessionRecord:
        normalized_payload = self._normalize_create_payload(payload)
        ensure_project_exists(
            project_id=normalized_payload.project_id,
            owner_user_id=actor_user_id,
        )
        return create_chat_session(owner_user_id=actor_user_id, payload=normalized_payload)

    def list_sessions(
        self,
        actor_user_id: UUID,
        project_id: str | None,
        limit: int,
        offset: int,
    ) -> list[ChatSessionRecord]:
        return list_chat_sessions(
            actor_user_id=actor_user_id,
            project_id=project_id,
            limit=limit,
            offset=offset,
        )

    def get_session(self, actor_user_id: UUID, session_id: UUID) -> ChatSessionRecord:
        session = get_chat_session(session_id=session_id, actor_user_id=actor_user_id)
        if not session:
            raise ChatSessionNotFoundError()
        return session

    def update_session(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: ChatSessionUpdateRequest,
    ) -> ChatSessionRecord:
        normalized_payload = self._normalize_update_payload(payload)
        updated = update_chat_session(
            session_id=session_id,
            actor_user_id=actor_user_id,
            payload=normalized_payload,
        )
        if not updated:
            raise ChatSessionNotFoundError()
        return updated

    def get_lifecycle_policy(
        self,
        actor_user_id: UUID,
        session_id: UUID,
    ) -> ChatLifecyclePolicy:
        session = self.get_session(actor_user_id=actor_user_id, session_id=session_id)
        return self._build_lifecycle_policy(session)

    def update_lifecycle_policy(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: ChatLifecyclePolicyUpdateRequest,
    ) -> ChatLifecyclePolicy:
        if not payload.model_fields_set:
            return self.get_lifecycle_policy(actor_user_id=actor_user_id, session_id=session_id)

        updated = self.update_session(
            actor_user_id=actor_user_id,
            session_id=session_id,
            payload=ChatSessionUpdateRequest(
                autosave_enabled=payload.autosave_enabled,
                autosave_strategy=payload.autosave_strategy,
                autosave_interval_minutes=payload.autosave_interval_minutes,
                autosave_min_messages=payload.autosave_min_messages,
                retention_days=payload.retention_days,
                retention_max_snapshots=payload.retention_max_snapshots,
            ),
        )
        return self._build_lifecycle_policy(updated)

    def list_timeline_events(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        limit: int,
        offset: int,
    ) -> list[ChatTimelineEvent]:
        self.get_session(actor_user_id=actor_user_id, session_id=session_id)
        linked = list_session_linked_engrams(
            session_id=session_id,
            actor_user_id=actor_user_id,
            limit=limit,
            offset=offset,
        )
        return [
            ChatTimelineEvent(
                event_id=item.engram_id,
                session_id=session_id,
                event_type=classify_timeline_event_type(item.tags),
                title=item.title,
                abstract=item.abstract,
                tags=item.tags,
                created_at=item.created_at,
            )
            for item in linked
        ]

    def list_messages(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        limit: int,
        offset: int,
    ) -> list[ChatMessageRecord]:
        self.get_session(actor_user_id=actor_user_id, session_id=session_id)
        return list_chat_messages(
            session_id=session_id,
            actor_user_id=actor_user_id,
            limit=limit,
            offset=offset,
        )

    def _list_pinned_resources(
        self,
        *,
        actor_user_id: UUID,
        session_id: UUID,
        list_resources: Callable[..., list[Any]],
    ) -> list[Any]:
        self.get_session(actor_user_id=actor_user_id, session_id=session_id)
        return list_resources(
            session_id=session_id,
            actor_user_id=actor_user_id,
        )

    def list_pinned_engrams(
        self,
        actor_user_id: UUID,
        session_id: UUID,
    ) -> list[EngramSummary]:
        return cast(
            list[EngramSummary],
            self._list_pinned_resources(
                actor_user_id=actor_user_id,
                session_id=session_id,
                list_resources=list_pinned_engram_summaries,
            ),
        )

    def list_pinned_documents(
        self,
        actor_user_id: UUID,
        session_id: UUID,
    ) -> list[PinnedDocumentRecord]:
        return cast(
            list[PinnedDocumentRecord],
            self._list_pinned_resources(
                actor_user_id=actor_user_id,
                session_id=session_id,
                list_resources=list_pinned_documents,
            ),
        )

    def _pin_resource(
        self,
        *,
        actor_user_id: UUID,
        session_id: UUID,
        resource_id: UUID,
        pin_resource: Callable[[UUID, UUID, UUID], Any],
        resource_name: str,
    ) -> Any:
        pinned = pin_resource(
            session_id,
            resource_id,
            actor_user_id,
        )
        if not pinned:
            raise ChatValidationError(f"Session or {resource_name} is not accessible for pinning")
        return pinned

    def pin_engram(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: PinEngramRequest,
    ):
        return self._pin_resource(
            actor_user_id=actor_user_id,
            session_id=session_id,
            resource_id=payload.engram_id,
            pin_resource=pin_engram_to_session,
            resource_name="engram",
        )

    def pin_document(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: PinDocumentRequest,
    ) -> PinnedDocumentRecord:
        return cast(
            PinnedDocumentRecord,
            self._pin_resource(
                actor_user_id=actor_user_id,
                session_id=session_id,
                resource_id=payload.document_id,
                pin_resource=pin_document_to_session,
                resource_name="document",
            ),
        )

    def unpin_engram(self, actor_user_id: UUID, session_id: UUID, engram_id: UUID) -> None:
        removed = unpin_engram_from_session(
            session_id,
            engram_id,
            actor_user_id,
        )
        if not removed:
            raise ChatSessionNotFoundError("Pinned engram not found for session")

    def unpin_document(self, actor_user_id: UUID, session_id: UUID, document_id: UUID) -> None:
        removed = unpin_document_from_session(
            session_id,
            document_id,
            actor_user_id,
        )
        if not removed:
            raise ChatSessionNotFoundError("Pinned document not found for session")

    @staticmethod
    def _lifecycle_dependencies() -> SessionLifecycleDependencies:
        return SessionLifecycleDependencies(
            list_session_linked_engrams=list_session_linked_engrams,
            list_chat_messages=list_chat_messages,
            count_session_messages_by_role=count_session_messages_by_role,
            delete_session_autosave_engrams=delete_session_autosave_engrams,
            create_engram=create_engram,
        )

    def _run_session_lifecycle_maintenance(
        self,
        *,
        actor_user_id: UUID,
        session: ChatSessionRecord,
    ) -> LifecycleMaintenanceResult:
        return run_session_lifecycle_maintenance(
            actor_user_id=actor_user_id,
            session=session,
            embedding_dim=self._embedding_dim,
            dependencies=self._lifecycle_dependencies(),
        )

    def save_session_as_engram(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: SaveSessionAsEngramRequest,
    ) -> SaveSessionAsEngramResponse:
        session = self.get_session(actor_user_id=actor_user_id, session_id=session_id)
        messages = list_chat_messages(
            session_id=session.session_id,
            actor_user_id=actor_user_id,
            limit=500,
            offset=0,
        )
        if not messages:
            raise ChatValidationError("Cannot save an empty chat session as engram")

        abstract = payload.abstract.strip()
        if is_generic_snapshot_abstract(abstract):
            derived_abstract = derive_chat_snapshot_abstract(messages)
            if derived_abstract:
                abstract = derived_abstract

        created = create_engram(
            payload=MemoryEngramCreate(
                project_id=session.project_id,
                thread_id=f"chat-session:{session.session_id}",
                title=payload.title,
                abstract=abstract,
                detailed_summary_markdown=transcript_markdown(session, messages),
                tags=payload.tags,
                keywords=payload.keywords,
                visibility_scope=payload.visibility_scope.value,
                source_session_id=session.session_id,
                retrieval_text=retrieval_text_from_messages(messages),
            ),
            embedding_dim=self._embedding_dim,
            owner_user_id=actor_user_id,
            enrichment_origin="chat.save_as_engram",
        )
        return SaveSessionAsEngramResponse(
            engram_id=created.engram_id,
            session_id=session.session_id,
            created_at=created.created_at,
        )

    def continue_session(
        self,
        actor_user_id: UUID,
        session_id: UUID,
        payload: ContinueSessionRequest,
    ) -> ContinueSessionResponse:
        session = self.get_session(actor_user_id=actor_user_id, session_id=session_id)
        continued = create_chat_session(
            owner_user_id=actor_user_id,
            payload=ChatSessionCreateRequest(
                project_id=session.project_id,
                title=payload.title or f"{session.title} (continued)",
                provider=session.provider,
                model_id=session.model_id,
                system_prompt=session.system_prompt,
                visibility_scope=session.visibility_scope,
                autosave_enabled=session.autosave_enabled,
                autosave_strategy=session.autosave_strategy,
                autosave_interval_minutes=session.autosave_interval_minutes,
                autosave_min_messages=session.autosave_min_messages,
                retention_days=session.retention_days,
                retention_max_snapshots=session.retention_max_snapshots,
            ),
        )

        carried_ids: list[UUID] = []
        for pinned in list_pinned_engrams(
            session_id=session.session_id, actor_user_id=actor_user_id
        ):
            copied = pin_engram_to_session(
                continued.session_id,
                pinned.engram_id,
                actor_user_id,
            )
            if copied:
                carried_ids.append(copied.engram_id)

        for pinned_document in list_pinned_documents(
            session_id=session.session_id, actor_user_id=actor_user_id
        ):
            pin_document_to_session(
                continued.session_id,
                pinned_document.document_id,
                actor_user_id,
            )

        return ContinueSessionResponse(
            session=continued,
            carried_engram_ids=carried_ids,
        )

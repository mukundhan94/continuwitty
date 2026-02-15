from __future__ import annotations

from uuid import uuid4

import pytest

from app.auth import hash_password
from app.config import get_settings
from app.models import EngramQueryRequest, MemoryEngramCreate, UserRole
from app.repository import create_engram, get_rehydration_bundle, list_engrams, query_engrams
from app.user_repository import create_user, get_user_auth_record


@pytest.mark.integration
def test_visibility_filters_for_list_query_and_rehydrate(clean_db) -> None:
    admin = get_user_auth_record(get_settings().ui_demo_username)
    assert admin is not None

    analyst = create_user(
        username=f"vis-{uuid4().hex[:8]}",
        password_hash=hash_password("analyst-pass-123"),
        role=UserRole.analyst,
        is_active=True,
    )

    private_engram = create_engram(
        MemoryEngramCreate(
            project_id="project-visibility",
            title="Private Engram",
            abstract="private",
            detailed_summary_markdown="private",
            visibility_scope="private",
        ),
        embedding_dim=get_settings().embedding_dim,
        owner_user_id=admin["user_id"],
    )

    project_engram = create_engram(
        MemoryEngramCreate(
            project_id="project-visibility",
            title="Project Engram",
            abstract="project shared",
            detailed_summary_markdown="project shared",
            visibility_scope="project",
            tags=["shared"],
        ),
        embedding_dim=get_settings().embedding_dim,
        owner_user_id=admin["user_id"],
    )

    visible_list = list_engrams(
        project_id="project-visibility",
        actor_user_id=analyst.user_id,
        limit=25,
        offset=0,
    )
    visible_ids = {item.engram_id for item in visible_list}
    assert project_engram.engram_id in visible_ids
    assert private_engram.engram_id not in visible_ids

    visible_query = query_engrams(
        EngramQueryRequest(
            query="shared",
            project_id="project-visibility",
            top_k=10,
            tags=["shared"],
        ),
        embedding_dim=get_settings().embedding_dim,
        actor_user_id=analyst.user_id,
    )
    query_ids = {item.engram_id for item in visible_query}
    assert project_engram.engram_id in query_ids
    assert private_engram.engram_id not in query_ids

    private_bundle = get_rehydration_bundle(private_engram.engram_id, actor_user_id=analyst.user_id)
    project_bundle = get_rehydration_bundle(project_engram.engram_id, actor_user_id=analyst.user_id)

    assert private_bundle is None
    assert project_bundle is not None
    assert project_bundle.engram_id == project_engram.engram_id

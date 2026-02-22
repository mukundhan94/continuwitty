from __future__ import annotations

from app.mcp.catalog import (
    _READ_TOOL_NAMES,
    _WRITE_TOOL_NAMES,
    _build_chat_tool_catalog,
    _build_engram_tool_catalog,
    _build_project_tool_catalog,
    _build_user_tool_catalog,
    build_tool_catalog,
)


def test_catalog_covers_declared_read_and_write_tools() -> None:
    catalog = build_tool_catalog()
    catalog_names = {item["name"] for item in catalog}
    declared_names = _READ_TOOL_NAMES | _WRITE_TOOL_NAMES
    assert declared_names.issubset(catalog_names)
    assert len(catalog_names) == len(catalog)


def test_catalog_entries_have_input_schema() -> None:
    for item in build_tool_catalog():
        assert "inputSchema" in item
        assert isinstance(item["inputSchema"], dict)
        assert item["inputSchema"].get("type") == "object"


def test_build_tool_catalog_matches_namespace_builders() -> None:
    assert build_tool_catalog() == [
        *_build_chat_tool_catalog(),
        *_build_engram_tool_catalog(),
        *_build_project_tool_catalog(),
        *_build_user_tool_catalog(),
    ]


def test_namespace_builders_emit_only_expected_prefixes() -> None:
    assert all(item["name"].startswith("chat.") for item in _build_chat_tool_catalog())
    assert all(item["name"].startswith("engram.") for item in _build_engram_tool_catalog())
    assert all(item["name"].startswith("project.") for item in _build_project_tool_catalog())
    assert all(item["name"].startswith("user.") for item in _build_user_tool_catalog())


def test_catalog_builder_returns_deep_copies() -> None:
    first = build_tool_catalog()
    first[0]["inputSchema"]["properties"]["project_id"]["type"] = "integer"
    second = build_tool_catalog()
    assert second[0]["inputSchema"]["properties"]["project_id"]["type"] == "string"

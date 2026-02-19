from __future__ import annotations

from app.mcp.catalog import _READ_TOOL_NAMES, _WRITE_TOOL_NAMES, build_tool_catalog


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

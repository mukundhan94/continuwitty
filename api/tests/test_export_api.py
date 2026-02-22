from __future__ import annotations

import json
import zipfile
from io import BytesIO

from app.export.api import _build_export_filename, _build_zip_payload
from app.export.models import ProjectExportFormat


def test_build_export_filename_uses_expected_suffix_and_sanitization() -> None:
    assert (
        _build_export_filename(
            project_id="engram-vault",
            export_format=ProjectExportFormat.json,
        )
        == "engram-export-engram-vault.json"
    )
    assert (
        _build_export_filename(
            project_id="my project/alpha",
            export_format=ProjectExportFormat.zip,
        )
        == "engram-export-my-project-alpha.zip"
    )


def test_build_zip_payload_contains_export_json_file() -> None:
    expected = {"schema_version": "1.0", "project": {"project_id": "engram-vault"}}
    payload = _build_zip_payload(json_payload=json.dumps(expected).encode("utf-8"))

    with zipfile.ZipFile(BytesIO(payload), mode="r") as archive:
        assert archive.namelist() == ["export.json"]
        loaded = json.loads(archive.read("export.json").decode("utf-8"))

    assert loaded == expected

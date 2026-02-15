from __future__ import annotations

import argparse
import json
import sys
from collections.abc import Sequence
from datetime import datetime
from pathlib import Path
from typing import Any
from uuid import UUID

from pydantic import ValidationError

from .config import get_settings
from .consolidation import run_project_consolidation
from .models import EngramQueryRequest, MemoryEngramCreate
from .repository import create_engram, get_rehydration_bundle, query_engrams


def _print_json(payload: Any) -> None:
    print(json.dumps(payload, indent=2))


def _iso_or_none(value: str | None) -> datetime | None:
    if not value:
        return None
    parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
    return parsed


def _load_json(path: str) -> Any:
    file_path = Path(path)
    raw = file_path.read_text(encoding="utf-8")
    return json.loads(raw)


def _upload(args: argparse.Namespace) -> int:
    data = _load_json(args.file)
    settings = get_settings()

    payloads = data if isinstance(data, list) else [data]
    results: list[dict[str, Any]] = []
    for item in payloads:
        payload = MemoryEngramCreate.model_validate(item)
        created = create_engram(payload, settings.embedding_dim)
        results.append(created.model_dump(mode="json"))

    _print_json({"created": len(results), "items": results})
    return 0


def _search(args: argparse.Namespace) -> int:
    settings = get_settings()
    request = EngramQueryRequest(
        query=args.query,
        top_k=args.top_k,
        project_id=args.project_id,
        tags=args.tags,
        keywords=args.keywords,
        created_after=_iso_or_none(args.created_after),
        created_before=_iso_or_none(args.created_before),
    )
    rows = query_engrams(request, settings.embedding_dim)
    _print_json([row.model_dump(mode="json") for row in rows])
    return 0


def _rehydrate(args: argparse.Namespace) -> int:
    engram_id = UUID(args.engram_id)
    bundle = get_rehydration_bundle(engram_id)
    if not bundle:
        print(f"Engram not found: {engram_id}", file=sys.stderr)
        return 1

    _print_json(bundle.model_dump(mode="json"))
    return 0


def _consolidate(args: argparse.Namespace) -> int:
    result = run_project_consolidation(
        project_id=args.project_id,
        source_limit=args.source_limit,
        min_items=args.min_items,
        dry_run=args.dry_run,
    )
    _print_json(result)
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="engram-cli",
        description="Local CLI for upload/search/rehydrate workflows.",
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    upload_parser = subparsers.add_parser("upload", help="Create one or more engrams from JSON")
    upload_parser.add_argument(
        "--file",
        required=True,
        help="Path to a JSON file containing a MemoryEngramCreate object or list",
    )

    search_parser = subparsers.add_parser("search", help="Semantic search for engrams")
    search_parser.add_argument("--query", required=True, help="Semantic query text")
    search_parser.add_argument("--project-id", help="Optional project filter")
    search_parser.add_argument("--tag", action="append", dest="tags", default=[], help="Tag filter")
    search_parser.add_argument(
        "--keyword",
        action="append",
        dest="keywords",
        default=[],
        help="Keyword filter",
    )
    search_parser.add_argument("--top-k", type=int, default=5, help="Result count (1-50)")
    search_parser.add_argument(
        "--created-after",
        help="ISO-8601 timestamp filter (inclusive)",
    )
    search_parser.add_argument(
        "--created-before",
        help="ISO-8601 timestamp filter (inclusive)",
    )

    rehydrate_parser = subparsers.add_parser(
        "rehydrate", help="Get rehydration bundle by engram id"
    )
    rehydrate_parser.add_argument("--engram-id", required=True, help="Engram UUID")

    consolidate_parser = subparsers.add_parser(
        "consolidate", help="Create an automated consolidation engram for a project"
    )
    consolidate_parser.add_argument("--project-id", required=True, help="Project scope")
    consolidate_parser.add_argument(
        "--source-limit",
        type=int,
        default=20,
        help="Max number of source engrams to consolidate",
    )
    consolidate_parser.add_argument(
        "--min-items",
        type=int,
        default=3,
        help="Minimum non-consolidated source engrams required",
    )
    consolidate_parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Preview consolidation without creating a new engram",
    )

    return parser


def main(argv: Sequence[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    try:
        if args.command == "upload":
            return _upload(args)
        if args.command == "search":
            return _search(args)
        if args.command == "rehydrate":
            return _rehydrate(args)
        if args.command == "consolidate":
            return _consolidate(args)
    except (OSError, json.JSONDecodeError, ValueError, ValidationError) as exc:
        print(f"CLI error: {exc}", file=sys.stderr)
        return 2

    parser.print_help()
    return 2


if __name__ == "__main__":
    raise SystemExit(main())

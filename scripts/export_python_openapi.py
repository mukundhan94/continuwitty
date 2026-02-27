#!/usr/bin/env python3

import argparse
import json
import sys
from pathlib import Path


def parse_args() -> argparse.Namespace:
	parser = argparse.ArgumentParser(
		description="Export the legacy Python FastAPI OpenAPI schema to a tracked JSON artifact.",
	)
	parser.add_argument(
		"--output",
		default="contracts/python-openapi.json",
		help="Output path for the exported schema (default: contracts/python-openapi.json).",
	)
	return parser.parse_args()


def main() -> int:
	args = parse_args()
	repo_root = Path(__file__).resolve().parents[1]
	output_path = (repo_root / args.output).resolve()

	python_api_root = repo_root / "api"
	if not python_api_root.exists():
		print(f"Python API directory not found: {python_api_root}", file=sys.stderr)
		return 1

	sys.path.insert(0, str(python_api_root))
	from app.main import app  # pylint: disable=import-error,import-outside-toplevel

	spec = app.openapi()
	output_path.parent.mkdir(parents=True, exist_ok=True)
	output_path.write_text(json.dumps(spec, indent=2, sort_keys=True) + "\n", encoding="utf-8")

	path_count = len(spec.get("paths", {}))
	print(f"Exported Python OpenAPI schema ({path_count} paths) to {output_path}")
	return 0


if __name__ == "__main__":
	raise SystemExit(main())

#!/usr/bin/env python3

import argparse
import json
import re
import sys
import urllib.error
import urllib.request
from pathlib import Path
from typing import Iterable


SUPPORTED_METHODS = {"get", "post", "put", "patch", "delete"}
PARAM_PATTERN = re.compile(r"{([^}/]+)}")
MISSING_ROUTE_MARKER = "404 page not found"


def parse_args() -> argparse.Namespace:
	parser = argparse.ArgumentParser(
		description="Validate that OpenAPI contract routes exist on the running Go API server.",
	)
	parser.add_argument(
		"--spec",
		default="contracts/python-openapi.json",
		help="Path to the OpenAPI JSON contract file.",
	)
	parser.add_argument(
		"--base-url",
		default="http://127.0.0.1:8000",
		help="Base URL for the running Go API server.",
	)
	parser.add_argument(
		"--timeout-seconds",
		type=float,
		default=5.0,
		help="HTTP timeout per request in seconds.",
	)
	return parser.parse_args()


def load_contract_paths(spec_path: Path) -> dict[str, list[str]]:
	data = json.loads(spec_path.read_text(encoding="utf-8"))
	raw_paths = data.get("paths", {})
	paths: dict[str, list[str]] = {}
	for path, operations in raw_paths.items():
		if path.startswith("/openapi.json") or path.startswith("/docs") or path.startswith("/redoc"):
			continue
		methods = [
			method.lower()
			for method in operations.keys()
			if method.lower() in SUPPORTED_METHODS
		]
		if methods:
			paths[path] = sorted(methods)
	return paths


def substitute_path_params(path: str) -> str:
	def resolve_parameter(match: re.Match[str]) -> str:
		name = match.group(1).lower()
		if "id" in name:
			return "not-a-uuid"
		return "contract-check"

	return PARAM_PATTERN.sub(resolve_parameter, path)


def request_route(base_url: str, method: str, path: str, timeout_seconds: float) -> tuple[int, str]:
	url = f"{base_url.rstrip('/')}{path}"
	request_data = None
	headers: dict[str, str] = {}
	if method in {"post", "put", "patch"}:
		request_data = b"{}"
		headers["Content-Type"] = "application/json"

	request = urllib.request.Request(url=url, method=method.upper(), data=request_data, headers=headers)
	try:
		with urllib.request.urlopen(request, timeout=timeout_seconds) as response:
			return int(response.status), response.read().decode("utf-8", errors="replace")
	except urllib.error.HTTPError as error:
		return int(error.code), error.read().decode("utf-8", errors="replace")


def is_router_404(status: int, body: str) -> bool:
	return status == 404 and MISSING_ROUTE_MARKER in body.lower()


def iter_contract_routes(contract_paths: dict[str, list[str]]) -> Iterable[tuple[str, str]]:
	for path in sorted(contract_paths.keys()):
		for method in contract_paths[path]:
			yield method, path


def main() -> int:
	args = parse_args()
	repo_root = Path(__file__).resolve().parents[1]
	spec_path = (repo_root / args.spec).resolve()
	if not spec_path.exists():
		print(f"OpenAPI contract not found: {spec_path}", file=sys.stderr)
		return 1

	contract_paths = load_contract_paths(spec_path)
	if not contract_paths:
		print(f"No contract paths found in {spec_path}", file=sys.stderr)
		return 1

	missing_routes: list[str] = []
	checked = 0
	for method, template_path in iter_contract_routes(contract_paths):
		checked += 1
		resolved_path = substitute_path_params(template_path)
		status, body = request_route(
			base_url=args.base_url,
			method=method,
			path=resolved_path,
			timeout_seconds=args.timeout_seconds,
		)
		if is_router_404(status, body):
			missing_routes.append(f"{method.upper()} {template_path}")

	if missing_routes:
		print(f"Route validation failed: {len(missing_routes)} missing contract routes")
		for route in missing_routes:
			print(f"  - {route}")
		return 1

	print(f"Validated {checked} contract routes against {args.base_url} (no router-level 404 mismatches).")
	return 0


if __name__ == "__main__":
	raise SystemExit(main())

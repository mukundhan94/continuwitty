#!/usr/bin/env python3

import argparse
import json
import re
import sys
import urllib.error
import urllib.request
from pathlib import Path


SUPPORTED_METHODS = {"get", "post", "put", "patch", "delete"}
PARAM_PATTERN = re.compile(r"{([^}/]+)}")


def parse_args() -> argparse.Namespace:
	parser = argparse.ArgumentParser(
		description="Compare Go and Python API HTTP status parity through the shadow proxy.",
	)
	parser.add_argument(
		"--spec",
		default="contracts/python-openapi.json",
		help="Path to the exported Python OpenAPI contract.",
	)
	parser.add_argument(
		"--proxy-base-url",
		default="http://127.0.0.1:8080",
		help="Base URL for the shadow proxy.",
	)
	parser.add_argument(
		"--timeout-seconds",
		type=float,
		default=5.0,
		help="HTTP timeout per request in seconds.",
	)
	return parser.parse_args()


def load_routes(spec_path: Path) -> list[tuple[str, str]]:
	data = json.loads(spec_path.read_text(encoding="utf-8"))
	raw_paths = data.get("paths", {})
	routes: list[tuple[str, str]] = []
	for path in sorted(raw_paths.keys()):
		if path.startswith("/openapi.json") or path.startswith("/docs") or path.startswith("/redoc"):
			continue
		for method in sorted(raw_paths[path].keys()):
			method_lower = method.lower()
			if method_lower not in SUPPORTED_METHODS:
				continue
			routes.append((method_lower, path))
	return routes


def resolve_path(path: str) -> str:
	def replace(match: re.Match[str]) -> str:
		name = match.group(1).lower()
		if "id" in name:
			return "not-a-uuid"
		return "shadow-compare"

	return PARAM_PATTERN.sub(replace, path)


def request_status(url: str, method: str, timeout_seconds: float) -> int:
	data = None
	headers: dict[str, str] = {}
	if method in {"post", "put", "patch"}:
		data = b"{}"
		headers["Content-Type"] = "application/json"
	request = urllib.request.Request(url=url, method=method.upper(), data=data, headers=headers)
	try:
		with urllib.request.urlopen(request, timeout=timeout_seconds) as response:
			return int(response.status)
	except urllib.error.HTTPError as error:
		return int(error.code)


def status_family(status: int) -> str:
	if 200 <= status < 300:
		return "2xx"
	if 300 <= status < 400:
		return "3xx"
	if 400 <= status < 500:
		return "4xx"
	return "5xx"


def main() -> int:
	args = parse_args()
	repo_root = Path(__file__).resolve().parents[1]
	spec_path = (repo_root / args.spec).resolve()
	if not spec_path.exists():
		print(f"Contract file not found: {spec_path}", file=sys.stderr)
		return 1

	routes = load_routes(spec_path)
	if not routes:
		print(f"No comparable routes found in {spec_path}", file=sys.stderr)
		return 1

	family_mismatches: list[str] = []
	exact_status_differences: list[str] = []
	for method, template_path in routes:
		path = resolve_path(template_path)
		go_status = request_status(
			url=f"{args.proxy_base_url.rstrip('/')}/go{path}",
			method=method,
			timeout_seconds=args.timeout_seconds,
		)
		py_status = request_status(
			url=f"{args.proxy_base_url.rstrip('/')}/py{path}",
			method=method,
			timeout_seconds=args.timeout_seconds,
		)
		go_family = status_family(go_status)
		py_family = status_family(py_status)
		if go_family != py_family:
			family_mismatches.append(
				f"{method.upper()} {template_path} -> go={go_status}({go_family}) py={py_status}({py_family})",
			)
			continue
		if go_status != py_status:
			exact_status_differences.append(
				f"{method.upper()} {template_path} -> go={go_status} py={py_status}",
			)

	if family_mismatches:
		print(f"Shadow compare found {len(family_mismatches)} status-family mismatches:")
		for mismatch in family_mismatches:
			print(f"  - {mismatch}")
		return 1

	if exact_status_differences:
		print(
			f"Shadow compare status-family parity passed for {len(routes)} routes "
			f"with {len(exact_status_differences)} exact status differences (informational):",
		)
		for difference in exact_status_differences:
			print(f"  - {difference}")
		return 0

	print(f"Shadow compare passed for {len(routes)} routes (exact status parity matched).")
	return 0


if __name__ == "__main__":
	raise SystemExit(main())

#!/usr/bin/env python3

import argparse
import statistics
import time
import urllib.error
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from pathlib import Path


@dataclass
class BenchmarkResult:
	label: str
	endpoint: str
	requests: int
	concurrency: int
	elapsed_seconds: float
	latencies_ms: list[float]
	status_counts: dict[int, int]

	@property
	def requests_per_second(self) -> float:
		if self.elapsed_seconds <= 0:
			return 0.0
		return self.requests / self.elapsed_seconds

	@property
	def p50_ms(self) -> float:
		return percentile(self.latencies_ms, 50)

	@property
	def p95_ms(self) -> float:
		return percentile(self.latencies_ms, 95)

	@property
	def p99_ms(self) -> float:
		return percentile(self.latencies_ms, 99)


def parse_args() -> argparse.Namespace:
	parser = argparse.ArgumentParser(
		description="Benchmark Go vs Python API latency/throughput through shadow proxy routes.",
	)
	parser.add_argument("--go-base-url", default="http://127.0.0.1:8080/go")
	parser.add_argument("--py-base-url", default="http://127.0.0.1:8080/py")
	parser.add_argument(
		"--endpoints",
		default="/healthz,/api/v1/version",
		help="Comma-separated endpoints to benchmark.",
	)
	parser.add_argument("--requests", type=int, default=300)
	parser.add_argument("--concurrency", type=int, default=20)
	parser.add_argument("--timeout-seconds", type=float, default=5.0)
	parser.add_argument("--output", default="findings/go-migration-benchmark.md")
	return parser.parse_args()


def percentile(values: list[float], p: int) -> float:
	if not values:
		return 0.0
	if len(values) == 1:
		return values[0]
	sorted_values = sorted(values)
	rank = (p / 100) * (len(sorted_values) - 1)
	lower = int(rank)
	upper = min(lower + 1, len(sorted_values) - 1)
	weight = rank - lower
	return sorted_values[lower] * (1 - weight) + sorted_values[upper] * weight


def run_single_request(url: str, timeout_seconds: float) -> tuple[int, float]:
	start = time.perf_counter()
	request = urllib.request.Request(url=url, method="GET")
	try:
		with urllib.request.urlopen(request, timeout=timeout_seconds) as response:
			status = int(response.status)
			_ = response.read()
	except urllib.error.HTTPError as error:
		status = int(error.code)
		_ = error.read()
	except Exception:
		status = 0
	elapsed_ms = (time.perf_counter() - start) * 1000
	return status, elapsed_ms


def run_benchmark(
	*,
	label: str,
	base_url: str,
	endpoint: str,
	requests: int,
	concurrency: int,
	timeout_seconds: float,
) -> BenchmarkResult:
	url = f"{base_url.rstrip('/')}{endpoint}"
	status_counts: dict[int, int] = {}
	latencies_ms: list[float] = []
	start = time.perf_counter()
	with ThreadPoolExecutor(max_workers=concurrency) as pool:
		futures = [pool.submit(run_single_request, url, timeout_seconds) for _ in range(requests)]
		for future in as_completed(futures):
			status, latency_ms = future.result()
			latencies_ms.append(latency_ms)
			status_counts[status] = status_counts.get(status, 0) + 1
	elapsed_seconds = time.perf_counter() - start
	return BenchmarkResult(
		label=label,
		endpoint=endpoint,
		requests=requests,
		concurrency=concurrency,
		elapsed_seconds=elapsed_seconds,
		latencies_ms=latencies_ms,
		status_counts=status_counts,
	)


def format_status_counts(counts: dict[int, int]) -> str:
	items = sorted(counts.items(), key=lambda item: item[0])
	return ", ".join([f"{code}:{count}" for code, count in items]) if items else "none"


def render_markdown(results: list[BenchmarkResult]) -> str:
	lines: list[str] = []
	lines.append("# Go Migration Benchmark Report")
	lines.append("")
	lines.append(
		f"- Generated at: {time.strftime('%Y-%m-%d %H:%M:%S UTC', time.gmtime())}",
	)
	lines.append("")
	lines.append(
		"| Runtime | Endpoint | Requests | Concurrency | Req/s | P50 (ms) | P95 (ms) | P99 (ms) | Mean (ms) | Statuses |",
	)
	lines.append("|---|---|---:|---:|---:|---:|---:|---:|---:|---|")
	for result in results:
		mean_ms = statistics.fmean(result.latencies_ms) if result.latencies_ms else 0.0
		lines.append(
			f"| {result.label} | `{result.endpoint}` | {result.requests} | {result.concurrency} | "
			f"{result.requests_per_second:.2f} | {result.p50_ms:.2f} | {result.p95_ms:.2f} | {result.p99_ms:.2f} | "
			f"{mean_ms:.2f} | {format_status_counts(result.status_counts)} |",
		)
	return "\n".join(lines) + "\n"


def main() -> int:
	args = parse_args()
	endpoints = [item.strip() for item in args.endpoints.split(",") if item.strip()]
	results: list[BenchmarkResult] = []
	for endpoint in endpoints:
		results.append(
			run_benchmark(
				label="go",
				base_url=args.go_base_url,
				endpoint=endpoint,
				requests=args.requests,
				concurrency=args.concurrency,
				timeout_seconds=args.timeout_seconds,
			),
		)
		results.append(
			run_benchmark(
				label="python",
				base_url=args.py_base_url,
				endpoint=endpoint,
				requests=args.requests,
				concurrency=args.concurrency,
				timeout_seconds=args.timeout_seconds,
			),
		)

	report = render_markdown(results)
	output_path = Path(args.output)
	output_path.parent.mkdir(parents=True, exist_ok=True)
	output_path.write_text(report, encoding="utf-8")
	print(report)
	print(f"Benchmark report written to {output_path.resolve()}")
	return 0


if __name__ == "__main__":
	raise SystemExit(main())

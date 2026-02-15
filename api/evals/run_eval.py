from __future__ import annotations

import argparse
import json
from pathlib import Path

from evals.harness import run_evaluations


def main() -> int:
    parser = argparse.ArgumentParser(description="Run Engram Vault local evaluation harness")
    parser.add_argument("--out", help="Optional JSON output path")
    args = parser.parse_args()

    summary = run_evaluations()
    output = json.dumps(summary, indent=2)
    print(output)

    if args.out:
        output_path = Path(args.out)
        output_path.parent.mkdir(parents=True, exist_ok=True)
        output_path.write_text(output + "\n", encoding="utf-8")

    return 0 if summary["passed"] else 1


if __name__ == "__main__":
    raise SystemExit(main())

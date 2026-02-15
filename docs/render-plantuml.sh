#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCS_DIR="$ROOT_DIR/docs"
OUT_DIR="$DOCS_DIR/rendered"
FORMAT="${1:-svg}"
shift || true

case "$FORMAT" in
  svg|png)
    ;;
  *)
    echo "Unsupported format: $FORMAT"
    echo "Usage: docs/render-plantuml.sh [svg|png] [puml_file ...]"
    exit 1
    ;;
esac

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker is required for rendering PlantUML without local Graphviz."
  exit 1
fi

mkdir -p "$OUT_DIR"

declare -a files

if [ $# -gt 0 ]; then
  for arg in "$@"; do
    if [[ "$arg" = /* ]]; then
      candidate="$arg"
    else
      candidate="$ROOT_DIR/$arg"
    fi
    if [ ! -f "$candidate" ]; then
      echo "PUML file not found: $arg"
      exit 1
    fi
    files+=("$candidate")
  done
else
  shopt -s nullglob
  files=("$DOCS_DIR"/*.puml)
  shopt -u nullglob
fi

if [ ${#files[@]} -eq 0 ]; then
  echo "No .puml files found in $DOCS_DIR"
  exit 0
fi

for input in "${files[@]}"; do
  base_name="$(basename "$input" .puml)"
  output="$OUT_DIR/$base_name.$FORMAT"
  echo "Rendering $(basename "$input") -> $(basename "$output")"
  docker run --rm -i plantuml/plantuml "-t$FORMAT" -pipe < "$input" > "$output"
done

echo "Done. Rendered files are in: $OUT_DIR"

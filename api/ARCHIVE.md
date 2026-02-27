# Legacy Python API Archive

The Python/FastAPI implementation under this `api/` directory is retained as a legacy reference archive during Go migration closeout.

## Status

- Runtime cutover target: Go (`cmd/api`, `internal/*`)
- Python runtime role: reference-only, optional parity/shadow validation
- Active default developer/runtime commands: Go (`make api`, `make dev`, `make test`)
- Legacy commands remain available under `make py-*`

## Purpose

- Preserve original implementation details for historical context and parity investigation.
- Support shadow comparison and contract export workflows during rollout hardening.

No new product features should be implemented in this archived Python surface.

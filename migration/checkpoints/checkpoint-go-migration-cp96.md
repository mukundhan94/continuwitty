# CP96 - Phase 4 Agent Workflow API Route Baseline

Date: 2026-02-26  
Status: Completed

## Summary

Ported agent workflow REST routes into Go:
- `POST /api/v1/agent-runs`
- `GET /api/v1/agent-runs/{thread_id}`
- `POST /api/v1/agent-runs/{thread_id}/resume`

This checkpoint wires the route layer to the new `internal/workflow` service and runtime engram persistence callback.

## Files

- Added: `internal/api/agent_workflow_api.go`
- Added: `internal/api/agent_workflow_api_test.go`
- Updated: `internal/api/router.go`
- Updated: `internal/api/router_test.go`
- Updated: `cmd/api/main.go`
- Updated: `checkpoint-go-migration.md`

## Tests Run

One-by-one:
- `go test ./internal/api -run TestMountAgentWorkflowRoutesRegistersEndpoints -count=1`
- `go test ./internal/api -run TestAgentWorkflowRoutesValidationErrors -count=1`
- `go test ./internal/api -run TestAgentWorkflowRoutesNotFoundAndErrors -count=1`
- `go test ./internal/api -run TestBuildAgentRunResponseCopiesSnapshotIDs -count=1`
- `go test ./internal/api -run TestAgentWorkflowRoutesMountedWithDependencies -count=1`

Focused/full:
- `go test ./internal/api ./internal/workflow ./cmd/api -count=1`
- `go test ./... -count=1`

## Code Health

- `internal/api/agent_workflow_api.go`: `10.0`
- `internal/api/agent_workflow_api_test.go`: `10.0`
- `internal/api/router.go`: `10.0`
- `internal/api/router_test.go`: `10.0`
- `cmd/api/main.go`: `10.0`

## Safeguard

- `pre_commit_code_health_safeguard`: `quality_gates=passed`, no findings.

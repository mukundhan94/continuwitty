# Go Migration Test Matrix

This matrix tracks active quality gates for the Go runtime.

| Area | Primary Gate | Supporting Gate |
|---|---|---|
| Core repository semantics (engram/chat/document/project/oauth/token/admin) | `go test ./internal/repository -count=1` | `go test ./... -count=1` |
| Session auth + UI login/logout + CSRF + lockout | `go test ./internal/api -count=1` | `make acceptance-test-mock-docker` |
| MCP transport, auth scopes, compatibility dispatch | `go test ./internal/mcp -count=1` | `make acceptance-test-mock-docker` |
| Admin memory lifecycle + linked engram semantics | `go test ./internal/admin -count=1` | `make acceptance-test-mock-docker` |
| Containerized runtime health | `make stack-smoke` | `make stack-up` + `docker compose ps` |
| End-to-end user workflows | `make acceptance-test-mock-docker` | `go test ./... -count=1` |

## Notes

- The migration is complete and only Go runtime gates are active.
- Enforce the release gate with `make check` plus acceptance smoke before deploy.

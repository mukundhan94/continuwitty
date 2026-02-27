# Go Migration Test Matrix

This matrix tracks how Python-era behaviors are now enforced in the Go runtime.

| Area | Primary Gate | Supporting Gate |
|---|---|---|
| Core repository semantics (engram/chat/document/project/oauth/token/admin) | `go test ./internal/repository -count=1` | `go test ./... -count=1` |
| Session auth + UI login/logout + CSRF + lockout | `go test ./internal/api -count=1` | `make acceptance-test-mock-docker` |
| MCP transport, auth scopes, compatibility dispatch | `go test ./internal/mcp -count=1` | `make acceptance-test-mock-docker` |
| Admin memory lifecycle + linked engram semantics | `go test ./internal/admin -count=1` | `make acceptance-test-mock-docker` |
| OpenAPI route contract parity vs Python | `make openapi-check` | `make shadow-compare` |
| Runtime parity across Go vs Python status behavior | `make shadow-compare` | `make benchmark-compare` |
| End-to-end user workflows | `make acceptance-test-mock-docker` | `go test ./... -count=1` |

## Notes

- Python `api/tests` are retained as legacy reference checks.
- Active quality gates for Go migration completion are Go unit/integration tests, acceptance tests, OpenAPI route validation, and shadow comparison.

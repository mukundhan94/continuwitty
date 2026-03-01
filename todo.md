# Engram Vault - Todo Tracker

> Forward-looking work items.
> See [Plan.md](Plan.md) for full roadmap. See [migration/checkpoints/checkpoint.md](migration/checkpoints/checkpoint.md) for completed milestones.

---

## Immediate (Current Sprint)

- [x] Phase 31 closeout: update `AGENT.md` + skills with final project-default/soft-delete/collection invariants and MCP organization contracts
- [x] Phase 31 closeout: run/record full acceptance mock execution (`make acceptance-test-mock`) with new admin-memory scenarios
- [x] Phase 18 follow-up: add explicit consolidation merge/grouping event semantics in timeline rendering
- [x] Phase 16 deferred: CLI smoke utility (`engram-cli mcp-call`)

## Near-Term (Next Phases)

- [x] Phase 33: portable export/import stash workflow (project + collections + engrams) with owner/admin authorization and round-trip tests
- [x] Phase 34: security audit remediation program (critical defaults, auth/session hardening, MCP/OAuth protections)
- [x] Phase 19: implement project membership and scoped sharing/revocation flows with audit trails
- [x] Phase 20: production security closeout (OIDC rollout hardening + centralized audit sink + auth abuse-path tests)
- [x] Phase 21: observability and reliability (request telemetry/metrics, stream/provider health categories, lifecycle tracing hooks, provider fallback/circuit strategy)
- [x] Phase 22: release automation and deployment profiles (CI staged gates + compose profiles + versioned release checklist/runbook)
- [x] Phase 23: EvalOps and prompt/policy governance

## Future (Post Phase 23)

- [x] Phase 24: engram graph foundations (link schema + repository)
- [x] Phase 25: link APIs, MCP tools, and suggestion pipeline
- [x] Phase 26: graph-aware context assembly (configurable recall depth)
- [x] Phase 27: traceability UX and link management UI
- [x] Phase 28: temporal dynamics, graph quality, and analytics
- [x] Phase 32: ContinuWitty query prefix (`cw>`) + federated linked engram recall

## Documentation Debt

- [x] Keep `migration/checkpoints/checkpoint.md` updated after each phase completion
- [x] Keep `AGENT.md` aligned with architecture changes
- [x] Update skills docs when domain behavior changes
- [x] Keep `docs/api-reference.md` current when API surface changes
- [x] Keep `docs/mcp-guide.md` current when MCP tools change

## Technical Debt

- [ ] Production security hardening follow-up (OIDC provider rollout validation + centralized audit sink integration)
- [x] Expand auth abuse-path tests (OIDC callback replay after pending-state consumption)
- [ ] Swap local deterministic embeddings for a real embedding model (sentence-transformers or hosted provider)
- [ ] Cross-provider engram reuse validation (same stored engram with different LLM providers)

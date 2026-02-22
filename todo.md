# Engram Vault - Todo Tracker

> Forward-looking work items.
> See [Plan.md](Plan.md) for full roadmap. See [checkpoint.md](checkpoint.md) for completed milestones.

---

## Immediate (Current Sprint)

- [x] Phase 31 closeout: update `AGENT.md` + skills with final project-default/soft-delete/collection invariants and MCP organization contracts
- [x] Phase 31 closeout: run/record full acceptance mock execution (`make acceptance-test-mock`) with new admin-memory scenarios
- [x] Phase 18 follow-up: add explicit consolidation merge/grouping event semantics in timeline rendering
- [x] Phase 16 deferred: CLI smoke utility (`engram-cli mcp-call`)

## Near-Term (Next Phases)

- [x] Phase 33: portable export/import stash workflow (project + collections + engrams) with owner/admin authorization and round-trip tests
- [x] Phase 34: security audit remediation program (critical defaults, auth/session hardening, MCP/OAuth protections)
- [ ] Phase 19: implement project membership and scoped sharing/revocation flows with audit trails
- [ ] Phase 20: OIDC integration + distributed rate-limit strategy + production auth hardening tests
- [ ] Phase 21: observability and reliability
- [ ] Phase 22: release automation and deployment profiles
- [ ] Phase 23: EvalOps and prompt/policy governance

## Future (Post Phase 23)

- [ ] Phase 24: engram graph foundations (link schema + repository)
- [ ] Phase 25: link APIs, MCP tools, and suggestion pipeline
- [ ] Phase 26: graph-aware context assembly (configurable recall depth)
- [ ] Phase 27: traceability UX and link management UI
- [ ] Phase 28: temporal dynamics, graph quality, and analytics
- [ ] Phase 32: ContinuWitty query prefix (`cw>`) + federated linked engram recall

## Documentation Debt

- [ ] Keep `checkpoint.md` updated after each phase completion
- [ ] Keep `AGENT.md` aligned with architecture changes
- [ ] Update skills docs when domain behavior changes
- [ ] Keep `docs/api-reference.md` current when API surface changes
- [ ] Keep `docs/mcp-guide.md` current when MCP tools change

## Technical Debt

- [ ] Production security hardening (full OIDC, centralized audit sink, distributed rate limits)
- [ ] Swap local deterministic embeddings for a real embedding model (sentence-transformers or hosted provider)
- [ ] Cross-provider engram reuse validation (same stored engram with different LLM providers)

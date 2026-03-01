# EvalOps + Governance (v1)

> Phase 23 operational reference for prompt/tool policy versioning and eval regression gates.

---

## Scope

This document defines the v1 governance contract across:

- chat prompt policy version metadata
- MCP tool policy version metadata
- deterministic EvalOps suite execution and release gating

---

## Versioned Policy Metadata

### Chat

- Config: `CHAT_PROMPT_POLICY_VERSION`
- Default: `chat-prompt-policy-v1`
- Surfaced in:
  - `POST /api/v1/chat/sessions/{session_id}/messages` response (`prompt_policy_version`)
  - stream `meta` and `done` events (`prompt_policy_version`)

### MCP

- Config: `MCP_TOOL_POLICY_VERSION`
- Default: `mcp-tool-policy-v1`
- Surfaced in:
  - MCP `initialize` result `policy.tool_policy_version`

### Eval Suite

- Config: `EVAL_SUITE_VERSION`
- Default: `eval-suite-v1`
- Surfaced in:
  - `GET /api/v1/version` (`eval_suite_version`)
  - MCP `initialize` result `policy.eval_suite_version`

---

## EvalOps Suite

Runner command:

```bash
make eval
```

Implementation:

- command: `cmd/evalops/main.go`
- suite engine: `internal/evalops`
- baseline: `evals/baselines/eval-suite-v1.json`

Dimensions:

- `continuity`
- `citation_trust`
- `memory_drift`

---

## Regression Delta Gate

`make eval` enforces score deltas against:

- baseline run (`evals/baselines/eval-suite-v1.json`)
- previous historical run (`data/evals/history.jsonl`, if present)

Default thresholds:

- overall: `min_overall_delta = -0.03`
- per dimension: `min_dimension_delta = -0.05`

Gate failure blocks:

- `make eval`
- `make check`
- CI Go backend job (`Run EvalOps delta gate`)

---

## Artifacts

Generated outputs:

- `data/evals/latest.json` (latest run result)
- `data/evals/history.jsonl` (time-series history)
- `data/evals/trend-report.md` (human-readable trend report)

Generate artifacts without enforcing deltas:

```bash
make eval-report
```

---

## Release Use

`make release-gate` includes `make check`, which includes `make eval`.

This ensures no release candidate passes deterministic gates when continuity/citation/memory-drift deltas regress beyond thresholds.

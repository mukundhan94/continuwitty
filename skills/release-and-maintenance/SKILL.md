---
name: release-and-maintenance
description: Use this skill when preparing release-quality changes, updating docs, and maintaining long-run project health.
---

# Release and Maintenance

## Use This Skill When
- Finalizing phases for commit/review.
- Updating README/plan status and operational notes.
- Doing periodic maintenance or dependency updates.

## Release Checklist
1. Confirm implementation matches plan scope.
2. Run full quality gate: `make check`.
3. Verify README sections: progress, file guide, quick start, implementation log.
4. Verify AGENT and skills docs are current for new modules.
5. Commit with phase-scoped message.

## Maintenance Cadence
- Weekly: run checks and evals.
- Monthly: dependency review and provider API compatibility review.
- Per phase: refresh docs and validate local reproducibility.

## Architecture Evaluation Notes
- Record deferred architecture options explicitly (example: FastMCP adapter evaluation).
- Track them as "future consideration" in README + relevant skills.
- Prefer incremental adapter pilots over full rewrites when core workflows are already stable.

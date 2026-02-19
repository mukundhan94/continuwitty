from __future__ import annotations

from pathlib import Path


def _repo_root() -> Path:
    return Path(__file__).resolve().parents[2]


def test_makefile_exposes_combined_local_dev_target() -> None:
    makefile = (_repo_root() / "Makefile").read_text(encoding="utf-8")

    assert "dev: ## Start DB (docker) + API + Web together in one terminal" in makefile
    assert "$(DOCKER_COMPOSE) up -d --build --force-recreate db" in makefile
    assert "uv run uvicorn app.main:app" in makefile
    assert "npm run dev -- --host --port" in makefile
    assert "make dev" in makefile
    assert "Container stack (use sparingly)" in makefile


def test_agent_guide_prefers_dev_or_split_terminal_workflow() -> None:
    agent_guide = (_repo_root() / "AGENT.md").read_text(encoding="utf-8")

    assert "Preferred local run mode: `make dev`" in agent_guide
    assert "run `make api` and `make web` separately" in agent_guide
    assert "Use `make stack-up` sparingly" in agent_guide


def test_readme_quick_start_uses_new_local_run_defaults() -> None:
    readme = (_repo_root() / "README.md").read_text(encoding="utf-8")

    assert "Preferred local run mode (single terminal, starts DB + API + Web)" in readme
    assert "make dev" in readme
    assert "Optional split-terminal mode" in readme
    assert "make api" in readme
    assert "make web" in readme
    assert "make stack-up" in readme
    assert "occasional debugging mode" in readme

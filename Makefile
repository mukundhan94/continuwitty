SHELL := /bin/zsh

-include .env
export

.PHONY: db-up db-down db-logs sync api cli lint format format-check check test test-unit test-integration eval

db-up:
	docker compose up -d db

db-down:
	docker compose down

db-logs:
	docker compose logs -f db

sync:
	cd api && uv sync --group dev

api:
	cd api && uv run uvicorn app.main:app --host $${API_HOST:-0.0.0.0} --port $${API_PORT:-8000} --reload

cli:
	cd api && uv run python -m app.cli $(ARGS)

lint:
	cd api && uv run ruff check .

format:
	cd api && uv run ruff format .

format-check:
	cd api && uv run ruff format --check .

test:
	cd api && uv run pytest -q

test-unit:
	cd api && uv run pytest -q -m "not integration"

test-integration:
	cd api && uv run pytest -q -m integration

eval:
	cd api && uv run python -m evals.run_eval --out evals/last_eval.json

check: lint format-check test eval

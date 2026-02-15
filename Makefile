SHELL := /bin/zsh

-include .env
export

.PHONY: db-up db-down db-logs stack-up stack-down stack-logs acceptance-sync acceptance-typecheck acceptance-test acceptance-test-docker sync api cli consolidate lint format format-check check test test-unit test-integration eval web-sync web-dev web-lint web-test web-build web-check

db-up:
	docker compose up -d db

db-down:
	docker compose down

db-logs:
	docker compose logs -f db

stack-up:
	docker compose up -d db api web

stack-down:
	docker compose down

stack-logs:
	docker compose logs -f db api web

acceptance-sync:
	cd acceptance-tests && npm install

acceptance-typecheck:
	cd acceptance-tests && npm run typecheck

acceptance-test:
	cd acceptance-tests && npm run test

acceptance-test-docker:
	@exit_code=0; \
	docker compose --profile acceptance up --build --abort-on-container-exit acceptance-tests || exit_code=$$?; \
	docker compose --profile acceptance down; \
	exit $$exit_code

sync:
	cd api && uv sync --group dev

api:
	cd api && uv run uvicorn app.main:app --host $${API_HOST:-0.0.0.0} --port $${API_PORT:-8000} --reload

cli:
	cd api && uv run python -m app.cli $(ARGS)

consolidate:
	cd api && uv run python -m app.cli consolidate $(ARGS)

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

web-sync:
	cd web && npm install

web-dev:
	cd web && npm run dev -- --host

web-lint:
	cd web && npm run lint

web-test:
	cd web && npm run test

web-build:
	cd web && npm run build

web-check: web-lint web-test web-build

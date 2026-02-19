SHELL := /bin/zsh

-include .env
export

DIAGRAM_PUML_FILES := docs/architecture-workflows.puml docs/model-switch-engram-usecases.puml

.PHONY: db-up db-down db-reset db-logs stack-up stack-down stack-reset stack-logs acceptance-sync acceptance-bddgen acceptance-typecheck acceptance-test acceptance-test-mock acceptance-test-bedrock-live acceptance-test-triage-live acceptance-test-docker acceptance-test-mock-docker acceptance-test-bedrock-live-docker acceptance-test-triage-live-docker sync api cli consolidate lint format format-check check test test-unit test-integration eval web-sync web web-lint web-test web-build web-check diagram-render diagram-render-png

db-up:
	docker compose up -d --build --force-recreate db

db-down:
	docker compose down

db-reset:
	docker compose down -v
	rm -f data/langgraph_checkpoints.sqlite data/audit_events.jsonl
	docker compose up -d --build --force-recreate db

db-logs:
	docker compose logs -f db

stack-up:
	docker compose up -d --build --force-recreate db api web

stack-down:
	docker compose down

stack-reset:
	docker compose down -v
	rm -f data/langgraph_checkpoints.sqlite data/audit_events.jsonl
	docker compose up -d --build --force-recreate db api web

stack-logs:
	docker compose logs -f db api web

acceptance-sync:
	cd acceptance-tests && npm install

acceptance-bddgen:
	cd acceptance-tests && npm run bdd:gen

acceptance-typecheck:
	cd acceptance-tests && npm run typecheck

acceptance-test:
	cd acceptance-tests && npm run test

acceptance-test-mock:
	cd acceptance-tests && npm run test:mock

acceptance-test-bedrock-live:
	cd acceptance-tests && npm run test:bedrock-live

acceptance-test-triage-live:
	cd acceptance-tests && npm run test:triage-live

acceptance-test-docker:
	@exit_code=0; \
	docker compose --profile acceptance up --build --force-recreate --abort-on-container-exit acceptance-tests || exit_code=$$?; \
	docker compose --profile acceptance down; \
	exit $$exit_code

acceptance-test-mock-docker:
	@exit_code=0; \
	ACCEPTANCE_BDD_TAGS='@mock' docker compose --profile acceptance up --build --force-recreate --abort-on-container-exit acceptance-tests || exit_code=$$?; \
	docker compose --profile acceptance down; \
	exit $$exit_code

acceptance-test-bedrock-live-docker:
	@exit_code=0; \
	ACCEPTANCE_BDD_TAGS='@bedrock-live' docker compose --profile acceptance up --build --force-recreate --abort-on-container-exit acceptance-tests || exit_code=$$?; \
	docker compose --profile acceptance down; \
	exit $$exit_code

acceptance-test-triage-live-docker:
	@exit_code=0; \
	ACCEPTANCE_BDD_TAGS='@triage-live' docker compose --profile acceptance up --build --force-recreate --abort-on-container-exit acceptance-tests || exit_code=$$?; \
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

web:
	cd web && npm run dev -- --host

web-lint:
	cd web && npm run lint

web-test:
	cd web && npm run test

web-build:
	cd web && npm run build

web-check: web-lint web-test web-build

diagram-render:
	./docs/render-plantuml.sh svg $(DIAGRAM_PUML_FILES)

diagram-render-png:
	./docs/render-plantuml.sh png $(DIAGRAM_PUML_FILES)

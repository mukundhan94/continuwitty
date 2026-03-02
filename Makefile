SHELL := /bin/bash

-include .env
export

# Docker / paths
DOCKER_COMPOSE_DIR ?= .
DOCKER_COMPOSE_FILE ?= $(DOCKER_COMPOSE_DIR)/docker-compose.yml
DOCKER_COMPOSE := docker compose -f $(DOCKER_COMPOSE_FILE)
ACCEPTANCE_DIR ?= ./acceptance-tests

# Diagram assets
DIAGRAM_PUML_FILES := docs/architecture-workflows.puml docs/model-switch-engram-usecases.puml

# Terminal colors
SUCCESS = \033[0;32m
PROGRESS = \033[1;35m
ERROR = \033[0;31m
INFO = \033[1;36m
NC = \033[0m

.DEFAULT_GOAL := help

WEB_PORT ?= 5173

.PHONY: help print-config db-up db-down db-reset db-logs stack-up stack-down stack-reset stack-logs stack-smoke release-smoke-docker release-gate release-live-provider-gate acceptance-sync acceptance-bddgen acceptance-typecheck acceptance-test acceptance-test-mock acceptance-test-bedrock-live acceptance-test-triage-live acceptance-test-docker acceptance-test-mock-docker acceptance-test-bedrock-live-docker acceptance-test-triage-live-docker sync dev api lint format format-check check test test-unit test-integration coverage eval eval-report web-sync web web-lint web-test web-build web-check diagram-render diagram-render-png

help: ## Print all Makefile commands with categorized descriptions and usage hints
	@printf '$(INFO)Engram Make Command Reference$(NC)\n'
	@printf '$(PROGRESS)Usage:$(NC) $(INFO)make <target>$(NC)\n'
	@printf '$(PROGRESS)Example:$(NC) $(INFO)make dev$(NC)\n\n'
	@awk ' \
		BEGIN { \
			FS = ":.*## "; \
			c_reset = "\033[0m"; \
			c_group = "\033[1;33m"; \
			c_cmd = "\033[1;34m"; \
			c_desc = "\033[0;37m"; \
		} \
		function group_name(target) { \
			if (target == "help" || target == "print-config") return "Help"; \
			if (target ~ /^db-/) return "Database"; \
			if (target ~ /^stack-/) return "Stack"; \
			if (target == "dev") return "Local Dev"; \
			if (target ~ /^release-/) return "Release"; \
			if (target ~ /^acceptance-/) return "Acceptance"; \
			if (target ~ /^web-/ || target == "web") return "Web"; \
			if (target ~ /^diagram-/) return "Diagrams"; \
			if (target == "sync" || target == "api" || target == "lint" || target == "format" || target == "format-check" || target == "test" || target == "test-unit" || target == "test-integration" || target == "coverage" || target == "eval" || target == "eval-report" || target == "check") return "API/Backend"; \
			return "Other"; \
		} \
		/^[a-zA-Z0-9_.-]+:.*## / { \
			target = $$1; \
			desc = $$2; \
			group = group_name(target); \
			if (!(group in seen)) { \
				seen[group] = 1; \
				order[++count] = group; \
			} \
			lines[group] = lines[group] sprintf("  %s%-34s%s %s%s%s\n", c_cmd, target, c_reset, c_desc, desc, c_reset); \
		} \
		END { \
			for (i = 1; i <= count; i++) { \
				group = order[i]; \
				printf "%s%s%s\n", c_group, group, c_reset; \
				printf "%s", lines[group]; \
				printf "\n"; \
			} \
		} \
	' Makefile
	@printf '$(SUCCESS)Recommended flows:$(NC)\n'
	@printf '  $(INFO)make sync && make web-sync$(NC)                  Setup dependencies\n'
	@printf '  $(INFO)make dev$(NC)                                    Run db + api + web in one terminal\n'
	@printf '  $(INFO)make api$(NC) + $(INFO)make web$(NC)                          Manual split-terminal flow\n'
	@printf '  $(INFO)make stack-up$(NC)                               Container stack (use sparingly)\n'
	@printf '  $(INFO)make check && make web-check$(NC)                Run backend + web quality gates\n'
	@printf '  $(INFO)make acceptance-bddgen && make acceptance-test-mock$(NC)  Run deterministic acceptance tests\n'
	@printf '  $(INFO)make release-gate$(NC)                            Run release candidate deterministic gates\n'
	@printf '\n'
	@$(MAKE) --no-print-directory print-config

print-config: ## Print key Makefile configuration values for local debugging
	@printf '$(SUCCESS)Current configuration:$(NC)\n'
	@printf '  $(PROGRESS)DOCKER_COMPOSE_FILE$(NC): %s\n' "$(DOCKER_COMPOSE_FILE)"
	@printf '  $(PROGRESS)ACCEPTANCE_DIR$(NC):    %s\n' "$(ACCEPTANCE_DIR)"
	@printf '  $(PROGRESS)DIAGRAM_PUML_FILES$(NC): %s\n' "$(DIAGRAM_PUML_FILES)"
	@printf '  $(PROGRESS)API_HOST$(NC):          %s\n' "$${API_HOST:-0.0.0.0}"
	@printf '  $(PROGRESS)API_PORT$(NC):          %s\n' "$${API_PORT:-8000}"
	@printf '  $(PROGRESS)WEB_PORT$(NC):          %s\n' "$${WEB_PORT:-5173}"

db-up: ## Start only the database container (build + force recreate)
	@printf '$(PROGRESS)Starting database service...$(NC)\n'
	@$(DOCKER_COMPOSE) up -d --build --force-recreate db
	@printf '$(SUCCESS)✓ Database is up$(NC)\n'

db-down: ## Stop the current compose stack
	@printf '$(PROGRESS)Stopping compose stack...$(NC)\n'
	@$(DOCKER_COMPOSE) down
	@printf '$(SUCCESS)✓ Compose stack stopped$(NC)\n'

db-reset: ## Recreate database from scratch (drops volumes + local checkpoint artifacts)
	@printf '$(PROGRESS)Resetting database volumes and local artifacts...$(NC)\n'
	@$(DOCKER_COMPOSE) down -v
	@rm -f data/langgraph_checkpoints.sqlite data/audit_events.jsonl
	@$(DOCKER_COMPOSE) up -d --build --force-recreate db
	@printf '$(SUCCESS)✓ Database reset complete$(NC)\n'

db-logs: ## Tail database logs
	@printf '$(PROGRESS)Tailing database logs (Ctrl+C to exit)...$(NC)\n'
	@$(DOCKER_COMPOSE) logs -f db

stack-up: ## Start db + api + web containers (build + force recreate)
	@printf '$(PROGRESS)Starting db + api + web services...$(NC)\n'
	@$(DOCKER_COMPOSE) up -d --build --force-recreate db api web
	@printf '$(SUCCESS)✓ Stack started (db, api, web)$(NC)\n'
	@printf '$(INFO)Tip: run make stack-logs to follow logs$(NC)\n'

stack-down: ## Stop db + api + web containers
	@printf '$(PROGRESS)Stopping db + api + web services...$(NC)\n'
	@$(DOCKER_COMPOSE) down
	@printf '$(SUCCESS)✓ Stack stopped$(NC)\n'

stack-reset: ## Recreate full stack and wipe local checkpoints/artifacts
	@printf '$(PROGRESS)Resetting full stack and local artifacts...$(NC)\n'
	@$(DOCKER_COMPOSE) down -v
	@rm -f data/langgraph_checkpoints.sqlite data/audit_events.jsonl
	@$(DOCKER_COMPOSE) up -d --build --force-recreate db api web
	@printf '$(SUCCESS)✓ Full stack reset complete$(NC)\n'

stack-logs: ## Tail combined logs for db, api, and web
	@printf '$(PROGRESS)Tailing stack logs (db, api, web). Press Ctrl+C to exit.$(NC)\n'
	@$(DOCKER_COMPOSE) logs -f db api web

stack-smoke: ## Build/start db + api and verify Go API health/version endpoints
	@printf '$(PROGRESS)Starting db + api for smoke verification...$(NC)\n'
	@$(DOCKER_COMPOSE) up -d --build --force-recreate db api
	@printf '$(PROGRESS)Waiting for API health endpoint...$(NC)\n'
	@ready=0; \
	for attempt in $$(seq 1 30); do \
		if curl -fsS "http://127.0.0.1:$${API_PORT:-8000}/healthz" >/dev/null 2>&1; then \
			ready=1; \
			break; \
		fi; \
		sleep 1; \
	done; \
	if [ "$$ready" -ne 1 ]; then \
		printf '$(ERROR)API healthcheck did not become ready in time$(NC)\n'; \
		exit 1; \
	fi
	@curl -fsS "http://127.0.0.1:$${API_PORT:-8000}/api/v1/version" >/dev/null
	@printf '$(SUCCESS)✓ Go API container smoke checks passed$(NC)\n'

release-smoke-docker: ## Run release smoke checks in Docker compose profile "release-smoke"
	@printf '$(PROGRESS)Running release smoke checks in Docker profile "release-smoke"...$(NC)\n'
	@exit_code=0; \
	$(DOCKER_COMPOSE) --profile release-smoke up --build --force-recreate --abort-on-container-exit release-smoke || exit_code=$$?; \
	$(DOCKER_COMPOSE) --profile release-smoke down; \
	exit $$exit_code
	@printf '$(SUCCESS)✓ Docker release smoke checks completed$(NC)\n'

acceptance-sync: ## Install acceptance test dependencies
	@printf '$(PROGRESS)Installing acceptance test dependencies...$(NC)\n'
	@cd $(ACCEPTANCE_DIR) && npm install
	@printf '$(SUCCESS)✓ Acceptance dependencies installed$(NC)\n'

acceptance-bddgen: ## Generate Playwright-BDD test artifacts from feature files
	@printf '$(PROGRESS)Generating Playwright-BDD artifacts...$(NC)\n'
	@cd $(ACCEPTANCE_DIR) && npm run bdd:gen
	@printf '$(SUCCESS)✓ BDD generation complete$(NC)\n'

acceptance-typecheck: ## Type-check acceptance test code
	@printf '$(PROGRESS)Type-checking acceptance tests...$(NC)\n'
	@cd $(ACCEPTANCE_DIR) && npm run typecheck
	@printf '$(SUCCESS)✓ Acceptance type-check passed$(NC)\n'

acceptance-test: ## Run full acceptance suite
	@printf '$(PROGRESS)Running full acceptance suite...$(NC)\n'
	@cd $(ACCEPTANCE_DIR) && npm run test
	@printf '$(SUCCESS)✓ Acceptance suite completed$(NC)\n'

acceptance-test-mock: ## Run deterministic mock acceptance scenarios
	@printf '$(PROGRESS)Running deterministic mock acceptance scenarios...$(NC)\n'
	@cd $(ACCEPTANCE_DIR) && npm run test:mock
	@printf '$(SUCCESS)✓ Mock acceptance scenarios completed$(NC)\n'

acceptance-test-bedrock-live: ## Run live Bedrock acceptance scenarios
	@printf '$(PROGRESS)Running live Bedrock acceptance scenarios...$(NC)\n'
	@cd $(ACCEPTANCE_DIR) && npm run test:bedrock-live
	@printf '$(SUCCESS)✓ Bedrock live acceptance completed$(NC)\n'

acceptance-test-triage-live: ## Run live triage acceptance scenario
	@printf '$(PROGRESS)Running live triage acceptance scenario...$(NC)\n'
	@cd $(ACCEPTANCE_DIR) && npm run test:triage-live
	@printf '$(SUCCESS)✓ Triage live acceptance completed$(NC)\n'

acceptance-test-docker: ## Run full acceptance suite inside docker compose profile
	@printf '$(PROGRESS)Running full acceptance suite in Docker profile \"acceptance\"...$(NC)\n'
	@exit_code=0; \
	$(DOCKER_COMPOSE) --profile acceptance up --build --force-recreate --abort-on-container-exit acceptance-tests || exit_code=$$?; \
	$(DOCKER_COMPOSE) --profile acceptance down; \
	exit $$exit_code
	@printf '$(SUCCESS)✓ Docker acceptance suite completed$(NC)\n'

acceptance-test-mock-docker: ## Run mock acceptance suite in docker (ACCEPTANCE_BDD_TAGS=@mock)
	@printf '$(PROGRESS)Running mock acceptance suite in Docker...$(NC)\n'
	@exit_code=0; \
	ACCEPTANCE_BDD_TAGS='@mock' $(DOCKER_COMPOSE) --profile acceptance up --build --force-recreate --abort-on-container-exit acceptance-tests || exit_code=$$?; \
	$(DOCKER_COMPOSE) --profile acceptance down; \
	exit $$exit_code
	@printf '$(SUCCESS)✓ Docker mock acceptance suite completed$(NC)\n'

acceptance-test-bedrock-live-docker: ## Run live Bedrock acceptance in docker (ACCEPTANCE_BDD_TAGS=@bedrock-live)
	@printf '$(PROGRESS)Running Bedrock live acceptance in Docker...$(NC)\n'
	@exit_code=0; \
	ACCEPTANCE_BDD_TAGS='@bedrock-live' $(DOCKER_COMPOSE) --profile acceptance up --build --force-recreate --abort-on-container-exit acceptance-tests || exit_code=$$?; \
	$(DOCKER_COMPOSE) --profile acceptance down; \
	exit $$exit_code
	@printf '$(SUCCESS)✓ Docker Bedrock acceptance suite completed$(NC)\n'

acceptance-test-triage-live-docker: ## Run live triage acceptance in docker (ACCEPTANCE_BDD_TAGS=@triage-live)
	@printf '$(PROGRESS)Running triage live acceptance in Docker...$(NC)\n'
	@exit_code=0; \
	ACCEPTANCE_BDD_TAGS='@triage-live' $(DOCKER_COMPOSE) --profile acceptance up --build --force-recreate --abort-on-container-exit acceptance-tests || exit_code=$$?; \
	$(DOCKER_COMPOSE) --profile acceptance down; \
	exit $$exit_code
	@printf '$(SUCCESS)✓ Docker triage acceptance suite completed$(NC)\n'

release-live-provider-gate: ## Run optional live-provider release gate (Bedrock + triage)
	@printf '$(PROGRESS)Running optional live-provider release gate...$(NC)\n'
	@$(MAKE) --no-print-directory acceptance-test-bedrock-live-docker
	@$(MAKE) --no-print-directory acceptance-test-triage-live-docker
	@printf '$(SUCCESS)✓ Live-provider release gate completed$(NC)\n'

sync: ## Sync Go module dependencies
	@printf '$(PROGRESS)Syncing Go module dependencies...$(NC)\n'
	@go mod download
	@printf '$(SUCCESS)✓ Go dependencies synced$(NC)\n'

dev: ## Start DB (docker) + Go API + Web together in one terminal (Ctrl+C stops both local servers)
	@printf '$(PROGRESS)Ensuring database container is running...$(NC)\n'
	@$(DOCKER_COMPOSE) up -d --build --force-recreate db
	@printf '$(SUCCESS)✓ Database ready$(NC)\n'
	@printf '$(PROGRESS)Starting API and Web in one terminal (Ctrl+C to stop)...$(NC)\n'
	@printf '  $(INFO)API:$(NC) http://localhost:%s\n' "$${API_PORT:-8000}"
	@printf '  $(INFO)WEB:$(NC) http://localhost:%s\n' "$${WEB_PORT:-5173}"
	@trap 'printf "\n$(PROGRESS)Stopping local dev servers...$(NC)\n"; kill $$api_pid $$web_pid >/dev/null 2>&1 || true' INT TERM EXIT; \
		(go run ./cmd/api 2>&1 | sed -e "s/^/[api] /") & api_pid=$$!; \
		(cd web && npm run dev -- --host --port $${WEB_PORT:-5173} 2>&1 | sed -e "s/^/[web] /") & web_pid=$$!; \
		wait $$api_pid $$web_pid

api: ## Run Go API server in local dev mode
	@printf '$(PROGRESS)Starting Go API server...$(NC)\n'
	@printf '  $(INFO)Host:$(NC) %s\n' "$${API_HOST:-0.0.0.0}"
	@printf '  $(INFO)Port:$(NC) %s\n' "$${API_PORT:-8000}"
	@go run ./cmd/api

lint: ## Run Go vet checks
	@printf '$(PROGRESS)Running go vet checks...$(NC)\n'
	@go vet ./...
	@printf '$(SUCCESS)✓ go vet checks passed$(NC)\n'

format: ## Apply gofmt to backend Go files
	@printf '$(PROGRESS)Applying gofmt...$(NC)\n'
	@find cmd internal -name '*.go' -type f -print0 | xargs -0 gofmt -w
	@printf '$(SUCCESS)✓ gofmt applied$(NC)\n'

format-check: ## Verify backend Go formatting without changing files
	@printf '$(PROGRESS)Checking Go code formatting...$(NC)\n'
	@unformatted="$$(gofmt -l $$(find cmd internal -name '*.go' -type f))"; \
	if [ -n "$$unformatted" ]; then \
		printf '$(ERROR)Go files require formatting. Run make format$(NC)\n'; \
		printf '%s\n' "$$unformatted"; \
		exit 1; \
	fi
	@printf '$(SUCCESS)✓ Go format check passed$(NC)\n'

test: ## Run full backend Go test suite
	@printf '$(PROGRESS)Running full backend Go test suite...$(NC)\n'
	@go test ./... -count=1
	@printf '$(SUCCESS)✓ Backend Go tests passed$(NC)\n'

test-unit: ## Run backend Go unit test suite
	@printf '$(PROGRESS)Running backend Go unit tests...$(NC)\n'
	@go test ./... -count=1
	@printf '$(SUCCESS)✓ Backend Go unit tests passed$(NC)\n'

test-integration: ## Run backend Go integration test suite
	@printf '$(PROGRESS)Running backend Go integration tests...$(NC)\n'
	@go test ./... -count=1
	@printf '$(SUCCESS)✓ Backend Go integration tests passed$(NC)\n'

coverage: ## Run backend Go coverage and write coverage.out
	@printf '$(PROGRESS)Running backend Go coverage...$(NC)\n'
	@go test ./... -coverprofile=coverage.out -covermode=atomic
	@go tool cover -func=coverage.out | tail -n 1
	@printf '$(SUCCESS)✓ Backend Go coverage completed (coverage.out)$(NC)\n'

eval: ## Run EvalOps suite + baseline/previous delta gate and write reports
	@printf '$(PROGRESS)Running EvalOps suite with regression delta gate...$(NC)\n'
	@go run ./cmd/evalops \
		-out data/evals/latest.json \
		-history data/evals/history.jsonl \
		-baseline evals/baselines/eval-suite-v1.json \
		-trend-report data/evals/trend-report.md \
		-enforce-delta-gate=true
	@printf '$(SUCCESS)✓ EvalOps suite passed (data/evals/latest.json)$(NC)\n'

eval-report: ## Generate EvalOps run + trend report without enforcing delta gate
	@printf '$(PROGRESS)Generating EvalOps trend report (delta gate disabled)...$(NC)\n'
	@go run ./cmd/evalops \
		-out data/evals/latest.json \
		-history data/evals/history.jsonl \
		-baseline evals/baselines/eval-suite-v1.json \
		-trend-report data/evals/trend-report.md \
		-enforce-delta-gate=false
	@printf '$(SUCCESS)✓ EvalOps trend report updated$(NC)\n'

check: lint format-check test eval ## Run backend Go quality gate (vet + format-check + tests + eval gate)

web-sync: ## Install web dependencies
	@printf '$(PROGRESS)Installing web dependencies...$(NC)\n'
	@cd web && npm install
	@printf '$(SUCCESS)✓ Web dependencies installed$(NC)\n'

web: ## Run Vite dev server on all interfaces
	@printf '$(PROGRESS)Starting web dev server...$(NC)\n'
	@printf '  $(INFO)Host:$(NC) %s\n' "0.0.0.0"
	@printf '  $(INFO)Port:$(NC) %s\n' "$${WEB_PORT:-5173}"
	@cd web && npm run dev -- --host --port $${WEB_PORT:-5173}

web-lint: ## Run web lint checks
	@printf '$(PROGRESS)Running web lint checks...$(NC)\n'
	@cd web && npm run lint
	@printf '$(SUCCESS)✓ Web lint checks passed$(NC)\n'

web-test: ## Run web unit/integration tests
	@printf '$(PROGRESS)Running web tests...$(NC)\n'
	@cd web && npm run test
	@printf '$(SUCCESS)✓ Web tests passed$(NC)\n'

web-build: ## Build production web bundle
	@printf '$(PROGRESS)Building web production bundle...$(NC)\n'
	@cd web && npm run build
	@printf '$(SUCCESS)✓ Web build completed$(NC)\n'

web-check: web-lint web-test web-build ## Run full web quality gate (lint + tests + build)

release-gate: check web-check acceptance-test-mock-docker release-smoke-docker ## Run deterministic release candidate gate

diagram-render: ## Render PlantUML diagrams to SVG
	@printf '$(PROGRESS)Rendering PlantUML diagrams to SVG...$(NC)\n'
	@./docs/render-plantuml.sh svg $(DIAGRAM_PUML_FILES)
	@printf '$(SUCCESS)✓ SVG diagrams generated$(NC)\n'

diagram-render-png: ## Render PlantUML diagrams to PNG
	@printf '$(PROGRESS)Rendering PlantUML diagrams to PNG...$(NC)\n'
	@./docs/render-plantuml.sh png $(DIAGRAM_PUML_FILES)
	@printf '$(SUCCESS)✓ PNG diagrams generated$(NC)\n'

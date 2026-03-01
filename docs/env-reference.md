# Engram Vault - Environment Variables

> Complete reference for all configuration environment variables.
> See [README](../README.md) for quick start. See [user-workflows.md](user-workflows.md) for usage context.
> Canonical starter file: [`.env.example`](../.env.example)

---

## Database

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_USER` | `engram` | Database username |
| `POSTGRES_PASSWORD` | `engram` | Database password |
| `POSTGRES_DB` | `engram` | Database name |
| `POSTGRES_PORT` | `5432` | Database port |

## API Runtime

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | Environment mode |
| `LOG_CONFIG_IN_DEV` | `true` | Print redacted parsed-config snapshot on startup in dev/local envs |
| `APP_SEMANTIC_VERSION` | - | Application version string |
| `APP_COMMIT_SHA` | - | Git commit SHA |
| `API_REQUEST_LOG_ENABLED` | `true` | Enable structured per-request telemetry logs |
| `API_METRICS_ENABLED` | `true` | Enable in-process request metrics endpoint (`/api/v1/metrics`) |
| `APP_SESSION_SECRET` | - | Session signing secret |

## Security

| Variable | Default | Description |
|----------|---------|-------------|
| `AUDIT_LOG_PATH` | `./data/audit_events.jsonl` | Local audit event log path |
| `AUDIT_LOG_STDOUT_ENABLED` | `false` | Emit audit events to stdout (production emits to stdout automatically) |
| `AUDIT_LOG_MAX_EVENT_BYTES` | `32768` | Maximum serialized bytes per audit event before truncation |
| `LOGIN_RATE_LIMIT_MAX_ATTEMPTS` | `5` | Max failed login attempts before lockout |
| `LOGIN_RATE_LIMIT_WINDOW_SECONDS` | `300` | Rate-limit sliding window |
| `LOGIN_LOCKOUT_SECONDS` | `900` | Lockout duration after max attempts |
| `MCP_TRANSPORT_RATE_LIMIT_MAX_REQUESTS` | `120` | Max MCP transport requests per window per actor key |
| `MCP_TRANSPORT_RATE_LIMIT_WINDOW_SECONDS` | `60` | MCP transport rate-limit window |
| `MCP_TRANSPORT_RATE_LIMIT_BLOCK_SECONDS` | `30` | MCP transport block duration once threshold is exceeded |

## OIDC Login

| Variable | Default | Description |
|----------|---------|-------------|
| `OIDC_ENABLED` | `false` | Enable external OIDC login flow (`/login/oidc`) |
| `OIDC_ISSUER_URL` | - | OIDC issuer URL used for discovery |
| `OIDC_CLIENT_ID` | - | OIDC client ID |
| `OIDC_CLIENT_SECRET` | - | OIDC client secret |
| `OIDC_REDIRECT_URL` | `http://localhost:8000/login/oidc/callback` | Redirect URI registered with the OIDC provider |
| `OIDC_SCOPES` | `openid profile email` | Requested OIDC scopes |
| `OIDC_USERNAME_CLAIM` | `email` | Preferred ID-token claim mapped to local username lookup |

## Debug / Observability

| Variable | Default | Description |
|----------|---------|-------------|
| `CHAT_DEBUG_ENABLED` | `true` | Attach debug trace payloads to chat responses |
| `CHAT_DEBUG_LOG_CONSOLE` | `true` | Print chat debug payloads in API logs |
| `CHAT_DEBUG_INCLUDE_RAW_TEXT` | `true` | Include request/response text in debug payloads |
| `LANGFUSE_ENABLED` | `false` | Publish traces to Langfuse when configured |
| `LANGFUSE_HOST` | `https://cloud.langfuse.com` | Langfuse endpoint |
| `LANGFUSE_PUBLIC_KEY` | - | Langfuse public key |
| `LANGFUSE_SECRET_KEY` | - | Langfuse secret key |

## Providers

| Variable | Default | Description |
|----------|---------|-------------|
| `DEFAULT_CHAT_PROVIDER` | `openai` | Default LLM provider |
| `DEFAULT_CHAT_MODEL` | `gpt-4o-mini` | Default LLM model ID |
| `OPENAI_API_KEY` | - | OpenAI API key |
| `OPENAI_BASE_URL` | - | OpenAI base URL override |
| `ANTHROPIC_API_KEY` | - | Anthropic API key |
| `ANTHROPIC_BASE_URL` | - | Anthropic base URL override |
| `ANTHROPIC_VERSION` | - | Anthropic API version |
| `AWS_REGION` | - | AWS region for Bedrock |
| `AWS_ACCESS_KEY_ID` | - | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | - | AWS secret key |
| `AWS_SESSION_TOKEN` | - | AWS session token (optional) |

## Embedding

| Variable | Default | Description |
|----------|---------|-------------|
| `EMBEDDING_DIM` | `256` | Embedding vector dimension |
| `EMBEDDING_PROVIDER` | `local` | Embedding provider (`local` or `openai`) |
| `EMBEDDING_MODEL` | - | External embedding model name |
| `EMBEDDING_FALLBACK_TO_LOCAL` | `true` | Fall back to local deterministic embeddings |

## Ingestion

| Variable | Default | Description |
|----------|---------|-------------|
| `INGESTION_MAX_FILE_BYTES` | - | Maximum file upload size |
| `INGESTION_MAX_TEXT_CHARS` | - | Maximum text ingestion length |
| `INGESTION_MAX_METADATA_JSON_BYTES` | `20000` | Maximum metadata JSON payload size for file ingestion |

## Web Runtime

| Variable | Default | Description |
|----------|---------|-------------|
| `WEB_PORT` | `5173` | Web dev server port |
| `VITE_API_PROXY_TARGET` | `http://localhost:8000` | API proxy target for Vite dev server |
| `VITE_ALLOWED_HOSTS` | - | Allowed hostnames for Vite dev server |
| `VITE_DEFAULT_*` | - | Frontend default configuration values |

## Acceptance Tests

| Variable | Default | Description |
|----------|---------|-------------|
| `ACCEPTANCE_WEB_BASE_URL` | - | Base URL for acceptance test web target |
| `ACCEPTANCE_API_BASE_URL` | - | Base URL for acceptance test API target |
| `ACCEPTANCE_BDD_TAGS` | - | BDD tag filter for acceptance runs |
| `BEDROCK_LIVE_EXPECTED_MODEL` | - | Expected model ID for Bedrock live tests |
| `BEDROCK_LIVE_PROMPT` | - | Prompt for Bedrock live test scenario |
| `BEDROCK_LIVE_MIN_RESPONSE_CHARS` | - | Minimum response length for Bedrock validation |
| `PW_HEADLESS` | `true` | Run Playwright in headless mode |
| `PW_TIMEOUT_MS` | - | Playwright test timeout |
| `UI_USERNAME` | - | Test UI username |
| `UI_PASSWORD` | - | Test UI password |

## OAuth

| Variable | Default | Description |
|----------|---------|-------------|
| `OAUTH_ENABLED` | `true` | Enable OAuth authorization server |
| `OAUTH_ISSUER_URL` | - | OAuth issuer URL |
| `OAUTH_REQUIRE_PROTECTED_REGISTRATION` | `false` | Require authenticated admin session for dynamic client registration (must be `true` in production) |
| `OAUTH_ACCESS_TOKEN_TTL_SECONDS` | - | OAuth access token lifetime |
| `OAUTH_AUTHORIZATION_CODE_TTL_SECONDS` | - | OAuth authorization code lifetime |
| `OAUTH_CLIENT_SECRET_PEPPER` | - | Pepper for OAuth client secret hashing |

## MCP Tokens

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_TOKEN_PEPPER` | - | Pepper for MCP token secret hashing |

## LangGraph

| Variable | Default | Description |
|----------|---------|-------------|
| `LANGGRAPH_CHECKPOINT_PATH` | `./data/langgraph_checkpoints.sqlite` | SQLite checkpoint file for LangGraph |

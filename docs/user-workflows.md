# Engram Vault - User Workflows

> Step-by-step workflows for local development, testing, and administration.
> See [UI User Flow](user-flow-engram-workflow.md) for visual screenshot walkthrough.
> See [Testing Guide](testing-guide.md) for test commands. See [Environment Variables](env-reference.md) for configuration.

---

## Dockerized Stack (API + Web + DB)

Use this when you want one-command local infrastructure with no host-level Python/Node runtimes.

1. Copy env:
```bash
cp .env.example .env
```

2. Start stack:
```bash
make stack-up
docker compose ps
```

3. Open:
- [http://localhost:8000/healthz](http://localhost:8000/healthz)
- [http://localhost:5173](http://localhost:5173) (React chat workbench)

4. Stop stack:
```bash
make stack-down
```

Fresh clean reset commands (drops DB volume and local runtime files):
```bash
make db-reset
make stack-reset
```

---

## Local Validation Sequence

Use this exact flow to validate current local behavior end-to-end.

1. Start infra:
```bash
make db-up
docker compose ps
```

2. Sync environment:
```bash
make sync
```

3. Run static quality checks:
```bash
make lint
make format-check
```

4. Run automated tests:
```bash
make test
```

5. (Optional legacy) Run memory evaluation harness:
```bash
make eval
```

6. Manual API smoke test:
```bash
make api
```
Open `http://localhost:8000/login`, sign in, then use `/ui` to run create/list/query/rehydrate from the dashboard.
For admin role validation, open `http://localhost:8000/ui/admin` and verify user list visibility.

Then test durable agent runs:

```bash
curl -X POST http://localhost:8000/api/v1/agent-runs \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "engram-vault",
    "thread_id": "thread-manual-001",
    "objective": "Validate checkpoint + resume flow",
    "notes": ["Initial run note"],
    "auto_persist_engram": true
  }'
```

```bash
curl -X POST http://localhost:8000/api/v1/agent-runs/thread-manual-001/resume \
  -H "Content-Type: application/json" \
  -d '{
    "notes": ["Resumed run note"],
    "auto_persist_engram": false
  }'
```

```bash
curl http://localhost:8000/api/v1/agent-runs/thread-manual-001
```

Then test periodic snapshots:

```bash
curl -X POST http://localhost:8000/api/v1/agent-runs \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "engram-vault",
    "thread_id": "thread-snapshot-001",
    "objective": "Validate periodic snapshot behavior",
    "notes": ["note-1", "note-2", "note-3"],
    "snapshot_enabled": true,
    "snapshot_every_n_notes": 2,
    "auto_persist_engram": false
  }'
```

7. CLI smoke test:
```bash
make cli ARGS="search --query 'local-first memory' --project-id engram-vault --top-k 3"
```

8. Consolidation dry-run:
```bash
make consolidate ARGS="--project-id engram-vault --dry-run"
```

9. Inspect recent audit events:
```bash
tail -n 10 data/audit_events.jsonl
```

If you want to use a hashed local UI password instead of plaintext, generate one with:
```bash
cd api
uv run python -c "from app.auth import hash_password; print(hash_password('admin123'))"
```

---

## Admin Console Test Runbook (Multi-Document Pin + MCP)

Use this sequence to validate the latest multi-document continuity path end-to-end with admin credentials.

1. Sign in as admin:
   - Open [http://localhost:5173](http://localhost:5173)
   - Login with `admin` / `admin123`

2. Optional admin-role sanity check:
   - Open [http://localhost:8000/ui/admin](http://localhost:8000/ui/admin)
   - Verify user list renders

3. Create a session:
   - Set `Project ID` to `engram-vault` (or your test project)
   - Click `Create Session`

4. Ingest two documents:
   - In `Document Ingestion` → `Show Upload Form`, ingest two text/file docs
   - Verify both appear in `Recent Documents`

5. Pin both docs:
   - Click `Pin to Chat` on both documents
   - Verify both cards show pinned state

6. Send a prompt:
   - Ask for a summary that should use both docs
   - Verify `Source references used` includes document entries

7. Continue chat:
   - Click `Continue in New Chat`
   - Verify pinned docs are still present in the continued session

8. MCP parity check:
   - Use `tools/list` and confirm document-pin tools are exposed
   - Call `chat.list_pinned_documents` and verify both document IDs are returned

---

## CLI Workflows

Use CLI mode when you want a terminal-only path (no browser).

### Upload Engram

```bash
cat > /tmp/engram.json <<'JSON'
{
  "project_id": "engram-vault",
  "thread_id": "cli-run-001",
  "title": "CLI upload sample",
  "abstract": "Uploaded from local CLI.",
  "detailed_summary_markdown": "Sample summary for CLI upload testing.",
  "tags": ["cli"],
  "keywords": ["upload", "local"]
}
JSON
```

```bash
make cli ARGS="upload --file /tmp/engram.json"
```

### Search Engrams

```bash
make cli ARGS="search --query 'uploaded from local cli' --project-id engram-vault --top-k 5"
```

### Rehydrate Engram

```bash
make cli ARGS="rehydrate --engram-id <engram_uuid>"
```

### Consolidation Maintenance

```bash
make consolidate ARGS="--project-id engram-vault --dry-run"
make consolidate ARGS="--project-id engram-vault"
```

### MCP Smoke Call

Use the CLI to smoke-test MCP JSON-RPC calls against `/api/v1/mcp/stream`.

```bash
make cli ARGS="mcp-call --method tools/list --username admin --password admin123"
```

```bash
make cli ARGS="mcp-call --method project.get_default --params-json '{}' --bearer-token engram_mcp_<token_id_hex>_<secret>"
```

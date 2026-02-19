# MCP Client Integrations (LibreChat, VS Code Copilot, Codex)

This guide provides copy-paste commands to connect external MCP clients to Engram.

## Prerequisites

- Engram API running at `http://localhost:8000`
- Admin credentials available (`admin` / `admin123` in local dev)
- `jq` installed for JSON parsing

## 1) Create an MCP Token (Admin)

```bash
BASE_URL=http://localhost:8000
COOKIE_JAR=/tmp/engram-admin.cookies
USERNAME=admin
PASSWORD=admin123

CSRF_TOKEN=$(
  curl -s -c "$COOKIE_JAR" "$BASE_URL/login" \
  | sed -n 's/.*name="csrf_token" value="\([^"]*\)".*/\1/p' \
  | head -n 1
)

curl -s -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
  -X POST "$BASE_URL/login" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "username=$USERNAME" \
  --data-urlencode "password=$PASSWORD" \
  --data-urlencode "csrf_token=$CSRF_TOKEN" >/dev/null

TOKEN_JSON=$(
  curl -s -b "$COOKIE_JAR" \
    -H "Content-Type: application/json" \
    -X POST "$BASE_URL/api/v1/mcp/tokens" \
    -d '{
      "name": "external-mcp-write",
      "scope": "write",
      "allowed_tools": [],
      "allowed_project_ids": ["engram-vault"],
      "expires_in_days": 90
    }'
)

export ENGRAM_MCP_TOKEN="$(echo "$TOKEN_JSON" | jq -r '.token')"
echo "$TOKEN_JSON" | jq
```

## 2) LibreChat Integration

Add the MCP server to your LibreChat config:

```yaml
mcpServers:
  engram:
    type: streamable-http
    url: http://host.docker.internal:8000/api/v1/mcp/stream
    headers:
      Authorization: "Bearer ${ENGRAM_MCP_TOKEN}"
      Accept: "text/event-stream"
    timeout: 60000
```

Notes:
- If LibreChat is not running in Docker, replace `host.docker.internal` with `localhost`.
- Restart LibreChat after config changes.

```bash
docker compose restart
```

## 3) VS Code Copilot Integration

Create `.vscode/mcp.json`:

```bash
mkdir -p .vscode
cat > .vscode/mcp.json <<EOF_VSCODE
{
  "servers": {
    "engram": {
      "type": "http",
      "url": "http://localhost:8000/api/v1/mcp/stream",
      "headers": {
        "Authorization": "Bearer ${ENGRAM_MCP_TOKEN}"
      }
    }
  }
}
EOF_VSCODE
```

Alternative CLI registration:

```bash
code --add-mcp "{\"name\":\"engram\",\"type\":\"http\",\"url\":\"http://localhost:8000/api/v1/mcp/stream\",\"headers\":{\"Authorization\":\"Bearer ${ENGRAM_MCP_TOKEN}\"}}"
```

## 4) Codex Integration

Append MCP server config to `~/.codex/config.toml`:

```bash
mkdir -p ~/.codex
cat >> ~/.codex/config.toml <<EOF_CODEX

[mcp_servers.engram]
url = "http://localhost:8000/api/v1/mcp/stream"
bearer_token_env_var = "ENGRAM_MCP_TOKEN"
startup_timeout_sec = 20
tool_timeout_sec = 60
enabled = true
EOF_CODEX
```

Verify Codex sees the server:

```bash
codex mcp list
```

## 5) MCP Connectivity Smoke Test

```bash
BASE_URL=http://localhost:8000

curl -sN \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ENGRAM_MCP_TOKEN" \
  -X POST "$BASE_URL/api/v1/mcp/stream" \
  -d '{
    "jsonrpc":"2.0",
    "id":"tools-list-1",
    "method":"tools/list",
    "params":{}
  }' \
  | sed -n 's/^data: //p' \
  | jq
```

If this succeeds, your external client integration should work with the same token and endpoint.

# MCP Finding: VS Code OAuth Cache and Re-Authentication

## Context
After clearing/resetting the DB, VS Code MCP can keep using a cached OAuth dynamic client registration (`client_id`).
This can cause errors like:

- `{"error":"invalid_client","error_description":"Unknown client_id."}`

## Resolution (VS Code)
Use this exact sequence to clear cached auth and force fresh registration:

1. Open Command Palette (`Cmd+Shift+P`).
2. Run `Authentication: Remove Dynamic Authentication Providers`.
3. Remove the provider linked to the Engram MCP host.
4. Open the Accounts menu (bottom-left profile icon).
5. If available, open trusted MCP server/auth entries and revoke the Engram server trust.
6. Run `MCP: List Servers` and restart the Engram MCP server (or remove + re-add it).
7. Run `Developer: Reload Window`.
8. Retry MCP authentication.

## Additional Recovery
If the issue persists:

- Recreate workspace config file: `.vscode/mcp.json`.
- Fully quit and relaunch VS Code.
- Verify API metadata endpoints are reachable:
  - `/.well-known/oauth-authorization-server`
  - `/.well-known/oauth-protected-resource`

## Notes for This Project
- This issue was cache-related, not a backend defect, after DB clear.
- No OAuth recovery code changes were kept for this specific incident.

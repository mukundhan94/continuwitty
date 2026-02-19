@mock @mcp @tokens
Feature: MCP bearer token authorization
  As an admin operator
  I want scoped MCP personal access tokens
  So external agents can access read/write workflows safely without session cookies

  Scenario: Read token can list tools but cannot execute write actions
    Given I am signed in
    And mocked MCP token backend is enabled
    When I create a read MCP token for project "engram-vault"
    And I call MCP tools list using bearer token only
    Then only read-safe tools should be visible for the bearer token
    When I attempt a write MCP tool call for project "engram-vault"
    Then the MCP response should deny the write action for token scope

  Scenario: Write token can execute project-scoped write workflow
    Given I am signed in
    And mocked MCP token backend is enabled
    When I create a write MCP token for project "acceptance-mcp-write"
    And I execute a bearer write workflow for project "acceptance-mcp-write"
    Then the bearer write workflow should create a session and engram

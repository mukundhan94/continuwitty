@admin-ui @mcp @auth @mock
Feature: Admin MCP token manager UI
  As an admin user
  I want to select tools and projects as chips when creating MCP tokens
  So token policies are easy to configure and verify from the UI

  Scenario: Admin creates an MCP token using selectable tool/project chips
    Given I am signed in
    And I create a session named "Admin token options seed"
    When I open the admin MCP token manager
    Then I should see selectable MCP tool and project options
    When I add tool and project chips and create a new MCP token
    Then I should see the one-time MCP token value and a persisted token row

  Scenario: Non-admin users do not see admin token controls
    Given I am signed in
    And I create a viewer user for admin token UI checks
    When I sign out and sign in as the created viewer
    Then I should not see the MCP Tokens admin action

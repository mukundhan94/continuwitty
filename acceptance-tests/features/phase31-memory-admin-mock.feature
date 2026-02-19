@mock @phase31 @memory-admin
Feature: Phase 31 project defaults and memory administration
  As an admin operator
  I want default-project fallback and memory lifecycle controls
  So that I can organize sessions and engrams safely across project boundaries

  Scenario: MCP engram create resolves default project when project_id is omitted
    Given I am signed in
    When I configure a default project for the phase31 scenario
    And I create an engram through MCP without project_id
    Then the MCP response should report default-project resolution

  Scenario: Admin memory session delete and restore keeps linked engrams by default
    Given I am signed in
    When I create a session and a linked engram for admin lifecycle checks
    And I soft-delete that session without deleting linked engrams
    Then I should be able to restore the session and still query the linked engram

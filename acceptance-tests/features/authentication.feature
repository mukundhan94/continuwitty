Feature: Authentication handoff
  Verify that login transitions directly to the workbench without manual refresh.

  Scenario: Successful sign in opens the workbench in one step
    Given I open the chat application
    When I sign in with default local credentials
    Then I should see the memory continuity workbench without refreshing

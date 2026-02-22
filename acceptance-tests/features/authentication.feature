Feature: Authentication handoff
  Verify that login transitions directly to the workbench without manual refresh.

  @mock
  Scenario: Successful sign in opens the workbench in one step
    Given I open the chat application
    When I sign in with default local credentials
    Then I should see the memory continuity workbench without refreshing

  @mock
  Scenario: Repeated failed sign in attempts trigger lockout
    Given I open the chat application
    When I attempt to sign in with an invalid password repeatedly
    Then I should see a login rate limit error

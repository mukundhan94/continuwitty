@mock @lifecycle
Feature: Lifecycle autosave policies (mocked)
  As an operator
  I want deterministic acceptance coverage for autosave behavior
  So that lifecycle policy wiring can be validated without provider nondeterminism

  Scenario: Message-count autosave triggers only at threshold
    Given I am signed in
    And mocked lifecycle backend is enabled
    When I create a message-count autosave session named "Mock lifecycle message-count"
    And I send a mocked lifecycle prompt "First incident update for timeline baseline."
    Then lifecycle timeline should show 0 autosave snapshots
    When I send a mocked lifecycle prompt "Second incident update that should trigger autosave snapshot."
    Then lifecycle timeline should show 1 autosave snapshots
    And the create session request should include autosave strategy "message_count" with minimum messages 2

  Scenario: Autosave-off policy does not emit snapshots
    Given I am signed in
    And mocked lifecycle backend is enabled
    When I create an autosave-off session named "Mock lifecycle autosave off"
    And I send a mocked lifecycle prompt "First update while autosave is disabled."
    And I send a mocked lifecycle prompt "Second update while autosave remains disabled."
    Then lifecycle timeline should show 0 autosave snapshots
    And the create session request should disable autosave

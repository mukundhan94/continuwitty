@mock @phase38 @memory-admin
Feature: Phase 38 contradiction alert quality
  As an operator
  I want deterministic coverage for contradiction alert refresh/list/resolve workflows
  So that contradiction warning quality remains high and actionable

  Scenario: Contradiction alert benchmark and resolution flow
    Given I am signed in
    When I seed deterministic contradiction links for warning quality checks
    And I refresh contradiction alerts for the seeded project
    Then contradiction alert precision and recall should meet threshold
    When I resolve one contradiction alert for the seeded project
    Then resolved contradiction alerts should include the actioned record

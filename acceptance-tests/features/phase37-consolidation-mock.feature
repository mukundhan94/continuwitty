@mock @phase37 @memory-admin
Feature: Phase 37 consolidation suggestion quality
  As an admin operator
  I want deterministic coverage for consolidation grouping and action workflows
  So that duplicate-memory suggestions stay precise and operationally safe

  Scenario: Consolidation refresh/list/action meets grouping quality thresholds
    Given I am signed in
    When I seed deterministic duplicate engrams for consolidation quality checks
    And I refresh consolidation suggestions for the seeded project
    Then consolidation grouping precision and recall should meet threshold
    When I merge one consolidation suggestion for the seeded project
    Then merged consolidation suggestions should include the actioned record

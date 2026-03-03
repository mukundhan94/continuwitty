@mock @phase40 @memory-admin
Feature: Phase 40 autonomous curation suggestions
  As an admin operator
  I want deterministic coverage for curation suggestion generation and action workflows
  So that autonomous memory guidance stays actionable and auditable

  Scenario: Curation generation and action flow
    Given I am signed in
    When I seed deterministic memory curation prerequisites
    And I refresh consolidation and contradiction workflows for memory curation
    Then curation suggestion type coverage should include consolidate and contradiction
    When I accept one curation suggestion for the seeded project
    Then accepted curation suggestions should include the actioned record

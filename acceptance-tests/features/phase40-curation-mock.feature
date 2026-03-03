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

  Scenario: Curation applied action cascades downstream workflows
    Given I am signed in
    When I seed deterministic memory curation prerequisites
    And I refresh consolidation and contradiction workflows for memory curation
    Then curation suggestion payloads should include consolidation and contradiction references
    When I apply consolidation and contradiction curation suggestions for the seeded project
    Then applied curation suggestions should include both actioned records
    And downstream consolidation and contradiction records should be actioned

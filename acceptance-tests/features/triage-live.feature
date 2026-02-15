@triage-live @bedrock-live
Feature: Incident triage continuity handoff
  Validate that live triage analysis can be saved as an engram and reused in a continued session.

  Background:
    Given I am signed in for bedrock live testing

  Scenario: Save triage memory and generate a commander handoff in a continued chat
    When I create a bedrock session named "Acceptance Triage Nebula P1" using the default bedrock model
    And I send a live triage seed prompt
    Then I should receive a triage-oriented assistant response
    When I save the current triage response as a project engram
    And I click continue in new chat
    And I pin the saved triage engram to the active session
    And I request a commander handoff brief from pinned triage memory
    Then I should receive a continuity-aware triage handoff response

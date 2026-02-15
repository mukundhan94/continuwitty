Feature: Session workflow layout stability
  Keep the chat workspace stable while session count grows.

  Background:
    Given I am signed in

  Scenario: Creating multiple sessions keeps the message pane height stable
    When I create a session named "Acceptance Layout Seed"
    And I capture the current chat pane height as baseline
    And I create a session named "Acceptance Layout Follow Up"
    And I create a session named "Acceptance Layout Third"
    Then the chat pane height drift should be at most 2 pixels

  Scenario: Continue in new chat creates a continued active session
    When I create a session named "Acceptance Continuation Seed"
    And I click continue in new chat
    Then a continued session should become active

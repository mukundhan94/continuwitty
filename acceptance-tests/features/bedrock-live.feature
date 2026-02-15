@bedrock-live
Feature: Bedrock live model response
  Validate the real Bedrock integration with the configured default model for the UI.

  Background:
    Given I am signed in for bedrock live testing

  Scenario: Bedrock default model returns a non-deterministic response
    When I create a bedrock session named "Acceptance Bedrock Live" using the default bedrock model
    Then the selected bedrock model should match the configured default when provided
    When I send a live bedrock prompt
    Then I should receive a non-empty assistant response from bedrock

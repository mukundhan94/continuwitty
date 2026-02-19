@mock @mcp @engram
Feature: Engram auto metadata enrichment via MCP
  As an agent operator
  I want conversation-only engram persistence to auto-fill missing metadata
  So that I can store sessions without manually crafting tags or abstract text

  Scenario: MCP create-from-conversation auto-fills metadata when empty
    Given I am signed in
    When I persist a conversation through MCP with empty metadata fields
    Then MCP enrichment report should indicate auto metadata was applied
    And the persisted engram should expose generated abstract tags and keywords

  Scenario: MCP create-from-conversation keeps explicit caller metadata
    Given I am signed in
    When I persist a conversation through MCP with explicit metadata fields
    Then MCP enrichment report should indicate no metadata overwrite
    And the persisted engram should keep caller abstract tags and keywords

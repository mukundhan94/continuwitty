@mock @export-import
Feature: Project export and import web workflow
  As a project operator
  I want to export and import engram bundles from the UI
  So that I can move memory stashes across projects

  Background:
    Given I am signed in

  Scenario: Transfer form exposes searchable project and collection suggestions
    When I prepare a project dataset for export tests
    And I open the project transfer page
    Then the transfer form should expose search suggestions for the prepared dataset

  Scenario: JSON export defaults to full project and embeddings disabled
    When I prepare a project dataset for export tests
    And I open the project transfer page
    And I export the prepared project as JSON
    Then the exported bundle should include full project data with embeddings disabled

  Scenario: JSON export can filter by collection IDs
    When I prepare a project dataset for export tests
    And I open the project transfer page
    And I export the prepared project using only the first collection ID
    Then the exported bundle should only include engrams from the selected collection

  Scenario: JSON export can include embeddings flag
    When I prepare a project dataset for export tests
    And I open the project transfer page
    And I export the prepared project with embeddings enabled
    Then the exported bundle should mark embeddings as included

  Scenario: ZIP export downloads archive format
    When I prepare a project dataset for export tests
    And I open the project transfer page
    And I export the prepared project as ZIP
    Then the exported download should be a ZIP bundle

  Scenario: Import skip policy keeps existing duplicates
    When I prepare source and target projects for import policy "skip"
    And I open the project transfer page
    And I import the prepared bundle with policy "skip"
    Then the import result should match "skip" policy expectations

  Scenario: Import rename policy creates renamed duplicates
    When I prepare source and target projects for import policy "rename"
    And I open the project transfer page
    And I import the prepared bundle with policy "rename"
    Then the import result should match "rename" policy expectations

  Scenario: Import overwrite policy replaces existing duplicates
    When I prepare source and target projects for import policy "overwrite"
    And I open the project transfer page
    And I import the prepared bundle with policy "overwrite"
    Then the import result should match "overwrite" policy expectations

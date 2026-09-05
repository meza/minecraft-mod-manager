Feature: Initialize configuration

  Scenario: A user cancels initialization before choosing a loader
    Given Alice uses MMM in an empty workspace
    When Alice starts interactive initialization
    Then Alice should see the i18n key "cmd.init.prompt.loader.question"
    When Alice cancels initialization
    Then Alice should observe a successful exit
    And Alice should find no configuration in the workspace

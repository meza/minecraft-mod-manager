Feature: Identify the running MMM release
  The user can obtain version information without a usable installation.

  # Source: ../../docs/commands/version.md
  Rule: Version information is a setup-independent utility
    Scenario Outline: The user requests version information with unusable installation metadata
      Given the user has <setup>
      And the user's existing workspace files have recorded content
      When the user requests the running MMM version
      Then they should see the running MMM release identified
      And they should not be offered initialization
      And they should not be told that MMM was updated
      And they should find their installation metadata and mod files unchanged

      Examples:
        | setup               |
        | no modlist          |
        | a malformed modlist |

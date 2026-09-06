Feature: Read command help without an installation
  The user can discover MMM commands and their contracts without starting an operation.

  # Source: ../../docs/commands/help.md#usage
  Rule: Help is independent of installation setup
    Scenario Outline: The user requests general help with unusable installation metadata
      Given the user has <setup>
      And the user's existing workspace files have recorded content
      When the user requests general help
      Then they should see available commands and shared options
      And they should not be offered initialization
      And they should find their installation metadata and mod files unchanged

      Examples:
        | setup               |
        | no modlist          |
        | a malformed modlist |

    Scenario: Bare MMM shows help
      Given the user has no modlist or lockfile
      When the user invokes MMM without a command
      Then they should see available commands and shared options
      And they should not be offered initialization
      And they should find no installation metadata created

  # Source: ../../docs/commands/help.md
  Rule: Command help explains the selected command without running it
    Scenario: The user reads change help
      Given the user has no modlist or lockfile
      And the user's existing workspace files have recorded content
      When the user requests help for change
      Then they should see change usage, target defaults and change-specific force semantics
      And they should not be offered initialization
      And they should find their workspace files unchanged and no installation metadata created

    Scenario: The user reads command help through the command help option
      Given the user has a malformed modlist
      And the user's existing workspace files have recorded content
      When the user requests change help with the change --help form
      Then they should see change usage, target defaults and change-specific force semantics
      And they should not be offered initialization
      And they should find their installation metadata and workspace files unchanged

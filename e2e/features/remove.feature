Feature: Remove managed mods
  The user removes only the managed mods they select while MMM preserves unrelated files and retryable state.

  Background:
    Given the user has an initialized installation

  # Source: ../../docs/intent.md#remove
  # Source: ../../docs/commands/remove.md#usage-and-options
  Rule: Removal applies once to the explicitly selected mod identities

    Scenario: The user removes one managed mod by project ID
      Given the user has a Modrinth Sodium mod config with a valid locked artifact installed
      And the user has an unrelated managed Iris artifact installed
      When the user authorizes MMM to remove Sodium by its project ID
      Then they should find the Sodium mod config absent from their modlist
      And they should find the Sodium lock entry absent from their lockfile
      And they should find the managed Sodium artifact absent
      And they should find the Iris mod config, lock entry and artifact unchanged

    Scenario: The user removes all mods matched by a case-insensitive glob
      Given the user has managed WorldEdit and WorldEdit CUI artifacts installed
      And the user has a managed Sodium artifact installed
      When the user authorizes MMM to remove mods matching "WORLD*EDIT*"
      Then they should find the WorldEdit and WorldEdit CUI mod configs absent from their modlist
      And they should find the WorldEdit and WorldEdit CUI lock entries absent from their lockfile
      And they should find both managed WorldEdit artifacts absent
      And they should find the Sodium mod config, lock entry and artifact unchanged

    Scenario Outline: Supported glob expressions select a managed mod
      Given the user has a managed Sodium artifact installed
      When the user authorizes MMM to remove mods matching "<pattern>"
      Then they should find the Sodium mod config, lock entry and managed artifact absent

      Examples:
        | pattern     |
        | sodiu?      |
        | sodi[ua]m   |
        | sodi[t-v]m  |

    Scenario: Overlapping selections remove a mod only once
      Given the user has a Modrinth Sodium mod config with a valid locked artifact installed
      When the user authorizes MMM to remove Sodium by its name, project ID and a matching glob
      Then they should find exactly one completed removal record for Sodium
      And they should find no Sodium mod config, lock entry or managed artifact

  # Source: ../../docs/intent.md#remove
  # Source: ../../docs/commands/remove.md#confirmation-and-no-prompt-execution
  Rule: Removal authority is limited to the resolved selection

    Scenario: Force authorizes removal without widening the selection
      Given the user has managed Sodium and Iris artifacts installed
      And the user has a visible unmanaged jar
      When the user removes Sodium with --force
      Then they should find the Sodium mod config, lock entry and managed artifact absent
      And they should find the Iris mod config, lock entry and artifact unchanged
      And they should find the visible unmanaged jar unchanged
      And they should find a durable record that removal of Sodium was authorized

    Scenario: An absent selection is a successful no-op
      Given the user has no mod config matching "absent-mod"
      When the user asks MMM to remove "absent-mod"
      Then they should be told that no mod matched the selection
      And they should find their modlist, lockfile and mod files unchanged
      And they should receive a successful operation result

  # Source: ../../docs/intent.md#remove
  # Source: ../../docs/commands/remove.md#removal-results-and-retry
  Rule: Removal failures retain truthful retryable state

    Scenario: A missing managed file does not prevent metadata removal
      Given the user has a Sodium mod config and valid lock entry
      And the managed Sodium artifact is already absent
      When the user authorizes MMM to remove Sodium
      Then they should find the Sodium mod config absent from their modlist
      And they should find the Sodium lock entry absent from their lockfile
      And they should be told that removal is complete

    Scenario: One failed file removal does not undo an independent removal
      Given the user has managed Sodium and Iris artifacts installed
      And the managed Sodium artifact cannot be deleted
      When the user authorizes MMM to remove Sodium and Iris
      Then they should find the Iris mod config, lock entry and artifact absent
      And they should not be told that Sodium was fully removed
      And they should be told that the Sodium artifact could not be deleted
      And they should receive guidance for retrying the Sodium removal
      And they should be told that the requested operation did not complete successfully

    Scenario: Retrying a failed removal completes only the remaining work
      Given the user previously attempted to remove Sodium and Iris
      And the Iris removal completed consistently
      And the Sodium removal remains retryable because its managed artifact could not be deleted
      And the managed Sodium artifact can now be deleted
      When the user authorizes MMM to remove Sodium and Iris again
      Then they should find no Sodium or Iris mod config, lock entry or managed artifact
      And they should find no duplicate removal work for Iris

    Scenario: Safe cancellation preserves a completed removal and retry finishes remaining work
      Given the user has managed Sodium, Iris and Lithium artifacts installed
      And the Iris removal can remain in progress after the Sodium removal settles
      When the user authorizes MMM to remove Sodium, Iris and Lithium
      Then they should observe the Sodium removal settled while the Iris removal remains in progress and the Lithium removal has not begun
      When the user safely cancels removal
      Then they should find the Sodium mod config, lock entry and managed artifact absent
      And they should find no new removal work started after cancellation
      And they should find the Lithium mod config, exact lock entry and managed artifact unchanged
      And they should find the reported Iris outcome agrees with its actual mod config, lock entry and managed artifact
      And they should receive a report of completed and unfinished removals
      And they should be told that the operation was cancelled
      When the user authorizes MMM to remove Sodium, Iris and Lithium again
      Then they should find no Sodium, Iris or Lithium mod config, lock entry or managed artifact
      And they should find no duplicate removal work for any previously completed mod

  # Source: ../../docs/intent.md#excluded-files
  # Source: ../../docs/commands/remove.md#removal-results-and-retry
  Rule: Removal never overrides file exclusions

    Scenario Outline: Force leaves an excluded selected file untouched
      Given the user has a selected file named "<filename>"
      And the file is protected by <protection>
      When the user attempts to remove its selected mod with --force
      Then they should find "<filename>" unchanged

      Examples:
        | filename            | protection                    |
        | sodium-private.jar  | a matching .mmmignore pattern |
        | sodium.jar.disabled | the .disabled suffix          |

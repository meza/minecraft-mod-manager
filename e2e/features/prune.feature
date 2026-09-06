Feature: Prune unmanaged jars
  The user deletes an authorized set of visible unmanaged jars without exposing protected or unrelated files.

  Background:
    Given the user has an initialized installation with a valid lockfile

  # Source: ../../docs/intent.md#prune
  # Source: ../../docs/commands/prune.md#deletion-boundary
  Rule: Prune considers only visible unmanaged jars immediately inside the mods directory

    Scenario: The user prunes the complete eligible deletion set
      Given the user has two visible unmanaged jars in their mods directory
      And the user has a managed jar in their mods directory
      And the user has an unmanaged text file in their mods directory
      And the user has an unmanaged jar in a subdirectory of their mods directory
      When the user authorizes MMM to prune the reported unmanaged jars
      Then they should find both visible unmanaged jars deleted
      And they should find the managed jar unchanged
      And they should find the unrelated text file unchanged
      And they should find the nested jar unchanged

    Scenario Outline: Force does not override protected-file exclusions
      Given the user has an unmanaged file named "<filename>" in their mods directory
      And the file is protected by <protection>
      When the user prunes with --force
      Then they should find "<filename>" unchanged
      And they should find no deletion record for "<filename>"
      And they should receive a successful operation result

      Examples:
        | filename             | protection                    |
        | private-library.jar  | a matching .mmmignore pattern |
        | optional.jar.disabled | the .disabled suffix         |

    Scenario: An empty deletion set is a successful no-op
      Given the user has no visible unmanaged jars in their mods directory
      When the user asks MMM to prune unmanaged jars
      Then they should be told that there are no unmanaged jars to delete
      And they should find their modlist, lockfile and mod files unchanged
      And they should receive a successful operation result

  # Source: ../../docs/intent.md#prune
  # Source: ../../docs/commands/prune.md#confirmation-and-missing-evidence
  Rule: Prune requires explicit deletion authority and trustworthy ownership evidence

    Scenario: A missing lockfile prevents pruning
      Given the user's lockfile is missing
      And the user has a jar that could be managed
      When the user attempts to prune with --force
      Then they should be told that ownership cannot be established without a lockfile
      And they should receive guidance to run install
      And they should find their modlist and mod files unchanged
      And they should be told that the requested operation did not complete successfully

    Scenario Outline: Untrustworthy resolution evidence prevents pruning
      Given the user's lockfile contains <problem>
      And the user has a jar whose ownership depends on that evidence
      When the user attempts to prune with --force
      Then they should be told that the resolution evidence is <problem>
      And they should receive recovery guidance
      And they should find their modlist, lockfile and mod files unchanged
      And they should be told that the requested operation did not complete successfully

      Examples:
        | problem                         |
        | malformed                       |
        | contradictory                   |
        | missing required integrity data |

  # Source: ../../docs/intent.md#excluded-files
  # Source: ../../docs/commands/prune.md#deletion-boundary
  Rule: Invalid ignore patterns fail closed before deletion

    Scenario: An invalid ignore pattern identifies its source line and preserves candidates
      Given the user's .mmmignore contains an invalid pattern on line 3
      And the user has visible unmanaged jars in their mods directory
      When the user attempts to prune with --force
      Then they should be told that the .mmmignore pattern on line 3 is invalid
      And they should find their modlist, lockfile and mod files unchanged
      And they should be told that the requested operation did not complete successfully

  # Source: ../../docs/intent.md#failure-retry-and-cancellation-promises
  # Source: ../../docs/commands/prune.md#results-failure-and-retry
  Rule: Independent prune failures remain visible and retryable

    Scenario: One failed deletion does not undo another deletion
      Given the user has visible unmanaged jars named "first.jar" and "second.jar"
      And "second.jar" cannot be deleted
      When the user authorizes MMM to prune both jars
      Then they should find "first.jar" deleted
      And they should find "second.jar" unchanged
      And they should be told that "second.jar" could not be deleted
      And they should receive guidance for retrying
      And they should be told that the requested operation did not complete successfully

    Scenario: Retrying prune deletes only the remaining unmanaged jar
      Given the user previously pruned "first.jar" successfully
      And "second.jar" remained because it could not be deleted
      And "second.jar" can now be deleted
      When the user authorizes MMM to prune unmanaged jars again
      Then they should find "second.jar" deleted
      And they should find no duplicate deletion work for "first.jar"

    Scenario: Safe cancellation preserves a completed prune deletion and retry finishes remaining work
      Given the user has visible unmanaged jars named "first.jar", "second.jar" and "third.jar"
      And deletion of "second.jar" can remain in progress after deletion of "first.jar" settles
      When the user authorizes MMM to prune the unmanaged jars
      Then they should observe deletion of "first.jar" settled while deletion of "second.jar" remains in progress and deletion of "third.jar" has not begun
      When the user safely cancels prune
      Then they should find "first.jar" deleted
      And they should find no new deletion work started after cancellation
      And they should find "third.jar" unchanged and unmanaged
      And they should find the reported outcome for "second.jar" agrees with whether that file remains
      And they should be told that the operation was cancelled
      When the user authorizes MMM to prune unmanaged jars again
      Then they should find every remaining visible unmanaged jar deleted
      And they should find no duplicate deletion work for "first.jar"

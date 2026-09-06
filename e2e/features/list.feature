Feature: Inspect declared and observed mod state
  The user asks MMM to report what can be established without repairing files or rewriting metadata.

  Background:
    Given the user has an initialized installation

  # Source: ../../docs/intent.md#list
  # Source: ../../docs/commands/list.md#what-the-report-distinguishes
  Rule: List distinguishes declared, resolved and observed installation evidence

    Scenario: List reports a verified installed artifact
      Given the user has a Modrinth Sodium mod config
      And the user's lockfile records an exact Sodium artifact with required integrity information
      And the installed Sodium file matches that artifact's filename and integrity information
      When the user lists their mods
      Then they should see Sodium reported as installed with its platform and project ID
      And they should find their modlist, lockfile and mod files unchanged

    Scenario: List reports a missing managed file
      Given the user has a Sodium mod config and valid lock entry
      And the managed Sodium file is missing
      When the user lists their mods
      Then they should see Sodium reported with its locked artifact and missing file state
      And they should receive guidance to reconcile the installation
      And they should find their modlist, lockfile and mod files unchanged

    Scenario Outline: List reports missing resolution evidence without creating it
      Given the user has a Sodium mod config
      And the user has <missing evidence>
      When the user lists their mods
      Then they should see Sodium reported with missing resolution evidence
      And they should receive guidance to run install
      And they should find no resolution evidence created
      And they should find their modlist and mod files unchanged

      Examples:
        | missing evidence     |
        | no lockfile          |
        | no Sodium lock entry |

    Scenario: A matching filename with different content is reported as a mismatch
      Given the user has a Sodium mod config and valid lock entry
      And the installed Sodium filename matches the lock entry
      But the installed Sodium content does not match the recorded integrity information
      When the user lists their mods
      Then they should see Sodium reported with a content mismatch
      And they should not see Sodium reported as correctly installed
      And they should receive guidance to reconcile the installation
      And they should find their modlist, lockfile and mod files unchanged

  # Source: ../../docs/intent.md#the-installation-model
  # Source: ../../docs/commands/list.md#what-the-report-distinguishes
  Rule: List preserves and reports untrustworthy resolution evidence

    Scenario Outline: List does not trust or repair invalid resolution evidence
      Given the user has a Sodium mod config
      And the user's resolution evidence is <problem>
      When the user lists their mods
      Then they should see the <problem> reported for Sodium
      And they should receive recovery guidance
      And they should find the original resolution evidence unchanged
      And they should find their modlist and mod files unchanged

      Examples:
        | problem                         |
        | malformed                       |
        | contradictory                   |
        | missing required integrity data |

  # Source: ../../docs/intent.md#shared-setup-and-recovery
  # Source: ../../docs/commands/list.md#what-the-report-distinguishes
  Rule: List reports duplicate declarations without choosing or rewriting them

    Scenario Outline: List preserves duplicate mod configs when the user requests inspection without correction
      Given the user's modlist contains <duplicate kind> Modrinth Sodium mod configs
      When the user lists their mods without accepting a mod config correction
      Then they should be told that the Sodium mod configs are <duplicate kind>
      And they should find every original Sodium mod config unchanged
      And they should find their lockfile and mod files unchanged

      Examples:
        | duplicate kind |
        | identical      |
        | conflicting    |

  # Source: ../../docs/intent.md#list
  # Source: ../../docs/commands/list.md#what-the-report-distinguishes
  Rule: List uses deterministic identity ordering and reports visible unmanaged jars

    Scenario: Entries use stable case-insensitive ordering with identity tie-breakers
      Given the user has mod configs whose names differ only by case
      And those mod configs have different platforms and project IDs
      When the user lists their mods
      Then they should see entries ordered case-insensitively by name
      And they should see equal names ordered by platform and then project ID
      And they should find their modlist, lockfile and mod files unchanged

    Scenario: Visible unmanaged jars do not prevent the managed report
      Given the user has a managed Sodium artifact installed
      And the user has a visible unmanaged jar
      And the user has an unmanaged jar matched by .mmmignore
      And the user has an unmanaged file ending in .disabled
      When the user lists their mods
      Then they should see the managed Sodium result
      And they should see the visible unmanaged jar reported as unmanaged
      And they should find the ignored jar and disabled file absent from unmanaged reporting
      And they should find their modlist, lockfile and mod files unchanged

  # Source: ../../docs/intent.md#list
  # Source: ../../docs/commands/list.md#read-only-guarantee-and-results
  Rule: List never repairs or adopts observed state

    Scenario: Inspection leaves every inconsistent surface unchanged
      Given the user has a missing managed file
      And the user has a mismatched managed file
      And the user has a visible unmanaged jar
      And the user has a lock entry missing required integrity information
      When the user lists their mods
      Then they should see each observed state reported with a useful next action
      And they should find no file repaired or removed
      And they should find no jar adopted
      And they should find their modlist and lockfile unchanged

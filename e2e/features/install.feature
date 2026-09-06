Feature: Reconcile a declared installation
  The user can make managed files and valid resolution evidence satisfy their modlist without opportunistic upgrades.

  Rule: Install preserves valid resolutions and reconciles declared changes
    # Source: ../../docs/intent.md#the-installation-model

    Scenario: A valid locked artifact is preserved when a newer artifact exists
      Given the user has Sodium declared from Modrinth
      And their lockfile records "sodium-0.5.5.jar" and its verified digest
      And "sodium-0.5.5.jar" has matching content "sodium-0.5.5" installed
      And Modrinth provides newer eligible Sodium 0.5.6 with content "sodium-0.5.6"
      When the user installs their declared mods
      Then they should find "sodium-0.5.5.jar" with content "sodium-0.5.5" still installed
      And they should find the Sodium lock entry unchanged
      And they should receive a report that Sodium is already satisfied
      And they should receive a successful operation result

    Scenario: New and changed mod configs resolve directly to their declared state
      Given the user has newly declared Lithium from Modrinth
      And the Sodium mod config changed from pin 0.5.5 to pin 0.5.6
      And Modrinth provides eligible Lithium 0.12.0 with content "lithium-0.12.0"
      And Modrinth provides pinned Sodium 0.5.6 with content "sodium-0.5.6"
      When the user installs their declared mods
      Then they should find content "lithium-0.12.0" and "sodium-0.5.6" installed
      And they should find the exact Lithium and Sodium artifacts recorded in their lockfile
      And they should find both mod configs unchanged in their modlist

    Scenario: A removed mod config removes its managed artifact and lock entry
      Given the user removed the Lithium mod config from their modlist
      And their lockfile still records managed artifact "lithium-0.12.0.jar"
      And "lithium-0.12.0.jar" with content "lithium-0.12.0" is installed
      When the user installs their declared mods
      Then they should find no Lithium managed artifact or lock entry
      And they should find the remaining mod configs and their artifacts unchanged
      And they should receive a report that Lithium was removed from the managed installation

  Rule: Dependency metadata does not expand the declared installation
    # Source: ../../docs/intent.md#dependency-management

    Scenario: Explicit dependency metadata does not declare or install another project
      Given the user has only Sodium declared from Modrinth without a lock entry or installed artifact
      And Modrinth provides an eligible Sodium artifact whose metadata names Fabric API as a required dependency
      And Modrinth provides an eligible Fabric API artifact
      When the user installs their declared mods
      Then they should find the exact Sodium artifact installed and recorded in their lockfile
      And they should find no Fabric API mod config, lock entry, or installed artifact
      And they should find exactly one declared mod in their modlist

  Rule: Install reevaluates valid locks under current effective release settings
    # Source: ../../docs/intent.md#the-installation-model

    Scenario: A changed inherited release default invalidates the previous resolution
      Given the user changed the installation policy from release-and-beta to release-only
      And Sodium has no per-mod release override
      And a beta Sodium artifact is installed and locked from the previous policy
      And Modrinth provides an eligible release Sodium artifact with different content
      When the user installs their declared mods
      Then they should find only the release Sodium artifact content installed
      And they should find the exact release artifact recorded in their lockfile
      And they should find no per-mod release override added to the Sodium mod config
      And they should find the installation-wide release-only policy unchanged

    Scenario: An unchanged per-mod override masks a changed installation default
      Given the user changed the installation policy from release-and-beta to release-only
      And the Sodium mod config still allows release and beta artifacts
      And a beta Sodium artifact satisfying that override is installed and locked
      And Modrinth provides a newer release Sodium artifact
      When the user installs their declared mods
      Then they should find the beta Sodium artifact content and lock entry unchanged
      And they should find the Sodium release-and-beta override unchanged
      And they should find the installation-wide release-only policy unchanged
      And they should receive a successful operation result

  Rule: A failed repair preserves the observed file and locked resolution
    # Source: ../../docs/intent.md#install

    Scenario: A failed repair download preserves a working managed file
      Given the user's lockfile records "sodium-0.5.5.jar" with content "sodium-0.5.5"
      And their managed Sodium file has modified content "locally-modified"
      And Modrinth cannot complete the recorded Sodium download after automatic retries
      When the user installs their declared mods
      Then they should find the locally modified Sodium content unchanged
      And they should find the original Sodium lock entry unchanged rather than adopted content
      And they should be told that Sodium repair failed and the installation remains unsatisfied
      And they should be told that the requested operation did not complete successfully

  Rule: Missing resolution evidence is reconciled but corrupt evidence is preserved
    # Source: ../../docs/intent.md#the-installation-model

    Scenario: An incomplete lockfile gains only its missing resolution
      Given the user has Sodium and Lithium mod configs
      And their lockfile contains a valid satisfied Sodium entry but no Lithium entry
      And Modrinth provides eligible Lithium 0.12.0 with content "lithium-0.12.0"
      When the user installs their declared mods
      Then they should find the Sodium artifact content and lock entry unchanged
      And they should find Lithium content "lithium-0.12.0" installed
      And they should find the exact Lithium artifact recorded in their lockfile

    Scenario: A missing lockfile resolves current constraints explicitly
      Given the user has Sodium declared from Modrinth without a lockfile
      And Modrinth provides eligible Sodium 0.5.6 with content "sodium-0.5.6"
      When the user installs their declared mods
      Then they should be told that previous exact resolutions could not be reproduced from missing evidence
      And they should find content "sodium-0.5.6" installed
      And they should find the exact Sodium artifact recorded in a new lockfile

    Scenario Outline: Existing untrustworthy resolution evidence is not reconstructed
      Given the user has Sodium declared with <bad evidence>
      And their installed Sodium file has content "sodium-existing"
      And Modrinth provides eligible Sodium 0.5.6 with content "sodium-0.5.6"
      When the user installs their declared mods
      Then they should find the <bad evidence> preserved for recovery
      And they should find installed content "sodium-existing" unchanged
      And they should not find Sodium 0.5.6 silently selected
      And they should be told why the resolution evidence cannot be trusted and how to recover
      And they should be told that the requested operation did not complete successfully

      Examples:
        | bad evidence                           |
        | malformed lock data                    |
        | contradictory Sodium lock entries      |
        | a Sodium entry without integrity data  |

    Scenario: An unavailable locked artifact is not substituted
      Given the user's valid lockfile records Sodium 0.5.5 but its managed file is missing
      And Modrinth no longer provides Sodium 0.5.5
      And Modrinth provides newer eligible Sodium 0.5.6 with content "sodium-0.5.6"
      When the user installs their declared mods
      Then they should find no Sodium artifact installed
      And they should find the Sodium 0.5.5 lock entry unchanged
      And they should not find Sodium 0.5.6 selected as a substitute
      And they should receive guidance about the unavailable locked artifact
      And they should be told that the requested operation did not complete successfully

  Rule: Independent failures remain retryable
    # Source: ../../docs/intent.md#failure-retry-and-cancellation-promises

    Scenario: A partial installation retains completed work and working replacements
      Given the user has newly declared Sodium and Lithium
      And Modrinth provides Sodium content "sodium-0.5.6"
      And Modrinth resolves Lithium but its download fails after automatic retries
      When the user installs their declared mods
      Then they should find Sodium content "sodium-0.5.6" installed and recorded in their lockfile
      And they should find no incomplete Lithium download installed as a jar
      And they should receive a report distinguishing completed Sodium from failed Lithium with retry guidance
      And they should be told that the requested operation did not complete successfully

    Scenario: Retrying a partial installation completes only remaining work
      Given the user has Sodium already satisfied after a partial installation
      And Lithium remains declared without an installed artifact
      And Modrinth now provides the resolved Lithium artifact with content "lithium-0.12.0"
      When the user installs their declared mods again
      Then they should find the Sodium artifact, mod config, and lock entry unchanged and singular
      And they should find Lithium content "lithium-0.12.0" installed with one mod config and one lock entry

    Scenario: A satisfied installation is a successful no-op
      Given all of the user's mod configs have singular valid lock entries and matching installed content
      When the user installs their declared mods
      Then they should find their modlist, lockfile, and mod files unchanged
      And they should receive a report that the installation is satisfied
      And they should receive a successful operation result

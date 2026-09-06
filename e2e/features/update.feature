Feature: Update declared mods
  The user can advance eligible unpinned artifacts under current constraints while preserving safe independent work.

  Rule: Update advances only eligible desired resolutions
    # Source: ../../docs/intent.md#update

    Scenario: An eligible unpinned artifact replaces the previous artifact
      Given the user has unpinned Sodium 0.5.5 published at "2025-01-10T10:00:00Z" with content "sodium-0.5.5" installed and locked
      And Modrinth provides eligible Sodium 0.5.6 published at "2025-02-10T10:00:00Z" with artifact "sodium-0.5.6.jar", content "sodium-0.5.6", and a verified digest
      When the user updates their declared mods
      Then they should find only "sodium-0.5.6.jar" with content "sodium-0.5.6" installed for Sodium
      And they should find the exact Sodium 0.5.6 artifact and its verified digest recorded in their lockfile
      And they should find the unpinned Sodium mod config unchanged

    Scenario: Current changed constraints resolve directly without an intermediate install
      Given the user changed the installation target from Minecraft 1.20.1 to 1.20.2
      And their lockfile records Sodium 0.5.5 for Minecraft 1.20.1 but its file is missing
      And Modrinth no longer provides Sodium 0.5.5
      And Modrinth provides eligible Sodium 0.5.6 for Minecraft 1.20.2 with content "sodium-0.5.6"
      When the user updates their declared mods
      Then they should find Sodium 0.5.6 content installed directly
      And they should find the Sodium 0.5.6 resolution recorded for Minecraft 1.20.2
      And they should not be told to install or download Sodium 0.5.5 first

    Scenario: A changed explicit pin is binding
      Given the user changed the Sodium mod config pin from 0.5.5 to 0.5.6
      And Sodium 0.5.5 content remains installed and locked
      And Modrinth provides pinned Sodium 0.5.6 with content "sodium-0.5.6"
      When the user updates their declared mods
      Then they should find pin 0.5.6 in the Sodium mod config
      And they should find only Sodium 0.5.6 content installed and recorded in their lockfile

    Scenario: An unchanged explicit pin remains fixed
      Given the user has Sodium pinned to Modrinth version 0.5.5 with matching installed and locked content
      And Modrinth provides newer Sodium 0.5.6 satisfying the loader, Minecraft target, and release policy but excluded by pin 0.5.5
      When the user updates their declared mods
      Then they should find pin 0.5.5 and its exact installed and locked artifact unchanged
      And they should receive a report that pinned Sodium was already satisfied
      And they should receive a successful operation result

    Scenario: Disabling fallback reevaluates a previously fallback-resolved artifact
      Given the user has a Fabric installation targeting Minecraft 1.20.2
      And the Sodium mod config changed from version fallback enabled to disabled
      And a Sodium artifact for Minecraft 1.20.1 remains installed and recorded as recovery evidence
      And Modrinth provides no eligible Sodium artifact for Minecraft 1.20.2
      When the user updates their declared mods
      Then they should find version fallback disabled in the Sodium mod config
      And they should find the Minecraft 1.20.1 artifact content and recovery evidence preserved
      And they should be told that Sodium remains unresolved under the current constraints
      And they should be told that the requested operation did not complete successfully

  Rule: An unavailable previous artifact does not block an eligible update
    # Source: ../../docs/intent.md#update

    Scenario: An unavailable previous artifact does not block a newer replacement
      Given the user has unpinned Sodium 0.5.5 locked but its file is missing
      And Modrinth no longer provides Sodium 0.5.5
      And Modrinth provides eligible newer Sodium 0.5.6 with content "sodium-0.5.6"
      When the user updates their declared mods
      Then they should find Sodium 0.5.6 content installed and recorded in their lockfile
      And they should receive a report that the newer eligible artifact satisfied Sodium

  Rule: Resolution evidence is reconciled conservatively
    # Source: ../../docs/intent.md#the-installation-model

    Scenario: A missing lock entry resolves without discarding unrelated resolutions
      Given the user has Sodium satisfied with valid lock evidence
      And Modrinth provides no newer eligible Sodium artifact
      And Lithium is declared without a lock entry
      And Modrinth provides eligible Lithium 0.12.0 with content "lithium-0.12.0"
      When the user updates their declared mods
      Then they should find Sodium content and lock evidence unchanged
      And they should find Lithium content "lithium-0.12.0" installed and recorded once

    Scenario Outline: Corrupt resolution evidence is preserved for recovery
      Given the user has Sodium declared with <bad evidence>
      And Sodium content "sodium-existing" is installed
      And Modrinth provides eligible Sodium 0.5.6 with content "sodium-0.5.6"
      When the user updates their declared mods
      Then they should find the <bad evidence> and installed content preserved
      And they should not find Sodium 0.5.6 silently selected
      And they should receive recovery guidance for the untrustworthy evidence
      And they should be told that the requested operation did not complete successfully

      Examples:
        | bad evidence                      |
        | malformed lock data               |
        | contradictory Sodium resolutions  |
        | a lock entry without integrity data |

  Rule: Update reconciles managed state as well as advancing artifacts
    # Source: ../../docs/intent.md#the-installation-model

    Scenario: Update removes managed state for a deleted mod config
      Given the user removed the Lithium mod config from their modlist
      And their lockfile still records managed artifact "lithium-0.12.0.jar"
      And "lithium-0.12.0.jar" with content "lithium-0.12.0" is installed
      And Modrinth and CurseForge provide no eligible updates for the remaining declared mods
      When the user updates their declared mods
      Then they should find no Lithium managed artifact or lock entry
      And they should find the remaining mod configs and their artifacts unchanged
      And they should receive a report that Lithium was removed from the managed installation

    Scenario Outline: Update repairs a pinned artifact without advancing its pin
      Given the user has Sodium pinned to Modrinth version 0.5.5 with an exact valid lock entry
      And the installed Sodium artifact is <observed state>
      And Modrinth provides the exact pinned Sodium 0.5.5 artifact with content "sodium-0.5.5"
      And Modrinth provides newer Sodium 0.5.6 satisfying the other constraints but excluded by pin 0.5.5
      When the user updates their declared mods
      Then they should find Sodium 0.5.5 with content "sodium-0.5.5" installed
      And they should find pin 0.5.5 and its exact lock entry unchanged
      And they should not find Sodium 0.5.6 installed or recorded
      And they should receive a report that pinned Sodium was repaired

      Examples:
        | observed state                        |
        | missing                               |
        | present with modified content "damaged" |

  Rule: Partial updates converge safely on retry
    # Source: ../../docs/intent.md#failure-retry-and-cancellation-promises

    Scenario: Successful updates remain installed when another update fails
      Given the user has eligible updates for Sodium and Lithium
      And the user has Lithium 0.11.2 with content "lithium-0.11.2" installed and locked
      And Modrinth provides Sodium 0.5.6 with content "sodium-0.5.6"
      And Modrinth cannot complete the Lithium replacement download after automatic retries
      When the user updates their declared mods
      Then they should find Sodium 0.5.6 content installed and recorded once
      And they should find Lithium's previous working content and lock entry preserved
      And they should receive a report distinguishing updated Sodium from failed Lithium with retry guidance
      And they should be told that the requested operation did not complete successfully

    Scenario: A retry completes only the previously failed update
      Given the user has Sodium 0.5.6 satisfied after a partial update
      And Modrinth provides no newer eligible Sodium artifact
      And Lithium's previous artifact remains installed and locked
      And Modrinth now provides the eligible Lithium replacement with content "lithium-0.12.0"
      When the user updates their declared mods again
      Then they should find Sodium's artifact, mod config, and lock entry unchanged and singular
      And they should find Lithium 0.12.0 content installed with one mod config and one lock entry

    Scenario: An installation with no eligible updates is a successful no-op
      Given every declared mod has a valid locked artifact with matching installed content
      And Modrinth and CurseForge provide no eligible updates under the current constraints
      When the user updates their declared mods
      Then they should find their modlist, lockfile, and mod files unchanged
      And they should receive a report that no eligible updates were found
      And they should receive a successful operation result

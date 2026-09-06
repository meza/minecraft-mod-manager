Feature: Validate installation configuration before using it
  The user can understand unusable metadata without MMM silently discarding evidence.

  Background:
    Given the user has a Fabric installation targeting Minecraft 1.20.2 with a release-only policy

  # Source: ../../docs/commands/README.md#modlist-and-setup
  Rule: Unusable modlists are reported and preserved
    Scenario: The user encounters an unknown modlist property
      Given the modlist contains the unknown property autoUpdate
      And the user has a lockfile and installed mod files
      When the user lists their declared mods without accepting a configuration correction
      Then they should be told that autoUpdate is not a supported modlist property
      And they should receive guidance for correcting the modlist
      And they should find their modlist, lockfile and mod files unchanged
      And they should be told that the requested operation did not complete successfully

    Scenario: The user encounters malformed modlist data
      Given the modlist contains malformed JSON
      And the user has a lockfile and installed mod files
      When the user lists their declared mods
      Then they should be told that the modlist could not be parsed
      And they should receive guidance for correcting the modlist
      And they should find their modlist, lockfile and mod files unchanged
      And they should be told that the requested operation did not complete successfully

    Scenario: The user encounters an unreadable modlist
      Given the modlist exists but cannot be read
      And the user has a lockfile and installed mod files
      When the user lists their declared mods
      Then they should be told that the modlist could not be read
      And they should receive guidance for restoring access to the modlist
      And they should find their lockfile and mod files unchanged
      And they should be told that the requested operation did not complete successfully

  # Source: ../../docs/commands/README.md#modlist-and-setup
  Rule: Invalid setup prevents modifying operations before actionable artifacts are changed
    Scenario Outline: A modifying operation preserves an invalid installation
      Given the user's modlist <invalid state>
      And the lockfile records an exact Sodium 1.0 artifact whose installed file is missing
      And Modrinth provides the exact Sodium 1.0 artifact and a later Sodium 2.0 release artifact for Fabric and Minecraft 1.20.2
      When the user <operation> without accepting a configuration correction
      Then they should be told <problem>
      And Modrinth should receive no artifact request
      And they should find their modlist, lockfile and mod files unchanged

      Examples:
        | operation                   | invalid state                                                                        | problem                                  |
        | installs their declared mods | contains an unpinned Modrinth Sodium config and the unknown property autoUpdate       | that autoUpdate is unsupported           |
        | updates their declared mods  | contains an unpinned Modrinth Sodium config and the unknown property autoUpdate       | that autoUpdate is unsupported           |
        | installs their declared mods | contains an unpinned Modrinth Sodium config but omits the required loader setting      | that the required loader setting is missing |
        | updates their declared mods  | contains an unpinned Modrinth Sodium config but omits the required loader setting      | that the required loader setting is missing |
        | installs their declared mods | contains two Modrinth Sodium configs requesting the conflicting exact pins 1.0 and 2.0 | that Sodium's duplicate pins conflict    |
        | updates their declared mods  | contains two Modrinth Sodium configs requesting the conflicting exact pins 1.0 and 2.0 | that Sodium's duplicate pins conflict    |

  # Source: ../../docs/commands/README.md#modlist-and-setup
  Rule: Identical duplicate mod configs have operation-specific treatment
    Scenario: A modifying operation deduplicates identical mod configs
      Given the modlist contains two identical Modrinth mod configs for Sodium
      And the lockfile records one exact Sodium artifact
      And that exact Sodium artifact is installed
      When the user installs their declared mods
      Then they should find one Sodium mod config in their modlist
      And they should find the exact Sodium artifact unchanged in their lockfile and mods directory
      And they should be told that an identical duplicate was removed

    Scenario: Inspection reports identical mod configs without rewriting them
      Given the modlist contains two identical Modrinth mod configs for Sodium
      And the lockfile records one exact Sodium artifact
      And that exact Sodium artifact is installed
      When the user lists their declared mods
      Then they should see the duplicate Sodium mod configs reported
      And they should find their modlist, lockfile and mod files unchanged
      And they should receive a successful operation result

  # Source: ../../docs/commands/README.md#lookup-and-lockfiles
  Rule: Inspection preserves incomplete resolution evidence
    Scenario: Inspection reports a missing lockfile without creating one
      Given the user declares a Modrinth project named Sodium
      And the user has no lockfile
      And the mods directory contains sodium.jar
      When the user lists their declared mods
      Then they should see Sodium reported with missing resolution evidence
      And they should not be told that the matching filename proves Sodium is installed correctly
      And they should find no lockfile created
      And they should find their modlist and mod files unchanged

    Scenario: Inspection preserves malformed lockfile evidence
      Given the user declares a Modrinth project named Sodium
      And the lockfile contains malformed resolution evidence for Sodium
      And the mods directory contains sodium.jar
      When the user lists their declared mods
      Then they should see that Sodium's installation state cannot be established from the malformed evidence
      And they should receive recovery guidance
      And they should find their modlist, lockfile and mod files unchanged

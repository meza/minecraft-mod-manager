Feature: Check platform-declared Minecraft compatibility
  The user can inspect known compatibility and unresolved evidence without changing the installation.

  Background:
    Given the user has a Fabric installation targeting Minecraft 1.20.1 with a release-only policy
    And Mojang provides a valid Minecraft version manifest containing 1.20.1, 1.20.2 and 24w14a

  # Source: ../../docs/commands/test.md#reported-outcomes
  Rule: Compatibility findings remain distinct
    Scenario: Every declared project is compatible with the target
      Given the user declares unpinned Modrinth projects named Sodium and Lithium
      And Modrinth provides eligible release artifacts for Sodium and Lithium on Fabric and Minecraft 1.20.2
      When the user checks compatibility with Minecraft 1.20.2
      Then they should see Sodium and Lithium reported as compatible with Minecraft 1.20.2
      And they should be told that platform metadata does not establish Minecraft runtime compatibility
      And they should find their modlist, lockfile and mod files unchanged

    Scenario: Every known incompatibility is reported
      Given the user declares unpinned Modrinth projects named Sodium and Lithium
      And Modrinth conclusively provides no eligible artifact for Sodium or Lithium on Fabric and Minecraft 1.20.2
      When the user checks compatibility with Minecraft 1.20.2
      Then they should see Sodium and Lithium reported as incompatible with Minecraft 1.20.2
      And they should find their modlist, lockfile and mod files unchanged
      And they should receive a completed compatibility report with incompatibilities

    Scenario: Service failure is reported as inconclusive
      Given the user declares an unpinned CurseForge project named JourneyMap
      And CurseForge cannot determine JourneyMap eligibility after automatic retries
      When the user checks compatibility with Minecraft 1.20.2
      Then they should see JourneyMap reported as inconclusive with the known service failure
      And they should not be told that JourneyMap is incompatible
      And they should find their modlist, lockfile and mod files unchanged
      And they should be told that the requested operation did not complete successfully

    Scenario: Incompatibilities and inconclusive checks produce an incomplete result
      Given the user declares a Modrinth project named Sodium and a CurseForge project named JourneyMap
      And Modrinth conclusively provides no eligible artifact for Sodium on Fabric and Minecraft 1.20.2
      And CurseForge cannot determine JourneyMap eligibility after automatic retries
      When the user checks compatibility with Minecraft 1.20.2
      Then they should see Sodium reported as incompatible with Minecraft 1.20.2
      And they should see JourneyMap reported as inconclusive with the known service failure
      And they should see both findings in the final report
      And they should find their modlist, lockfile and mod files unchanged
      And they should be told that the compatibility check is incomplete

  # Source: ../../docs/commands/test.md#usage
  Rule: Effective mod constraints determine compatibility from platform metadata
    Scenario Outline: Version fallback changes the compatibility finding
      Given the user declares an unpinned Modrinth project named Sodium that <fallback policy>
      And Modrinth provides a Sodium release artifact for Fabric and Minecraft 1.20.1 but no Sodium artifact for Minecraft 1.20.2
      When the user checks compatibility with Minecraft 1.20.2
      Then they should see Sodium reported as <finding> with Minecraft 1.20.2
      And they should find their modlist, lockfile and mod files unchanged
      And they should be told that the compatibility check completed conclusively

      Examples:
        | fallback policy                  | finding      |
        | does not allow version fallback  | incompatible |
        | allows version fallback           | compatible   |

    Scenario Outline: A per-mod release override takes precedence over the inherited policy
      Given the user declares an unpinned Modrinth project named Sodium with <release policy>
      And Modrinth provides a Sodium beta artifact for Fabric and Minecraft 1.20.2 but no release artifact
      When the user checks compatibility with Minecraft 1.20.2
      Then they should see Sodium reported as <finding> with Minecraft 1.20.2
      And they should find their modlist, lockfile and mod files unchanged
      And they should be told that the compatibility check completed conclusively

      Examples:
        | release policy                            | finding      |
        | no per-mod release override               | incompatible |
        | a per-mod override allowing beta artifacts | compatible   |

    Scenario Outline: An exact pin constrains the artifact used for compatibility
      Given the user declares a Modrinth project named Sodium that is <pin policy>
      And Modrinth provides Sodium release artifact 0.9 for Fabric and Minecraft 1.20.1
      And Modrinth provides Sodium release artifact 1.0 for Fabric and Minecraft 1.20.2
      When the user checks compatibility with Minecraft 1.20.2
      Then they should see Sodium reported as <finding> with Minecraft 1.20.2
      And they should find their modlist, lockfile and mod files unchanged
      And they should be told that the compatibility check completed conclusively

      Examples:
        | pin policy                  | finding      |
        | unpinned                    | compatible   |
        | pinned to exact version 0.9 | incompatible |

  # Source: ../../docs/commands/test.md#usage
  Rule: Manifest-listed Minecraft targets are accepted
    Scenario: The user checks a manifest-listed snapshot
      Given the user declares an unpinned Modrinth project named Sodium
      And Modrinth provides an eligible release artifact for Sodium on Fabric and Minecraft 24w14a
      When the user checks compatibility with Minecraft 24w14a
      Then they should see Minecraft 24w14a accepted as the checked target
      And they should see Sodium reported as compatible with Minecraft 24w14a
      And they should find their modlist, lockfile and mod files unchanged

  # Source: ../../docs/commands/test.md#read-only-and-recovery-behavior
  Rule: Compatibility inspection never repairs installation state
    Scenario: The current target is checked while its managed artifact has drifted
      Given the user declares a Modrinth project named Sodium
      And the lockfile records an exact Sodium artifact for Minecraft 1.20.1
      And the installed Sodium file does not match that locked artifact
      And Modrinth provides an eligible Sodium artifact on Fabric and Minecraft 1.20.1
      When the user checks compatibility with Minecraft 1.20.1
      Then they should see Sodium reported as compatible with Minecraft 1.20.1
      And they should find the mismatched file unrepaired
      And they should find their modlist and lockfile unchanged

    Scenario: Cancellation preserves completed compatibility findings
      Given the user declares a Modrinth project named Sodium and a CurseForge project named JourneyMap
      And Modrinth provides an eligible Sodium artifact on Fabric and Minecraft 1.20.2
      And CurseForge keeps the JourneyMap compatibility lookup pending
      When the user checks compatibility with Minecraft 1.20.2 until Sodium is reported compatible while JourneyMap remains pending
      And the user cancels the compatibility check
      Then they should see Sodium's compatible finding preserved
      And they should see JourneyMap reported as not completed
      And they should be told that the compatibility check is incomplete
      And they should find their modlist, lockfile and mod files unchanged
      And they should be told that the operation was cancelled

    Scenario: Compatibility inspection reports platform findings without creating a missing lockfile
      Given the user declares unpinned Modrinth projects named Sodium and Lithium
      And the user has no lockfile
      And the mods directory contains sodium.jar and lithium.jar
      And Modrinth provides a Sodium release artifact for Fabric and Minecraft 1.20.2
      And Modrinth provides only a Lithium beta artifact for Fabric and Minecraft 1.20.2
      When the user checks compatibility with Minecraft 1.20.2
      Then they should see Sodium reported as compatible with Minecraft 1.20.2
      And they should see Lithium reported as incompatible with Minecraft 1.20.2
      And they should see missing resolution evidence reported
      And they should find no lockfile created or reconciled
      And they should find their modlist and mod files unchanged
      And they should receive a completed compatibility report with incompatibilities

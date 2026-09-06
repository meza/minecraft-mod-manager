Feature: Apply artifact eligibility during mod operations
  Add, install, and update apply the same declared constraints before changing an installation.

  Rule: Lookup applies the installation and mod config constraints together
    # Source: ../../docs/intent.md#platforms-and-artifact-lookup

    Scenario: Add installs a release artifact satisfying all inherited constraints
      Given the user has a Fabric installation targeting Minecraft 1.20.2 with a release-only policy
      And Modrinth identifies project "lithium" as Lithium
      And Modrinth provides Lithium 0.12.0 for Fabric and Minecraft 1.20.2 as a release with artifact content "lithium-0.12.0"
      When the user adds project "lithium" from Modrinth without changing the search
      Then they should find Lithium content "lithium-0.12.0" installed
      And they should find one Lithium mod config in their modlist
      And they should find that exact artifact with its verified digest recorded in their lockfile

    Scenario Outline: An incompatible constraint leaves an ordinary addition unresolved
      Given the user has a <loader> installation targeting Minecraft <target> with a <policy> policy
      And Modrinth identifies project "ferritecore" as FerriteCore
      And Modrinth provides only FerriteCore 6.0.1 for <artifact loader> and Minecraft <artifact target> as a <release type> with content "ferritecore-6.0.1"
      When the user adds project "ferritecore" from Modrinth without force and does not change the search
      Then they should be told that no eligible artifact satisfies the declared constraints
      And they should find no FerriteCore mod config, lock entry, or installed artifact
      And they should be told that the requested operation did not complete successfully

      Examples:
        | loader | target | policy       | artifact loader | artifact target | release type |
        | Fabric | 1.20.2 | release-only | Forge           | 1.20.2         | release      |
        | Fabric | 1.20.2 | release-only | Fabric          | 1.20.1         | release      |
        | Fabric | 1.20.2 | release-only | Fabric          | 1.20.2         | beta         |

    Scenario: Install applies a per-mod release policy instead of the installation policy
      Given the user has a Fabric installation targeting Minecraft 1.20.2 with a release-only policy
      And the Sodium mod config allows release and beta artifacts but has no lock entry
      And Modrinth provides Sodium 0.5.6 for Fabric and Minecraft 1.20.2 as a beta with content "sodium-0.5.6-beta"
      When the user installs their declared mods
      Then they should find Sodium content "sodium-0.5.6-beta" installed and recorded in their lockfile
      And they should find the Sodium release-and-beta override unchanged
      And they should find the installation-wide release-only policy unchanged

    Scenario Outline: Add accepts artifacts for manifest-listed Minecraft targets
      Given the user has a Fabric installation targeting Minecraft <target>
      And Mojang identifies <target> as <kind>
      And Modrinth identifies project "snapshot-helper" as Snapshot Helper
      And Modrinth provides Snapshot Helper 2 for Fabric and Minecraft <target> as a release with content "snapshot-helper-2"
      When the user adds project "snapshot-helper" from Modrinth without changing the search
      Then they should find Snapshot Helper content "snapshot-helper-2" installed
      And they should find its mod config and exact artifact lock entry

      Examples:
        | target      | kind         |
        | 1.20.2      | a release    |
        | 23w31a      | a snapshot   |
        | 1.20.2-pre1 | a prerelease |

  Rule: Pins identify exact platform artifacts
    # Source: ../../docs/intent.md#platforms-and-artifact-lookup

    Scenario: Install resolves a Modrinth pin by platform version number
      Given the user has Sounds Be Gone pinned to Modrinth version 1.3.1 without a lock entry or installed artifact
      And Modrinth provides version 1.3.1 in artifact "soundsbegone-v1.3.1-fabric.jar" with content "soundsbegone-1.3.1"
      When the user installs their declared mods
      Then they should find "soundsbegone-v1.3.1-fabric.jar" with content "soundsbegone-1.3.1" installed
      And they should find the exact artifact and its verified digest recorded in their lockfile
      And they should find Modrinth pin 1.3.1 unchanged in the mod config

    Scenario: Install resolves a CurseForge pin by complete artifact filename
      Given the user has Sounds Be Gone pinned to CurseForge filename "soundsbegone-1.3.1.jar" without a lock entry or installed artifact
      And CurseForge provides "soundsbegone-1.3.1.jar" with content "soundsbegone-1.3.1"
      When the user installs their declared mods
      Then they should find "soundsbegone-1.3.1.jar" with content "soundsbegone-1.3.1" installed
      And they should find the exact artifact and its verified digest recorded in their lockfile
      And they should find the CurseForge filename pin unchanged in the mod config

  Rule: Version fallback is explicit and bounded
    # Source: ../../docs/intent.md#platforms-and-artifact-lookup

    Scenario: Install falls back to the nearest earlier release in the same series
      Given the user has a Fabric installation targeting Minecraft 1.20.2
      And the Lithium mod config has version fallback enabled and no lock entry
      And Modrinth has no eligible Lithium artifact for Minecraft 1.20.2
      And Modrinth provides eligible Lithium artifacts for Minecraft 1.20.1 and 1.20
      When the user installs their declared mods
      Then they should find the Lithium artifact for Minecraft 1.20.1 installed and recorded in their lockfile
      And they should be told that version fallback was used from Minecraft 1.20.2 to 1.20.1

    Scenario: Install does not fall back when permission is omitted
      Given the user has a Fabric installation targeting Minecraft 1.20.2
      And the Lithium mod config has no version fallback setting or lock entry
      And Modrinth has no eligible Lithium artifact for Minecraft 1.20.2
      And Modrinth provides an eligible Lithium artifact for Minecraft 1.20.1
      When the user installs their declared mods
      Then they should be told that no eligible artifact satisfies the declared constraints
      And they should find no Lithium lock entry or installed artifact
      And they should find the Lithium mod config unchanged
      And they should be told that the requested operation did not complete successfully

    Scenario Outline: Install does not cross an unsupported fallback boundary
      Given the user has a Fabric installation targeting Minecraft <target>
      And the Lithium mod config has version fallback enabled and no lock entry
      And Modrinth has no eligible Lithium artifact for Minecraft <target>
      And Modrinth provides an eligible Lithium artifact for Minecraft <available target>
      When the user installs their declared mods
      Then they should be told that no eligible artifact satisfies the declared constraints
      And they should find no Lithium lock entry or installed artifact
      And they should find the Lithium mod config unchanged
      And they should be told that the requested operation did not complete successfully

      Examples:
        | target | available target |
        | 1.20.2 | 1.19.4           |
        | 23w31a | 23w30a           |

  Rule: Unusable metadata and unavailable services are not absence
    # Source: ../../docs/intent.md#platforms-and-artifact-lookup

    Scenario Outline: Update does not bypass the newest eligible artifact when required metadata is missing
      Given the user has unpinned Lithium 0.11.0 with content "lithium-0.11.0" installed and locked
      And Modrinth provides Lithium 0.12.0 as the newest eligible artifact but without <metadata>
      And Modrinth provides older eligible Lithium 0.11.2 with complete metadata and content "lithium-0.11.2"
      When the user updates their declared mods
      Then they should find Lithium 0.11.0 content and its lock entry unchanged
      And they should not find Lithium 0.11.2 installed or recorded as a substitute
      And they should be told that the newest eligible artifact lacks required metadata
      And they should be told that the requested operation did not complete successfully

      Examples:
        | metadata              |
        | download information  |
        | integrity information |

    Scenario: An update service failure remains unknown
      Given the user has unpinned Sodium 0.5.5 with content "sodium-0.5.5" installed and locked
      And Modrinth cannot provide Sodium project or artifact metadata after automatic retries
      When the user updates their declared mods
      Then they should be told that the Sodium lookup failed
      And they should not be told that Sodium is absent or incompatible
      And they should find Sodium content and its lock entry unchanged
      And they should receive guidance for retrying
      And they should be told that the requested operation did not complete successfully

  Rule: Update requires both a later publication time and different content
    # Source: ../../docs/intent.md#platforms-and-artifact-lookup

    Scenario Outline: Update applies both parts of the unpinned candidate rule
      Given the user has unpinned Lithium 0.11.2 published at "2025-01-10T10:00:00Z" with content "content-a" installed and locked
      And Modrinth provides Lithium 0.12.0 published at "<candidate time>" with content "<candidate content>"
      When the user updates their declared mods
      Then they should find Lithium <installed version> with content "<installed content>" installed and locked
      And they should receive a report that <reported outcome>
      And they should receive a successful operation result

      Examples:
        | candidate time       | candidate content | installed version | installed content | reported outcome             |
        | 2025-02-10T10:00:00Z | content-b         | 0.12.0            | content-b         | Lithium was updated          |
        | 2025-01-10T10:00:00Z | content-b         | 0.11.2            | content-a         | no eligible update was found |
        | 2024-12-10T10:00:00Z | content-b         | 0.11.2            | content-a         | no eligible update was found |
        | 2025-02-10T10:00:00Z | content-a         | 0.11.2            | content-a         | no eligible update was found |

    Scenario: Update uses publication time rather than ordering nonsemantic version labels
      Given the user has unpinned Lithium version label "release-20" published at "2025-01-10T10:00:00Z" with content "content-a" installed and locked
      And Modrinth provides Lithium version label "build-alpha" published at "2025-02-10T10:00:00Z" with different content "content-b"
      When the user updates their declared mods
      Then they should find Lithium version label "build-alpha" with content "content-b" installed and locked
      And they should receive a report that Lithium was updated

    Scenario Outline: Update does not advance an artifact for display metadata alone
      Given the user has unpinned Lithium 0.11.2 published at "2025-01-10T10:00:00Z" with content "content-a" installed and locked
      And Modrinth provides the same publication time and content with a changed <field>
      When the user updates their declared mods
      Then they should find Lithium 0.11.2 with content "content-a" installed and locked
      And they should receive a report that no eligible update was found
      And they should receive a successful operation result

      Examples:
        | field          |
        | display name   |
        | version string |

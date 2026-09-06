Feature: Add a mod
  The user can declare an explicitly selected platform project and reconcile its artifact into an installation.

  Rule: A resolved addition declares and installs one exact artifact
    # Source: ../../docs/intent.md#add

    Scenario: The user adds a Modrinth project
      Given the user has a Fabric installation targeting Minecraft 1.20.2 with a release-only policy
      And Modrinth identifies project "AANobbMI" as Sodium
      And Modrinth provides eligible Sodium 0.5.6 artifact "sodium-0.5.6.jar" with content "sodium-0.5.6" and a verified digest
      When the user adds project "AANobbMI" from Modrinth without changing the search
      Then they should find "sodium-0.5.6.jar" with content "sodium-0.5.6" installed
      And they should find one Sodium mod config for Modrinth project "AANobbMI" in their modlist
      And they should find the exact Sodium artifact and its verified digest recorded once in their lockfile
      And they should receive a report that Sodium was declared and installed

    Scenario: The user adds an exact CurseForge pin
      Given the user has a Fabric installation targeting Minecraft 1.20.2 with a release-only policy
      And CurseForge identifies project "306612" as Fabric API
      And CurseForge provides eligible artifact "fabric-api-0.91.6+1.20.2.jar" with content "fabric-api-0.91.6" and a verified digest
      When the user adds CurseForge project "306612" pinned to filename "fabric-api-0.91.6+1.20.2.jar" without changing the search
      Then they should find "fabric-api-0.91.6+1.20.2.jar" with content "fabric-api-0.91.6" installed
      And they should find the filename pin in the Fabric API mod config
      And they should find the exact supplied artifact and its verified digest recorded in their lockfile

    Scenario: Repeating an already satisfied addition is a no-op
      Given the user has Sodium declared from Modrinth project "AANobbMI"
      And "sodium-0.5.6.jar" with content "sodium-0.5.6" satisfies its lock entry
      When the user adds project "AANobbMI" from Modrinth again without changing the search
      Then they should find one Sodium mod config in their modlist
      And they should find one Sodium lock entry in their lockfile
      And they should find the installed Sodium content unchanged
      And they should receive a report that Sodium was already satisfied
      And they should receive a successful operation result

  Rule: Explicit settings update only the selected mod config
    # Source: ../../docs/intent.md#add

    Scenario: Supplied settings replace matching settings and omitted settings are preserved
      Given the user has Sodium declared with Modrinth pin 0.5.5, release-only override, and version fallback disabled
      And Modrinth provides eligible Sodium 0.5.6 as a beta with content "sodium-0.5.6"
      When the user adds Sodium with pin 0.5.6 and a release-and-beta override without changing the fallback setting or search
      Then they should find pin 0.5.6 and the release-and-beta override in the Sodium mod config
      And they should find version fallback still disabled in the Sodium mod config
      And they should find the installation-wide release policy unchanged
      And they should find Sodium content "sodium-0.5.6" installed and recorded in their lockfile

    Scenario: Omitting every optional setting preserves nondefault existing settings
      Given the user has Sodium declared with Modrinth pin 0.5.5, a release-and-beta override, and version fallback enabled
      And Sodium 0.5.5 is installed with matching lock evidence
      And Modrinth provides newer artifacts outside the existing pin
      When the user adds Sodium again without supplying a pin, release types, or version fallback and does not change the search
      Then they should find pin 0.5.5, the release-and-beta override, and version fallback enabled in the Sodium mod config
      And they should find Sodium 0.5.5 content and its lock evidence unchanged
      And they should receive a successful operation result

    Scenario: Explicit unpin selects the newest eligible artifact
      Given the user has Sodium pinned to Modrinth version 0.5.5 with installed content "sodium-0.5.5"
      And Modrinth provides eligible Sodium 0.5.6 with content "sodium-0.5.6"
      When the user adds Sodium with its pin explicitly cleared and without changing the search
      Then they should find no pin in the Sodium mod config
      And they should find Sodium content "sodium-0.5.6" installed and recorded in their lockfile

    Scenario: Explicit false clears version fallback for one mod
      Given the user has Sodium declared with version fallback enabled
      And Modrinth provides an eligible Sodium artifact for the exact Minecraft target with content "sodium-exact-target"
      When the user adds Sodium with version fallback explicitly disabled and without changing the search
      Then they should find version fallback disabled in the Sodium mod config
      And they should find the installation-wide settings unchanged
      And they should find content "sodium-exact-target" installed and recorded in their lockfile

    Scenario: An ordinary unresolved fallback removal preserves the prior setting and artifact
      Given the user has a Fabric installation targeting Minecraft 1.20.2
      And Sodium has version fallback enabled with a Minecraft 1.20.1 artifact installed and recorded as recovery evidence
      And Modrinth provides no eligible Sodium artifact for Minecraft 1.20.2
      When the user adds Sodium with version fallback explicitly disabled without force and does not change the search
      Then they should find version fallback still enabled in the Sodium mod config
      And they should find the Minecraft 1.20.1 artifact content and recovery evidence unchanged
      And they should be told that the requested setting change could not resolve an eligible artifact
      And they should be told that the requested operation did not complete successfully

    Scenario: Force saves an unresolved fallback removal while preserving working bytes
      Given the user has a Fabric installation targeting Minecraft 1.20.2
      And Sodium has version fallback enabled with a Minecraft 1.20.1 artifact installed and recorded as recovery evidence
      And Modrinth provides no eligible Sodium artifact for Minecraft 1.20.2
      When the user adds Sodium with version fallback explicitly disabled with force and does not change the search
      Then they should find version fallback disabled in the Sodium mod config
      And they should find the Minecraft 1.20.1 artifact content and recovery evidence preserved
      And they should find no artifact resolution invented for the unresolved Minecraft 1.20.2 request
      And they should be told that the declared settings and installed artifact do not yet match
      And they should be told that the requested operation did not complete successfully

    Scenario: A version and unpin request is invalid
      Given the user has Sodium pinned to Modrinth version 0.5.5 with installed content "sodium-0.5.5"
      When the user adds Sodium with both version 0.5.6 and unpin
      Then they should be told that version and unpin cannot be combined
      And they should find their Sodium mod config, lock entry, and installed content unchanged
      And they should receive a rejection of the invalid request

  Rule: Ordinary unresolved additions preserve established state
    # Source: ../../docs/intent.md#lookup-failure-and-search-correction

    Scenario Outline: A first-time ordinary lookup miss declares nothing
      Given the user has a Fabric installation targeting Minecraft 1.20.2
      And Modrinth <lookup result> for project "missing-sodium"
      When the user adds project "missing-sodium" from Modrinth without force and does not change the search
      Then they should receive a report that <reported outcome>
      And they should find no mod config or lock entry for project "missing-sodium"
      And they should find no new artifact installed
      And they should be told that the requested operation did not complete successfully

      Examples:
        | lookup result                                           | reported outcome                                      |
        | confirms that the project does not exist                | the project was not found                             |
        | confirms the project but has no eligible artifact       | no artifact satisfied the declared constraints        |

    Scenario: Service failure does not become a missing project
      Given the user has a Fabric installation targeting Minecraft 1.20.2
      And Modrinth cannot confirm project "AANobbMI" after automatic retries
      When the user adds project "AANobbMI" from Modrinth and does not change the search
      Then they should be told that Modrinth could not complete the lookup
      And they should not be told that the project or artifact is absent
      And they should find no Sodium mod config, lock entry, or installed artifact
      And they should receive guidance for retrying
      And they should be told that the requested operation did not complete successfully

    Scenario: An unresolved pin change preserves the previous pin
      Given the user has Sodium pinned to Modrinth version 0.5.5 with installed content "sodium-0.5.5" and matching recovery evidence
      And Modrinth confirms Sodium but has no eligible version 0.5.6
      When the user adds Sodium pinned to 0.5.6 without force and does not change the search
      Then they should find pin 0.5.5 in the Sodium mod config
      And they should find the previous lock entry and installed content "sodium-0.5.5" unchanged
      And they should be told that pin 0.5.6 could not be resolved
      And they should be told that the requested operation did not complete successfully

  Rule: Force can declare only a verified unresolved project
    # Source: ../../docs/intent.md#lookup-failure-and-search-correction

    Scenario: Force saves a verified project without manufacturing an installation
      Given the user has a Fabric installation targeting Minecraft 1.20.2 with a release-only policy
      And Modrinth confirms project "AANobbMI" as Sodium
      And Modrinth provides only a beta Sodium artifact for Fabric and a release Sodium artifact for Forge
      When the user adds project "AANobbMI" from Modrinth with force and does not change the search
      Then they should find an unresolved Sodium mod config with the requested constraints in their modlist
      And they should find no Sodium lock entry or installed Sodium artifact
      And they should be told that the mod config was saved but installation remains incomplete
      And they should receive guidance to adjust the mod config or retry install
      And they should be told that the requested operation did not complete successfully

    Scenario: Force records an unresolved pin while preserving a working artifact
      Given the user has Sodium pinned to Modrinth version 0.5.5 with installed content "sodium-0.5.5" and matching recovery evidence
      And Modrinth confirms Sodium but has no eligible version 0.5.6
      When the user adds Sodium pinned to 0.5.6 with force and does not change the search
      Then they should find pin 0.5.6 in the Sodium mod config
      And they should find installed content "sodium-0.5.5" and its recovery evidence preserved
      And they should find no artifact resolution invented for the unresolved 0.5.6 request
      And they should be told that the declared pin and installed artifact do not yet match
      And they should be told that the requested operation did not complete successfully

    Scenario: Force does not save an unverified identity
      Given the user has a Fabric installation targeting Minecraft 1.20.2
      And CurseForge cannot confirm project "999999" after automatic retries
      When the user adds project "999999" from CurseForge with force and does not change the search
      Then they should find no mod config, lock entry, or installed artifact for project "999999"
      And they should be told that force cannot verify the project identity
      And they should be told that the requested operation did not complete successfully

    Scenario: Force does not accept corrupt content for a resolved artifact
      Given the user has a Fabric installation targeting Minecraft 1.20.2
      And Modrinth confirms project "AANobbMI" as Sodium and resolves eligible Sodium 0.5.6 with a verified digest
      And Modrinth serves content that does not match the resolved Sodium digest
      When the user adds project "AANobbMI" from Modrinth with force and does not change the search
      Then they should find the Sodium mod config in their modlist
      And they should find no corrupt Sodium content installed
      And they should not be told that force bypassed artifact integrity or completed the installation
      And they should receive guidance to retry the unresolved installation
      And they should be told that the requested operation did not complete successfully

  Rule: Resolved download failures remain safely retryable
    # Source: ../../docs/intent.md#add

    Scenario: A first-time download failure keeps the confirmed declaration
      Given the user has a Fabric installation targeting Minecraft 1.20.2
      And Modrinth identifies project "AANobbMI" as Sodium
      And Modrinth resolves eligible Sodium 0.5.6 but its download fails after automatic retries
      When the user adds project "AANobbMI" from Modrinth without changing the search
      Then they should find the Sodium mod config in their modlist
      And they should find no installed Sodium artifact
      And they should be told that the confirmed project was declared but installation is unresolved
      And they should receive guidance to retry install
      And they should be told that the requested operation did not complete successfully

    Scenario: A pin-change download failure keeps the requested pin and previous working bytes
      Given the user has Sodium pinned to Modrinth version 0.5.5 with installed content "sodium-0.5.5" and matching recovery evidence
      And Modrinth resolves eligible Sodium 0.5.6 but its download fails after automatic retries
      When the user adds Sodium pinned to 0.5.6 without changing the search
      Then they should find pin 0.5.6 in the Sodium mod config
      And they should find installed content "sodium-0.5.5" and its recovery evidence preserved
      And they should be told that the installation does not satisfy requested pin 0.5.6
      And they should receive guidance to retry install
      And they should be told that the requested operation did not complete successfully

    Scenario: Install completes a previously failed pin change
      Given the user has Sodium requesting Modrinth pin 0.5.6 while content "sodium-0.5.5" remains installed with recovery evidence
      And Modrinth now provides version 0.5.6 with artifact "sodium-0.5.6.jar", content "sodium-0.5.6", and a verified digest
      When the user installs their declared mods
      Then they should find pin 0.5.6 in the Sodium mod config
      And they should find only "sodium-0.5.6.jar" with content "sodium-0.5.6" installed for Sodium
      And they should find the exact 0.5.6 artifact and its verified digest recorded once in their lockfile

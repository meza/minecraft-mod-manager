Feature: Minecraft version metadata remains useful when refresh fails
  The user can distinguish validated version information from unavailable evidence.

  Background:
    Given the user has a Fabric installation targeting Minecraft 1.20.1 with a release-only policy
    And the user has an empty modlist and an empty lockfile

  # Source: ../../docs/intent.md#operational-expectations
  Rule: Successfully validated manifests are reusable across invocations
    Scenario: The user obtains current metadata before using an older cached manifest
      Given the user has a cached Minecraft version manifest fetched on 2024-01-01 identifying 1.20.1 as the latest stable release
      And Mojang provides a valid Minecraft version manifest identifying 1.20.2 as the latest stable release
      When the user checks compatibility with the latest Minecraft release
      Then they should see Minecraft 1.20.2 as the checked target
      And they should find the validated manifest cached with its fetch time
      And they should find their modlist, lockfile and mod files unchanged
      And they should receive a successful operation result

    Scenario: The user reuses a manifest fetched by an earlier invocation
      Given the user has no cached Minecraft version manifest
      And Mojang provides a valid Minecraft version manifest identifying 1.20.2 as the latest stable release
      When the user checks compatibility with the latest Minecraft release
      Then they should see Minecraft 1.20.2 as the checked target
      And they should find the validated manifest cached with its fetch time
      And they should receive a successful operation result
      Given Mojang cannot provide the Minecraft version manifest after automatic retries
      When the user checks compatibility with the latest Minecraft release again
      Then they should see Minecraft 1.20.2 as the checked target
      And they should be warned that cached metadata is being used
      And they should see the original fetch time of that cached manifest
      And they should receive a successful operation result

    Scenario: The user receives the manifest after a temporary service failure
      Given the user has no cached Minecraft version manifest
      And Mojang temporarily fails to provide the Minecraft version manifest
      And Mojang provides a valid manifest identifying 1.20.2 as the latest stable release on an automatic retry
      When the user checks compatibility with the latest Minecraft release
      Then they should see Minecraft 1.20.2 as the checked target
      And they should find the validated manifest cached with its fetch time
      And they should receive a successful operation result

    Scenario: The user uses one refreshed manifest throughout a compatibility check
      Given the user declares two unpinned Modrinth projects named Atlas and Beacon
      And Mojang provides a valid manifest identifying 1.20.2 as the latest stable release
      And Modrinth provides eligible release artifacts for Atlas and Beacon on Fabric and Minecraft 1.20.2
      When the user checks compatibility with the latest Minecraft release
      Then they should see both projects checked against Minecraft 1.20.2
      And Mojang should receive only one successful manifest request during that invocation
      And the user should receive a successful operation result

  # Source: ../../docs/intent.md#operational-expectations
  Rule: Failed refresh uses valid cached evidence without a hard expiry
    Scenario: The user uses an old cached latest release with a staleness warning
      Given the user has a valid cached Minecraft version manifest fetched on 2015-01-01 identifying 1.8.1 as the latest stable release
      And Mojang cannot provide the Minecraft version manifest after automatic retries
      When the user checks compatibility with the latest Minecraft release
      Then they should see Minecraft 1.8.1 as the checked target
      And they should be warned that cached metadata is being used
      And they should see 2015-01-01 as the manifest fetch date
      And they should be warned that a newer Minecraft release may exist
      And they should receive a successful operation result

    Scenario: The user validates an explicit version found in cached metadata
      Given the user has a valid cached Minecraft version manifest fetched on 2024-01-01 containing 1.20.2
      And Mojang cannot provide the Minecraft version manifest after automatic retries
      When the user checks compatibility with Minecraft 1.20.2
      Then they should see Minecraft 1.20.2 accepted as the checked target
      And they should be warned that cached metadata fetched on 2024-01-01 is being used
      And they should receive a successful operation result

    Scenario: The user cannot conclude that a version absent from stale metadata is invalid
      Given the user has a valid cached Minecraft version manifest that does not contain 1.20.2
      And Mojang cannot provide the Minecraft version manifest after automatic retries
      When the user checks compatibility with Minecraft 1.20.2 without choosing an alternative target
      Then they should be told that Minecraft 1.20.2 could not be validated
      And they should not be told that Minecraft 1.20.2 is invalid or incompatible
      And they should find their modlist, lockfile and mod files unchanged
      And they should be told that the requested operation did not complete successfully

  # Source: ../../docs/intent.md#operational-expectations
  Rule: Missing or corrupt evidence cannot establish a Minecraft target
    Scenario Outline: The user receives actionable failure when no valid manifest is available
      Given the user has <cache> Minecraft version manifest cache
      And Mojang cannot provide the Minecraft version manifest after automatic retries
      When the user checks compatibility with the latest Minecraft release without choosing an alternative target
      Then they should be told that the target version could not be determined
      And they should receive guidance for retrying
      And they should find their modlist, lockfile and mod files unchanged
      And they should be told that the requested operation did not complete successfully

      Examples:
        | cache     |
        | no        |
        | a corrupt |

    Scenario: The user retains valid cached metadata when a refresh supplies invalid data
      Given the user has a valid cached Minecraft version manifest fetched on 2024-01-01 containing 1.20.2
      And Mojang supplies invalid manifest data on every refresh attempt
      When the user checks compatibility with Minecraft 1.20.2
      Then they should be warned that cached metadata fetched on 2024-01-01 is being used
      And they should find their previously validated cached manifest unchanged
      And they should receive a successful operation result

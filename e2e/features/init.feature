Feature: Initialize a managed installation
  The user can create installation metadata without adopting or deleting existing jars.

  Background:
    Given the user has an existing usable mods directory

  # Source: ../../docs/commands/init.md#inputs-and-defaults
  Rule: Initialization records complete validated inputs
    Scenario: The user initializes with complete explicit values
      Given Mojang provides a valid Minecraft version manifest containing 1.20.2
      When the user initializes server/modlist.json with Fabric, Minecraft 1.20.2, release and beta artifacts, and the existing mods directory
      Then they should find server/modlist.json containing those installation settings and no mod configs
      And they should find an empty server/modlist-lock.json
      And they should find no jars downloaded or adopted

    # Source: ../../docs/intent.md#active-display-and-permanent-transcript
    Scenario: The user initializes with documented defaults
      Given Mojang provides a valid Minecraft version manifest identifying 1.20.2 as the latest stable release
      When the user initializes a Fabric installation using the documented defaults
      Then they should find a modlist targeting Minecraft 1.20.2 with a release-only policy
      And they should find the mods directory recorded as mods
      And they should find an empty lockfile
      And they should see the resolved loader, Minecraft target, release policy and mods directory reported
      And they should see equivalent resolved initialization settings use the same durable text and formatting under matching locale and character support whether values were supplied explicitly, accepted from offered defaults, or applied as documented defaults

  # Source: ../../docs/commands/init.md#inputs-and-defaults
  Rule: Paths are based on the selected modlist
    Scenario: A relative mods path is resolved from a custom modlist directory
      Given the user has an existing server/client-mods directory
      And Mojang provides a valid Minecraft version manifest containing 1.20.2
      When the user initializes server/client.json with Fabric, Minecraft 1.20.2, a release-only policy, and the relative mods path client-mods
      Then they should find server/client.json containing client-mods as its mods directory
      And they should find an empty server/client-lock.json
      And they should see server/client-mods established as the managed mods directory

    Scenario: An absolute mods path remains absolute
      Given the user has an existing usable mods directory outside the modlist directory
      And Mojang provides a valid Minecraft version manifest containing 1.20.2
      When the user initializes server/client.json with Fabric, Minecraft 1.20.2, a release-only policy, and that absolute mods path
      Then they should find server/client.json containing that absolute mods path
      And they should find an empty server/client-lock.json

  # Source: ../../docs/commands/init.md#existing-metadata-and-reset-behavior
  Rule: Reset requires explicit authority and preserves jars
    Scenario: The user explicitly resets an existing installation
      Given the user has an existing modlist and lockfile
      And the mods directory contains managed and unmanaged jars
      And Mojang provides a valid Minecraft version manifest containing 1.20.2
      When the user authorizes resetting the metadata with Fabric, Minecraft 1.20.2, a release-only policy, and the existing mods directory
      Then they should be told that the modlist and lockfile were reset
      And they should find a modlist containing the supplied settings and no mod configs
      And they should find an empty lockfile
      And they should find every existing jar unchanged

    Scenario: The user explicitly resets an orphan lockfile
      Given the user has no modlist
      And the user has an orphan lockfile containing an exact artifact
      And the mods directory contains that artifact and an unmanaged jar
      And Mojang provides a valid Minecraft version manifest containing 1.20.2
      When the user authorizes resetting the metadata with Fabric, Minecraft 1.20.2, a release-only policy, and the existing mods directory
      Then they should be told that the orphan lockfile was reset
      And they should find a modlist containing the supplied settings and no mod configs
      And they should find an empty lockfile
      And they should find both existing jars unchanged

  # Source: ../../docs/commands/init.md#interactive-collection
  Rule: Complete explicit setup is validated before it is written
    Scenario: The user supplies a complete valid setup
      Given Mojang provides a valid Minecraft version manifest containing 24w14a
      When the user initializes with Quilt, Minecraft 24w14a, alpha and release artifacts, and the existing mods directory
      Then they should see the complete resolved setup reported
      And they should find the accepted setup in the modlist
      And they should find an empty lockfile

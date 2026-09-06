Feature: Preserve installation file ownership boundaries
  The user relies on MMM to distinguish managed, unmanaged and excluded files and to contain every file operation.

  Background:
    Given the user has an initialized installation

  # Source: ../../docs/intent.md#file-ownership-and-adoption
  # Source: ../../docs/commands/README.md#file-ownership-and-exclusions
  Rule: Visible unmanaged jars coexist with unrelated managed work

    Scenario: An unmanaged jar does not block an independent installation
      Given the user has a visible unmanaged jar in their mods directory
      And the user has a declared Sodium mod ready to install
      And Modrinth provides its eligible artifact with verifiable content
      When the user installs their declared mods
      Then they should find the exact Sodium artifact installed and locked
      And they should find the visible unmanaged jar unchanged and still unmanaged
      And they should see the visible unmanaged jar reported as unmanaged

    Scenario: An unmanaged jar does not block an independent update
      Given the user has a visible unmanaged jar in their mods directory
      And the user has an installed unpinned Sodium artifact
      And Modrinth provides a newer eligible Sodium artifact with different verifiable content
      When the user updates their declared mods
      Then they should find the newer Sodium artifact installed and locked
      And they should find the visible unmanaged jar unchanged and still unmanaged
      And they should see the visible unmanaged jar reported as unmanaged

    Scenario: An unmanaged jar does not block an independent removal
      Given the user has a visible unmanaged jar in their mods directory
      And the user has a managed Sodium artifact installed
      When the user authorizes MMM to remove Sodium
      Then they should find the Sodium mod config, lock entry and managed artifact absent
      And they should find the visible unmanaged jar unchanged and still unmanaged
      And they should see the visible unmanaged jar reported as unmanaged

    Scenario: An unmanaged jar does not block an independent version change
      Given the user has a visible unmanaged jar in their mods directory
      And the user's managed artifacts have eligible replacements for another Minecraft target
      When the user changes their installation to that Minecraft target
      Then they should find the target and managed artifacts changed consistently
      And they should find the visible unmanaged jar unchanged and still unmanaged
      And they should see the visible unmanaged jar reported as unmanaged

    Scenario: Repairing a modified managed file does not adopt its content
      Given the user has a Sodium mod config and a valid lock entry for artifact A
      And the managed Sodium file contains locally modified content
      And Modrinth can provide the exact verified content for artifact A
      When the user installs their declared mods
      Then they should find the exact content of artifact A installed
      And they should find the existing lock entry for artifact A unchanged
      And they should not find the locally modified content recorded as a locked artifact

  # Source: ../../docs/intent.md#platforms-and-artifact-lookup
  # Source: ../../docs/commands/README.md#lookup-and-lockfiles
  Rule: A managed mod retains its adopted platform association

    Scenario: An update does not silently migrate a managed mod to another platform
      Given the user has a Modrinth Sodium mod config with a valid locked artifact installed
      And Modrinth provides no newer eligible Sodium artifact
      And CurseForge provides a newer eligible artifact for its own Sodium project
      When the user updates their declared mods
      Then they should find Sodium still associated with Modrinth in their modlist and lockfile
      And they should find the existing Modrinth Sodium artifact unchanged
      And they should find no CurseForge Sodium artifact installed or locked
      And they should receive a successful operation result

  # Source: ../../docs/intent.md#excluded-files
  # Source: ../../docs/commands/README.md#file-ownership-and-exclusions
  Rule: Ignore patterns make matching files virtually invisible to MMM

    Scenario: Ignore patterns are read beside the selected modlist and apply relative to its mods directory
      Given the user selects "instances/server/server.json" as their modlist
      And that modlist selects "mods" as a relative mods directory
      And "instances/server/.mmmignore" matches "private-*.jar"
      And the user has "instances/server/mods/private-library.jar"
      And the user has "instances/server/mods/public-library.jar"
      When the user lists their mods
      Then they should find "private-library.jar" excluded from unmanaged reporting
      And they should see "public-library.jar" reported as unmanaged
      And they should find both files unchanged

    Scenario: Scan does not report or adopt an ignored unmanaged jar
      Given the user has "private.jar" matched by .mmmignore
      And Modrinth would recognize "private.jar" if it were visible
      When the user scans with --add
      Then they should find "private.jar" unreported, unchanged and excluded
      And they should find no mod config or lock entry created from "private.jar"

    Scenario: Update does not replace an ignored managed file
      Given the user has an installed Sodium artifact matched by .mmmignore
      And Modrinth provides a newer eligible Sodium artifact
      When the user updates their declared mods
      Then they should find the ignored Sodium file unchanged

    Scenario: Removal does not delete an ignored selected file
      Given the user has a selected Sodium file matched by .mmmignore
      When the user attempts to remove Sodium with --force
      Then they should find the ignored Sodium file unchanged

    Scenario: Prune does not report or delete an ignored unmanaged jar
      Given the user has "private.jar" matched by .mmmignore
      When the user prunes with --force
      Then they should find "private.jar" unreported, unchanged and excluded

    Scenario: An invalid ignore pattern stops mutation before exclusions can be exposed
      Given the user's .mmmignore contains an invalid pattern on line 4
      And the user has both managed and unmanaged jars in their mods directory
      When the user updates their declared mods
      Then they should be told that the .mmmignore pattern on line 4 is invalid
      And they should find their modlist, lockfile and mod files unchanged
      And they should be told that the requested operation did not complete successfully

  # Source: ../../docs/intent.md#excluded-files
  # Source: ../../docs/commands/README.md#file-ownership-and-exclusions
  Rule: Disabled files use baseline exclusion without satisfying enabled declared state

    Scenario: Scan does not report or adopt a disabled file
      Given the user has "optional.jar.disabled" in their mods directory
      And Modrinth would recognize its content if the file were visible
      When the user scans with --add
      Then they should find "optional.jar.disabled" unreported, unchanged and excluded
      And they should find no mod config or lock entry created from that file

    Scenario: Update does not replace a disabled managed file
      Given the user has a Sodium mod config and a corresponding file named "sodium.jar.disabled"
      And Modrinth provides a newer eligible Sodium artifact
      When the user updates their declared mods
      Then they should find "sodium.jar.disabled" unchanged

    Scenario: Removal does not delete a disabled selected file
      Given the user has a selected Sodium file named "sodium.jar.disabled"
      When the user attempts to remove Sodium with --force
      Then they should find "sodium.jar.disabled" unchanged

    Scenario: Prune does not report or delete a disabled file
      Given the user has "optional.jar.disabled" in their mods directory
      When the user prunes with --force
      Then they should find "optional.jar.disabled" unreported and unchanged

    Scenario: Installing a declared enabled artifact preserves its disabled counterpart
      Given the user has a Sodium mod config with a valid lock entry for "sodium.jar"
      And the user renamed the managed file to "sodium.jar.disabled"
      And Modrinth can provide the exact verified content for "sodium.jar"
      When the user installs their declared mods
      Then they should find "sodium.jar.disabled" unchanged
      And they should find the exact locked content installed as "sodium.jar"
      And they should find both files present

  # Source: ../../docs/intent.md#paths-and-installation-boundaries
  # Source: ../../docs/commands/README.md#file-ownership-and-exclusions
  Rule: File operations remain within the trusted installation boundary

    Scenario: A relative mods directory resolves from the selected modlist directory
      Given the user selects "instances/server/server.json" as their modlist
      And that modlist selects "mods" as its mods directory
      And Modrinth provides a resolved Sodium artifact with verifiable content
      When the user installs their declared Sodium mod
      Then they should find the Sodium artifact installed in "instances/server/mods"
      And they should find no Sodium artifact written relative to the shell working directory

    Scenario: A trusted absolute mods directory defines the installation boundary
      Given the user's modlist selects a trusted absolute mods directory
      And Modrinth provides a resolved Sodium artifact with verifiable content
      When the user installs their declared Sodium mod
      Then they should find the Sodium artifact installed in the selected absolute mods directory
      And they should find no Sodium artifact written beside the modlist

    Scenario: Unsafe artifact metadata cannot escape the mods directory
      Given the user has a declared Sodium mod
      And Modrinth supplies artifact metadata whose filename would escape the mods directory
      And a file exists at the resulting path outside the installation boundary
      When the user installs their declared mods
      Then they should be told that the Sodium artifact path is unsafe
      And they should find the outside file unchanged
      And they should find no Sodium lock entry created from the unsafe metadata
      And they should be told that the requested operation did not complete successfully

  # Source: ../../docs/intent.md#paths-and-installation-boundaries
  # Source: ../../docs/commands/README.md#file-ownership-and-exclusions
  Rule: Destination collisions preserve protected files while independent work continues

    Scenario: An unrelated destination collision fails only the affected install
      Given the user has declared Sodium and Iris mod configs without lock entries
      And Modrinth provides eligible Sodium and Iris artifacts with verifiable content
      And an unrelated file already occupies the Sodium artifact's destination
      When the user installs their declared mods
      Then they should find the unrelated destination file unchanged
      And they should be told that Sodium could not be installed because of the collision
      And they should find no lock evidence claiming that the unrelated file is Sodium
      And they should find the exact Iris artifact installed and recorded in the lockfile
      And they should be told that the requested operation did not complete successfully

    Scenario: A forced version change does not overwrite an excluded destination collision
      Given the user has a declared Sodium mod with a replacement for another Minecraft target
      And the replacement destination is excluded by .mmmignore
      And an unrelated file already occupies that destination
      When the user changes their installation to that Minecraft target with --force
      Then they should find the excluded destination file unchanged
      And they should be told that force cannot override exclusion or collision protection
      And they should be told that the requested operation did not complete successfully

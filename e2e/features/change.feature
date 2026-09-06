Feature: Change an installation's Minecraft target
  The user can prepare and switch the managed installation while preserving recoverable state.

  Background:
    Given the user has a Fabric installation targeting Minecraft 1.20.1 with a release-only policy
    And the user has a valid Minecraft version manifest containing 1.20.1, 1.20.2 and 24w14a

  # Source: ../../docs/commands/change.md#ordinary-change
  Rule: Ordinary change prepares every replacement before switching
    Scenario: The user changes to a compatible Minecraft target
      Given the user declares unpinned Modrinth projects named Sodium and Lithium
      And the lockfile records exact installed Sodium and Lithium artifacts for Minecraft 1.20.1
      And Modrinth provides eligible Sodium and Lithium replacements for Fabric and Minecraft 1.20.2
      When the user changes the installation to Minecraft 1.20.2
      Then they should find both target replacements installed with their supplied content
      And they should find Minecraft 1.20.2 and the exact replacements recorded in the modlist and lockfile
      And they should find the previous managed artifacts removed

    Scenario: A failed staged replacement preserves the original installation
      Given the user declares unpinned Modrinth projects named Sodium and Lithium
      And the lockfile records exact installed Sodium and Lithium artifacts for Minecraft 1.20.1
      And Modrinth provides eligible Sodium and Lithium replacements for Fabric and Minecraft 1.20.2
      And downloading the Lithium replacement fails after automatic retries
      When the user changes the installation to Minecraft 1.20.2
      Then they should be told that Lithium could not be prepared
      And they should find the Minecraft 1.20.1 modlist, lockfile and installed artifacts unchanged
      And they should receive guidance for retrying the change
      And they should be told that the requested operation did not complete successfully

    Scenario: Every compatibility blocker is reported before ordinary change stops
      Given the user declares Modrinth projects named Atlas and Beacon and a CurseForge project named Compass
      And Modrinth conclusively provides no eligible artifact for Atlas or Beacon on Fabric and Minecraft 1.20.2
      And CurseForge cannot determine Compass eligibility after automatic retries
      When the user changes the installation to Minecraft 1.20.2 without forcing incompatibilities
      Then they should see Atlas and Beacon reported as incompatible with Minecraft 1.20.2
      And they should see Compass reported as inconclusive with the known service failure
      And they should see all three blockers in the final report
      And they should find the Minecraft 1.20.1 modlist, lockfile and installed artifacts unchanged
      And they should be told that the compatibility check is incomplete and the version change did not proceed

    Scenario: A concrete release-policy blocker preserves the original target
      Given the user declares an unpinned Modrinth project named Sodium with no per-mod release override
      And the lockfile records an exact installed Sodium release artifact for Minecraft 1.20.1
      And Modrinth provides a Sodium beta artifact for Fabric and Minecraft 1.20.2 but no release artifact
      When the user changes the installation to Minecraft 1.20.2 without forcing incompatibilities
      Then they should see Sodium reported as incompatible because the release-only policy excludes the beta artifact
      And they should find the Minecraft 1.20.1 target, lockfile and installed Sodium artifact unchanged
      And they should receive a completed compatibility report with incompatibilities

  # Source: ../../docs/commands/change.md#ordinary-change
  Rule: Same-target change is a successful no-op
    Scenario: The user requests the configured target while a managed file needs repair
      Given the user declares a Modrinth project named Sodium
      And the lockfile records an exact Sodium artifact for Minecraft 1.20.1
      And the installed Sodium file does not match that locked artifact
      When the user changes the installation to Minecraft 1.20.1
      Then they should be told that the target is already Minecraft 1.20.1
      And they should be directed to install to repair the managed file
      And they should find their modlist, lockfile and mod files unchanged
      And they should receive a successful operation result

  # Source: ../../docs/commands/change.md#forced-retention
  Rule: Forced change retains exact working artifacts when replacements are unavailable
    Scenario: Available replacements install while an unavailable replacement is retained
      Given the user declares unpinned Modrinth projects named Sodium and Lithium
      And the lockfile records exact installed Sodium and Lithium artifacts for Minecraft 1.20.1
      And Modrinth provides an eligible Sodium replacement for Fabric and Minecraft 1.20.2
      And Modrinth conclusively provides no eligible Lithium artifact for Fabric and Minecraft 1.20.2
      When the user authorizes changing the installation to Minecraft 1.20.2 despite incompatibilities
      Then they should find the target Sodium artifact installed and recorded
      And they should find the exact previous Lithium artifact still installed and recorded for Minecraft 1.20.2
      And they should find both mod configs retained in the modlist
      And they should be told that Lithium was retained without platform-declared compatibility
      And they should not find Lithium disabled or removed
      And they should receive a successful operation result

    Scenario: Force cannot satisfy a mod without a replacement or existing file
      Given the user declares an unpinned Modrinth project named Lithium
      And the lockfile records an exact Lithium artifact for Minecraft 1.20.1
      And the installed Lithium file is missing
      And Modrinth conclusively provides no eligible Lithium artifact for Fabric and Minecraft 1.20.2
      When the user authorizes changing the installation to Minecraft 1.20.2 despite incompatibilities
      Then they should be told that Lithium cannot be prepared or retained
      And they should find the Minecraft 1.20.1 modlist and lockfile unchanged
      And they should receive guidance for restoring the file or choosing another target
      And they should be told that the requested operation did not complete successfully

    Scenario: Force does not turn a failed lookup into known incompatibility or retention authority
      Given the user declares an unpinned CurseForge project named JourneyMap
      And the lockfile records an exact installed JourneyMap artifact for Minecraft 1.20.1
      And CurseForge cannot determine JourneyMap eligibility for Minecraft 1.20.2 after automatic retries
      When the user authorizes changing the installation to Minecraft 1.20.2 despite incompatibilities
      Then they should see JourneyMap reported as inconclusive with the known service failure
      And they should not be told that JourneyMap is incompatible or authorized for retention
      And they should find the Minecraft 1.20.1 modlist, lockfile and installed JourneyMap artifact unchanged
      And they should be told that the requested operation did not complete successfully

    Scenario: Force does not turn a failed replacement download into successful retention
      Given the user declares an unpinned Modrinth project named Sodium
      And the lockfile records an exact installed Sodium artifact for Minecraft 1.20.1
      And Modrinth provides an eligible Sodium replacement for Fabric and Minecraft 1.20.2
      And downloading the Sodium replacement fails after automatic retries
      When the user authorizes changing the installation to Minecraft 1.20.2 despite incompatibilities
      Then they should be told that the Sodium replacement download failed
      And they should not be told that the previous Sodium artifact was retained for Minecraft 1.20.2
      And they should find the Minecraft 1.20.1 modlist, lockfile and installed Sodium artifact unchanged
      And they should be told that the requested operation did not complete successfully

    Scenario: A retained artifact can be reproduced without force
      Given the user declares an unpinned Modrinth project named Lithium
      And the user's modlist now targets Minecraft 1.20.2
      And the user previously authorized retaining an exact Lithium artifact for Minecraft 1.20.2
      And that exact artifact is recorded in the lockfile for Minecraft 1.20.2
      And the installed Lithium file is missing
      And Modrinth can still provide that exact retained artifact
      When the user installs their declared mods
      Then they should find the exact retained Lithium artifact reproduced
      And they should find its retention authorization unchanged in the lockfile
      And they should not be required to force the installation
      And they should receive a successful operation result

    Scenario: Changed effective constraints require reassessment of a retained artifact
      Given the user declares an unpinned Modrinth project named Lithium
      And the user's modlist now targets Minecraft 1.20.2
      And the user previously authorized retaining an exact Lithium artifact for Fabric and Minecraft 1.20.2
      And that exact artifact is installed and recorded in the lockfile
      And the user changes the modlist loader to Quilt
      And Modrinth conclusively provides no eligible Lithium artifact for Quilt and Minecraft 1.20.2
      When the user installs their declared mods without authorizing a new compatibility exception
      Then they should be told that the previous retention does not authorize Lithium under the changed loader
      And they should not find an arbitrary replacement recorded or installed
      And they should find the previous Lithium file and recovery evidence preserved
      And they should be told that the requested operation did not complete successfully

    Scenario: Changing the effective release policy requires retained-artifact reassessment
      Given the user declares an unpinned Modrinth project named Lithium with no per-mod release override
      And the user's modlist targets Minecraft 1.20.2 and allows release and beta artifacts
      And the user previously authorized retaining an exact Lithium beta artifact for Fabric and Minecraft 1.20.2
      And that exact artifact is installed and recorded in the lockfile
      And the user changes the modlist default release policy to release-only
      And Modrinth provides no Lithium release artifact for Fabric and Minecraft 1.20.2
      When the user installs their declared mods without authorizing a new compatibility exception
      Then they should be told that the previous retention does not authorize Lithium under the changed effective release policy
      And they should not find an arbitrary replacement recorded or installed
      And they should find the previous Lithium file and recovery evidence preserved
      And they should be told that the requested operation did not complete successfully

    Scenario: A masked default-policy change preserves retained-artifact authorization
      Given the user declares an unpinned Modrinth project named Lithium with a per-mod override allowing beta artifacts
      And the user's modlist targets Minecraft 1.20.2 with a release-only default policy
      And the user previously authorized retaining an exact Lithium beta artifact for Fabric and Minecraft 1.20.2
      And that exact artifact is installed and recorded in the lockfile
      And the user changes the modlist default release policy to alpha-only
      When the user installs their declared mods
      Then they should find the exact retained Lithium artifact unchanged in the lockfile and mods directory
      And they should not be told that the default-policy change invalidated the retained artifact
      And they should not be required to authorize the exact artifact again
      And they should receive a successful operation result

    Scenario: Update replaces a retained artifact when a newer eligible artifact becomes available
      Given the user declares an unpinned Modrinth project named Lithium
      And the user's modlist now targets Minecraft 1.20.2
      And the user previously authorized retaining an exact Lithium artifact for Fabric and Minecraft 1.20.2
      And that exact artifact is installed and recorded in the lockfile
      And Modrinth provides a later eligible Lithium artifact with different content for Fabric and Minecraft 1.20.2
      When the user updates their declared mods
      Then they should find the later Lithium artifact installed with its supplied content
      And they should find that exact later artifact recorded in the lockfile
      And they should find the previously retained Lithium artifact removed

    Scenario: Update preserves a retained artifact when no newer eligible artifact exists
      Given the user declares an unpinned Modrinth project named Lithium
      And the user's modlist now targets Minecraft 1.20.2
      And the user previously authorized retaining an exact Lithium artifact for Fabric and Minecraft 1.20.2
      And that exact artifact is installed and recorded in the lockfile
      And Modrinth provides no newer eligible Lithium artifact for Fabric and Minecraft 1.20.2
      When the user updates their declared mods
      Then they should find the exact retained Lithium artifact unchanged in the lockfile and mods directory
      And they should be told that Lithium remains retained without platform-declared compatibility
      And they should receive a successful operation result

  # Source: ../../docs/commands/change.md#failure-cancellation-and-retry
  Rule: Switching failure reports the actual recovery outcome
    Scenario: Cancellation during preparation leaves the original installation unchanged
      Given the user declares unpinned Modrinth projects named Sodium and Lithium
      And the lockfile records exact installed Sodium and Lithium artifacts for Minecraft 1.20.1
      And Modrinth provides eligible Sodium and Lithium replacements for Fabric and Minecraft 1.20.2
      And replacement preparation remains in progress
      When the user cancels the version change before switching begins
      Then they should be told that the version change was cancelled during preparation
      And they should find the Minecraft 1.20.1 modlist, lockfile and installed artifacts unchanged
      And they should find no target replacement installed
      And they should be told that the operation was cancelled

    Scenario: Safe cancellation during switching recovers the original installation
      Given the user declares an unpinned Modrinth project named Sodium
      And the lockfile records an exact installed Sodium artifact for Minecraft 1.20.1
      And Modrinth provides an eligible Sodium replacement for Fabric and Minecraft 1.20.2
      And switching the prepared Sodium replacement fails after the original file is moved aside
      And recovery can restore the original metadata and artifact
      When the user requests safe cancellation while version-change recovery is in progress
      Then they should be told that cancellation is waiting for recovery to complete
      And they should find the Minecraft 1.20.1 modlist, lockfile and installed Sodium artifact restored
      And they should be told that recovery completed
      And they should be told that the operation was cancelled

    Scenario: Safe cancellation reports incomplete switching recovery
      Given the user declares an unpinned Modrinth project named Sodium
      And the lockfile records an exact installed Sodium artifact for Minecraft 1.20.1
      And Modrinth provides an eligible Sodium replacement for Fabric and Minecraft 1.20.2
      And switching the prepared Sodium replacement fails after the original file is moved aside
      And recovery cannot restore the original Sodium file
      When the user requests safe cancellation while version-change recovery is in progress
      Then they should be told that cancellation recovery was incomplete
      And they should see the actual modlist, lockfile and file state reported
      And they should not be told that recovery succeeded
      And they should receive a concrete recovery action
      And they should be told that the operation was cancelled

    Scenario: A switching failure recovers the original installation
      Given the user declares an unpinned Modrinth project named Sodium
      And the lockfile records an exact installed Sodium artifact for Minecraft 1.20.1
      And Modrinth provides an eligible Sodium replacement for Fabric and Minecraft 1.20.2
      And switching the prepared Sodium replacement fails after the original file is moved aside
      And recovery can restore the original metadata and artifact
      When the user changes the installation to Minecraft 1.20.2
      Then they should be told that switching failed and recovery completed
      And they should find the Minecraft 1.20.1 modlist, lockfile and installed Sodium artifact restored
      And they should receive guidance for retrying the change
      And they should be told that the requested operation did not complete successfully

    Scenario: Incomplete switching recovery reports the remaining state
      Given the user declares an unpinned Modrinth project named Sodium
      And the lockfile records an exact installed Sodium artifact for Minecraft 1.20.1
      And Modrinth provides an eligible Sodium replacement for Fabric and Minecraft 1.20.2
      And switching the prepared Sodium replacement fails after the original file is moved aside
      And recovery cannot restore the original Sodium file
      When the user changes the installation to Minecraft 1.20.2
      Then they should be told that switching and recovery were incomplete
      And they should see the actual modlist, lockfile and file state reported
      And they should not be told that nothing changed or recovery succeeded
      And they should receive a concrete recovery action
      And they should be told that the requested operation did not complete successfully

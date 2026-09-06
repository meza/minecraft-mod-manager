Feature: Discover and adopt unmanaged artifacts
  The user asks MMM to recognize visible unmanaged jars and grants ownership only through explicit adoption intent.

  Background:
    Given the user has an initialized installation

  # Source: ../../docs/intent.md#scan-and-adopt
  # Source: ../../docs/commands/scan.md#discovery-and-platform-fallback
  Rule: Discovery reports conclusive and inconclusive recognition distinctly

    Scenario: Scan prefers Modrinth by default
      Given the user does not supply a platform preference
      And Modrinth recognizes a visible unmanaged jar with exact artifact evidence
      When the user asks MMM to recognize the jar without adopting it
      Then they should see the jar reported as known from Modrinth
      And they should find a durable record that Modrinth was the preferred platform
      And they should find the jar, modlist and lockfile unchanged

    Scenario: Scan honors an explicit CurseForge preference
      Given the user supplies CurseForge as their platform preference
      And CurseForge recognizes a visible unmanaged jar with exact artifact evidence
      When the user asks MMM to recognize the jar without adopting it
      Then they should see the jar reported as known from CurseForge
      And they should find a durable record that CurseForge was the preferred platform
      And they should find the jar, modlist and lockfile unchanged

    Scenario: Discovery reports known, unknown and uncertain candidates without adopting them
      Given the user has three visible unmanaged jars in their mods directory
      And Modrinth recognizes the first jar with exact artifact evidence
      And Modrinth and CurseForge conclusively do not recognize the second jar
      And Modrinth and CurseForge cannot complete recognition of the third jar after automatic retries
      When the user asks MMM to recognize the jars without adopting any result
      Then they should see the first jar reported as known
      And they should see the second jar reported as unknown
      And they should see the third jar reported as uncertain
      And they should find all three jar files unchanged
      And they should find their modlist and lockfile unchanged

    Scenario: Scan falls back after the preferred platform fails
      Given the user prefers Modrinth for recognition
      And Modrinth cannot complete recognition of a visible unmanaged jar after automatic retries
      And CurseForge recognizes the jar with exact artifact evidence
      When the user asks MMM to recognize the jar without adopting it
      Then they should see the jar reported as known from CurseForge
      And they should not be told that the jar is unknown
      And they should find the jar, modlist and lockfile unchanged

    Scenario: Scan falls back after the preferred platform reports no match
      Given the user prefers Modrinth for recognition
      And Modrinth conclusively does not recognize a visible unmanaged jar
      And CurseForge recognizes the jar with exact artifact evidence
      When the user asks MMM to recognize the jar without adopting it
      Then they should see the jar reported as known from CurseForge
      And they should not be told that the jar is unknown
      And they should find the jar, modlist and lockfile unchanged

    Scenario: A failed preferred lookup and alternate no-match remain uncertain
      Given the user prefers Modrinth for recognition
      And Modrinth cannot complete recognition of a visible unmanaged jar after automatic retries
      And CurseForge conclusively does not recognize the jar
      When the user scans with --add
      Then they should be told that recognition remains uncertain
      And they should not be told that the jar is unknown
      And they should find the jar unchanged and unmanaged
      And they should find no mod config or lock entry created from the jar
      And they should receive guidance for retrying
      And they should be told that the requested operation did not complete successfully

    Scenario: Exhausted recognition failures remain uncertain
      Given neither Modrinth nor CurseForge can complete recognition of a visible unmanaged jar after automatic retries
      When the user asks MMM to recognize the jar
      Then they should be told that recognition remains uncertain
      And they should receive guidance for retrying
      And they should not be told that the jar is unknown
      And they should find the jar, modlist and lockfile unchanged
      And they should be told that the requested operation did not complete successfully

  # Source: ../../docs/intent.md#scan-and-adopt
  # Source: ../../docs/commands/scan.md#adoption
  Rule: Explicit adoption records the exact discovered artifact without replacing its bytes

    Scenario: The user explicitly adopts an unambiguous recognized artifact
      Given the user has a visible unmanaged Sodium jar with known content
      And Modrinth recognizes it as an exact Sodium artifact with a filename, hash and download source
      When the user scans with --add
      Then they should find a Modrinth Sodium mod config in their modlist
      And they should find that exact filename, hash and Modrinth source recorded in their lockfile
      And they should find the original Sodium content unchanged on disk
      And they should find no newer Sodium artifact downloaded

    Scenario: Explicit adoption skips unknown and uncertain candidates
      Given the user has three visible unmanaged jars in their mods directory
      And Modrinth recognizes the first jar unambiguously
      And Modrinth and CurseForge conclusively do not recognize the second jar
      And Modrinth and CurseForge cannot complete recognition of the third jar after automatic retries
      When the user scans with --add
      Then they should find the recognized project's mod config in their modlist
      And they should find the exact recognized artifact in their lockfile
      And they should find no mod config or lock entry created for the unknown jar
      And they should find no mod config or lock entry created for the uncertain jar
      And they should find all three jar files unchanged
      And they should be told that the requested operation did not complete successfully

    Scenario: Adoption does not duplicate an already managed project
      Given the user has a Modrinth Sodium mod config and valid locked artifact
      And the user has another visible jar recognized as the same Modrinth project
      When the user scans with --add
      Then they should find exactly one Modrinth Sodium mod config in their modlist
      And they should find exactly one valid Sodium lock entry
      And they should find the existing locked artifact unchanged
      And they should find the other jar unmanaged

    Scenario: Add intent does not authorize changing an existing pin
      Given the user has a Modrinth Sodium mod config pinned to version number "1.0.0"
      And the exact locked artifact A with version number "1.0.0" is installed
      And the user has a visible unmanaged jar recognized as artifact B with version number "2.0.0" of the same project
      And artifact B conflicts with the existing pin
      When the user scans with --add and authorizes adoption without authorizing a pin change
      Then they should be told that artifact B conflicts with the existing pin
      And they should find the version pin "1.0.0" unchanged in their modlist
      And they should find the exact lock entry and installed content for artifact A unchanged
      And they should find artifact B unchanged and unmanaged
      And they should find no lock evidence adopting artifact B

  # Source: ../../docs/intent.md#scan-and-adopt
  # Source: ../../docs/commands/scan.md#adoption
  Rule: Existing valid resolution evidence makes duplicate discovery deterministic

    Scenario: A valid locked artifact wins when several discovered jars identify one project
      Given the user has a Modrinth Sodium mod config with a valid locked artifact installed as a managed file
      And the user has several visible jars recognized as artifacts of that Sodium project
      When the user scans with --add
      Then they should find the existing Sodium lock entry unchanged
      And they should find the managed locked artifact unchanged
      And they should find exactly one Sodium mod config
      And they should find every discovered Sodium jar still unmanaged
      And they should find a durable record that the locked artifact was kept

  # Source: ../../docs/intent.md#scan-and-adopt
  # Source: ../../docs/commands/scan.md#adoption
  Rule: Explicit first-time adoption can preserve a documented compatibility exception

    Scenario: An explicitly adopted incompatible artifact remains reproducible for its target
      Given the user has a visible unmanaged Sodium jar with verifiable content
      And the user has no existing Sodium mod config
      And Modrinth recognizes and can provide that exact artifact but does not report it compatible with the user's Minecraft target
      When the user scans with --add
      Then they should find a Modrinth Sodium mod config in their modlist
      And they should find that exact artifact recorded in their lockfile for the current target
      And they should be told about the platform-reported compatibility limitation
      When the user deletes the adopted Sodium file
      Then they should find the adopted Sodium file absent
      And they should find its exact lock entry unchanged
      When the user later installs their declared mods
      Then they should find the same Sodium artifact content installed without additional force
      And they should find the compatibility exception limited to that artifact and target
      And they should receive a successful operation result

    Scenario: An adopted compatibility exception does not transfer to another target
      Given the user targets Minecraft version X
      And Mojang lists Minecraft versions X and Y in the Minecraft version manifest
      And the user has no existing Sodium mod config
      And the user has a visible unmanaged Sodium artifact A with verifiable content
      And Modrinth recognizes artifact A but does not report it compatible with Minecraft version X
      And Modrinth conclusively provides no eligible Sodium artifact for Minecraft version Y
      When the user scans with --add
      Then they should find artifact A recorded as an adopted compatibility exception for Minecraft version X
      When the user changes their declared Minecraft target to version Y without authorizing another compatibility exception
      Then they should find Minecraft version Y declared in their modlist
      When the user installs their declared mods
      Then they should be told that Sodium is not satisfied for Minecraft version Y
      And they should not be told that the exception for Minecraft version X satisfies version Y
      And they should find artifact A and its exact recovery evidence preserved
      And they should receive guidance for resolving or retrying Sodium
      And they should be told that the requested operation did not complete successfully

  # Source: ../../docs/intent.md#excluded-files
  # Source: ../../docs/commands/scan.md#discovery-and-platform-fallback
  Rule: Scan ignores protected and nested files

    Scenario: Scan examines only visible immediate jars
      Given the user has a visible unmanaged jar in their mods directory
      And the user has an unmanaged jar matched by .mmmignore
      And the user has an unmanaged file ending in .disabled
      And the user has an unmanaged jar in a subdirectory of their mods directory
      When the user scans with --add
      Then they should find only the visible immediate jar considered for recognition
      And they should find the ignored, disabled and nested files unreported and unchanged
      And they should find no mod config or lock entry created for those excluded files

    Scenario: An invalid ignore pattern stops adoption before lookup or mutation
      Given the user's .mmmignore contains an invalid pattern on line 2
      And the user has a visible unmanaged jar in their mods directory
      When the user scans with --add
      Then they should be told that the .mmmignore pattern on line 2 is invalid
      And they should find the jar, modlist and lockfile unchanged
      And they should be told that the requested operation did not complete successfully

  # Source: ../../docs/intent.md#failure-retry-and-cancellation-promises
  # Source: ../../docs/commands/scan.md#results-failure-and-retry
  Rule: Independent adoption results remain consistent across failure and retry

    Scenario: One metadata failure does not undo an independent adoption
      Given the user has two visible unmanaged jars recognized as different projects
      And metadata for the second project cannot be saved
      When the user scans with --add
      Then they should find the first project's mod config and exact lock entry saved consistently
      And they should be told that adoption of the second project failed
      And they should find the second jar unchanged and unmanaged
      And they should receive guidance for retrying
      And they should be told that the requested operation did not complete successfully

    Scenario: Retrying adoption does not duplicate completed metadata
      Given the user previously adopted the first of two recognized projects
      And the second project's adoption remains incomplete
      And metadata can now be saved
      When the user scans with --add again
      Then they should find one mod config and one exact lock entry for each project
      And they should find no duplicate metadata for the first project

    Scenario: Safe cancellation preserves a completed adoption and retry finishes remaining work
      Given the user has visible unmanaged Sodium, Iris and Lithium jars
      And Modrinth can recognize each jar as a different project
      And Iris recognition can remain in progress after Sodium adoption settles
      When the user scans with --add
      Then they should observe the Sodium adoption settled while Iris recognition remains in progress and Lithium recognition has not begun
      When the user safely cancels scan
      Then they should find the Sodium mod config and exact lock entry saved consistently
      And they should find no new recognition or adoption work started after cancellation
      And they should find the Iris and Lithium jars unchanged and unmanaged
      And they should find no Iris or Lithium mod config or lock entry
      And they should receive a report of completed and unfinished scan work
      And they should be told that the operation was cancelled
      Given Modrinth can now complete recognition of every unfinished project
      When the user scans with --add again
      Then they should find one mod config and one exact lock entry for each project
      And they should find no duplicate metadata for Sodium

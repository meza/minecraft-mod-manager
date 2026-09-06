Feature: Failed and interrupted work leaves understandable recovery outcomes
  The user can retain completed work and retry unfinished operations safely.

  Background:
    Given the user has a Fabric installation targeting Minecraft 1.20.2 with a release-only policy

  # Source: ../../docs/intent.md#failure-retry-and-cancellation-promises
  Rule: Replacement requires verified complete artifact content
    Scenario Outline: The user keeps their working artifact after a failed replacement transfer
      Given the user has Atlas artifact A installed and recorded in their lockfile
      And the user declares Atlas as an unpinned Modrinth project
      And Modrinth identifies a newer eligible Atlas artifact B with different content
      And Modrinth <transfer failure> when serving artifact B on every download attempt
      When the user updates their installation
      Then they should find the original Atlas artifact A content still installed
      And they should not find artifact B installed as a loadable jar
      And they should find resolution evidence identifying the retained artifact A
      And they should see that the Atlas update failed because of <reported reason>
      And they should receive guidance for retrying the update
      And they should be told that the requested operation did not complete successfully

      Examples:
        | transfer failure                                  | reported reason            |
        | closes the connection before the file is complete  | an incomplete download     |
        | serves complete bytes that do not match B's hash   | a failed integrity check   |

  # Source: ../../docs/intent.md#failure-retry-and-cancellation-promises
  Rule: Required metadata writes are part of the operation outcome
    Scenario: The user is told when installation cannot persist resolution evidence
      Given the user declares Atlas as an unpinned Modrinth project with no lock entry or installed file
      And Modrinth provides an eligible Atlas artifact A with verifiable content
      And the user has a lockfile that cannot be written
      When the user installs their declared mods
      Then they should see a failure identifying the lockfile write problem
      And they should not be told that the Atlas installation completed successfully
      And they should receive guidance describing the remaining installation state and required recovery
      And they should be told that the requested operation did not complete successfully

    Scenario: The user can retry after restoring metadata write access
      Given the user declares Atlas as an unpinned Modrinth project with no lock entry or installed file
      And Modrinth provides an eligible Atlas artifact A with verifiable content
      And the user has a lockfile that cannot be written
      When the user installs their declared mods
      Then they should see a failure identifying the lockfile write problem
      And they should be told that the requested operation did not complete successfully
      When the user restores write access to their lockfile
      Then they should find their lockfile writable
      When the user retries installing their declared mods
      Then they should find exactly one Atlas mod config in their modlist
      And they should find exactly one Atlas lock entry identifying artifact A
      And they should find artifact A's expected content installed without an extra loadable Atlas jar

  # Source: ../../docs/intent.md#failure-retry-and-cancellation-promises
  Rule: Harmless residue differs from incomplete installation consistency
    Scenario: The user does not need to clean up a backup when automatic cleanup recovers
      Given the user has Atlas artifact A installed and recorded in their lockfile
      And the user declares Atlas as an unpinned Modrinth project
      And Modrinth provides a newer eligible Atlas artifact B with different verifiable content
      And the user has a temporary filesystem restriction that prevents the first backup cleanup attempt but permits its retry
      When the user updates their installation
      Then they should find artifact B's expected content installed and recorded in their lockfile
      And they should find no leftover Atlas backup
      And they should not receive a request to clean up that backup
      And they should receive a successful operation result

    Scenario: The user receives a warning about an unremovable non-loadable backup
      Given the user has Atlas artifact A installed and recorded in their lockfile
      And the user declares Atlas as an unpinned Modrinth project
      And Modrinth provides a newer eligible Atlas artifact B with different verifiable content
      And the user has a filesystem restriction that prevents removal of A's non-loadable backup through all automatic cleanup retries
      When the user updates their installation
      Then they should find artifact B's expected content installed and recorded in their lockfile
      And they should find no extra loadable Atlas jar
      And they should receive a warning identifying the remaining backup and how to remove it
      And they should receive a successful operation result

    Scenario: The user is not told an extra loadable jar is harmless residue
      Given the user has Atlas artifact A installed and recorded in their lockfile
      And the user declares Atlas as an unpinned Modrinth project
      And Modrinth provides a newer eligible Atlas artifact B with different verifiable content
      And the user encounters a filesystem failure during replacement that leaves both A and B as loadable jars and prevents recovery
      When the user updates their installation
      Then they should be told that the installation remains inconsistent
      And they should see the actual remaining Atlas files and the action needed for recovery
      And they should not receive an overall success result
      And they should be told that the requested operation did not complete successfully

  # Source: ../../docs/intent.md#failure-retry-and-cancellation-promises
  Rule: Safe cancellation retains completed independent work
    Scenario: The user cancels installation while another artifact is downloading
      Given the user declares unpinned Modrinth projects Atlas and Beacon with no installed artifacts or lock entries
      And Modrinth provides eligible artifacts A and B with verifiable content for those projects respectively
      And Modrinth can keep the Beacon download unfinished while independent work progresses
      When the user starts installing their declared mods
      Then they should find artifact A installed and recorded for Atlas while Beacon is still downloading
      When the user requests safe cancellation
      Then they should find the completed Atlas artifact and lock entry preserved
      And they should find both requested mod configs preserved
      And they should not find an incomplete Beacon download installed as a jar
      And Modrinth should observe cancellation of the unfinished Beacon download without new downloads starting after cancellation
      And the user should see Atlas as completed and Beacon as unfinished
      And they should be told that the operation was cancelled after necessary consistency work completes
      When the user retries installing their declared mods with all downloads able to complete
      Then they should find the expected content of A and B installed
      And they should find exactly one mod config and one exact artifact lock entry for each project

    Scenario: The user remains informed while safe cancellation waits for required consistency work
      Given the user declares Atlas from Modrinth with a requested pin for artifact B
      And the user has an Atlas replacement in progress with complete artifact B bytes installed and the previous artifact A still recorded in their lockfile
      And the user has a temporary filesystem restriction preventing both the lockfile write for B and restoration of artifact A
      When the user requests safe cancellation
      Then they should be told that cleanup is still in progress
      And they should be warned that a further interruption forces termination and may leave recovery unfinished
      And they should observe MMM waiting for required consistency work without an automatic shutdown deadline
      When the user restores access needed for the consistency work
      Then they should find exactly one loadable Atlas jar whose bytes match the recorded Atlas artifact before MMM returns control
      And they should find one of these consistent Atlas outcomes:
        | installed artifact | locked artifact | reported outcome       |
        | B                  | B               | replacement completed  |
        | A                  | A               | previous state restored |
      And they should find their requested pin for artifact B preserved
      And they should be told that the operation was cancelled

    Scenario: The user explicitly ends recovery through emergency termination
      Given the user declares Atlas from Modrinth with a requested pin for artifact B
      And the user has an Atlas replacement in progress with complete artifact B bytes installed and the previous artifact A still recorded in their lockfile
      And the user has a filesystem restriction preventing both the lockfile write for B and restoration of artifact A
      When the user requests safe cancellation
      Then they should be warned that a further interruption forces termination and may leave recovery unfinished
      When the user requests emergency termination while recovery is unfinished
      Then they should regain control without waiting for recovery to finish
      And they should not receive a claim that recovery completed successfully

  # Source: ../../docs/intent.md#results-and-exit-status
  Rule: Errors identify the failed work without duplicating handled failures
    Scenario: The user receives one actionable result for an exhausted download failure
      Given the user declares Atlas as an unpinned Modrinth project with no lock entry or installed file
      And Modrinth identifies an eligible Atlas artifact A but cannot serve it after automatic retries
      When the user installs their declared mods
      Then they should receive one settled failure record identifying Atlas and the failed download
      And they should receive a summary identifying the unfinished work and a useful retry action
      And they should not see the handled Atlas failure repeated as another independent error
      And they should not see unrelated command usage presented as the cause
      And they should be told that the requested operation did not complete successfully

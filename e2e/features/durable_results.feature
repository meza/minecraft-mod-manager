Feature: Durable results explain the operation after it finishes
  The user can rely on settled results and relevant decisions remaining available.

  Background:
    Given the user has a Fabric installation targeting Minecraft 1.20.2 with a release-only policy

  # Source: ../../docs/intent.md#active-display-and-permanent-transcript
  Rule: Results become permanent once in completion order
    Scenario: The user receives results as independent work settles
      Given the user declares unpinned Modrinth projects Atlas and Beacon with no installed artifacts or lock entries
      And Modrinth provides eligible artifacts A and B with verifiable content for Atlas and Beacon respectively
      And Modrinth keeps the Atlas download pending while the Beacon download can complete
      When the user starts installing their declared mods
      Then they should find Beacon artifact B installed and recorded in their lockfile while Atlas is still pending
      And they should find one settled Beacon installation result in the permanent transcript
      Given Modrinth allows the Atlas download to complete
      Then the user should find Atlas artifact A installed and recorded in their lockfile
      And they should find one settled Atlas installation result after the Beacon result in the permanent transcript
      And they should find the Beacon record unchanged
      And they should receive a summary of two completed installations without replaying either result
      And they should receive the same durable result and summary text and formatting as equivalent executions under the same locale and character capabilities

    Scenario: The user retains completed and failed results after partial success
      Given the user declares unpinned Modrinth projects Atlas and Beacon with no installed artifacts or lock entries
      And Modrinth provides an eligible Atlas artifact A with verifiable content
      And Modrinth cannot complete the Beacon lookup after automatic retries
      When the user installs their declared mods
      Then they should find one settled Atlas success record and one settled Beacon failure record in their permanent transcript
      And they should find each record in the order its outcome settled
      And they should receive a summary identifying one completed installation and one unfinished installation with a useful next action
      And they should not find the result list replayed by the summary
      And they should receive the same durable success, failure and summary text and formatting as equivalent executions under the same locale and character capabilities
      And they should be told that the requested operation did not complete successfully

    Scenario: The user retains relevant records after cancellation
      Given the user has pre-command shell output containing a recognizable message
      And the user declares unpinned Modrinth projects Atlas and Beacon with no installed artifacts or lock entries
      And Modrinth provides eligible artifacts A and B with verifiable content for Atlas and Beacon respectively
      And Modrinth keeps Beacon downloading while Atlas can complete
      When the user starts installing their declared mods
      Then they should find artifact A installed and one settled Atlas result in their permanent transcript while Beacon is pending
      When the user requests safe cancellation
      Then they should find the settled Atlas result preserved exactly once
      And they should see which requested work remains unfinished
      And they should find the cancellation outcome recorded without claiming that all requested work succeeded
      And they should find their pre-command message and the operation's durable records still available in normal terminal history
      And they should receive the same durable completion and cancellation text and formatting as equivalent executions under the same locale and character capabilities
      And they should be told that the operation was cancelled after required consistency work completes

  # Source: ../../docs/intent.md#active-display-and-permanent-transcript
  Rule: Relevant resolved decisions have durable records independent of input collection
    Scenario: The user retains the meaning of a per-mod release-policy decision
      Given Modrinth provides an Atlas beta artifact A for Fabric and Minecraft 1.20.2 with verifiable content
      And Modrinth provides no eligible release artifact for Atlas
      When the user adds Atlas from Modrinth with an explicit per-mod policy allowing releases and betas
      Then they should find a durable decision record identifying the Atlas release-policy override
      And they should receive the same durable decision text and formatting as equivalent executions under the same locale and character capabilities
      And they should find the Atlas artifact A content installed and recorded in their lockfile
      And they should find their installation-wide release-only policy unchanged

    Scenario: The user retains a fallback warning alongside the installed result
      Given the user declares Atlas from Modrinth with version fallback allowed
      And Modrinth provides no Atlas artifact for Minecraft 1.20.2
      And Modrinth provides an eligible Atlas release artifact A for Fabric and Minecraft 1.20.1 with verifiable content
      When the user installs their declared mods
      Then they should find a durable warning identifying Atlas and the fallback from Minecraft 1.20.2 to 1.20.1
      And they should find a settled Atlas installation result exactly once
      And they should find both records available after MMM exits
      And they should receive the same durable warning and result text and formatting as equivalent executions under the same locale and character capabilities
      And they should receive a successful operation result

  # Source: ../../docs/intent.md#active-display-and-permanent-transcript
  Rule: Completed invocations preserve existing history and accessible results
    Scenario Outline: The user returns to their existing history after an operation
      Given the user has pre-command shell output containing a recognizable message
      And the user declares Atlas as an unpinned Modrinth project with no installed artifact or lock entry
      And Modrinth <response>
      When the user installs their declared mods
      Then they should regain terminal control
      And they should find their pre-command message and the operation's durable records still available in normal terminal history
      And they should find each settled result exactly once
      And they should receive a report that <reported outcome>

      Examples:
        | response                                                   | reported outcome        |
        | provides an eligible Atlas artifact A with verifiable content | Atlas was installed     |
        | cannot complete the Atlas lookup after automatic retries     | Atlas installation failed |

    Scenario: The user can access every result from a list longer than their terminal display
      Given the user has more declared mods than can be displayed together in their terminal
      And the user has valid lock entries and matching installed artifact content for every declared mod
      When the user lists their declared mods
      Then they should be able to access every declared mod's reported installation state
      And they should find no mod omitted or duplicated to fit the display
      And they should find the full durable list available after MMM exits

  # Source: ../../docs/intent.md#controls-language-and-accessibility
  Rule: Product messages preserve meaning under localization
    Scenario: The user receives English when a selected translation is missing
      Given the user selects a supported locale with a missing translation for an Atlas lookup failure
      And the user declares Atlas as an unpinned Modrinth project
      And Modrinth cannot complete the Atlas lookup after automatic retries
      When the user installs their declared mods
      Then they should receive the Atlas lookup failure message in English
      And they should be told that the requested operation did not complete successfully

    Scenario: The user can distinguish success and failure from durable text
      Given the user declares unpinned Modrinth projects Atlas and Beacon with no installed artifacts or lock entries
      And Modrinth provides an eligible Atlas artifact A with verifiable content
      And Modrinth cannot complete the Beacon lookup after automatic retries
      When the user installs their declared mods
      Then they should be able to identify Atlas as completed and Beacon as failed from the durable text alone
      And they should be told that the requested operation did not complete successfully

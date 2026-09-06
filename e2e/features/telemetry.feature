Feature: Operational telemetry respects the user's collection boundary
  PostHog is the telemetry service receiving permitted operational metrics from MMM.

  Background:
    Given the user has a Fabric installation targeting Minecraft 1.20.2 with a release-only policy
    And the user has an empty modlist and an empty lockfile
    And PostHog can receive MMM telemetry

  # Source: ../../docs/intent.md#operational-expectations
  # Source: ../../README.md#telemetry
  Rule: Default telemetry and explicit opt-out have distinct observable effects
    Scenario: The user runs MMM with its default telemetry policy
      Given the user has not set MMM_DISABLE_TELEMETRY
      When the user lists their declared mods
      Then they should receive the requested list result
      And PostHog should receive operational telemetry for the invocation

    Scenario: The user disables telemetry explicitly
      Given the user sets MMM_DISABLE_TELEMETRY to 1
      When the user lists their declared mods
      Then they should receive the requested list result
      And PostHog should receive no telemetry from the invocation

    Scenario: The user uses the same machine across invocations
      Given the user has not disabled telemetry
      When the user lists their declared mods
      Then they should receive the requested list result
      And PostHog should receive a stable machine identifier
      When the user lists their declared mods again on the same machine
      Then they should receive the requested list result
      And PostHog should receive the same machine identifier for the second invocation

  # Source: ../../docs/intent.md#operational-expectations
  Rule: Telemetry emits only permitted operational information
    Scenario: The user performs an operation with telemetry enabled
      Given the user has not disabled telemetry
      When the user lists their declared mods
      Then they should receive the requested list result
      And PostHog should receive no collected information outside these permitted categories:
        | permitted category          |
        | MMM app version             |
        | operating system            |
        | execution mode              |
        | command outcomes            |
        | command timings             |
        | Minecraft version           |
        | loader                      |
        | mod count                   |
        | request and download counts |
        | bounded error categories    |
        | stable machine identifier   |

    Scenario: The user does not disclose personal values embedded in otherwise permitted fields
      Given the user has not disabled telemetry
      And the user has a synthetic username embedded in their configuration path and artifact download address
      And the user declares a Modrinth project with a synthetic private project ID and display name
      And the user supplies a synthetic API credential and a private modlist path as command input
      And Modrinth fails the artifact lookup with raw error text containing that username, project identity, address and credential
      When the user installs their declared mods with debug diagnostics and a performance recording requested
      Then they should be told that the requested operation did not complete successfully
      And PostHog should receive no personal information, credentials, mod identities, private filesystem paths, arbitrary command arguments or raw error text in any telemetry value
      And PostHog should receive no debug log or detailed local performance recording

  # Source: ../../docs/intent.md#operational-expectations
  Rule: Telemetry failure cannot fail the requested operation or retain terminal control
    Scenario Outline: The user completes an operation despite a telemetry service problem
      Given the user has not disabled telemetry
      And PostHog <service problem>
      When the user lists their declared mods
      Then they should receive the requested list result
      And they should regain control without waiting indefinitely for telemetry delivery

      Examples:
        | service problem                         |
        | rejects telemetry requests              |
        | cannot be reached                       |
        | accepts a connection but never responds |

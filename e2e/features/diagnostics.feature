Feature: Optional diagnostics help the user explain an operation
  Diagnostic records do not replace ordinary results or required installation metadata.

  Background:
    Given the user has a Fabric installation targeting Minecraft 1.20.2 with a release-only policy
    And the user has an empty modlist and an empty lockfile

  # Source: ../../docs/commands/README.md#debug-log-reference
  Rule: The user explicitly requests a local debug audit
    Scenario: The user performs an ordinary operation without creating a debug log
      When the user lists their declared mods without requesting debug diagnostics
      Then they should receive the requested list result
      And they should find no debug log created by that invocation

    Scenario: The user records debug events beside the selected modlist
      Given the user selects a modlist named server.json in an existing configuration directory separate from their working directory
      When the user lists their declared mods with debug diagnostics enabled
      Then they should find a debug log beside server.json
      And they should find one event per line in logfmt format
      And they should find these common fields on every event:
        | field | required content                                   |
        | ts    | an RFC3339Nano timestamp                            |
        | level | debug, info, warn or error                          |
        | event | a stable event name                                |
        | cmd   | the root command name and applicable subcommand    |
      And they should find no control characters in the logged keys or values
      And they should receive the requested list result

    Scenario: The user receives useful failure context without exposing a credential
      Given the user declares Atlas as an unpinned Modrinth project with no installed artifact
      And the user supplies a synthetic runtime Modrinth credential
      And Modrinth fails the Atlas lookup after automatic retries
      When the user installs their declared mods with debug diagnostics enabled
      Then they should receive an ordinary failure report identifying the affected Atlas lookup and a useful next action
      And they should find additional failure context in their debug log
      And they should not find their credential or authorization header values in user output or the debug log
      And they should be told that the requested operation did not complete successfully

  # Source: ../../docs/intent.md#operational-expectations
  # Source: ../../README.md#performance-logs
  Rule: Detailed performance recordings require an explicit request
    Scenario: The user performs an ordinary operation without a local performance recording
      When the user lists their declared mods without requesting a performance recording
      Then they should find no mmm-perf.json created by that invocation
      And they should receive the requested list result

    Scenario: The user records performance beside their selected modlist by default
      Given the user selects a modlist named server.json in a configuration directory separate from their working directory
      When the user lists their declared mods with a performance recording requested
      Then they should find mmm-perf.json in the selected configuration directory
      And they should find performance information for that invocation in the recording
      And they should find recorded filesystem paths normalized relative to the configuration directory
      And they should not find machine-specific absolute path prefixes in the recording
      And they should receive the requested list result

    Scenario: The user selects the performance recording directory
      Given the user has an existing diagnostics directory
      When the user lists their declared mods with a performance recording requested in that diagnostics directory
      Then they should find mmm-perf.json in the selected diagnostics directory
      And they should receive the requested list result

    Scenario Outline: The user retains a successful operation when an optional diagnostic export fails
      Given the user requests <diagnostic>
      And the user has a filesystem restriction preventing that diagnostic export
      When the user lists their declared mods
      Then they should receive the requested list result
      And they should find their modlist, lockfile and mod files unchanged

      Examples:
        | diagnostic                 |
        | debug diagnostics          |
        | a performance recording    |

  # Source: ../../docs/intent.md#operational-expectations
  # Source: ../../README.md#using-your-own-api-keys
  Rule: Ordinary distribution access and explicit overrides protect credential values
    Scenario Outline: The user installs from an official distribution without personal credentials
      Given the user runs an official MMM distribution without personal API credential overrides
      And <platform> accepts the distribution's access defaults and provides an eligible Atlas artifact A with verifiable content
      When the user adds Atlas from <platform> to their installation
      Then they should find artifact A's expected content installed and recorded in their lockfile
      And they should find an Atlas mod config associated with <platform>
      And they should not find credential values in user output

      Examples:
        | platform   |
        | Modrinth   |
        | CurseForge |

    Scenario Outline: The user selects a runtime credential for platform access
      Given the user supplies a synthetic <platform> credential through <setting source>
      And <platform> accepts only that supplied credential and provides an eligible Atlas artifact A with verifiable content
      When the user adds Atlas from <platform> with debug diagnostics and a performance recording requested
      Then they should find artifact A's expected content installed and recorded in their lockfile
      And they should not find credential values in user output, debug logs or the performance recording

      Examples:
        | platform   | setting source                                |
        | Modrinth   | the MODRINTH_API_KEY environment variable      |
        | CurseForge | the CURSEFORGE_API_KEY environment variable    |
        | Modrinth   | MODRINTH_API_KEY in their working-directory .env |
        | CurseForge | CURSEFORGE_API_KEY in their working-directory .env |

    Scenario Outline: An explicitly empty runtime override does not restore an embedded credential
      Given the user runs an official MMM distribution with a synthetic embedded credential for <platform>
      And the user supplies an explicitly empty value through <setting source> with no other runtime override for that platform
      And <platform> accepts the embedded credential for Atlas lookup but rejects access without it
      When the user adds Atlas from <platform> with debug diagnostics and a performance recording requested
      Then <platform> should receive no request using the embedded credential
      And the user should receive a non-success result explaining why platform access could not be established
      And they should find no new Atlas mod config, lock entry or installed artifact
      And they should not find credential values in user output, debug logs or the performance recording

      Examples:
        | platform   | setting source                                    |
        | Modrinth   | the MODRINTH_API_KEY environment variable            |
        | CurseForge | the CURSEFORGE_API_KEY environment variable          |
        | Modrinth   | MODRINTH_API_KEY in their working-directory .env     |
        | CurseForge | CURSEFORGE_API_KEY in their working-directory .env   |

  # Source: ../../docs/intent.md#operational-expectations
  # Source: ../../docs/commands/README.md#network-and-proxy-settings
  Rule: Standard proxy settings apply to HTTP requests and downloads
    Scenario: An HTTP-only proxy setting does not route HTTPS platform requests
      Given the user sets HTTP_PROXY to an unreachable proxy as their only proxy setting
      And Modrinth provides an eligible Atlas artifact A with verifiable content through HTTPS addresses reachable directly
      When the user adds Atlas from Modrinth to their installation
      Then they should observe the HTTPS platform lookup and artifact download reached directly without the HTTP proxy
      And they should find artifact A's expected content installed and recorded in their lockfile

    Scenario: The user reaches a platform and its download service through a configured HTTPS proxy
      Given the user configures HTTPS_PROXY for the HTTPS addresses used by Modrinth and its artifact downloads
      And Modrinth provides an eligible Atlas artifact A with verifiable content reachable only through that proxy
      When the user adds Atlas from Modrinth to their installation
      Then they should find artifact A's expected content installed and recorded in their lockfile
      And they should observe the platform lookup and artifact download routed through the configured proxy

    Scenario: The user bypasses their proxy for explicitly excluded hosts
      Given the user configures a proxy and NO_PROXY for the hosts used by Modrinth and its artifact downloads
      And Modrinth provides an eligible Atlas artifact A with verifiable content reachable directly
      And the configured proxy cannot reach those excluded hosts
      When the user adds Atlas from Modrinth to their installation
      Then they should find artifact A's expected content installed and recorded in their lockfile
      And they should observe the excluded hosts reached without the proxy

  # Source: ../../docs/intent.md#operational-expectations
  # Source: ../../internal/httpclient/README.md#doer-and-rate-limited-client
  Rule: Platform requests respect service limits during automatic retry
    Scenario: The user completes an addition after the platform permits a retry
      Given Modrinth temporarily rate-limits the Atlas lookup and specifies when another request is permitted
      And Modrinth provides an eligible Atlas artifact A with verifiable content on the permitted retry
      When the user adds Atlas from Modrinth to their installation
      Then Modrinth should receive no retry before the permitted time
      And the user should find artifact A's expected content installed and recorded in their lockfile

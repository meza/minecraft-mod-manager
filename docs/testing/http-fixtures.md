# HTTP fixtures for E2E tests

## Status and purpose

This is the agreed design for HTTP-dependent E2E scenarios. The fixture servers, endpoint overrides and verification described here are not implemented yet. The variable names below describe the planned interface; setting them does not currently redirect MMM's requests.

Contributors implementing this design should keep `make e2e` as the entry point. The existing command runs the current suite, whose product smoke scenario cancels initialization before making HTTP requests. See the [terminal E2E guide](terminal-harness.md) for the available harness and prerequisites.

Use scenario-owned Go HTTP servers with explicit response handlers. Recordings are not the E2E fixture model. The [existing VCR helper](http-vcr.md) is an unused in-process helper outside its own tests; it cannot control HTTP calls from a separately launched MMM process.

## Ownership and request flow

[Product intent](../intent.md#acceptance-and-evidence) requires scenarios to exercise the real MMM process with deterministic network fixtures. The selected design is:

```text
Godog scenario -> fixture server setup and endpoint environment
              -> terminal driver -> MMM process -> local HTTP fixture server
              <- product outcomes and independent fixture observations
```

- Godog owns each scenario's server, response data, request observations and cleanup.
- Go's [httptest.Server](https://pkg.go.dev/net/http/httptest#Server) owns the listening server. [http.ServeMux](https://pkg.go.dev/net/http#ServeMux) supplies ordinary method/path routing; scenario handlers encode JSON or serve bytes and files.
- MMM retains platform-specific request construction, headers, parsing and selection. Its shared HTTP and download code retains retries, rate limiting, response validation and filesystem operations.
- The terminal driver owns terminal interaction and process control. HTTP fixtures do not depend on whether the driver is reached through a CLI or a language binding.

Keep fixture handlers alongside the E2E scenarios and reusable product actions. Do not build an expectation DSL, matcher registry, request planner or general mocking framework. Share concrete fixture data and handlers where scenarios need the same platform behavior.

## Planned endpoint configuration

The E2E harness will start one server per scenario on an automatically allocated loopback port, then pass all four values to each MMM process it launches. Distinct path prefixes can separate platform APIs, the Minecraft manifest and downloads on that server.

| Planned variable | Meaning |
| --- | --- |
| `MMM_E2E_MODRINTH_BASE_URL` | Base before Modrinth's `/v2/...` paths; an optional fixture path prefix is retained |
| `MMM_E2E_CURSEFORGE_BASE_URL` | Complete CurseForge API base, including `/v1`; request paths are appended to it |
| `MMM_E2E_MINECRAFT_MANIFEST_URL` | Complete Minecraft version-manifest URL |
| `MMM_E2E_DOWNLOAD_ORIGIN` | Exact allowed download origin: scheme, loopback address and port, with no path |

These are environment variables for `e2e`-tagged executables only, not user-facing command flags or modlist fields. Read and validate them before an E2E executable runs a command. All four are required, including for launches that are not expected to use HTTP; missing or invalid values are fixture-configuration failures, with no fallback to production endpoints.

Accept absolute `http` URLs with the literal loopback address `127.0.0.1` or `[::1]` and an explicit valid port. Reject credentials, query strings and fragments. API bases may contain path prefixes, and the manifest URL contains its resource path. The download origin contains no path. The harness allocates addresses per scenario rather than using fixed ports or inheriting endpoint values from the developer's shell.

Fixture responses must point download URLs at the configured download origin. The tagged build's HTTP download exception applies only to that exact origin, including its port. E2E requests and redirects must remain within configured fixture origins; fixtures must never forward unmatched requests to live services. Preserve credential clearing and telemetry opt-out in the child environment.

Normal builds ignore all four variables and retain their production API endpoints, HTTPS download requirement and trusted download hosts. Do not introduce a general insecure-download switch or disable TLS verification. These fixtures exercise real loopback HTTP; they do not establish production DNS, TLS or proxy behavior.

## Responses and observable outcomes

Use fixed synthetic metadata and small deterministic file payloads. Platform responses must follow the relevant [Modrinth](https://docs.modrinth.com/api/) and [CurseForge](https://docs.curseforge.com/rest-api/) contracts, including hashes matching the served download bytes. No live recording, service credentials or upstream access is required.

Register handlers before launching MMM. Match the request properties that matter to the scenario, including method, path, query, headers or body as applicable. Independent routes must work in any order. Synchronize mutable fixture state without holding a shared lock while waiting or writing a response, so one delayed download does not block unrelated requests.

Keep product outcomes separate from fixture correctness:

- Gherkin describes what a named actor does and observes. [BDD actions](bdd.md) arrange concrete fixture responses and exercise MMM.
- Assert the product through terminal observations, exit status and resulting files. For downloads, check the expected bytes and relevant modlist or lockfile state.
- Retain unexpected requests, invalid request contents and fixture-handler errors in scenario-owned, synchronized state. They must fail the scenario even if MMM handles the resulting HTTP error successfully.
- Verify requests required to establish the scenario occurred. Do not impose an order on independent requests or assert incidental call counts. Where retries are the behavior under test, define the response sequence and required attempts explicitly.
- An intentionally configured error response is valid fixture behavior. A missing handler or fixture failure must not satisfy a scenario expecting a platform failure.

## HTTP errors, delays and transfer failures

Ordinary handlers can return successful responses, unavailable projects and HTTP error statuses. Keep the real retry policy active; a retryable failure needs responses for the subsequent attempts as well.

A complete response with an error status does not demonstrate a stalled or interrupted download. Scenarios for those behaviors need controlled response streaming, request cancellation or connection termination. Coordinate the handler and scenario through explicit signals and observe the request context so cleanup can release blocked work. Do not use arbitrary sleeps to guess when MMM has reached a state, or treat a terminal's idle screen as proof that a request completed.

Introduce such controls only for a scenario that needs them. Use Go's HTTP response and connection facilities directly rather than adding a generic fault-injection framework. A fixture-induced connection failure or expected cancellation must be distinguished from an accidental handler error.

## Lifecycle and diagnostics

1. Create the scenario's isolated workspace and server; register its handlers and expected observations.
2. Pass the allocated endpoints to MMM and drive the product through the terminal harness.
3. Establish product outcomes, then stop any remaining MMM process through the terminal driver before closing its server. Release controlled handlers so cleanup cannot hang.
4. Collect terminal diagnostics, fixture errors and request observations before discarding state. Verify required requests after the client has stopped, when no further requests can arrive.
5. Close the server and remove the workspace. Cleanup failures also fail the scenario; retain the original product or fixture failure when reporting additional cleanup errors.

Keep diagnostic data limited to synthetic fixture traffic and the existing terminal evidence. Do not record real credentials. No server, request ledger or mutable response state is shared between scenarios.

## Evidence needed before adoption

The first implementation should demonstrate initialization using the local manifest, a compatible mod download from each platform, and project-not-found responses that leave the modlist unchanged. Independently verify unexpected-request detection, missing required requests and concurrent handlers.

Verify endpoint validation, exact download-origin restrictions, redirect containment and normal builds ignoring the test-only variables. Run the scenarios on Windows, macOS and Linux. The Godog acceptance gate must fail undefined and pending steps. These are implementation acceptance requirements, not claims about current coverage.

The normal contributor and [terminal E2E workflows](terminal-harness.md#running-the-suite) remain the entry points. Implementing this design does not require replacing the terminal binding or removing the legacy VCR helper.

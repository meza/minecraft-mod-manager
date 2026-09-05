# `mmm test`

> This guide describes the Go-port target defined in [product intent](../intent.md), not a claim that every released build already implements it.

`test` checks whether each configured project has a platform-eligible artifact for a target Minecraft version. It is a read-only way to assess a version change before running [`mmm change`](change.md).

See [shared command behavior](README.md), especially [configuration and setup](README.md#configuration-and-setup), [execution modes](README.md#execution-modes), and [results and retry](README.md#results-and-retry).

```bash
mmm test 1.20.4
```

## Usage

```text
mmm test [game-version]
mmm t [game-version]
```

If no target is supplied, MMM looks up the latest stable Minecraft release. An explicit target can be any version listed in Mojang's manifest, including snapshots and prereleases. The selected platforms must still provide artifacts that satisfy the configured loader, release types, pins, and per-mod fallback choices.

An invalid target can be corrected in an interactive run while other inputs are preserved. If the latest-version lookup fails, the interactive flow can collect an explicit target that MMM can validate. Unattended or redirected execution reports invalid input or lookup failure without prompting. An unavailable validation service is reported as inconclusive; it is not proof that a supplied target is invalid.

Use the shared `--config` option to inspect another installation:

```bash
mmm --config ./server/modlist.json test 1.21.1
```

## Reported outcomes

For each declared mod, `test` reports one of these meanings:

- **Compatible:** the platform reports an artifact eligible under the declaration for the target.
- **Incompatible:** the platform conclusively reports that no eligible artifact is available for the target.
- **Inconclusive:** MMM could not determine eligibility because an API request, lookup, or required service failed.

These results describe platform metadata, not a Minecraft launch or runtime compatibility test. A known incompatible result has a distinct non-success outcome suitable for a scriptable compatibility gate. Inconclusive checks are also non-successful and remain distinguishable from known incompatibility. When both occur, the overall result is incomplete and the report preserves both kinds of finding.

MMM explains which mods could not be established and why when known. It does not label a network or service failure as incompatibility.

If the target equals the configured Minecraft version, the compatibility report still remains an inspection operation; use `install` to repair missing or changed managed files.

## Read-only and recovery behavior

The `test` operation does not change the configuration, lockfile, or mods directory. A missing lockfile is reported as missing resolution evidence and is not created.

If configuration is missing, an interactive invocation may offer the separate shared initialization flow. If the configuration contains conflicting duplicate declarations, it may offer the shared correction flow. Only an explicitly accepted setup or correction changes metadata; after it succeeds, the original `test` request resumes. Declining, cancelling, or failing recovery leaves the unresolved state untouched and does not run the compatibility check against it.

Unattended and redirected runs never prompt or initialize implicitly. They report actionable setup or correction guidance and return a non-success result.

Cancellation stops unfinished checks and preserves every conclusive result already obtained. The final outcome identifies incomplete work rather than treating it as compatibility success.

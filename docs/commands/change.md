# `mmm change`

> This guide describes the Go-port target defined in [product intent](../intent.md), not a claim that every released build already implements it.

`change` changes the modlist's Minecraft target and prepares the corresponding managed installation.

See [shared command behavior](README.md), especially [lookup and lockfiles](README.md#lookup-and-lockfiles), [results and retry](README.md#results-and-retry), and [cancellation and terminal output](README.md#cancellation-and-terminal-output). Use [`mmm test`](test.md) first when you want a read-only compatibility report.

```bash
mmm change 1.21.1
```

## Usage and options

```text
mmm change [game-version]
```

| Option | Meaning |
| --- | --- |
| `--force` | Proceed past platform-reported incompatibilities and retain existing installed files when no eligible replacement is available. |

If no target is supplied, MMM looks up the latest stable Minecraft release. An explicit target can be any version listed in Mojang's manifest, including snapshots and prereleases. Platform artifact availability remains subject to the modlist's loader, release types, pins and per-mod fallback choices.

An invalid target can be corrected in an interactive run while other input is preserved. If the latest-version lookup fails, the interactive flow can collect an explicit target that MMM can validate. Unattended or redirected execution reports invalid input or lookup failure without prompting or changing the installation. A service outage must not be reported as proof that an explicit version is invalid.

## Ordinary change

Before switching, MMM checks whether each project in the modlist has an eligible artifact for the requested target. It reports platform-declared compatibility; it does not prove that Minecraft will run successfully.

An ordinary change proceeds only when the compatibility gate passes. MMM prepares target artifacts before switching the working installation. If compatibility checks or preparation fail, it preserves the original modlist and working installation.

Finding a compatibility blocker does not end the report early. Without force, cancel unnecessary outstanding downloads and clean up staging, but finish the remaining compatibility checks so the operator sees all known blockers and any inconclusive checks from the attempted change. Report the non-success outcome after those checks finish. An operator-requested cancellation still follows the safe-cancellation contract; it does not require completing the report.

Once switching starts, MMM completes the metadata and file consistency work or attempts recovery to the original installation. If recovery cannot complete, the result identifies the actual remaining state and the next action. It does not claim that nothing changed unless that was established.

If the requested target already equals the modlist's Minecraft target, `change` succeeds as a no-op even if a managed file is missing or damaged. Run `mmm install` to repair the current target.

## Forced retention

`change --force` bypasses known platform-reported incompatibilities. It installs every eligible target replacement it can prepare. When no eligible replacement exists and the current installed file is available, MMM retains that file in place: it does not disable it, delete it, or remove its mod config.

```bash
mmm change --force 1.21.1
```

The mod config remains in the modlist and the exact retained artifact remains in the lockfile for the new target. Later `install` runs preserve or reproduce that exact retained artifact without requiring force again. A later `update` may replace it when a newer eligible artifact becomes available.

This authorization applies only to that retained artifact and target. A future artifact resolution or a change to its effective lookup constraints must be assessed under the normal lookup and force rules. Effective constraints include the modlist-wide Minecraft target, loader and default release policy together with the mod config's per-mod settings. The report identifies retained artifacts and the missing platform compatibility declaration so you can decide whether they work when Minecraft starts.

If neither a replacement nor an existing installed file is available, MMM cannot satisfy that mod. A request or download failure also does not authorize a silent retention decision. Preparation remains incomplete, and the command reports a non-success result rather than claiming the version change succeeded.

Force does not bypass integrity requirements, exclusions, collision protection, or recovery guarantees.

## Failure, cancellation, and retry

Preparation leaves the original state intact until MMM can switch coherently. After switching begins, cancellation may need to finish consistency work or recover the previous installation before returning control. Completed recovery has no automatic timeout; a second interruption after the explicit warning is an emergency exit and may leave recovery unfinished.

Retry after a reported failure or incomplete recovery follows the state and next action reported by MMM. Existing unmanaged jars and files excluded by `.mmmignore` or the `.disabled` suffix remain untouched throughout the change.

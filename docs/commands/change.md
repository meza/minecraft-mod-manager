# `mmm change`

> This guide describes the Go-port target defined in [product intent](../intent.md), not a claim that every released build already implements it.

`change` changes the configured Minecraft version and prepares the corresponding managed installation.

See [shared command behavior](README.md), especially [selection and lockfiles](README.md#selection-and-lockfiles), [results and retry](README.md#results-and-retry), and [cancellation and terminal output](README.md#cancellation-and-terminal-output). Use [`mmm test`](test.md) first when you want a read-only compatibility report.

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

If no target is supplied, MMM looks up the latest stable Minecraft release. An explicit target can be any version listed in Mojang's manifest, including snapshots and prereleases. Platform artifact availability remains subject to the configured loader, release types, pins, and per-mod fallback choices.

An invalid target can be corrected in an interactive run while other input is preserved. If the latest-version lookup fails, the interactive flow can collect an explicit target that MMM can validate. Unattended or redirected execution reports invalid input or lookup failure without prompting or changing the installation. A service outage must not be reported as proof that an explicit version is invalid.

## Ordinary change

Before switching, MMM checks whether each configured project has an eligible artifact for the requested target. It reports platform-declared compatibility; it does not prove that Minecraft will run successfully.

An ordinary change proceeds only when the compatibility gate passes. MMM prepares target artifacts before switching the working installation. If compatibility checks or preparation fail, it preserves the original configuration and working installation.

Once switching starts, MMM completes the metadata and file consistency work or attempts recovery to the original installation. If recovery cannot complete, the result identifies the actual remaining state and the next action. It does not claim that nothing changed unless that was established.

If the requested target already equals the configured Minecraft version, `change` succeeds as a no-op even if a managed file is missing or damaged. Run `mmm install` to repair the current target.

## Forced retention

`change --force` bypasses known platform-reported incompatibilities. It installs every eligible target replacement it can prepare. When no eligible replacement exists and the current installed file is available, MMM retains that file in place: it does not disable it, delete it, or remove its declaration.

```bash
mmm change --force 1.21.1
```

The retained artifact remains declared and locked for the new target. Later `install` runs preserve or reproduce that exact retained selection without requiring force again. A later `update` may replace it when a newer eligible artifact becomes available.

This authorization applies only to that retained artifact and target. A future artifact selection or a change to its declaration constraints must be assessed under the normal selection and force rules. The report identifies retained artifacts and the missing platform compatibility declaration so you can decide whether they work when Minecraft starts.

If neither a replacement nor an existing installed file is available, MMM cannot satisfy that mod. A request or download failure also does not authorize a silent retention decision. Preparation remains incomplete, and the command reports a non-success result rather than claiming the version change succeeded.

Force does not bypass integrity requirements, exclusions, collision protection, or recovery guarantees.

## Failure, cancellation, and retry

Preparation leaves the original state intact until MMM can switch coherently. After switching begins, cancellation may need to finish consistency work or recover the previous installation before returning control. Completed recovery has no automatic timeout; a second interruption after the explicit warning is an emergency exit and may leave recovery unfinished.

Retry after a reported failure or incomplete recovery follows the state and next action reported by MMM. Existing unmanaged jars and files excluded by `.mmmignore` or the `.disabled` suffix remain untouched throughout the change.

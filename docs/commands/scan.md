# `mmm scan`

> This guide describes the Go-port target defined in [product intent](../intent.md), not a claim that every released build already implements it.

`mmm scan` identifies visible unmanaged jars and can explicitly adopt recognized artifacts into `modlist.json` and its lockfile.

```bash
mmm scan
mmm scan --prefer curseforge --add
```

## Options

| Option | Meaning | Accepted values | Default |
| --- | --- | --- | --- |
| `-p, --prefer` | Platform to search first | `modrinth`, `curseforge` | `modrinth` |
| `-a, --add` | Adopt recognized, unambiguous results without an adoption prompt | flag | off |

Global options, setup recovery, mod config correction and invalid-option correction are covered in [shared command behavior](README.md#modlist-and-setup).

## Discovery and platform fallback

Scan examines only `.jar` files immediately inside the modlist's mods directory; it does not recurse into subdirectories. It skips managed artifacts, files matched by `.mmmignore`, and files ending in `.disabled`.

MMM identifies candidates by their artifact evidence. It searches the preferred platform first and uses the other supported platform as fallback. If timeout or connection failures continue after retries on the preferred platform, fallback can still produce a conclusive match. A candidate whose lookups cannot be completed remains uncertain; a conclusive no-match is unknown. Service failure is not reported as proof that a file is unknown.

The report separates known, unknown and uncertain candidates. Discovery alone never grants MMM ownership of a file.

## Adoption

In an interactive run without `--add`, MMM offers a choice before adopting recognized candidates. `--add` is explicit adoption intent for recognized, unambiguous results. Without prompting and without `--add`, scan reports candidates without changing metadata.

Adoption creates or updates the project's mod config in the modlist. It records the exact artifact already on disk in the lockfile and does not download a newer version. Unknown or uncertain candidates are never silently adopted, and declining adoption preserves both files and metadata. Already managed projects are not duplicated.

When multiple discovered jars identify the same project, MMM keeps an existing valid locked artifact. Without one, an interactive run asks which artifact to adopt. In an unattended or redirected run, MMM leaves that project unresolved while still adopting independent unambiguous projects when `--add` was supplied. The order in which files are processed never decides ownership; unselected jars remain unmanaged.

Adoption does not silently override an existing version pin. If a discovered artifact conflicts with a pin, MMM reports the conflict and requires explicit consent to change it. Without prompting, the project remains unchanged.

Explicit adoption of an artifact without an existing mod config can authorize that exact artifact even when the platform does not report it as compatible with the configured Minecraft target. MMM reports that limitation and records enough evidence to preserve or reproduce the adopted artifact on later installs. This authorization applies only to that artifact and target; it does not waive integrity checks or authorize future incompatible artifacts.

## Results, failure and retry

Each candidate can complete independently. Successful adoptions remain consistent if another lookup or metadata change fails, and retries do not create duplicate mod configs or lock entries. Invalid ignore patterns stop mutation and identify the offending line so intended exclusions are not exposed.

See the shared guidance for [execution modes](README.md#execution-modes), [results and retry](README.md#results-and-retry), and [safe cancellation](README.md#cancellation-and-terminal-output). Cancellation stops new lookup and adoption work while retaining completed independent changes and restoring consistent metadata before returning control.

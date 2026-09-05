# `mmm install`

> This guide describes the Go-port target defined in [product intent](../intent.md), not a claim that every released build already implements it.

`mmm install` makes the managed installation match `modlist.json` while preserving exact valid selections already recorded in the lockfile.

```bash
mmm install
mmm i
```

Use it after changing declarations, pulling a modpack's configuration and lockfile, or whenever a managed file is missing or damaged. Global options, setup recovery and declaration correction are covered in [shared command behavior](README.md#configuration-and-setup).

## What it reconciles

MMM reads three distinct sources of state:

- `modlist.json` declares the mods and their selection constraints.
- The matching lockfile records exact resolved artifacts. For example, `server.json` uses `server-lock.json`.
- The configured mods directory contains the files currently on disk.

For each declaration, `install` preserves a valid locked resolution that still satisfies the declaration. It resolves newly declared mods and declarations whose constraints have changed, installs missing locked files, and replaces managed files whose contents do not match the recorded integrity information. It also removes lock entries and previously managed files for mods removed from the declaration.

It does not choose a newer release for an already satisfied lock entry. Run [`mmm update`](update.md) when you intend to advance eligible unpinned selections.

Resolution uses the configured Minecraft version, loader and allowed release types, plus any per-mod pin, release-type override or `allowVersionFallback` setting. A permitted fallback stays within the same Minecraft release series and is reported when selected. Stored project names may refresh from platform metadata; platform and project ID define identity.

## Lockfile behavior

A missing or incomplete lockfile is reconciled as part of installation. Missing entries are resolved from the declarations without discarding unrelated valid resolutions. When the whole lockfile is missing, MMM cannot reproduce earlier exact choices; it resolves the current constraints and reports the result.

Missing evidence is different from corrupt evidence. MMM preserves and reports malformed or contradictory lock data, or an existing resolution without required integrity information, rather than silently selecting a possibly different artifact. Follow the reported recovery guidance and retry.

If a recorded artifact is no longer available or its download fails, MMM reports that failure instead of substituting another artifact for the locked selection.

To reproduce the same managed selection on another machine, keep `modlist.json` and its lockfile together.

## Files outside MMM's ownership

Visible unmanaged jars are legitimate and do not block unrelated installation work. `install` reports them but does not adopt, replace or delete them. A concrete destination collision fails the affected mod without overwriting the unrelated file.

Files matched by `.mmmignore` and files ending in `.disabled` are excluded. `--force` does not override those protections. A disabled file does not satisfy its enabled declaration, so `install` may recreate the enabled `.jar` beside its `.disabled` counterpart.

## Results, failure and retry

Mods are processed independently. Successful work remains consistent if another mod fails, and the final report distinguishes completed, failed and unresolved work. A retry continues toward the declared state without duplicating completed entries. Already satisfied installations succeed without material changes.

A failed or incomplete download never replaces a working managed file. Metadata failures and unsafe extra loadable jars are operation failures. See the shared guidance for [execution modes](README.md#execution-modes), [results and retry](README.md#results-and-retry), and [safe cancellation](README.md#cancellation-and-terminal-output).

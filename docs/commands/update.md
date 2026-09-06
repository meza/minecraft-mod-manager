# `mmm update`

> This guide describes the Go-port target defined in [product intent](../intent.md), not a claim that every released build already implements it.

`mmm update` advances eligible unpinned mods and reconciles the resulting managed installation.

```bash
mmm update
mmm u
```

The command has no command-specific options. Use the global options and setup described in [shared command behavior](README.md#modlist-and-setup), including `--config` for an alternative modlist and `--unattended` when prompts are unavailable.

## Lookup behavior

`update` reads the current modlist and existing resolution evidence, resolves the desired artifacts under the update lookup policy, and then applies those resolutions. It does not run an ordinary `install` first. A newly added mod config or a changed loader, Minecraft version, release policy, fallback setting or pin participates directly in lookup, without installing an intermediate locked artifact.

For an unpinned mod, an update must have a later publication date and different file content. It must also satisfy the modlist's Minecraft target, loader and release types, including per-mod overrides. A changed display name or version string alone does not establish an update.

The per-mod `allowVersionFallback` setting defaults to disabled. When enabled, fallback stays within the same Minecraft release series and MMM reports the fallback resolution. It does not allow unrelated constraint overrides.

If the newest matching release lacks required download or integrity information, the affected update fails. MMM does not silently choose an older release to work around unusable newest metadata.

## Pins and existing artifacts

Pinned mods stay pinned during ordinary updates. If the mod config's pin changed since the lockfile was written, the new explicit pin is binding and is resolved directly.

A missing old jar, or an old artifact that can no longer be downloaded, does not prevent MMM from installing an eligible newer artifact. Each replacement is prepared and verified before the previous working file is removed. If preparation fails, the previous working artifact remains in place and the report identifies the unsatisfied update.

Missing lock entries are reconciled normally without discarding unrelated valid resolutions. Corrupt or contradictory resolution evidence is preserved and reported with recovery guidance rather than silently reconstructed.

## Ownership and exclusions

Unmanaged jars remain untouched and do not block unrelated updates. MMM does not adopt them or move a managed mod to another platform without explicit operator intent. A destination collision fails the affected update instead of overwriting an unrelated file.

Files matched by `.mmmignore` and files ending in `.disabled` are excluded. A disabled counterpart does not satisfy a declared enabled file, so reconciliation may create the enabled `.jar` beside it.

## Results, failure and retry

Successful updates remain installed when another mod fails. The overall result is incomplete when any work fails or remains unresolved, and the report identifies the affected mods and useful next actions. Retrying preserves completed work and does not create duplicate mod configs or lock entries. A fully satisfied installation with no eligible updates is a successful no-op.

See the shared guidance for [execution modes](README.md#execution-modes), [results and retry](README.md#results-and-retry), and [safe cancellation](README.md#cancellation-and-terminal-output). A broken output pipe does not cancel an authorized update; MMM continues the consistency work even when the reader can no longer receive output.

# `mmm init`

> This guide describes the Go-port target defined in [product intent](../intent.md), not a claim that every released build already implements it.

`init` creates a modlist and an empty lockfile for a new managed installation. It collects the mod loader, Minecraft version, allowed release types, and mods directory. It does not download, delete, or adopt any mods.

See [modlist and setup](README.md#modlist-and-setup), [execution modes](README.md#execution-modes), and [cancellation and terminal output](README.md#cancellation-and-terminal-output) for behavior shared with other commands.

```bash
mmm init --loader fabric
```

## Inputs and defaults

| Short | Long | Meaning | Default |
| --- | --- | --- | --- |
| `-l` | `--loader` | Minecraft mod loader | None |
| `-g` | `--game-version` | A version listed in Mojang's manifest, or `latest` | `latest`, meaning the latest stable release |
| `-r` | `--release-types` | Nonempty comma-separated list of `alpha`, `beta`, and `release` | `release` |
| `-m` | `--mods-folder` | Existing directory for installed mods | `mods`, relative to the modlist directory |
| `-c` | `--config` | Modlist file to create | `./modlist.json` |
| `-f` | `--force` | Skip confirmation before resetting an existing modlist or orphan lockfile | Off |

Relative mods paths are resolved from the directory containing the selected modlist. Absolute paths are supported. The mods directory must already exist and be usable; `init` does not offer to create it. Parent directories for a new modlist file may be created separately from the mods directory.

Loader has no default. In a no-prompt run, supply it explicitly:

```bash
mmm --unattended init --loader fabric
```

This uses the default latest stable Minecraft release, release-only artifacts, and a `mods` directory beside `modlist.json`.

## Interactive collection

In an interactive terminal, `init` skips each explicitly supplied valid field and collects the rest in this order:

1. loader
2. Minecraft version
3. release types
4. mods directory

Defaults are offered for omitted fields. They are still collected in the interactive flow; a default is not treated as an explicitly supplied value. For example, `mmm init --loader fabric` still asks about Minecraft version, release types, and mods directory.

An invalid supplied value reopens only that field for correction and preserves the other inputs. Minecraft versions can be selected from suggestions, and release types use a selection that requires at least one choice. If the latest-version lookup fails, enter an explicit manifest-listed version that MMM can validate.

When collection is needed, the interactive flow ends with a final confirmation before writing. This confirmation still appears with `--force`, because force only skips the separate reset confirmation. Declining the final confirmation or pressing Escape cancels without saving. The Go-port flow has no step-back navigation; Escape only leaves filtering when a list control is currently filtering.

When every required value is explicitly supplied and valid, `init` writes directly without opening the collection flow, provided no existing metadata requires reset confirmation.

In unattended or redirected execution, defaults are used directly, loader remains required, and invalid or missing required input fails without writing or prompting.

## Existing metadata and reset behavior

Initialization writes the selected modlist and its matching empty lockfile. For example, `--config server.json` writes `server.json` and `server-lock.json`.

If either an existing modlist or an orphan lockfile would be reset, `init` explains that the metadata will be replaced while existing jars remain untouched. It then requires reset confirmation or `--force`.

If you decline the reset in an interactive run, `init` offers another modlist path instead of silently cancelling. Changing that path also changes the base directory for relative mods paths, so MMM revalidates the location and preserves the entered text if it needs correction.

`--force` does not provide a missing loader, accept invalid settings, bypass the final interactive confirmation, adopt jars, or authorize deleting jars.

When another command finds no modlist, it can offer this same initialization flow. After successful initialization, the original command resumes with its original inputs and the resulting modlist path. If initialization is declined, cancelled, or fails, the original command does not run.

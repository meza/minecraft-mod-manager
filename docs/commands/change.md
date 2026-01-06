# `mmm change`

`mmm change` switches your configuration to a new Minecraft version and reinstalls every configured mod for that version.
It downloads the new jars first and only switches after all downloads succeed.

Use it when you are ready to move your modpack to a different Minecraft release.

```bash
mmm change 1.21.1
```

With `--force`, the command proceeds even if some mods do not support the target version. Those mods are skipped during install.
If incompatible jars already exist, MMM keeps your current config and removes those files unless you supply a policy flag.
In interactive runs, MMM will ask which policy to use when you do not set one.

```bash
mmm change --force latest
```

## What it does

- Reuses the `test` checks to verify that your configured mods support the target version (unless you pass `--force`).
- Downloads the compatible jars into `mods/.mmm-staging` without touching your current mods.
- If all downloads succeed, swaps in the new jars and updates `modlist-lock.json` and `modlist.json`.
- Cleans up staging and backup files.

If you attempt to change to the current version, the command exits with code `0` and makes no changes.

## Usage

```bash
mmm change [game_version]
```

## Options

| Flag                  | Meaning                                                                 | Allowed values | Example                                    |
|-----------------------|-------------------------------------------------------------------------|----------------|--------------------------------------------|
| `-f, --force`         | Proceed even if some mods lack support; unsupported mods are skipped    | `true/false`   | `mmm change --force 1.21.1`                |
| `--keep-config`       | Keep incompatible mods in the config and remove their files (requires `--force`) | `true/false`   | `mmm change --force --keep-config 1.21.1`  |
| `--prune-config`      | Remove incompatible mods from the config and remove their files (requires `--force`) | `true/false`   | `mmm change --force --prune-config 1.21.1` |
| `--disable-skipped`   | Keep incompatible mods in the config and disable their files (requires `--force`) | `true/false`   | `mmm change --force --disable-skipped 1.21.1` |

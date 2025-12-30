# `mmm change`

`mmm change` switches your configuration to a new Minecraft version and reinstalls every configured mod for that version.

Use it when you are ready to move your modpack to a different Minecraft release.

```bash
mmm change 1.21.1
```

With `--force`, the command proceeds even if some mods do not support the target version. Those mods are skipped during install.

```bash
mmm change --force latest
```

## What it does

- Reuses the `test` checks to verify that your configured mods support the target version (unless you pass `--force`).
- Removes the currently installed mod jars and clears `modlist-lock.json`.
- Updates `modlist.json` with the new `gameVersion`, then runs `install` to fetch compatible releases.

If you attempt to change to the current version, the command exits with code `2` and makes no changes.

## Usage

```bash
mmm change [game_version]
```

## Options

| Flag          | Meaning                                                              | Allowed values | Example                     |
|---------------|----------------------------------------------------------------------|----------------|-----------------------------|
| `-f, --force` | Proceed even if some mods lack support; unsupported mods are skipped | `true/false`   | `mmm change --force 1.21.1` |

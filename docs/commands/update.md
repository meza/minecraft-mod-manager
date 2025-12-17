# `mmm update`

`mmm update` checks each configured mod for a newer compatible release and installs it for you.

Most of the time you run this after a while to bring your mods folder up to date without re-adding everything.

```bash
mmm update
```

## What it does

- Runs `mmm install` first to make sure your lock file and mods folder are consistent.
- For each configured mod that is not pinned to a specific `version`, checks for a newer release matching your configured Minecraft version, loader, and allowed release types.
- Downloads the newer jar, removes the previous jar, and updates `modlist-lock.json`.
- Updates mod names in `modlist.json` to stay in sync.

If a mod is pinned to a specific `version`, `update` will not look for newer releases. The pinned version is only re-downloaded if the file is missing or does not match the lock file hash (during the initial `install` phase).

## Usage

```bash
mmm update
mmm u
```

## Options

This command only uses the global options:

| Flag | Meaning | Allowed values | Example |
| --- | --- | --- | --- |
| `-c, --config` | Path to `modlist.json` | file path | `mmm --config ./server/modlist.json update` |
| `-q, --quiet` | Suppress normal output | `true/false` | `mmm --quiet update` |
| `-d, --debug` | Print debug details | `true/false` | `mmm --debug update` |


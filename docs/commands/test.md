# `mmm test`

`mmm test` checks whether all your configured mods have versions available for a target Minecraft release. Use this to see if you can safely upgrade before making any changes.

```bash
mmm test 1.20.4
```

## Flags

| Flag | Meaning | Allowed values | Example |
| --- | --- | --- | --- |
| `-c, --config` | Use a specific `modlist.json` path | File path | `mmm test -c ./modlist.json 1.20.4` |
| `-q, --quiet` | Suppress non-essential output | `true` or `false` | `mmm test --quiet 1.20.4` |
| `-d, --debug` | Print debug details | `true` or `false` | `mmm test --debug 1.20.4` |

## What it does

- Resolves the target game version (from your argument or defaults to the latest stable Minecraft release).
- Queries each configured mod to see if a compatible version exists for that target.
- Reports which mods lack support so you know what is blocking an upgrade.
- Makes no changes to your configuration, lock file, or mods folder.

If you are on 1.19.2 and want to see whether you could upgrade to 1.19.3, run `mmm test 1.19.3`. If you omit the game version, the command uses the latest stable Minecraft version automatically. If the command cannot fetch the latest version (for example, when offline), provide an explicit version instead.

In unattended environments, MMM fails fast if it cannot determine the latest version, so you should supply an explicit version.
If the target version matches your configured version, the command prints a no-op message and exits successfully.
If MMM cannot determine compatibility for a mod because a platform check fails, it reports that mod as inconclusive and exits non-zero.

## Exit codes for automation

For server operators and script automation, the command returns specific exit codes:

| Exit code | Meaning |
| --- | --- |
| 0 | All mods support the target version, or the target version matches your current version |
| 1 | One or more mods lack support for the target version, or compatibility could not be determined |

You could run `mmm test` daily in a cron job. When it returns exit code 0, you know a version upgrade is possible. If it returns 1, some mods are incompatible or MMM could not confirm compatibility due to a platform error.

## Usage

```bash
mmm test [game_version]
mmm t [game_version]
```

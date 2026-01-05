This package implements the `mmm change` command: verify a target Minecraft version, download replacement jars into a staging area, and only then switch the mods folder and configuration to that version.

The command runs the same compatibility checks as `mmm test` (unless `--force` is provided), stages downloads under `mods/.mmm-staging`, updates `modlist.json` and `modlist-lock.json`, and attempts rollback if switching fails.

If you adjust behavior, keep `docs/specs/change.md` and `docs/commands/change.md` in sync with the code.

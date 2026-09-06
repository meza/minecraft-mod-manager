This package implements the `mmm change` command: verify a target Minecraft version, download replacement artifacts into a staging area, and only then switch the mods folder and modlist to that version.

The command runs the same compatibility checks as `mmm test` (unless `--force` is provided), stages downloads under `mods/.mmm-staging`, updates `modlist.json` and `modlist-lock.json`, and attempts rollback if switching fails.

If you adjust behavior, follow the target in [`docs/intent.md`](../../../docs/intent.md) and keep the [`change` command guide](../../../docs/commands/change.md) accurate.

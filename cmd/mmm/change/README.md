This package implements the `mmm change` command: verify a target Minecraft version, clear existing installations, update the config to that version, and rerun install for the new release.

The command runs the same compatibility checks as `mmm test` (unless `--force` is provided) and uses `install` to fetch the appropriate mod files after switching versions.

If you adjust behavior, keep `docs/specs/change.md` and `docs/commands/change.md` in sync with the code.

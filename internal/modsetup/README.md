# internal/modsetup

This package contains idempotent workflows for setting up a mod config, a lock
entry for its resolved artifact, and the artifact's local file across:

- `modlist.json` (modlist)
- `modlist-lock.json` (lockfile)
- the mods directory on disk

Commands like `mmm add` and `mmm scan --add` should treat this package as the
single source of truth for setting up those records and the local file, while
the commands themselves remain responsible for CLI parsing, UI, and telemetry.

## Idempotency & reconciliation

The "ensure" methods are idempotent in the sense that they do not create
duplicate entries for the same `(platform,id)` pair, and they fill in missing
entries between the modlist and lockfile.

They do **not** currently reconcile conflicting or "mismatched" states (for
example: modlist and lock entries are present but the local jar hash differs). That behavior
will be implemented where the command UX requires it (for example `scan`).

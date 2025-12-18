# internal/modsetup

This package contains the idempotent workflow that ensures a mod is correctly
represented across:

- `modlist.json` (configuration)
- `modlist-lock.json` (lock file)
- the mods directory on disk

Commands like `mmm add` and `mmm scan --add` should treat this package as the
single source of truth for "making a mod fully set up", while the commands
themselves remain responsible for CLI parsing, UI, and telemetry.

## Idempotency & reconciliation

The "ensure" methods are idempotent in the sense that they do not create
duplicate entries for the same `(platform,id)` pair, and they fill in missing
entries between config and lock.

They do **not** currently reconcile conflicting or "mismatched" states (for
example: config and lock present but the local jar hash differs). That behavior
will be implemented where the command UX requires it (for example `scan`).

# internal/modinstall

This package contains reusable logic for ensuring the local file for a locked
artifact in `modlist-lock.json` is present on disk.

It is extracted from `mmm install` so other commands (for example `mmm add`)
can reuse the same behavior without re-implementing it.

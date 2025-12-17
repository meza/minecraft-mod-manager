This package implements the `mmm update` command: reconcile the workspace by running `install`, then check each configured mod for a newer compatible release and swap in the updated jar when found.

This command is intentionally non-interactive (no prompts). It reuses the same config/lock helpers and platform selection logic as `add` and `install`.

If you change behavior, update `docs/specs/update.md` first (or alongside the code) so reviewers have one source of truth.

This package implements the `mmm update` command: reconcile the workspace by running `install`, then check each configured mod for a newer compatible release and swap in the updated jar when found.

This command is intentionally unattended (no prompts). It reuses the same config/lock helpers and platform selection logic as `add` and `install`.

If you change behavior, update `docs/specs/update.md` first (or alongside the code) so reviewers have one source of truth.

Interaction contract and frames live in:
- `docs/interactions/interaction-guidelines.md`
- `docs/interactions/flows/update.md`

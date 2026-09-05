This package implements the `mmm update` command: reconcile the workspace by running `install`, then check each configured mod for a newer compatible release and swap in the updated jar when found.

This command is intentionally unattended (no prompts). It reuses the same config/lock helpers and platform selection logic as `add` and `install`.

If you change behavior, follow the target in [`docs/intent.md`](../../../docs/intent.md) and keep the [`update` command guide](../../../docs/commands/update.md) accurate.

Shared terminal implementation, conventions and annotated examples live in the [terminal corpus](../../../docs/interactions/README.md).

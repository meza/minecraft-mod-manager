This package implements the `mmm update` command: reconcile the installation by running `install`, then check each mod config for a newer compatible artifact and swap in the updated local file when found.

This command currently runs without prompts. It reuses the same modlist/lockfile helpers and platform artifact lookup logic as `add` and `install`.

If you change behavior, follow the target in [`docs/intent.md`](../../../docs/intent.md) and keep the [`update` command guide](../../../docs/commands/update.md) accurate.

Shared terminal implementation, conventions and annotated examples live in the [terminal corpus](../../../docs/interactions/README.md).

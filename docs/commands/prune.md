# `mmm prune`

> This guide describes the Go-port target defined in [product intent](../intent.md), not a claim that every released build already implements it.

`mmm prune` deletes visible unmanaged `.jar` files from the modlist's mods directory.

```bash
mmm prune
mmm prune --force
```

## Options

| Option | Meaning | Default |
| --- | --- | --- |
| `-f, --force` | Delete the identified unmanaged jars without confirmation | off |

`--force` supplies deletion authority; it does not enable prompting or override exclusions. Use `--unattended --force` when a no-prompt invocation should delete the reported set. Global options and setup are covered in [shared command behavior](README.md#modlist-and-setup).

## Deletion boundary

Prune considers only unmanaged `.jar` files immediately inside the modlist's mods directory. It does not recurse into subdirectories and does not delete unrelated file types.

The following remain untouched:

- artifacts identified as managed;
- files matched by `.mmmignore`;
- files ending in `.disabled`;
- files in subdirectories; and
- unrelated directory contents.

`--force` does not override these protections. An invalid ignore pattern stops deletion and identifies the offending line rather than silently exposing an intended exclusion.

## Confirmation and missing evidence

In an interactive terminal, prune shows the complete intended deletion set and asks for confirmation. Declining or cancelling preserves every candidate. `--force` authorizes deletion of that shown set without confirmation.

In unattended or redirected execution, MMM never prompts. Without `--force`, it reports that deletion needs explicit authorization and removes nothing.

Prune requires a lockfile to distinguish managed artifacts from unmanaged jars safely. If the lockfile is missing, it refuses deletion and explains that [`mmm install`](install.md) is needed to establish resolution evidence. A missing modlist follows the [shared setup behavior](README.md#modlist-and-setup); a missing lockfile is not treated as missing setup.

Corrupt or contradictory resolution evidence is preserved and reported; it is not evidence that a possibly managed jar is unmanaged. See [lookup and lockfiles](README.md#lookup-and-lockfiles) for the shared recovery contract.

## Results, failure and retry

An empty deletion set is a successful no-op. Each deletion can succeed or fail independently: successful removals remain complete, while failures identify the affected files and a useful next action. Retrying operates on the remaining unmanaged jars.

See the shared guidance for [execution modes](README.md#execution-modes), [result categories](README.md#results-and-retry), and [safe cancellation](README.md#cancellation-and-terminal-output). Cancellation stops scheduling new deletions, retains completed independent removals, and returns control only after required consistency work.

# `mmm remove`

> This guide describes the Go-port target defined in [product intent](../intent.md), not a claim that every released build already implements it.

`remove` deletes explicitly selected mods from the modlist, lockfile and managed files.

See [shared command behavior](README.md), especially [execution modes](README.md#execution-modes), [file ownership and exclusions](README.md#file-ownership-and-exclusions), and [results and retry](README.md#results-and-retry).

```bash
mmm remove sodium
```

## Usage and options

```text
mmm remove <id-or-name>...
```

Selections can be mod IDs, names, or supported case-insensitive glob patterns. Multiple selections may match the same mod; MMM removes it only once.

| Option | Meaning |
| --- | --- |
| `--force` | Remove the matched selection without confirmation. |

Examples:

```bash
mmm remove mod1 mod2 "mod with spaces"
mmm remove "world*edit*"
```

Quote names and patterns so the shell passes them to MMM unchanged.

## Confirmation and no-prompt execution

In an interactive terminal, MMM shows the matched mods and asks for confirmation. Declining or cancelling makes no changes. `--force` authorizes exactly the displayed or supplied selection and skips that confirmation.

In unattended or redirected execution, `--force` is required because MMM cannot ask for removal authorization. `--unattended` disables prompts but does not itself authorize deletion.

An already-absent target is a successful no-op. Force does not widen the selection, bypass exclusions, or allow an unrelated file collision.

## Removal results and retry

For each selected mod, MMM removes the mod config, lock entry and managed artifact while keeping enough consistent state to represent completed and incomplete work. If the managed jar is already absent, MMM still removes the corresponding metadata cleanly.

If a file cannot be removed, MMM does not report that mod as fully removed. It preserves enough state to identify and retry the remaining work. Independent removals that completed successfully remain completed, and rerunning the same request does not create duplicate work.

Visible unmanaged jars remain untouched. Files matched by `.mmmignore` and files ending in `.disabled` are excluded from MMM operations, including removal; `--force` does not override that protection.

Cancellation stops scheduling new removals but retains completed independent changes. MMM completes required metadata consistency before returning control, unless the operator uses the explicitly warned second interruption as an emergency exit. The final report identifies completed, failed, and unresolved items and gives the next useful action.

## Glob patterns

Patterns are case-insensitive. `*` matches any sequence of characters, `?` matches one character, and bracket expressions such as `[abc]` or `[a-z]` match a character from a set or range. Use a literal ID or name when you want the narrowest selection.

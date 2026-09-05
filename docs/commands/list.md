# `mmm list`

> This guide describes the Go-port target defined in [product intent](../intent.md), not a claim that every released build already implements it.

`mmm list` reports every declared mod and the installation evidence MMM can establish without changing configuration, lock data or files.

```bash
mmm list
```

The command has no command-specific options. Global options, missing setup and declaration correction are covered in [shared command behavior](README.md#configuration-and-setup).

## What the report distinguishes

For each declaration, the report distinguishes:

- an installed artifact whose file and integrity information match its valid lock entry;
- a missing managed file;
- missing lockfile or lock-entry evidence; and
- a content mismatch between the managed file and its recorded integrity information.

A matching filename alone does not prove that a mod is correctly installed. When the lockfile is missing, `list` reports the missing resolution evidence and does not create a lockfile or resolve new selections.

Malformed lock data, contradictory resolutions and entries without required integrity information are reported without trusting or rewriting that evidence. Conflicting duplicate declarations are likewise reported rather than resolved by inspection. Identical duplicates are not silently deduplicated by `list`.

Entries use stable, case-insensitive name ordering, with platform and project ID as tie-breakers. Platform and project ID identify a mod; its stored display name is metadata.

Visible unmanaged jars are reported after or alongside the managed results without preventing the declared list from being shown. Files matched by `.mmmignore` and files ending in `.disabled` are excluded from unmanaged reporting.

## Read-only guarantee and results

`list` does not repair files, reconcile lock entries, adopt jars, or rewrite configuration. If interactive setup or a duplicate-declaration correction is explicitly accepted, that shared flow is a separate change; `list` then resumes against the accepted setup or correction. Declining, cancelling or failing that flow leaves the original list operation unrun.

Missing, mismatched, corrupt or inconclusive evidence is shown with the next useful action, such as running [`mmm install`](install.md) for ordinary reconciliation. See the shared guidance for [execution modes](README.md#execution-modes), [result categories](README.md#results-and-retry), and [cancellation](README.md#cancellation-and-terminal-output).

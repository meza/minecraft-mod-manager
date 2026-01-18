# remove

Remove one or more mods from your configuration and (if present) delete their jar files from your mods folder.

Quick start:

```bash
mmm remove sodium
```

## Usage

`mmm remove <mods...>`

You can pass one or more mod lookups. Each lookup is matched against the lockfile first by mod ID and name.
MMM also removes any matching config entries that do not have a lock entry, even when the lookup matches the lockfile.

You can also use [glob patterns](#glob-primer) to describe multiple mods.

If a lookup does not match anything, MMM skips it and keeps going.

If the mod file is already missing on disk, MMM skips the file removal and still removes the mod from your config.

When a lock entry matches, MMM removes the config entry with the same ID and then removes the lock entry.

In interactive terminals, MMM lists the matched mods and asks you to confirm before it removes anything.
Use `--force` to skip the confirmation. In unattended or non-tty runs, MMM only removes when you pass `--force` or `--unattended`.
The confirmation prompt still appears when `--quiet` is set.

## Examples

Remove multiple mods (quote names with spaces):

```bash
mmm remove mod1 mod2 "mod with space in its name"
```

Remove a group of mods using a [glob pattern](#glob-primer):

```bash
mmm remove "world*edit*"
```

Tip: quote your patterns so your shell does not expand them before MMM sees them.


## Flags

| Short | Long        | Meaning                              | Allowed values | Example                   |
|------:|-------------|--------------------------------------|----------------|---------------------------|
|  `-f` | `--force`   | Skip the confirmation prompt         | true/false     | `mmm remove --force sodium` |


## Glob primer

```
// Patterns:
term ['/' term]*
term:
'*'         matches any sequence of non-Separator characters
'?'         matches any single non-Separator character
'[' [ '^' ] { character-range } ']'
// Character classes (must be non-empty):
c           matches character c (c != '*', '?', '\\', '[', '/')
'\\' c      matches character c
// Character-ranges:
c           matches character c (c != '\\', '-', ']')
'\\' c      matches character c
lo '-' hi   matches character c for lo <= c <= hi
```

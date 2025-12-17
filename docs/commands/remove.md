# remove

Remove one or more mods from your configuration and (if present) delete their jar files from your mods folder.

Quick start:

```bash
mmm remove sodium
```

## Usage

`mmm remove <mods...>`

You can pass one or more mod lookups. Each lookup is matched against both the configured mod ID and the mod name.
MMM checks the ID first, and then the name.

You can also use [glob patterns](#glob-primer) to describe multiple mods.

If a lookup does not match anything, MMM skips it and keeps going.

If the mod file is already missing on disk, MMM skips the file removal and still removes the mod from your config.

If you use `--dry-run`, MMM prints what it would remove without changing any files (it will not delete jars, and it will
not create a missing lock file).

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

| Short | Long        | Meaning                                               | Allowed values | Example                |
|------:|-------------|-------------------------------------------------------|----------------|------------------------|
|  `-n` | `--dry-run` | Show what would be removed without deleting anything  | true/false     | `mmm remove -n sodium` |


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

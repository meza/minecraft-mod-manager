# `prune` frame snapshots

This document specifies `prune` as state-by-state terminal frame snapshots.

All requirements in `docs/interactions/consolidation/shared.md` apply.

## User goal

- Delete unmanaged jar files from the mods folder safely.

## Success conditions

- Only unmanaged jar files are deleted.
- User intent is confirmed unless `--force` is used.
- Default behavior is non-destructive.

## State model

States:
- PRUNE-01: config present check (may delegate to init)
- PRUNE-02: scan for unmanaged jars
- PRUNE-03: show unmanaged list
- PRUNE-04: confirmation prompt (unless --force)
- PRUNE-05: delete files
- PRUNE-06: success
- PRUNE-NOOP: no unmanaged files
- PRUNE-CANCEL: user cancels (`ctrl+c`)
- PRUNE-ERR: failure

## Frame snapshots

### PRUNE-01 Missing config gate (delegates to init)

#### Command used
`prune`

```
No configuration found.
MMM looked for a config file at: ./modlist.json
This command needs config to continue.
? Initialize now? (y/N):

enter accept • ctrl+c/esc quit
```

If the user answers Yes, MMM MUST run the init flow exactly as specified in:
- `docs/interactions/consolidation/init.md`

After init success, MMM resumes prune at PRUNE-02.

### PRUNE-NOOP No unmanaged files

#### Command used
`prune`

```
No unmanaged files found.
```

### PRUNE-03 Unmanaged list and confirmation prompt

#### Command used
`prune`

```
Unmanaged files:
- mods/unmanaged.jar
- mods/old-mod.jar
? Delete these files? (y/N):

enter accept • ctrl+c/esc quit
```

### PRUNE-04 Decline deletion

#### Command used
`prune`

```
Unmanaged files:
- mods/unmanaged.jar
? Delete these files? (y/N): n
```

### PRUNE-05 Confirm deletion and success

#### Command used
`prune`

```
Unmanaged files:
- mods/unmanaged.jar
? Delete these files? (y/N): y
Deleted 1 files.
```

### PRUNE-CANCEL Cancel at confirmation

#### Command used
`prune`

```
Unmanaged files:
- mods/unmanaged.jar
? Delete these files? (y/N):

enter accept • ctrl+c/esc quit
^C
Cancelled.
```

## Non-interactive behavior

Non-interactive mode MUST NOT prompt.

### Non-interactive missing config

#### Command used
`--non-interactive prune`

```
Error: no configuration file found at ./modlist.json.
Next: run `mmm init` to create one.
```

### Non-interactive no unmanaged files

#### Command used
`--non-interactive prune`

```
No unmanaged files found.
```

### Non-interactive without `--force` (safe no-op)

#### Command used
`--non-interactive prune`

```
Unmanaged files:
- mods/unmanaged.jar
Unmanaged files detected. Run `mmm prune --force` to delete them.
```

### Non-interactive with `--force`

#### Command used
`--non-interactive prune --force`

```
Unmanaged files:
- mods/unmanaged.jar
Deleted 1 files.
```

## Quiet flag

### `--quiet` success (`--force`)

#### Command used
`prune --quiet --force`

```
Output: none
Exit code: 0
```

### `--quiet` unmanaged exist (without `--force`)

#### Command used
`prune --quiet`

```
- mods/unmanaged.jar
```

### `--quiet` no unmanaged (without `--force`)

#### Command used
`prune --quiet`

```
Output: none
Exit code: 0
```

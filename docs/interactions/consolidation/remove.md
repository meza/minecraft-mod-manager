# `remove` frame snapshots

This document specifies `remove` as terminal frame snapshots.

All requirements in `docs/interactions/consolidation/shared.md` apply.

## User goal

- Remove mods from config and delete their jars when present.

## Success conditions

- Config and lock updated.
- Matching jars deleted when present.
- Lookups that match nothing are safe and do not fail the run.

## State model

States:
- REMOVE-01: config present check (may delegate to init)
- REMOVE-02: argument capture (one or more lookups)
- REMOVE-03: resolve lookups against config
- REMOVE-04: apply removals (edit config, edit lock, delete jars when present)
- REMOVE-05: success
- REMOVE-DRY-RUN: dry run output
- REMOVE-ERR: failure

## Frame snapshots

### REMOVE-01 Missing config gate (delegates to init)

#### Command used
`remove sodium`

```
No configuration found.
MMM looked for a config file at: ./modlist.json
This command needs config to continue.
? Initialize now? (y/N):

enter accept • ctrl+c/esc quit
```

If the user answers Yes, MMM MUST run the init flow exactly as specified in:
- `docs/interactions/consolidation/init.md`

After init success, MMM resumes remove at REMOVE-02.

### REMOVE-05 Success

#### Command used
`remove inventory-sorting soundsbegone`

```
Removed Inventory Sorting
Removed SoundsBeGone
```

### REMOVE-05 Lookup not found (non-fatal)

#### Command used
`remove not-a-mod`

```
No matching mods for: not-a-mod
```

### REMOVE-05 Jar missing (non-fatal)

#### Command used
`remove inventory-sorting`

```
Removed Inventory Sorting
Note: mods/inventorysorter.jar was not found, skipping file deletion.
```

### REMOVE-DRY-RUN Dry run

#### Command used
`remove --dry-run inventory-sorting`

```
Would remove Inventory Sorting
Would delete mods/inventorysorter.jar
```

## Quiet flag

### `--quiet` success

#### Command used
`remove --quiet inventory-sorting`

```
Output: none
Exit code: 0
```

`--quiet` failure:
- Errors still print.

## Non-interactive behavior

Non-interactive mode MUST NOT prompt.

### Non-interactive missing config

#### Command used
`--non-interactive remove inventory-sorting`

```
Error: no configuration file found at ./modlist.json.
Next: run `mmm init` to create one.
```

### Non-interactive success

#### Command used
`--non-interactive remove inventory-sorting`

```
Removed Inventory Sorting
```

### Non-interactive dry run

#### Command used
`--non-interactive remove --dry-run inventory-sorting`

```
Would remove Inventory Sorting
Would delete mods/inventorysorter.jar
```

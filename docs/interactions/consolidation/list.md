# `list` frame snapshots

This document specifies `list` as terminal frame snapshots.

All requirements in `docs/interactions/consolidation/shared.md` apply.

## User goal

- See what is configured and whether it is installed correctly.

## Success conditions

- User sees each configured mod and its status.
- Output is readable without color.

## State model

States:
- LIST-01: config present check (may delegate to init)
- LIST-02: render list
- LIST-03: success
- LIST-ERR: failure

## Frame snapshots

### LIST-01 Missing config gate (delegates to init)

#### Command used
`list`

```
No configuration found.
MMM looked for a config file at: ./modlist.json
This command needs config to continue.
? Initialize now? (y/N):

enter accept • ctrl+c/esc quit
```

If the user answers Yes, MMM MUST run the init flow exactly as specified in:
- `docs/interactions/consolidation/init.md`

After init success, MMM resumes list at LIST-02.

### LIST-02 Render list

#### Command used
`list`

```
V Inventory Sorting
X SoundsBeGone (hash mismatch, run `mmm install`)
```

### LIST-02 Empty list

#### Command used
`list`

```
No mods configured.
```

## Quiet flag

`--quiet` does not meaningfully apply to list.

### `--quiet` list output

#### Command used
`list --quiet`

```
V Inventory Sorting
```

## Non-interactive behavior

Non-interactive mode MUST NOT prompt.

### Non-interactive success

#### Command used
`--non-interactive list`

```
V Inventory Sorting
```

### Non-interactive missing config

#### Command used
`--non-interactive list`

```
Error: no configuration file found at ./modlist.json.
Next: run `mmm init` to create one.
```

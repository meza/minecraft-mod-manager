# `change` frame snapshots

This document specifies `change` as state-by-state terminal frame snapshots.

All requirements in `docs/interactions/consolidation/shared.md` apply.

## User goal

- Switch the configured Minecraft version and reinstall mods for that version.

## Success conditions

- Config is updated to the new game version.
- Mods folder contains compatible mod jars for that version.
- Unsupported mods are handled predictably, especially under `--force`.

## State model

States:
- CHANGE-01: config present check (may delegate to init)
- CHANGE-02: target version capture
- CHANGE-NOOP: target equals current version
- CHANGE-03: compatibility check (skipped under --force)
- CHANGE-04: remove installed jars
- CHANGE-05: install for target
- CHANGE-06: write config and lock
- CHANGE-07: success
- CHANGE-SKIP: forced skipped mods list (actionable)
- CHANGE-ERR: failure
- CHANGE-CANCEL: user cancels (`ctrl+c`)

## Frame snapshots

### CHANGE-01 Missing config gate (delegates to init)

#### Command used
`change 1.21.11`

```
No configuration found.
MMM looked for a config file at: ./modlist.json
This command needs config to continue.
? Initialize now? (y/N):

enter accept • ctrl+c/esc quit
```

If the user answers Yes, MMM MUST run the init flow exactly as specified in:
- `docs/interactions/consolidation/init.md`

After init success, MMM resumes change at CHANGE-02.

### CHANGE-NOOP Same version

#### Command used
`change 1.21.11`

```
The target version 1.21.11 is the same as your current version.
```

### CHANGE-07 Success (interactive tui-lite wrapper allowed)

#### Command used
`change 1.21.11`

```
Changing Minecraft version to 1.21.11...
- Testing compatibility
- Removing installed jars
- Installing for 1.21.11
- Writing config and lock
Version changed. Now targeting 1.21.11.
```

### CHANGE-ERR Blocked by unsupported mods (no --force)

#### Command used
`change 1.21.11`

```
Changing Minecraft version to 1.21.11...
- Testing compatibility
Change failed: some mods do not support 1.21.11.
Blocking mods:
- <mod 1>
- <mod 2>
Next: run `mmm test 1.21.11`, remove blockers, or use `mmm change --force 1.21.11` if you accept missing mods.
```

### CHANGE-SKIP Force mode proceeds and reports skipped mods

#### Command used
`change --force 1.21.11`

```
Changing Minecraft version to 1.21.11...
- Testing compatibility (skipped, --force)
- Removing installed jars
- Installing for 1.21.11
Skipped unsupported mods:
- <mod 1>
- <mod 2>
Version changed. Now targeting 1.21.11.
```

### CHANGE-CANCEL Cancel during run

#### Command used
`change 1.21.11`

```
Changing Minecraft version to 1.21.11...
^C
Cancelled.
```

## Non-interactive behavior

Non-interactive mode MUST NOT prompt.

### Non-interactive missing config

#### Command used
`--non-interactive change 1.21.11`

```
Error: no configuration file found at ./modlist.json.
Next: run `mmm init` to create one.
```

### Non-interactive no-op

#### Command used
`--non-interactive change 1.21.11`

```
The target version 1.21.11 is the same as your current version.
```

### Non-interactive success

#### Command used
`--non-interactive change 1.21.11`

```
Changing Minecraft version to 1.21.11...
Version changed. Now targeting 1.21.11.
```

### Non-interactive blocked (no --force)

#### Command used
`--non-interactive change 1.21.11`

```
Change failed: some mods do not support 1.21.11.
Blocking mods:
- <mod 1>
- <mod 2>
```

### Non-interactive force success with skipped mods

#### Command used
`--non-interactive change --force 1.21.11`

```
Version changed. Now targeting 1.21.11.
Skipped unsupported mods:
- <mod 1>
- <mod 2>
```

## Quiet flag

### `--quiet` success

#### Command used
`change --quiet 1.21.11`

```
Output: none
Exit code: 0
```

### `--quiet` force with skipped mods (actionable)

#### Command used
`change --quiet --force 1.21.11`

```
Skipped unsupported mods:
- <mod 1>
- <mod 2>
```

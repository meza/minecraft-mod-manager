# `install` frame snapshots

This document specifies `install` as state-by-state terminal frame snapshots.

All requirements in `docs/interactions/consolidation/shared.md` apply.

## User goal

- Make the mods folder match the config and lock file.

## Success conditions

- Every configured mod is installed and matches the lock file.
- Lock file reflects the exact installed files.
- If unmanaged jar files exist, MMM proceeds and then reports them with remediation.

## State model

States:
- INSTALL-01: config present check (may delegate to init)
- INSTALL-02: install running (progress)
- INSTALL-NOTICE-UNMANAGED: unmanaged files notice
- INSTALL-03: success
- INSTALL-ERR: failure
- INSTALL-CANCEL: user cancels (`ctrl+c`)

## Frame snapshots

### INSTALL-01 Missing config gate (delegates to init)

#### Command used
`install`

```
No configuration found.
MMM looked for a config file at: ./modlist.json
This command needs config to continue.
? Initialize now? (y/N):

enter accept • ctrl+c/esc quit
```

If the user answers Yes, MMM MUST run the init flow exactly as specified in:
- `docs/interactions/consolidation/init.md`

After init success, MMM resumes install at INSTALL-02.

### INSTALL-02 Running (interactive tui-lite wrapper allowed)

#### Command used
`install`

```
Installing mods...
- Loading configuration
- Checking mods folder
- Downloading missing mods
- Verifying hashes
- Writing lock file
Install complete.
```

### INSTALL-NOTICE-UNMANAGED Unmanaged files detected after install

#### Command used
`install`

```
Installing mods...
Install complete.
Unmanaged files detected:
MMM found jar files in your mods folder that are not in your lock file.
- mods/unmanaged.jar
Run `mmm scan` to adopt or resolve these files.
```

### INSTALL-ERR Failure

#### Command used
`install`

```
Installing mods...
Install failed: <short reason>
Next: <one actionable next step>
```

### INSTALL-CANCEL Cancel during run

#### Command used
`install`

```
Installing mods...
^C
Cancelled.
```

## Non-interactive behavior

Non-interactive mode MUST NOT prompt.

### Non-interactive missing config

#### Command used
`--non-interactive install`

```
Error: no configuration file found at ./modlist.json.
Next: run `mmm init` to create one.
```

### Non-interactive success

#### Command used
`--non-interactive install`

```
Installing mods...
Install complete.
```

### Non-interactive unmanaged

#### Command used
`--non-interactive install`

```
Install complete.
Unmanaged files detected:
- mods/unmanaged.jar
Run `mmm scan` to adopt or resolve these files.
```

### Non-interactive failure

#### Command used
`--non-interactive install`

```
Install failed: <short reason>
Next: <one actionable next step>
```

## Quiet flag

### `--quiet` success (no unmanaged)

#### Command used
`install --quiet`

```
Output: none
Exit code: 0
```

### `--quiet` unmanaged

#### Command used
`install --quiet`

```
Unmanaged files detected:
- mods/unmanaged.jar
Run `mmm scan` to adopt or resolve these files.
```

# `update` frame snapshots

This document specifies `update` as state-by-state terminal frame snapshots.

All requirements in `docs/interactions/consolidation/shared.md` apply.

## User goal

- Upgrade configured mods to newer compatible releases.

## Success conditions

- Mods are updated when newer compatible releases exist.
- Lock file reflects the installed files.
- If unmanaged jar files exist, MMM proceeds and then reports them with remediation.

## State model

States:
- UPDATE-01: config present check (may delegate to init)
- UPDATE-02: baseline install phase
- UPDATE-03: check for updates
- UPDATE-04: apply updates
- UPDATE-NOTICE-UNMANAGED: unmanaged files notice
- UPDATE-05: success
- UPDATE-ERR: failure
- UPDATE-CANCEL: user cancels (`ctrl+c`)

## Frame snapshots

### UPDATE-01 Missing config gate (delegates to init)

#### Command used
`update`

```
No configuration found.
MMM looked for a config file at: ./modlist.json
This command needs config to continue.
? Initialize now? (y/N):

enter accept • ctrl+c/esc quit
```

If the user answers Yes, MMM MUST run the init flow exactly as specified in:
- `docs/interactions/consolidation/init.md`

After init success, MMM resumes update at UPDATE-02.

### UPDATE-02 to UPDATE-05 Success (interactive tui-lite wrapper allowed)

#### Command used
`update`

```
Updating mods...
- Installing baseline
- Checking for updates
- Downloading updates
- Writing lock file
Update complete.
```

### UPDATE-NOTICE-UNMANAGED Unmanaged files detected after update

#### Command used
`update`

```
Updating mods...
Update complete.
Unmanaged files detected:
MMM found jar files in your mods folder that are not in your lock file.
- mods/unmanaged.jar
Run `mmm scan` to adopt or resolve these files.
```

### UPDATE-ERR Failure

#### Command used
`update`

```
Updating mods...
Update failed: <short reason>
Next: <one actionable next step>
```

### UPDATE-CANCEL Cancel during run

#### Command used
`update`

```
Updating mods...
^C
Cancelled.
```

## Non-interactive behavior

Non-interactive mode MUST NOT prompt.

### Non-interactive missing config

#### Command used
`--non-interactive update`

```
Error: no configuration file found at ./modlist.json.
Next: run `mmm init` to create one.
```

### Non-interactive success

#### Command used
`--non-interactive update`

```
Updating mods...
Update complete.
```

### Non-interactive failure

#### Command used
`--non-interactive update`

```
Update failed: <short reason>
Next: <one actionable next step>
```

## Quiet flag

### `--quiet` success (no unmanaged)

#### Command used
`update --quiet`

```
Output: none
Exit code: 0
```

### `--quiet` unmanaged

#### Command used
`update --quiet`

```
Unmanaged files detected:
- mods/unmanaged.jar
Run `mmm scan` to adopt or resolve these files.
```

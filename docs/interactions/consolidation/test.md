# `test` frame snapshots

This document specifies `test` as state-by-state terminal frame snapshots.

All requirements in `docs/interactions/consolidation/shared.md` apply.

## User goal

- Decide whether a target Minecraft version is safe before changing anything.

## Success conditions

- User learns whether all configured mods support the target version.
- No files are changed.
- Exit codes support automation.

## State model

States:
- TEST-01: config present check (may delegate to init)
- TEST-02: target version resolution (explicit or latest)
- TEST-03: querying mods
- TEST-04: success
- TEST-05: failure (blocking mods)
- TEST-NOOP: target equals current configured version
- TEST-ERR-LATEST-OFFLINE: latest cannot be resolved

## Frame snapshots

### TEST-01 Missing config gate (delegates to init)

#### Command used
`test 1.21.11`

```
No configuration found.
MMM looked for a config file at: ./modlist.json
This command needs config to continue.
? Initialize now? (y/N):

enter accept • ctrl+c/esc quit
```

If the user answers Yes, MMM MUST run the init flow exactly as specified in:
- `docs/interactions/consolidation/init.md`

After init success, MMM resumes test at TEST-02.

### TEST-04 Success

#### Command used
`test 1.21.11`

```
Testing compatibility for Minecraft 1.21.11...
All mods support 1.21.11.
Next: run `mmm change 1.21.11` when ready.
```

### TEST-05 Failure (blocking mods)

#### Command used
`test 1.21.11`

```
Testing compatibility for Minecraft 1.21.11...
Some mods do not support 1.21.11:
- <mod 1>
- <mod 2>
Next: wait for updates, remove blockers, or run `mmm change --force 1.21.11` if you accept missing mods.
```

### TEST-NOOP Target equals current version

#### Command used
`test 1.21.11`

```
The target version 1.21.11 is the same as your current version.
```

### TEST-ERR-LATEST-OFFLINE latest cannot be resolved

#### Command used
`test latest`

```
Error: could not resolve latest Minecraft version.
Next: run `mmm test <version>` with an explicit version.
```

## Non-interactive behavior

Non-interactive mode MUST NOT prompt.

Exit codes:
- 0: all mods support the target
- 1: one or more mods lack support
- 2: target equals current version

### Non-interactive missing config

#### Command used
`--non-interactive test 1.21.11`

```
Error: no configuration file found at ./modlist.json.
Next: run `mmm init` to create one.
```

### Non-interactive success

#### Command used
`--non-interactive test 1.21.11`

```
Testing compatibility for Minecraft 1.21.11...
All mods support 1.21.11.
```

### Non-interactive failure

#### Command used
`--non-interactive test 1.21.11`

```
Testing compatibility for Minecraft 1.21.11...
Some mods do not support 1.21.11:
- <mod 1>
- <mod 2>
```

### Non-interactive no-op

#### Command used
`--non-interactive test 1.21.11`

```
The target version 1.21.11 is the same as your current version.
```

## Quiet flag

### `--quiet` success

#### Command used
`test --quiet 1.21.11`

```
Output: none
Exit code: 0
```

### `--quiet` failure

#### Command used
`test --quiet 1.21.11`

```
- <mod 1>
- <mod 2>
```

# `scan` frame snapshots

This document specifies `scan` as state-by-state terminal frame snapshots.

All requirements in `docs/interactions/consolidation/shared.md` apply.

## User goal

- Identify jar files in the mods folder that MMM does not manage.

## Success conditions

- User sees which files are recognized, unknown, or unsure.
- User can adopt recognized unmanaged files into config and lock.
- Unknown and unsure files are clearly explained.

## State model

States:
- SCAN-01: config present check (may delegate to init)
- SCAN-02: scan running
- SCAN-03: results rendered (recognized, unknown, unsure)
- SCAN-04: adoption prompt (eligible only)
- SCAN-05: write updates (when confirmed or --add)
- SCAN-NOT-ELIGIBLE: unsure results block adoption
- SCAN-CANCEL: user cancels (`ctrl+c`)
- SCAN-ERR: failure

## Frame snapshots

### SCAN-01 Missing config gate (delegates to init)

#### Command used
`scan`

```
No configuration found.
MMM looked for a config file at: ./modlist.json
This command needs config to continue.
? Initialize now? (y/N):

enter accept • ctrl+c/esc quit
```

If the user answers Yes, MMM MUST run the init flow exactly as specified in:
- `docs/interactions/consolidation/init.md`

After init success, MMM resumes scan at SCAN-02.

### SCAN-02 Scan running

#### Command used
`scan`

```
Scanning mods folder...
```

### SCAN-03 Results rendered

#### Command used
`scan`

```
Scanning mods folder...
Scan results:

Recognized:
- inventorysorter.jar -> Inventory Sorting (modrinth inventory-sorting)

Unknown:
- unmanaged.jar (no match on modrinth or curseforge)

Unsure:
- flaky.jar (platform lookup failed)
```

## Adoption frames

### SCAN-04 Adoption prompt (eligible: recognized present, unsure empty)

#### Command used
`scan`

```
Scanning mods folder...
Scan results:

Recognized:
- inventorysorter.jar -> Inventory Sorting (modrinth inventory-sorting)

? Add recognized mods to your configuration? (y/N):

enter accept • ctrl+c/esc quit
```

### SCAN-05 Adoption accepted -> configuration updated

#### Command used
`scan`

```
Scanning mods folder...
Scan results:

Recognized:
- inventorysorter.jar -> Inventory Sorting (modrinth inventory-sorting)

? Add recognized mods to your configuration? (y/N): y
Configuration updated.
```

### SCAN-04 Adoption declined -> exit with no side effects

#### Command used
`scan`

```
Scanning mods folder...
Scan results:

Recognized:
- inventorysorter.jar -> Inventory Sorting (modrinth inventory-sorting)

? Add recognized mods to your configuration? (y/N): n
```

### SCAN-NOT-ELIGIBLE Unsure results block adoption

#### Command used
`scan`

```
Scanning mods folder...
Scan results:

Recognized:
- inventorysorter.jar -> Inventory Sorting (modrinth inventory-sorting)

Unsure:
- flaky.jar (platform lookup failed)

Some files could not be verified due to platform errors. Fix the errors and run scan again.
```

## Cancellation frames

### SCAN-CANCEL Cancel during scan

#### Command used
`scan`

```
Scanning mods folder...
^C
Cancelled.
```

### SCAN-ERR Failure

#### Command used
`scan`

```
Scan failed: <short reason>
Next: <one actionable next step>
```

## Non-interactive behavior

Non-interactive mode MUST NOT prompt.

### Non-interactive missing config

#### Command used
`--non-interactive scan`

```
Error: no configuration file found at ./modlist.json.
Next: run `mmm init` to create one.
```

### Non-interactive: no writes without `--add`

#### Command used
`--non-interactive scan`

```
Scan results:
Unknown:
- mods/unmanaged.jar
Next: run `mmm scan --add` to adopt recognized results.
```

### Non-interactive: `--add` eligible writes

#### Command used
`--non-interactive scan --add`

```
Scan results:
Recognized:
- inventorysorter.jar -> Inventory Sorting (modrinth inventory-sorting)
Configuration updated.
```

## Quiet flag

With `--quiet` set, MMM prints only actionable results.

### `--quiet`: unknown files only

#### Command used
`scan --quiet`

```
Unknown files:
- mods/unmanaged.jar
```

### `--quiet`: no unknown or unsure

#### Command used
`scan --quiet`

```
Output: none
Exit code: 0
```

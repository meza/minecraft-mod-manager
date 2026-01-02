# Flow: scan

All requirements in `docs/interactions/interaction-guidelines.md` apply.

## User contract

### User goal and success conditions

You want to identify jar files in your mods folder that are not managed by MMM, and optionally add recognized results to your config and lock.

Success looks like (for the user):
- You see which files are unmanaged
- You can add recognized results to config and lock without guessing

### Entry points

- Command: `mmm scan [-p <platform>] [--add]`
- User guide: `docs/commands/scan.md`
- Behavior spec: `docs/specs/scan.md`

### Primary flow

1. You run `mmm scan`.
2. MMM lists candidate jar files and filters out ignored and managed files, applying `.mmmignore` rules.
3. MMM tries to identify each candidate by checking the preferred platform first (modrinth by default).
4. MMM prints recognized files, unknown files, and unsure files.

### Alternate and error flows

- If no config is found, MMM follows the missing-config gate as defined in the [guidelines](../interaction-guidelines.md#missing-config).
- If recognized files exist and prompts are allowed, MMM asks whether to add the recognized results to config and lock.
- Unknown and unsure files are informational only and do not block adopting recognized results.
- When `--add` is set, MMM writes recognized results without prompting.
- If scan cannot read the mods folder, MMM prints an actionable error and exits non-zero.

---

## `scan` frame snapshots

This document specifies `scan` as state-by-state terminal frame snapshots.

### State model

States:
- SCAN-01: running (scanning and identifying)
- SCAN-02: results (no adoption)
- SCAN-03: adoption prompt
- SCAN-04: adoption written
- SCAN-ERR: failure

### Frame snapshots

#### SCAN-01 Running (tty)

##### Command used
`scan`

In tty mode, MMM shows all candidate files at the same time.
As file status changes, MMM updates the icons in place.

```
Scanning mods folder:
⏳ inventorysorter.jar
⏳ unmanaged.jar
⏳ flaky.jar
... (one row per file, all candidates shown)
```

#### SCAN-02 Results rendered (tty and non-tty)

##### Command used
`scan`

```
Scan results:

Recognized:
✅ inventorysorter.jar -> Inventory Sorting (inventory-sorting) [modrinth]

Unknown: (no match on modrinth or curseforge)
❌ unmanaged.jar

Unsure: (platform lookup failed)
❔ flaky.jar
```

#### SCAN-03 Adoption prompt (tty)

##### Command used
`scan`

```
Scan results:

Recognized:
✅ inventorysorter.jar -> Inventory Sorting (inventory-sorting) [modrinth]

Unknown: (no match on modrinth or curseforge)
❌ unmanaged-A.jar
❌ unmanaged-B.jar

Unsure: (platform lookup failed)
❔ flaky.jar

? Add recognized mods to your configuration? (<yesShort>/<noShort>) [default: <noShort>]:

enter accept • ctrl+c/esc quit
```

#### SCAN-04 Adoption written (tty)

##### Command used
`scan`

```
Scan results:

Unknown: (no match on modrinth or curseforge)
❌ unmanaged-A.jar
❌ unmanaged-B.jar

Unsure: (platform lookup failed)
❔ flaky.jar

? Add recognized mods to your configuration? (<yesShort>/<noShort>) [default: <noShort>]: <yesShort>

Added:
✅ Inventory Sorting (inventory-sorting) [modrinth]
```

### Error frames

#### SCAN-ERR Failure (tty and non-tty)

##### Command used
`scan`

```
‼️ Scan failed: <reason>
```

### Unattended behavior

Unattended mode MUST NOT prompt.
This applies when `--unattended` is set.

For `scan`, `--unattended` differs from tty behavior only when a prompt would be shown.

#### Unattended results (no --add)

##### Command used
`--unattended scan`

```
Same as SCAN-02.
```

#### Unattended add recognized results (--add)

##### Command used
`--unattended scan --add`

```
Scan results:

Unknown: (no match on modrinth or curseforge)
❌ unmanaged-A.jar
❌ unmanaged-B.jar

Unsure: (platform lookup failed)
❔ flaky.jar

Added:
✅ Inventory Sorting (inventory-sorting) [modrinth]
```

### Non-interactive (non-tty) behavior

This applies when stdin or stdout is not a TTY.

In non-tty mode:
- MMM MUST NOT prompt.
- MMM MUST NOT emit terminal control sequences.
- Output is rendered as a plain transcript.

Behavior matches unattended mode for adoption.

### Quiet flag

With `--quiet` set, MMM prints only actionable results.

#### `--quiet` unknown and unsure only

##### Command used
`scan --quiet`

```
Unknown: (no match on modrinth or curseforge)
❌ unmanaged-A.jar
❌ unmanaged-B.jar

Unsure: (platform lookup failed)
❔ flaky.jar
```

#### `--quiet --add` wrote recognized results

##### Command used
`scan --quiet --add`

```
Output: none
Exit code: 0
```

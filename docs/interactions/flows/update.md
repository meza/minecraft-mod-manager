# Flow: update

All requirements in `docs/interactions/interaction-guidelines.md` apply.

## User contract

### User goal and success conditions

You want to update your configured mods to newer compatible versions without re-adding everything.

Success looks like (for the user):
- Mods that have updates available are updated on disk
- `modlist-lock.json` is updated deterministically to match what is installed
- Pinned mods are not updated
- Failures are clearly reported and do not stop other mods from updating

### Entry points

- Command: `mmm update`
- User guide: `docs/commands/update.md`
- Behavior spec: `docs/specs/update.md`

### Primary flow

1. You run `mmm update`.
2. MMM runs the `install` procedure first to ensure your current setup is consistent.
3. MMM checks each configured mod for newer compatible releases.
   - MMM respects the configured Minecraft version, loader, release types, platform, and pinning rules.
4. MMM updates mods that have a newer compatible version available.
5. MMM continues updating other mods even if a specific mod cannot be updated.
6. MMM prints results grouped by outcome:
   - Already up to date mods
   - Updated mods
   - Skipped mods (pinned)
   - Failed mods (with reasons)
7. MMM exits with a non-zero code if any mod failed to update.

### Alternate and error flows

- If no config is found, MMM follows the missing-config gate as defined in the [guidelines](../interaction-guidelines.md#missing-config).
- If unmanaged jars are detected before checking for configured mods, MMM prints the unmanaged files notice and exits with code 1 (even when no mods are configured).
- If there are no configured mods and no unmanaged jars, MMM exits successfully and reports `No mods configured.`.
- If no updates are available, MMM exits successfully and reports results with only the "Already up to date" segment (and "Skipped" if applicable).
- If unmanaged jars are detected during the install prerequisite when mods are configured, MMM prints the unmanaged files notice and exits with code 1.
- If the `install` prerequisite fails, MMM exits non-zero with an actionable error.
- If writing lock updates fails, MMM exits non-zero with an actionable error.
- If writing config updates fails, MMM exits non-zero with an actionable error.

---

## `update` frame snapshots

This document specifies `update` as state-by-state terminal frame snapshots.

### State model

States:
- UPDATE-01: running (install prerequisite, then update checks and downloads)
- UPDATE-01B: running (mixed statuses)
- UPDATE-02: success (updates applied or nothing to do)
- UPDATE-03: partial success (some updates failed)
- UPDATE-NOOP-NO-MODS: no mods configured
- UPDATE-NOOP-NO-UPDATES: no updates available
- UPDATE-ERR-INSTALL: install prerequisite failed
- UPDATE-ERR-WRITE-LOCK: failed to write lock updates
- UPDATE-ERR-WRITE-CONFIG: failed to write config updates

### Frame snapshots

When reusing the [install](./install.md) flow, `update` reuses the same frames as install for the prerequisite step both in tty and non-tty modes.
This includes the final frame renders for non-tty.

#### UPDATE-01 Running (tty)

In tty mode, MMM shows all mods at the same time, grouped into segments.
As mod status changes, MMM updates the icons in place.
If a segment has no mods, MMM does not render that segment.

##### Command used
`update`

##### UPDATE-01-1 Installing prerequisite

First, MMM runs the install prerequisite, reusing the install flow output:

```
Installing mods:
⏳ Inventory Sorting (inventory-sorting) [modrinth]
⬇️ Fabric API (fabric-api) [modrinth]
█████░░░░░
50% (512 KB / 1 MB)
⬇️ Mod Menu (modmenu) [modrinth]
██░░░░░░░░
20% (200 KB / 1 MB)
⏳ Lithium (lithium) [modrinth]
... (one row per mod, all mods shown)

Updating (⠋ waiting for install to complete)
```

In non-tty mode, MMM prints a header before the install transcript:

```
Updating your mods to their newest versions:
Installing potentially missing mods:
```

##### UPDATE-01-2 Updating mods (tty)

UPDATE-01-1 runs to completion, then the view transitions to updating mods:

```
Updating your mods to their newest versions:
⠋ Inventory Sorting (inventory-sorting) [modrinth]
⠋ Fabric API (fabric-api) [modrinth]
⬇️ Mod Menu (modmenu) [modrinth]
██░░░░░░░░
20% (200 KB / 1 MB)
⏳ Some Mod (some-mod) [curseforge]
... (one row per mod, all mods shown)
```

In tty mode, MMM MAY update this view in place.

#### UPDATE-01B Running (tty, mid-run mixed statuses)

As progress is made, the same view updates to reflect the final segments the mods will be sorted into.

##### Command used
`update`

```
Updating your mods to their newest versions:
Already up to date:
✅ Fabric API (fabric-api) [modrinth]
... (one row per up to date mod)

Updated mods:
✅ Inventory Sorting (inventory-sorting) [modrinth]
... (one row per updated mod)

Skipped:
📌 Pinned Mod (pinned-mod) [curseforge] pinned (skipped)
... (one row per skipped mod)

Failed:
❌ Mod Menu (modmenu) [modrinth] update failed: <reason>
... (one row per failed mod)

Updating:
⠋ Some Mod (some-mod) [curseforge]
... (one row per in-progress mod)
```

#### UPDATE-02 Success (tty and non-tty)

##### Command used
`update`

```
Updating your mods to their newest versions:
Already up to date:
✅ Fabric API (fabric-api) [modrinth]
✅ Mod Menu (modmenu) [modrinth]
... (one row per up to date mod)

Updated mods:
✅ Inventory Sorting (inventory-sorting) [modrinth]
... (one row per updated mod)

Skipped:
📌 Pinned Mod (pinned-mod) [curseforge] pinned (skipped)
... (one row per skipped mod)

✅ Update complete.
```

Exit code: 0

If there are no configured mods, MMM SHOULD print:

```
Updating your mods to their newest versions:
No mods configured.
```

Exit code: 0

#### UPDATE-NOOP-NO-MODS No mods configured (tty and non-tty)

##### Command used
`update`

```
Updating your mods to their newest versions:
No mods configured.
```

Exit code: 0

#### UPDATE-NOOP-NO-UPDATES No updates available (tty)

##### Command used
`update`

```
Updating your mods to their newest versions:
Already up to date:
✅ Fabric API (fabric-api) [modrinth]
✅ Mod Menu (modmenu) [modrinth]
... (one row per up to date mod)

Skipped:
📌 Pinned Mod (pinned-mod) [curseforge] pinned (skipped)
... (one row per skipped mod)

✅ Update complete.
```

Exit code: 0

#### UPDATE-NOOP-NO-UPDATES No updates available (non-tty)

##### Command used
`update`

```
Updating your mods to their newest versions:
✅ Fabric API (fabric-api) [modrinth]
✅ Mod Menu (modmenu) [modrinth]
📌 Pinned Mod (pinned-mod) [curseforge] pinned (skipped)
... (one row per skipped mod)

✅ Update complete.
```

Exit code: 0

#### UPDATE-03 Partial success (tty)

##### Command used
`update`

```
Updating your mods to their newest versions:
Already up to date:
✅ Fabric API (fabric-api) [modrinth]
... (one row per up to date mod)

Updated mods:
✅ Inventory Sorting (inventory-sorting) [modrinth]
... (one row per updated mod)

Skipped:
📌 Pinned Mod (pinned-mod) [curseforge] pinned (skipped)
... (one row per skipped mod)

Failed:
❌ Mod Menu (modmenu) [modrinth] update failed: <reason>
... (one row per failed mod)

‼️ Update incomplete.

Fix the reasons and rerun `mmm update`.
```

Exit code: 1

#### UPDATE-03 Partial success (non-tty)

##### Command used
`update`

```
Updating your mods to their newest versions:
✅ Fabric API (fabric-api) [modrinth]
✅ Inventory Sorting (inventory-sorting) [modrinth]
📌 Pinned Mod (pinned-mod) [curseforge] pinned (skipped)
❌ Mod Menu (modmenu) [modrinth] update failed: <reason>

‼️ Update incomplete.

Fix the reasons and rerun `mmm update`.
```

Exit code: 1

### Error frames

#### UPDATE-ERR-INSTALL install prerequisite failed

##### Command used
`update`

When the install prerequisite fails, MMM prints the install failure transcript first, then the update error below.

```
‼️ Install prerequisite failed. Update aborted.

Fix the reason and rerun `mmm update`.
```

Exit code: 1

#### UPDATE-ERR-WRITE-LOCK failed to write lock updates

##### Command used
`update`

```
‼️ Could not write lock file at ./modlist-lock.json.

Fix the file permissions and rerun `mmm update`.
```

Exit code: 1

#### UPDATE-ERR-WRITE-CONFIG failed to write config updates

##### Command used
`update`

```
‼️ Could not write config file at ./modlist.json.

Fix the file permissions and rerun `mmm update`.
```

Exit code: 1

### Unattended behavior

Unattended mode MUST NOT prompt.
This applies when `--unattended` is set.

For `update`, `--unattended` output is identical to tty output; missing-config prompts are only shown in interactive mode and fail fast in unattended.
Message shapes and exit codes match the tty frames above.

### Non-interactive (non-tty) behavior

This applies when stdin or stdout is not a TTY.

In non-tty mode:
- MMM MUST NOT prompt.
- MMM MUST NOT emit terminal control sequences.
- Output is rendered as a plain transcript.

### Quiet flag

With `--quiet`, MMM prints only actionable results.
On success, MMM is silent.

Per the guidelines, `update --quiet` is silent unless there are errors.

`--quiet` MUST NOT change error message shapes. Failures print the same errors and exit non-zero.

#### `--quiet` success

##### Command used
`update --quiet`

```
Output: none
Exit code: 0
```

#### `--quiet` partial success

##### Command used
`update --quiet`

```
Updating your mods to their newest versions:
‼️ Update incomplete.

Failed:
❌ Mod Menu (modmenu) [modrinth] update failed: <reason>
... (one row per failed mod)
```

Exit code: 1

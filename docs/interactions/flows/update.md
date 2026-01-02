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
- If there are no configured mods, MMM exits successfully and reports `No mods configured.`.
- If no updates are available, MMM exits successfully and reports results with only the "Already up to date" segment (and "Skipped" if applicable).
- If the `install` prerequisite fails, MMM exits non-zero with an actionable error.
- If writing lock updates fails, MMM exits non-zero with an actionable error.

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

### Frame snapshots

#### UPDATE-01 Running (tty)

In tty mode, MMM shows all mods at the same time, grouped into segments.
As mod status changes, MMM updates the icons in place.
If a segment has no mods, MMM does not render that segment.

##### Command used
`update`

```
Updating:
⠋ Inventory Sorting (inventory-sorting) [modrinth]
⠋ Fabric API (fabric-api) [modrinth]
⏳ Mod Menu (modmenu) [modrinth]
⏳ Some Mod (some-mod) [curseforge]
... (one row per mod, all mods shown)
```

In tty mode, MMM MAY update this view in place.

#### UPDATE-01B Running (tty, mid-run mixed statuses)

As progress is made, the same view updates to reflect the final segments the mods will be sorted into.

##### Command used
`update`

```
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
No mods configured.
```

Exit code: 0

#### UPDATE-NOOP-NO-MODS No mods configured (tty and non-tty)

##### Command used
`update`

```
No mods configured.
```

Exit code: 0

#### UPDATE-NOOP-NO-UPDATES No updates available (tty and non-tty)

##### Command used
`update`

```
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

#### UPDATE-03 Partial success (tty and non-tty)

##### Command used
`update`

```
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

### Error frames

#### UPDATE-ERR-INSTALL install prerequisite failed

##### Command used
`update`

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

### Unattended behavior

Unattended mode MUST NOT prompt.
This applies when `--unattended` is set.

For `update`, `--unattended` output is identical to tty output because this flow has no prompts.
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
‼️ Update incomplete.

Failed:
❌ Mod Menu (modmenu) [modrinth] update failed: <reason>
... (one row per failed mod)
```

Exit code: 1

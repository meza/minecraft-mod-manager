# Flow: remove

All requirements in `docs/interactions/interaction-guidelines.md` apply.

## User contract

### User goal and success conditions

You want to remove one or more mods from your configuration, and delete their jar files when present.

Success looks like (for the user):
- The targeted mods are removed from `modlist.json` and `modlist-lock.json`
- The jar files are deleted from disk when present

### Entry points

- Command: `mmm remove <mods...>`
- User guide: `docs/commands/remove.md`
- Behavior spec: `docs/specs/remove.md`

### Primary flow

1. You run `mmm remove <mods...>`.
2. MMM resolves each lookup against the lockfile by ID and name, and also removes any matching config entries that do not have lock entries.
3. MMM deletes jar files when possible, removes the matching config entries by ID, and then removes the lock entries.

### Alternate and error flows

- If no config is found, MMM follows the missing-config gate as defined in the [guidelines](../interaction-guidelines.md#missing-config).
- When `--dry-run` is set, MMM prints what would be removed and makes no changes.
- If a lookup matches nothing, MMM skips it.
- Missing jars MUST NOT block config and lock updates.
- If a file delete fails, MMM reports it and exits non-zero.

---

## `remove` frame snapshots

This document specifies `remove` as state-by-state terminal frame snapshots.

### State model

States:
- REMOVE-01: running (deleting)
- REMOVE-02: success
- REMOVE-DRY-RUN: dry run
- REMOVE-NOOP: no matches
- REMOVE-ERR-DELETE: delete failed

### Frame snapshots

#### REMOVE-01 Running (tty)

##### Command used
`remove inventory-sorting soundsbegone`

In tty mode, MMM shows all matched mods at the same time.
As mod status changes, MMM updates the icons in place.

```
Removing mods:
⏳ Inventory Sorting (inventory-sorting)
⏳ Sounds Be Gone! (soundsbegone)
... (one row per mod, all matched mods shown)
```

#### REMOVE-02 Success (tty and non-tty)

##### Command used
`remove inventory-sorting soundsbegone`

```
✅ Inventory Sorting (inventory-sorting)
✅ Sounds Be Gone! (soundsbegone)
... (one row per mod, all matched mods shown)

✅ Remove complete.
```

#### REMOVE-NOOP No matches (tty and non-tty)

##### Command used
`remove not-a-mod`

```
No matching mods found.
```

#### REMOVE-DRY-RUN Dry run (tty and non-tty)

##### Command used
`remove --dry-run inventory-sorting soundsbegone`

```
Would remove:
❔ Inventory Sorting (inventory-sorting)
❔ Sounds Be Gone! (soundsbegone)
... (one row per mod, all matched mods shown)
```

### Error frames

#### REMOVE-ERR-DELETE Delete failed (tty and non-tty)

##### Command used
`remove inventory-sorting soundsbegone`

```
✅ Inventory Sorting (inventory-sorting)
❌ Sounds Be Gone! (soundsbegone) delete failed: <reason>
... (one row per mod, all matched mods shown)

‼️ Remove incomplete.
Fix the reason and rerun mmm remove.
```

### Unattended behavior

Unattended mode MUST NOT prompt.
This applies when `--unattended` is set.

For `remove`, `--unattended` output is identical to tty output because this flow has no prompts.
Message shapes and exit codes match the tty frames above.

### Non-interactive (non-tty) behavior

This applies when stdin or stdout is not a TTY.

In non-tty mode:
- MMM MUST NOT prompt.
- MMM MUST NOT emit terminal control sequences.
- Output is rendered as a plain transcript.

### Quiet flag

#### `--quiet` success

##### Command used
`remove --quiet inventory-sorting`

```
Output: none
Exit code: 0
```

`--quiet` failure:
- Errors still print.

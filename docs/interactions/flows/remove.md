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
2. If unmanaged jars are detected in the mods folder, MMM prints the unmanaged files notice and exits with code 1.
3. MMM resolves each lookup against the lockfile by ID and name, and also removes any matching config entries that do not have lock entries.
4. MMM lists the matched mods and asks for confirmation unless `--force` or `--unattended` is set.
5. When confirmed, MMM deletes jar files when possible, removes the matching config entries by ID, and then removes the lock entries.

### Alternate and error flows

- If no config is found, MMM follows the missing-config gate as defined in the [guidelines](../interaction-guidelines.md#missing-config).
- If you decline or cancel the confirmation, MMM exits without changing files and prints a cancellation message.
- If a lookup matches nothing, MMM skips it.
- Missing jars MUST NOT block config and lock updates.
- If a file delete fails, MMM reports it and exits non-zero.

---

## `remove` frame snapshots

This document specifies `remove` as state-by-state terminal frame snapshots.

### State model

States:
- REMOVE-01: confirmation prompt
- REMOVE-02: running (deleting)
- REMOVE-03: success
- REMOVE-NOOP: no matches
- REMOVE-CANCEL: canceled
- REMOVE-ERR-DELETE: delete failed

### Frame snapshots

#### REMOVE-01 Confirmation (tty)

##### Command used
`remove inventory-sorting soundsbegone`

```
Mods to remove:
❔ Inventory Sorting (inventory-sorting)
❔ Sounds Be Gone! (soundsbegone)
... (one row per mod, all matched mods shown)

? Remove these mods? (<yesShort>/<noShort>) [default: <noShort>]:

enter accept • ctrl+c/esc quit
```

#### REMOVE-02 Running (tty)

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

#### REMOVE-03 Success (tty and non-tty)

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

#### REMOVE-CANCEL Canceled (tty)

##### Command used
`remove inventory-sorting soundsbegone`

```
Remove canceled. No changes were made.
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

For `remove`, `--unattended` output matches the tty running/success frames (no prompt).
Message shapes and exit codes match the tty frames above.

### Non-interactive (non-tty) behavior

This applies when stdin or stdout is not a TTY.

In non-tty mode:
- MMM MUST NOT prompt.
- MMM MUST NOT emit terminal control sequences.
- Output is rendered as a plain transcript.
- If `--force` is not set, MMM refuses to remove mods and prints an actionable error.

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
In interactive terminals, `--quiet` still prompts for confirmation.

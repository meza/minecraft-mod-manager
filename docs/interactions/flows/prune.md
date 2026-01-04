# Flow: prune

All requirements in `docs/interactions/interaction-guidelines.md` apply.

## User contract

### User goal and success conditions

You want to delete unmanaged files from your mods folder so your installation matches what's in the configuration.

Success looks like (for the user):
- Unmanaged files are identified using the lock file and `.mmmignore`
- Deletions only happen after explicit confirmation, unless `--force` is supplied
- Deleted files are reported in output

### Entry points

- Command: `mmm prune`
- User guide: `docs/commands/prune.md`
- Behavior spec: `docs/specs/prune.md`

### Primary flow

1. You run `mmm prune`.
2. MMM reads lock to determine which filenames are managed.
3. MMM looks in the mods folder for filenames outside of that list, applying `.mmmignore` rules.
4. MMM prints the unmanaged file list.
5. MMM asks for confirmation before deleting, unless `--force` is set.
6. MMM deletes the listed files and prints what was deleted.

### Alternate and error flows

- If no unmanaged files are found, MMM prints a message and exits successfully.
- If the lock file does not exist, MMM prints an error and instructs the user to run `mmm install`.
- If the lock file exists but is empty, all non-ignored jar files are considered unmanaged.
- In `--unattended` mode and in non-tty mode, MMM MUST NOT prompt. Deletions happen only when `--force` is set.
- If deletion fails, MMM prints an actionable error and exits non-zero.

---

## `prune` frame snapshots

This document specifies `prune` as state-by-state terminal frame snapshots.

### State model

States:
- PRUNE-01: show unmanaged list (no deletion)
- PRUNE-02: confirmation prompt
- PRUNE-03: deleting
- PRUNE-04: success
- PRUNE-NOOP: no unmanaged files
- PRUNE-ERR-NO-LOCK: lock file missing
- PRUNE-ERR-DELETE: deletion failed

### Frame snapshots

#### PRUNE-01 Unmanaged list (tty, before confirmation)

##### Command used
`prune`

```
Unmanaged files:
❌ old-mod.jar
❌ unmanaged.jar

? Delete these files? (<yesShort>/<noShort>) [default: <noShort>]:

enter accept • ctrl+c/esc quit
```

#### PRUNE-02 Confirm deletion (tty)

##### Command used
`prune`

```
Unmanaged files:
❌ old-mod.jar
❌ unmanaged.jar

? Delete these files? (<yesShort>/<noShort>) [default: <noShort>]: <yesShort>
```

#### PRUNE-03 Deleting (tty, --force)

##### Command used
`prune --force`

```
Deleting files:
⏳ old-mod.jar
⏳ unmanaged.jar
```

#### PRUNE-04 Success (tty and non-tty, --force)

##### Command used
`prune --force`

```
Deleted files:
✅ old-mod.jar
✅ unmanaged.jar

✅ Prune complete.
Run mmm install to ensure your mods folder matches your configuration.
```

#### PRUNE-NOOP No unmanaged files (tty and non-tty)

##### Command used
`prune`

```
No unmanaged files found.
Run mmm install to ensure your mods folder matches your configuration.
```

### Error frames

#### PRUNE-ERR-NO-LOCK Lock file missing (tty and non-tty)

##### Command used
`prune`

```
‼️ Lock file not found at ./modlist-lock.json.

Run mmm install to create it.
```

#### PRUNE-ERR-DELETE Deletion failed (tty and non-tty)

##### Command used
`prune --force`

```
Deleting files:
✅ old-mod.jar
❌ unmanaged.jar delete failed: <reason>

‼️ Prune incomplete.
Please delete the files manually and rerun mmm prune --force.
```

### Unattended behavior

Unattended mode MUST NOT prompt.
This applies when `--unattended` is set.

For `prune`, `--unattended` output differs from tty behavior only when a confirmation would be required.

#### Unattended unmanaged files found (tty and non-tty, no --force)

##### Command used
`--unattended prune`

```
Unmanaged files:
❌ old-mod.jar
❌ unmanaged.jar

‼️ Refusing to delete files in unattended mode without --force.
Rerun with mmm prune --force.
```

#### Unattended success (tty and non-tty, --force)

##### Command used
`--unattended prune --force`

```
Deleted files:
✅ old-mod.jar
✅ unmanaged.jar

✅ Prune complete.
Run mmm install to ensure your mods folder matches your configuration.
```

### Non-interactive (non-tty) behavior

This applies when stdin or stdout is not a TTY.

In non-tty mode:
- MMM MUST NOT prompt.
- MMM MUST NOT emit terminal control sequences.

Behavior matches unattended mode for deletion confirmation.

### Quiet flag

#### `--quiet --force` success

##### Command used
`prune --quiet --force`

```
Output: none
Exit code: 0
```

#### `--quiet` unmanaged files found (no --force)

##### Command used
`prune --quiet`

```
Unmanaged files:
❌ old-mod.jar
❌ unmanaged.jar
```

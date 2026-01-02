# Flow: install

All requirements in `docs/interactions/interaction-guidelines.md` apply.

## User contract

### User goal and success conditions

You want to ensure that whatever is in the lock file (or in the config when a lock entry is missing) is present on disk.

Success looks like (for the user):
- Each managed mod has a matching jar on disk
- The lock entries reflect what is installed on disk
- No config or lock changes unless lock entries must be added

### Entry points

- Command: `mmm install`
- User guide: `docs/commands/install.md`
- Behavior spec: `docs/specs/install.md`

### Primary flow

1. You run `mmm install`.
2. MMM reads config and lock.
3. MMM installs managed mods by ensuring every mod in the lock file (or in config when a lock entry is missing) is present on disk.
4. If any config entries are missing from the lock file, MMM adds lock entries for the downloaded mods. Otherwise, MMM leaves `modlist.json` and `modlist-lock.json` unchanged.

### Alternate and error flows

- If the lock file is missing entries for some configured mods, MMM resolves files for those mods and adds lock entries after download.
- If downloads fail, MMM prints an actionable error and exits non-zero.
- If MMM cannot write new lock entries when needed, MMM prints an actionable error and exits non-zero.

---

## `install` frame snapshots

This document specifies `install` as state-by-state terminal frame snapshots.

### State model

States:
- INSTALL-01: running (installing and verifying)
- INSTALL-02: success
- INSTALL-ERR-DOWNLOAD: download failed
- INSTALL-ERR-WRITE-LOCK: failed to write lock updates

### Frame snapshots

#### INSTALL-01 Running (tty)

##### Command used
`install`

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
```

#### INSTALL-02 Success (tty and non-tty)

##### Command used
`install`

```
✅ Inventory Sorting (inventory-sorting) [modrinth]
✅ Fabric API (fabric-api) [modrinth]
✅ Mod Menu (modmenu) [modrinth]
✅ Lithium (lithium) [modrinth]
... (one row per mod, all mods shown)

✅ Install complete.
```

### Error frames

#### INSTALL-ERR-DOWNLOAD Download failed (tty)

##### Command used
`install`

```
Installing mods:
✅ Inventory Sorting (inventory-sorting) [modrinth]
❌ Fabric API (fabric-api) [modrinth] download failed: <reason>
⏳ Mod Menu (modmenu) [modrinth]
... (one row per mod, all mods shown)

‼️ Download failed. Installation incomplete.
Your configuration was not modified.

Check your network and rerun mmm install.
```

#### INSTALL-ERR-WRITE-LOCK Failed to write lock updates (tty)

##### Command used
`install`

```
Installing mods:
✅ Inventory Sorting (inventory-sorting) [modrinth]
✅ Fabric API (fabric-api) [modrinth]
... (one row per mod, all mods shown)

‼️ Could not write lock file at ./modlist-lock.json.
Your configuration was not modified.

Fix the file permissions and rerun mmm install.
```

#### INSTALL-ERR-DOWNLOAD Download failed (non-tty)

##### Command used
`install`

In non-tty mode, MMM MUST NOT emit terminal control sequences and MUST avoid spinners.

```
✅ Inventory Sorting (inventory-sorting) [modrinth]
❌ Fabric API (fabric-api) [modrinth] download failed: <reason>
❌ Mod Menu (modmenu) [modrinth] not installed (install aborted)
... (one row per mod, all mods shown)

‼️ Download failed. Installation incomplete.
Your configuration was not modified.
```

#### INSTALL-ERR-WRITE-LOCK Failed to write lock updates (non-tty)

##### Command used
`install`

In non-tty mode, MMM MUST NOT emit terminal control sequences and MUST avoid spinners.

```
✅ Inventory Sorting (inventory-sorting) [modrinth]
✅ Fabric API (fabric-api) [modrinth]
... (one row per mod, all mods shown)

‼️ Could not write lock file at ./modlist-lock.json.
Your configuration was not modified.
```

### Unattended behavior

Unattended mode MUST NOT prompt.
This applies when `--unattended` is set.

For `install`, `--unattended` output is identical to tty output because this flow has no prompts.
Message shapes and exit codes match the tty frames above.

### Non-interactive (non-tty) behavior

This applies when stdin or stdout is not a TTY.

In non-tty mode:
- MMM MUST NOT prompt.
- MMM MUST NOT emit terminal control sequences.
- MMM MUST avoid spinners and other dynamic output.

On success, MMM prints the same final per-mod list and summary line as tty.
Only the running output differs, and it is rendered as a plain transcript.

### Quiet flag

#### `--quiet` success

##### Command used
`--quiet install`

```
Output: none
Exit code: 0
```

#### `--quiet` failure

##### Command used
`--quiet install`

```
‼️ Download failed.

❌ Fabric API (fabric-api) [modrinth] download failed: <reason>

Exit code: 1
```

In `--quiet` mode:
- Errors still print.
- Success prints nothing.

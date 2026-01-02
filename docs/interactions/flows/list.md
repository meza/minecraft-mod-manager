# Flow: list

All requirements in `docs/interactions/interaction-guidelines.md` apply.

## User contract

### User goal and success conditions

You want to see which mods are configured and whether they are installed correctly.

Success looks like (for the user):
- You can tell which mods are installed
- You can tell what to do when a mod is missing or mismatched

### Entry points

- Command: `mmm list`
- User guide: `docs/commands/list.md`
- Behavior spec: `docs/specs/list.md`

### Primary flow

1. You run `mmm list`.
2. MMM renders a view that shows installed and missing states.
3. If MMM detects unmanaged jar files, MMM prints the unmanaged files notice and recommends running `mmm scan`.

### Alternate and error flows

- If no config is found, MMM follows the missing-config gate as defined in the [guidelines](../interaction-guidelines.md#missing-config).
- If a mod has a hash mismatch, MMM reports it as not installed and suggests running `mmm install`.

---

## `list` frame snapshots

This document specifies `list` as state-by-state terminal frame snapshots.

### State model

States:
- LIST-01: render list
- LIST-02: empty list
- LIST-03: unmanaged files detected notice (after list)
- LIST-ERR: failure

### Frame snapshots

#### LIST-01 Render list

##### Command used
`list`

```
Mods:

✅ Inventory Sorting (inventory-sorting) [modrinth]
❌ Some Mod (some-mod) [curseforge] not installed
❌ Gamma Mod (mod-c) [modrinth] hash mismatch (run mmm install)
... (one row per mod, all mods shown)
```

#### LIST-02 Empty list

##### Command used
`list`

```
No mods configured.
```

#### LIST-03 Unmanaged files detected (after list)

##### Command used
`list`

```
Mods:

✅ Inventory Sorting (inventory-sorting) [modrinth]
... (one row per mod, all mods shown)

Unmanaged files detected:

There are jar files in your mods folder that are not in your lock file.

❌ unmanaged-A.jar
❌ unmanaged-B.jar

Run mmm scan to adopt or resolve these files.
```

### Error frames

#### LIST-ERR Failure

##### Command used
`list`

```
‼️ Could not show mod list: <reason>

Fix the configuration and rerun mmm list.
```

### Unattended behavior

Unattended mode MUST NOT prompt.
This applies when `--unattended` is set.

For `list`, `--unattended` output is identical to tty output because this flow has no prompts.

### Non-interactive (non-tty) behavior

This applies when stdin or stdout is not a TTY.

In non-tty mode:
- MMM MUST NOT prompt.
- MMM MUST NOT emit terminal control sequences.
- Output is rendered as a plain transcript.

### Quiet flag

`--quiet` does not meaningfully apply to `list`.
The list still prints.

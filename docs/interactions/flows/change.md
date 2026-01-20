# Flow: change

All requirements in `docs/interactions/interaction-guidelines.md` apply.

## User contract

### User goal and success conditions

You want to update your mods to support a different Minecraft version.

Success looks like (for the user):
- MMM now targets the requested Minecraft version
- The mods folder contains a compatible set of jars for that version

### Entry points

- Command: `mmm change [game_version]`
- User guide: `docs/commands/change.md`
- Behavior spec: `docs/specs/change.md`

### Primary flow

1. You run `mmm change [game_version]`.
2. If unmanaged jars are detected in the mods folder, MMM prints the unmanaged files notice and exits with code 1.
3. MMM resolves the target version, defaulting to `latest` when not provided.
4. MMM starts compatibility checks for the target version unless `--force` is set.
5. As mods are confirmed compatible, MMM downloads them into a staging area without touching your current setup.
6. Only if downloads succeed, MMM switches you to the new version (config + mods folder swap).
7. MMM cleans up temporary and backup artifacts.

### Alternate and error flows

- If no config is found, MMM follows the missing-config gate as defined in the [guidelines](../interaction-guidelines.md#missing-config).
- If you attempt to change to the current version, the command exits with code `0` and makes no changes.
- If some mods do not support the target version and `--force` is not set, MMM cancels downloads and switching (if started), continues checking compatibility to surface all offenders, then fails after all checks complete.
- With `--force`, MMM proceeds even if some mods are unsupported, and skipped mods are reported.
- If downloads fail, MMM MUST roll back (no user-visible changes) and print an actionable error.
- If switching fails after partially switching, MMM MUST attempt rollback and print an actionable error.

---

## `change` frame snapshots

### State model

States:
- CHANGE-01: running (compatibility + download)
- CHANGE-02: switching
- CHANGE-03: success
- CHANGE-ERR-COMPAT: compatibility failed (no --force), downloads and switching canceled, compatibility continues to surface all offenders
- CHANGE-ERR-DOWNLOAD: download failed (rollback)
- CHANGE-ERR-SWITCH: switching failed (rollback attempt)

### Frame snapshots

#### CHANGE-01 Running (tty)

##### Command used
`change 1.19.4`

In tty mode, MMM shows all mods at the same time, grouped into segments.
As mod status changes, MMM updates the icons in place.
If a segment has no mods, MMM does not render that segment.

```
Change Minecraft version to 1.19.4
Your current setup will not be modified until all downloads succeed.

Compatibility:
⠋ Sounds Be Gone! (sounds-be-gone) [modrinth]
⏳ Better Clouds (better-clouds) [modrinth]
⏳ Lithium (lithium) [modrinth]
... (one row per mod, all mods shown)

Downloading:
⬇️ Inventory Sorting (inventory-sorting) [modrinth]
█████░░░░░
50% (512 KB / 1 MB)
⬇️ Fabric API (fabric-api) [modrinth]
██░░░░░░░░
20% (200 KB / 1 MB)
⏳ Mod Menu (modmenu) [modrinth]
... (one row per mod, all mods shown)

Switching: (⠋ waiting for downloads to complete)
⏳ Cloth Config API (cloth-config) [modrinth]
⏳ Sodium (sodium) [modrinth]
⏳ Sodium Extra (sodium-extra) [modrinth]
... (one row per mod, all mods shown)
```

#### CHANGE-03 Success (tty)

##### Command used
`change 1.19.4`

```
✅ Sounds Be Gone! (sounds-be-gone) [modrinth]
✅ Inventory Sorting (inventory-sorting) [modrinth]
✅ Better Clouds (better-clouds) [modrinth]
... (one row per mod, all mods shown)

✅ Now targeting 1.19.4
```

#### CHANGE-03 Success with skipped mods (--force, tty)

##### Command used
`change --force 1.19.4`

```
✅ Sounds Be Gone! (sounds-be-gone) [modrinth]
✅ Inventory Sorting (inventory-sorting) [modrinth]
❌ Some Mod (some-mod) [modrinth] unsupported for 1.19.4 (skipped)
... (one row per mod, all mods shown)

✅ Now targeting 1.19.4

Skipped unsupported mods:
❌ Some Mod (some-mod) [modrinth]
```

#### CHANGE-FORCE-POLICY Choose incompatible jar policy (--force, tty)

##### Command used
`change --force 1.19.4`

```
Change Minecraft version to 1.19.4
Your current setup will not be modified until all downloads succeed.

Compatibility:
✅ Sounds Be Gone! (sounds-be-gone) [modrinth]
✅ Inventory Sorting (inventory-sorting) [modrinth]
❌ Some Mod (some-mod) [modrinth] unsupported for 1.19.4 (skipped)
... (one row per mod, all mods shown)

Downloading:
✅ Sounds Be Gone! (sounds-be-gone) [modrinth]
✅ Inventory Sorting (inventory-sorting) [modrinth]
... (one row per mod, all mods shown)

Switching: (⠋ waiting for your choice)
⏳ Cloth Config API (cloth-config) [modrinth]
⏳ Sodium (sodium) [modrinth]
⏳ Sodium Extra (sodium-extra) [modrinth]
... (one row per mod, all mods shown)

? Choose what to do with the incompatible mods already present
❯ Keep the mods in the config and remove the incompatible files
  Remove the mods from the config and remove the incompatible files
  Keep the mods in the config and disable the incompatible files (*.jar.disabled)
↑/k up • ↓/j down • / filter • esc clear filter • enter apply filter • esc cancel • q quit • ? more

? Choose what to do with the incompatible mods already present Kept in config
```

#### CHANGE-03 Success with skipped mods (--force --prune-config, tty)

##### Command used
`change --force --prune-config 1.19.4`

```
✅ Sounds Be Gone! (sounds-be-gone) [modrinth]
✅ Inventory Sorting (inventory-sorting) [modrinth]
❌ Some Mod (some-mod) [modrinth] unsupported for 1.19.4
... (one row per mod, all mods shown)

✅ Now targeting 1.19.4

Removed unsupported mods:
❌ Some Mod (some-mod) [modrinth]
```

#### CHANGE-03 Success with skipped mods (--force --disable-skipped, tty)

##### Command used
`change --force --disable-skipped 1.19.4`

```
✅ Sounds Be Gone! (sounds-be-gone) [modrinth]
✅ Inventory Sorting (inventory-sorting) [modrinth]
❌ Some Mod (some-mod) [modrinth] unsupported for 1.19.4
... (one row per mod, all mods shown)

✅ Now targeting 1.19.4

Disabled unsupported mods:
❌ Some Mod (some-mod) [modrinth]
```

#### CHANGE-NOOP Target equals current version

##### Command used
`change 1.19.4`

```
✅ Already targeting 1.19.4. No changes were made.
```

Exit code: 0

#### CHANGE-03 Success (non-tty)

##### Command used
`change 1.19.4`

```
✅ Sounds Be Gone! (sounds-be-gone) [modrinth]
✅ Inventory Sorting (inventory-sorting) [modrinth]
✅ Better Clouds (better-clouds) [modrinth]
... (one row per mod, all mods shown)

✅ Now targeting 1.19.4
```

#### CHANGE-03 Success with skipped mods (--force, non-tty)

##### Command used
`change --force 1.19.4`

```
✅ Sounds Be Gone! (sounds-be-gone) [modrinth]
✅ Inventory Sorting (inventory-sorting) [modrinth]
❌ Some Mod (some-mod) [modrinth] unsupported for 1.19.4 (skipped)
... (one row per mod, all mods shown)

✅ Now targeting 1.19.4

Skipped unsupported mods:
❌ Some Mod (some-mod) [modrinth]
```

#### CHANGE-03 Success with skipped mods (--force --prune-config, non-tty)

##### Command used
`change --force --prune-config 1.19.4`

```
✅ Sounds Be Gone! (sounds-be-gone) [modrinth]
✅ Inventory Sorting (inventory-sorting) [modrinth]
❌ Some Mod (some-mod) [modrinth] unsupported for 1.19.4
... (one row per mod, all mods shown)

✅ Now targeting 1.19.4

Removed unsupported mods:
❌ Some Mod (some-mod) [modrinth]
```

#### CHANGE-03 Success with skipped mods (--force --disable-skipped, non-tty)

##### Command used
`change --force --disable-skipped 1.19.4`

```
✅ Sounds Be Gone! (sounds-be-gone) [modrinth]
✅ Inventory Sorting (inventory-sorting) [modrinth]
❌ Some Mod (some-mod) [modrinth] unsupported for 1.19.4
... (one row per mod, all mods shown)

✅ Now targeting 1.19.4

Disabled unsupported mods:
❌ Some Mod (some-mod) [modrinth]
```

### Error and recovery frames

#### CHANGE-ERR-COMPAT Compatibility failed (no --force, tty, compatibility continues)

##### Command used
`change 1.19.4`

In tty mode, once the first compatibility check fails and `--force` is not set, MMM cancels downloads and switching and removes those sections.
MMM continues compatibility checks so you can see all incompatible mods.

```
Change Minecraft version to 1.19.4
Your current setup will not be modified until all downloads succeed.

Compatibility:
❌ Some Mod (some-mod) [modrinth] unsupported for 1.19.4
⠋ Better Clouds (better-clouds) [modrinth]
⏳ Lithium (lithium) [modrinth]
... (one row per mod, all mods shown)
```

#### CHANGE-ERR-COMPAT Compatibility failed (no --force, tty, final)

##### Command used
`change 1.19.4`

```
Change Minecraft version to 1.19.4
Your current setup will not be modified until all downloads succeed.

Compatibility:
❌ Some Mod (some-mod) [modrinth] unsupported for 1.19.4
❌ Another Mod (another-mod) [curseforge] unsupported for 1.19.4

‼️ Compatibility check failed for 1.19.4
No changes were made.

Remove blockers, or rerun with `mmm change --force 1.19.4`.
```

#### CHANGE-ERR-COMPAT Compatibility failed (non-tty, no --force)

##### Command used
`change 1.19.4`

```
❌ Sounds Be Gone! (sounds-be-gone) [modrinth]
❌ Some Mod (some-mod) [modrinth] unsupported for 1.19.4
❌ Another Mod (another-mod) [curseforge] unsupported for 1.19.4

‼️ Compatibility check failed for 1.19.4
No changes were made.
```

#### CHANGE-ERR-DOWNLOAD Download failed (rollback)

##### Command used
`change 1.19.4`

```
Change Minecraft version to 1.19.4
Your current setup will not be modified until all downloads succeed.

Downloading:
⬇️ Inventory Sorting (inventory-sorting) [modrinth]
█████░░░░░
50% (512 KB / 1 MB)
❌ Some Mod (some-mod) [modrinth] download failed: <reason>
... (one row per mod, all mods shown)

‼️ Downloads failed. No changes were made.

Check your network and rerun `mmm change 1.19.4`.
```

#### CHANGE-ERR-SWITCH Switching failed (rollback attempt)

##### Command used
`change 1.19.4`

```
Switching:
⠋ Sounds Be Gone! (sounds-be-gone) [modrinth]
⏳ Inventory Sorting (inventory-sorting) [modrinth]
❌ Better Clouds (better-clouds) [modrinth] switch failed: <reason>
... (one row per mod, all mods shown)

‼️ Switching failed. Attempted rollback. No changes were made.
```

#### CHANGE-ERR-FORCE-POLICY Multiple policy flags (--force)

##### Command used
`change --force --keep-config --prune-config 1.19.4`

```
‼️ Invalid flags

--keep-config, --prune-config, and --disable-skipped are mutually exclusive.
Choose exactly one policy flag under --force.
```

### Unattended behavior

Unattended mode MUST NOT prompt.
This applies when `--unattended` is set.

In unattended mode:
- If no config is found, MMM MUST fail fast (as defined in the missing config gate in `docs/interactions/interaction-guidelines.md`).
- If compatibility fails and `--force` is not set, MMM MUST exit non-zero after printing an actionable error.
- If downloads fail or switching fails, MMM MUST roll back and print an actionable error.

### Non-tty mode

This applies when stdin or stdout is not a TTY.

In non-tty mode:
- MMM MUST NOT prompt.
- MMM MUST NOT emit terminal control sequences.
- Frames are printed as a plain transcript.

### Quiet flag

#### `--quiet` success

##### Command used
`change --quiet 1.19.4`

```
Output: none
Exit code: 0
```

When the target equals the current version, `--quiet` still produces no output and exits 0.

#### `--quiet --force` success with skipped mods

##### Command used
`change --quiet --force 1.19.4`

```
Skipped unsupported mods:
❌ Some Mod (some-mod) [modrinth] unsupported for 1.19.4
```

#### `--quiet` compatibility failed (no --force)

##### Command used
`change --quiet 1.19.4`

```
‼️ Compatibility failed for 1.19.4

❌ Some Mod (some-mod) [modrinth] unsupported for 1.19.4
❌ Another Mod (another-mod) [curseforge] unsupported for 1.19.4
```

#### `--quiet` download failed (rollback)

##### Command used
`change --quiet 1.19.4`

```
‼️ Downloads failed. No changes were made.

❌ Some Mod (some-mod) [modrinth] download failed: <reason>
```

#### `--quiet` switching failed (rollback attempt)

##### Command used
`change --quiet 1.19.4`

```
‼️ Switching failed. Attempted rollback. No changes were made.

❌ Some Mod (some-mod) [modrinth] switch failed: <reason>
```

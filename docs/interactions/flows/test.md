# Flow: test

All requirements in `docs/interactions/interaction-guidelines.md` apply.

## User contract

### User goal and success conditions

You want to know if your configured mods support a target Minecraft version before you change anything.

Success looks like (for the user):
- You can tell whether the target version is safe to switch to
- You can identify which mods block the upgrade
- Exit codes are stable and usable in scripts
- No files are changed

### Entry points

- Command: `mmm test [game_version]`
- User guide: `docs/commands/test.md`
- Behavior spec: `docs/specs/test.md`

### Primary flow

1. You run `mmm test [game_version]`.
2. MMM resolves the target version, defaulting to `latest` when not provided.
3. MMM checks each configured mod for a compatible release for the target version.
4. MMM prints results and exits:
   - `0` when all mods support the target
   - `1` when one or more mods block the target, or compatibility cannot be determined
   - `0` when the target version equals the configured version (well-defined no-op)

### Alternate and error flows

- If the target version equals the configured version, MMM makes no changes, prints a no-op message, and exits with code `0`.
- If MMM resolves `latest` and it equals the configured version, MMM behaves as a no-op.
- If `latest` cannot be resolved (offline or platform error), MMM behaves as follows:
  - In interactive tty mode, MMM prompts for an explicit target version.
  - In unattended and non-tty modes, MMM fails fast with an actionable error.
- If the user provides an invalid Minecraft version, MMM behaves as follows:
  - In interactive tty mode, MMM prompts for a valid Minecraft version.
  - In unattended and non-tty modes, MMM fails fast with an actionable error.
- If MMM cannot determine compatibility for one or more mods due to platform errors, MMM reports the inconclusive mods and exits non-zero.

---

## `test` frame snapshots

This document specifies `test` as state-by-state terminal frame snapshots.

### State model

States:
- TEST-02: target version resolution (explicit or latest)
- TEST-ERR-LATEST-OFFLINE: latest cannot be resolved
- TEST-ERR-VERSION: invalid Minecraft version
- TEST-03: running (checking compatibility)
- TEST-04: success (all mods supported)
- TEST-05: failure (blocking mods)
- TEST-ERR-INCONCLUSIVE: platform errors (compatibility unknown for one or more mods)
- TEST-NOOP: target equals configured version

### Frame snapshots

#### TEST-02 Target version resolution (tty, no argument provided)

When the user does not provide a target version, MMM defaults to `latest`.
This is not a prompt. It is version resolution before running the check.

##### Command used
`test`

```
Testing compatibility for Minecraft <resolvedLatest>...
```

#### TEST-03 Running (tty)

In tty mode, MMM shows all mods at the same time.
As mod status changes, MMM updates the icons in place.

##### Command used
`test 1.21.11`

```
Testing compatibility for Minecraft 1.21.11:
⠋ Inventory Sorting (inventory-sorting) [modrinth]
⠋ Fabric API (fabric-api) [modrinth]
⠋ Mod Menu (modmenu) [modrinth]
⠋ Some Mod (some-mod) [curseforge]
... (one row per mod, all mods shown)
```

#### TEST-04 Success (tty and non-tty)

##### Command used
`test 1.21.11`

```
Compatible mods:
✅ Inventory Sorting (inventory-sorting) [modrinth]
✅ Fabric API (fabric-api) [modrinth]
✅ Mod Menu (modmenu) [modrinth]
... (one row per compatible mod)

✅ All mods support 1.21.11.
```

Exit code: 0

#### TEST-05 Failure (blocking mods, tty and non-tty)

##### Command used
`test 1.21.11`

```
Compatible mods:
✅ Inventory Sorting (inventory-sorting) [modrinth]
✅ Mod Menu (modmenu) [modrinth]
... (one row per compatible mod)

Not compatible mods:
❌ Some Mod (some-mod) [curseforge]
... (one row per not compatible mod)

‼️ Some mods do not support 1.21.11.

Wait for updates, remove blockers, then rerun `mmm test 1.21.11`.
```

Exit code: 1

#### TEST-ERR-INCONCLUSIVE Platform errors (tty and non-tty)

If compatibility cannot be determined for one or more mods, MMM MUST treat this as a failure to accomplish the intended outcome.

##### Command used
`test 1.21.11`

```
Compatible mods:
✅ Inventory Sorting (inventory-sorting) [modrinth]
... (one row per compatible mod)

Could not be checked:
❔ Some Mod (some-mod) [curseforge] <reason>
... (one row per inconclusive mod)

‼️ Compatibility check incomplete for 1.21.11.

Rerun later or run `mmm test --debug 1.21.11` to capture a diagnostic log.
```

Exit code: 1

#### TEST-05 Failure with platform errors (tty and non-tty)

When both not compatible mods and platform errors exist, MMM MUST render both sections.

##### Command used
`test 1.21.11`

```
Compatible mods:
✅ Inventory Sorting (inventory-sorting) [modrinth]
... (one row per compatible mod)

Not compatible mods:
❌ Some Mod (some-mod) [curseforge]
... (one row per not compatible mod)

Could not be checked:
❔ Flaky Mod (flaky-mod) [modrinth] <reason>
... (one row per inconclusive mod)

‼️ Some mods do not support 1.21.11.

Wait for updates, remove blockers, then rerun `mmm test 1.21.11`.

‼️ Compatibility check incomplete for 1.21.11.

Rerun later or run `mmm test --debug 1.21.11` to capture a diagnostic log.
```

Exit code: 1

#### TEST-NOOP Target equals configured version (tty and non-tty)

##### Command used
`test 1.21.11`

```
The target version 1.21.11 is the same as your configured version.
Nothing to test.
```

Exit code: 0

#### TEST-ERR-LATEST-OFFLINE latest cannot be resolved (tty)

In interactive tty mode, MMM MAY offer recovery by prompting for an explicit version.

##### Command used
`test`

```
‼️ Could not resolve latest Minecraft version.
? Enter an explicit Minecraft version to test: 1.21.11

tab complete • enter accept • ctrl+c/esc quit
```

If the user enters a version, MMM resumes at TEST-03.
If the user cancels, MMM exits safely.

#### TEST-ERR-VERSION invalid Minecraft version (tty)

In interactive tty mode, MMM MAY offer recovery by prompting for an explicit version.
This prompt MUST follow the init version prompt format, but the question text should reflect the test use case.

##### Command used
`test 1.21.12`

```
‼️ Invalid Minecraft version 1.21.12.
? What exact Minecraft version are you trying to test against? (eg: 1.18.2, 1.19, 1.19.1) 1.21.12     <- That Minecraft version does not exist

tab complete • enter accept • ctrl+c/esc quit
```

If the user enters a valid version, MMM resumes at TEST-03.
If the user cancels, MMM exits safely.

### Unattended behavior

Unattended mode MUST NOT prompt.
This applies when `--unattended` is set.

In unattended mode:
- If `latest` cannot be resolved, MMM MUST fail fast.
- If the version is invalid, MMM MUST fail fast.

#### Unattended success

##### Command used
`--unattended test 1.21.11`

```
Compatible mods:
✅ Inventory Sorting (inventory-sorting) [modrinth]
... (one row per compatible mod)

✅ All mods support 1.21.11.
```

Exit code: 0

#### Unattended failure (blocking mods)

##### Command used
`--unattended test 1.21.11`

```
Compatible mods:
✅ Inventory Sorting (inventory-sorting) [modrinth]
... (one row per compatible mod)

Not compatible mods:
❌ Some Mod (some-mod) [curseforge]
... (one row per not compatible mod)

‼️ Some mods do not support 1.21.11.
```

Exit code: 1

#### Unattended no-op (target equals configured version)

##### Command used
`--unattended test 1.21.11`

```
The target version 1.21.11 is the same as your configured version.
Nothing to test.
```

Exit code: 0

#### Unattended latest cannot be resolved

##### Command used
`--unattended test`

```
‼️ Could not resolve latest Minecraft version.
Run `mmm test <version>` with an explicit version.
```

Exit code: 1

#### Unattended invalid Minecraft version

##### Command used
`--unattended test 1.21.12`

```
‼️ Invalid Minecraft version 1.21.12.
Rerun with a valid version (example: 1.21.1).
```

Exit code: 1

### Non-interactive (non-tty) behavior

This applies when stdin or stdout is not a TTY.

In non-tty mode:
- MMM MUST NOT prompt.
- MMM MUST NOT emit terminal control sequences.
- MMM MUST avoid spinners and other dynamic output.

For `test`, non-tty behavior matches unattended behavior for prompts and message shapes.

### Quiet flag

With `--quiet`, MMM prints only actionable results.
`--quiet` MUST NOT change error message shapes. Failures print the same errors and exit non-zero.

#### `--quiet` success

##### Command used
`test --quiet 1.21.11`

```
Output: none
Exit code: 0
```

#### `--quiet` failure (blocking mods)

##### Command used
`test --quiet 1.21.11`

```
Not compatible mods:
❌ Some Mod (some-mod) [curseforge]
❌ Another Mod (another-mod) [modrinth]

Could not be checked:
❔ Flaky Mod (flaky-mod) [modrinth]
```

Exit code: 1

#### `--quiet` no-op (target equals configured version)

##### Command used
`test --quiet 1.21.11`

```
Output: none
Exit code: 0
```

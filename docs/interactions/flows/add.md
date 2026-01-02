# Flow: add

All requirements in `docs/interactions/interaction-guidelines.md` apply.

## User contract

### User goal and success conditions

You want to add a mod to your config and download a compatible jar into your mods folder.

Success looks like (for the user):
- The mod is listed in the config file
- The jar exists on disk in the configured mods folder

### Entry points

- Command: `mmm add <platform> <id>`
- User guide: `docs/commands/add.md`
- Behavior spec: `docs/specs/add.md`

### Primary flow

1. You run `mmm add <platform> <id>`.
2. MMM resolves a compatible file for <id> from the <platform> and downloads it.

### Alternate and error flows

- If no config is found, MMM follows the missing-config gate as defined in the [guidelines](../interaction-guidelines.md#missing-config).
- If the mod id is not found on the platform, MMM prompts to try a different search.
- If no compatible file is found, MMM prompts to try a different search.
- If the download fails, MMM prompts to try a different search.

---

## `add` frame snapshots

This document specifies `add` as state-by-state terminal frame snapshots.

### State model

States:
- ADD-01: platform selection (when in recovery)
- ADD-02: id text input
- ADD-03: success
- ADD-ERR-NOT-FOUND: project not found
- ADD-ERR-NO-COMPAT: no compatible file
- ADD-ERR-DOWNLOAD: download failed

### Frame snapshots

#### ADD-01 Platform selection list (waiting for input)

With bubbletea tui-lite platform selector.

##### Command used

No command, this state is not reachable from the CLI currently.
This frame is only reachable from error recovery flows.

```
? Which platform should MMM use?
❯ <platform>
  <platform>

  ↑/k up • ↓/j down • / filter • esc clear filter • enter apply filter • esc cancel • q quit • ? more
```

#### ADD-02 Platform selected -> id prompt

With bubbletea tui-lite input field

##### Command used
No command, this state is not reachable from the CLI currently.
This frame is only reachable from error recovery flows.

```
? Which platform should we use? <platform>
? What is the mod id? inventory-sorting

tab complete • enter accept • ctrl+c/esc quit
```

#### ADD-03 Success

With bubbletea tui-lite mod progress display.

##### Command used
This frame can be reached directly from the CLI.
`add <platform> inventory-sorting`

Or it can be reached from recovery flows.

##### Started

```
️⌛ Downloading Inventory Sorting (inventory-sorting) from <platform>
```

##### In progress

```
⬇️ Downloading Inventory Sorting (inventory-sorting) from <platform>
█████░░░░░
50% (512 KB / 1 MB)
```

##### Finished

```
✅ Added Inventory Sorting (inventory-sorting) for <platform>
```

### Error and recovery frames

#### ADD-ERR-NOT-FOUND Project not found, retry id

##### Command used
`add <platform> <id>`

```
‼️ <platform> doesn't have a project for id <id>.
? Would you like to modify your search? (<yesShort>/<noShort>) [default: <noShort>]: <yesShort>

enter accept • ctrl+c/esc quit
```

If yes: the app returns to ADD-01
If no: the app exits with code 1

#### ADD-ERR-NO-COMPAT No compatible file, try alternate platform

##### Command used
`add <platform> <id>`

```
‼️ No file found for <loader> and Minecraft <gameVersion>.
? Would you like to modify your search? (<yesShort>/<noShort>) [default: <noShort>]:

enter accept • ctrl+c/esc quit
```

If Yes: the app returns to ADD-01
If no: the app exits with code 1

#### ADD-ERR-DOWNLOAD Download failed, retry

##### Command used
`add <platform> <id>`

```
‼️ Download failed for <platform> <id>

We retried <N> times but could not complete the download.
It is possible that <platform> is experiencing issues.

Check <platform> and your network and try again later.

? Would you like to modify your search? (<yesShort>/<noShort>) [default: <noShort>]:

enter accept • ctrl+c/esc quit
```

If Yes: the app returns to ADD-01
If no: the app exits with code 1

### Unattended behavior

Unattended mode MUST NOT prompt.

#### Unattended success

##### Command used
`--unattended add <platform> inventory-sorting`

```
✅ Added Inventory Sorting (inventory-sorting) for <platform>
```

#### Unattended missing config

##### Command used
`--unattended add <platform> inventory-sorting`

```
‼️ No configuration file found at ./modlist.json.

Run `mmm init` to create one.
```

#### Unattended unknown platform

##### Command used
`--unattended add not-a-platform inventory-sorting`

Let cobra handle this case

#### Unattended project not found

##### Command used
`--unattended add <platform> <id>`

```
‼️ Could not find project for <platform> <id>

Verify the id on the platform and try again.
```

the app exits with code 1

#### Unattended no compatible file

##### Command used
`--unattended add <platform> <id>`

```
‼️ No file found for <loader> and Minecraft <gameVersion>.
```

the app exits with code 1

#### Unattended download failed

##### Command used
`--unattended add <platform> <id>`

```
‼️ Download failed for <platform> <id>

We retried <N> times but could not complete the download.
It is possible that <platform> is experiencing issues.

Check <platform> and your network and try again later.
```

the app exits with code 1

### Non-interactive (non-tty) behavior

For `add`, non-interactive (non-tty) behavior is the same as unattended behavior:
- no prompts
- transcript-only output (no control sequences)
- same messages and exit codes as the unattended cases above

### Quiet flag

#### `--quiet` success

##### Command used
`--quiet add <platform> <id>`

```
Output: none
Exit code: 0
```

#### `--quiet` failure

- Errors still print.
- Exit code is non-zero as applicable.

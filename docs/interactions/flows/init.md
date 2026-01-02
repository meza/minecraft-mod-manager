# Flow: init

All requirements in `docs/interactions/interaction-guidelines.md` apply.

## User contract

### User goal and success conditions

You want to create a valid MMM configuration so you can manage mods in a folder.

Success looks like (for the user):
- `modlist.json` exists at the configured path
- The configuration is valid and points at an existing mods folder

### Entry points

- Command: `mmm init`
- User guide: `docs/commands/init.md`
- Behavior spec: `docs/specs/init.md`

### Primary flow

1. You run `mmm init` in a terminal.
2. MMM collects required values:
   - loader
   - game version
   - release types
   - mods folder
3. MMM validates the inputs and writes the configuration.

### Alternate and error flows

- If you provide all required flags, MMM runs without prompting.
- If the config path already exists, MMM asks what to do (overwrite or choose another path).
- If the user provides an invalid Minecraft version, MMM rejects it and keeps the user in the version input step.
- If the user selects `latest` but MMM cannot resolve it (offline), MMM rejects it and asks for an explicit version.
- If the mods folder path does not exist, MMM rejects it and keeps the user in the mods folder step.
  - MMM MUST NOT offer to create the folder. The folder must already exist.

---

## `init` frame snapshots

This document specifies `init` as state-by-state terminal frame snapshots.

### State model

States:
- INIT-01: config exists check
- INIT-02: config exists resolution (overwrite or new path)
- INIT-03: loader selection list
- INIT-04: Minecraft version text input
- INIT-ERR-VERSION: invalid Minecraft version (inline)
- INIT-ERR-LATEST-OFFLINE: latest cannot be resolved (inline)
- INIT-05: release types multi-select list
- INIT-06: mods folder text input
- INIT-ERR-MODS-FOLDER: invalid mods folder path (inline)
- INIT-07: confirm write
- INIT-09: success

### Frame snapshots

#### INIT-01 Config exists gate

If the config file already exists, MMM MUST ask what to do next.

##### Command used
`init`

```
? ./modlist.json already exists. Overwrite it? (<yesShort>/<noShort>) [default: <noShort>]:

enter accept • ctrl+c/esc quit
```

#### INIT-02 Config exists resolution

If overwrite is accepted, MMM continues to INIT-03.

If overwrite is declined, MMM asks for a new config path and then continues to INIT-03 using that new path.

##### Overwrite accepted (continues to loader selection)

```
? ./modlist.json already exists. Overwrite it? (<yesShort>/<noShort>) [default: <noShort>]: <yesShort>
? Which loader would you like to use?
❯ bukkit
  bungeecord
  cauldron
  datapack
  fabric
  folia
  forge
  liteloader

  ↑/k up • ↓/j down • / filter • esc clear filter • enter apply filter • esc cancel • q quit • ? more
```

##### Overwrite declined and new config path entered

```
? ./modlist.json already exists. Overwrite it? (<yesShort>/<noShort>) [default: <noShort>]: <noShort>
? Enter a new config path: ./modlist2.json

tab complete • enter accept • ctrl+c/esc quit
```

#### INIT-03 Loader selection list

##### Command used
`init`

```
? Which loader would you like to use?
❯ bukkit
  bungeecord
  cauldron
  datapack
  fabric
  folia
  forge
  liteloader

  •••

  ↑/k up • ↓/j down • / filter • esc clear filter • enter apply filter • esc cancel • q quit • ? more
```

##### Loader chosen (collapsed into answered prompt)

```
? Which loader would you like to use? fabric
? What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1) 1.21.11

tab complete • enter accept • ctrl+c/esc quit
```

#### INIT-04 Minecraft version input

##### INIT-ERR-VERSION Invalid Minecraft version (inline)

##### Command used
`init`

```
? Which loader would you like to use? fabric
? What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1) 1.21.12     <- That Minecraft version does not exist

tab complete • enter accept • ctrl+c/esc quit
```

##### INIT-ERR-LATEST-OFFLINE latest cannot be resolved (inline)

##### Command used
`init`

```
? Which loader would you like to use? fabric
? What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1) latest     <- Could not resolve latest, enter an explicit version

tab complete • enter accept • ctrl+c/esc quit
```

#### INIT-05 Release types selection list

##### Command used
`init`

```
? Which loader would you like to use? fabric
? What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1) 1.21.11
? Which types of releases would you like to consider to download?
❯   alpha
    beta
  ✓ release







  ↑/k up • ↓/j down • / filter • esc clear filter • enter apply filter • esc cancel • space toggle • enter accept • ctrl+c/esc quit •
```

##### Release types chosen (collapsed into answered prompt)

```
? Which loader would you like to use? fabric
? What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1) 1.21.11
? Which types of releases would you like to consider to download? beta, release
? Which folder should MMM download mods into? ./mods

tab complete • enter accept • ctrl+c/esc quit
```

#### INIT-06 Mods folder input

##### INIT-ERR-MODS-FOLDER Invalid mods folder (inline)

##### Command used
`init`

```
? Which loader would you like to use? fabric
? What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1) 1.21.11
? Which types of releases would you like to consider to download? release
? Which folder should MMM download mods into? ./mods-does-not-exist     <- Folder does not exist

tab complete • enter accept • ctrl+c/esc quit
```

#### INIT-07 Confirm write

##### Command used
`init`

```
? Which loader would you like to use? fabric
? What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1) 1.21.11
? Which types of releases would you like to consider to download? release
? Which folder should MMM download mods into? ./mods
? Write configuration now? (<yesShort>/<noShort>) [default: <noShort>]: <yesShort>

enter accept • ctrl+c/esc quit
```

#### INIT-09 Success

##### Command used
`init`

```
Created new configuration for <loader> and Minecraft <gameVersion> at ./modlist.json
Run mmm add <platform> <id> to add your first mod.
```

### Unattended behavior

Unattended mode MUST NOT prompt.
This applies when `--unattended` is set.

#### Unattended success

##### Command used
`--unattended init <flags...>`

```
Created new configuration for <loader> and Minecraft <gameVersion> at ./modlist.json
Run mmm add <platform> <id> to add your first mod.
```

#### Unattended missing required input

##### Command used
`--unattended init`

```
‼️ Missing required values for init in unattended mode.
Run mmm init interactively or provide --loader, --game-version, and --mods-folder.
```

#### Unattended config already exists

##### Command used
`--unattended init <flags...>`

```
‼️ Configuration file already exists at ./modlist.json.
Rerun with --force to overwrite, use --config to choose a different path, or remove the existing file and try again.
```

#### Unattended config already exists, overwrite with --force

##### Command used
`--unattended init --force <flags...>`

```
Created new configuration for <loader> and Minecraft <gameVersion> at ./modlist.json
Run mmm add <platform> <id> to add your first mod.
```

#### Unattended latest cannot be resolved (offline)

##### Command used
`--unattended init --game-version latest <other flags...>`

```
‼️ Could not resolve latest Minecraft version.
Rerun with --game-version <version>.
```

#### Unattended invalid Minecraft version

##### Command used
`--unattended init --game-version 1.21.12 <other flags...>`

```
‼️ Invalid Minecraft version 1.21.12.
Rerun with a valid --game-version (example: 1.21.1).
```

#### Unattended mods folder does not exist

##### Command used
`--unattended init --mods-folder ./mods-does-not-exist <other flags...>`

```
‼️ Mods folder does not exist at ./mods-does-not-exist.
Create the folder or rerun with --mods-folder <path-to-existing-folder>.
```

### Non-interactive (non-tty) behavior

This applies when stdin or stdout is not a TTY.

For `init`, non-tty behavior is the same as unattended behavior:
- no prompts
- transcript-only output (no control sequences)
- same messages and exit codes as the unattended cases above

### Quiet flag

#### `--quiet` success

##### Command used
`init --quiet`

```
Output: none
Exit code: 0
```

`--quiet` failure:
- Errors still print.

#### `--quiet` missing required input (unattended)

##### Command used
`init --quiet --unattended`

```
‼️ Missing required values for init in unattended mode.
Run mmm init interactively or provide --loader, --game-version, and --mods-folder.
```

#### `--quiet` config already exists (unattended, no --force)

##### Command used
`init --quiet --unattended <flags...>`

```
‼️ Configuration file already exists at ./modlist.json.
Rerun with --force to overwrite, use --config to choose a different path, or remove the existing file and try again.
```

#### `--quiet` latest cannot be resolved (unattended)

##### Command used
`init --quiet --unattended --game-version latest <other flags...>`

```
‼️ Could not resolve latest Minecraft version.
Rerun with --game-version <version>.
```

#### `--quiet` invalid Minecraft version (unattended)

##### Command used
`init --quiet --unattended --game-version 1.21.12 <other flags...>`

```
‼️ That Minecraft version does not exist.
Rerun with a valid --game-version.
```

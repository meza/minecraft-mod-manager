# `init` frame snapshots

This document specifies `init` as state-by-state terminal frame snapshots.

All requirements in `docs/interactions/consolidation/shared.md` apply.

## User goal

- Create a valid config and lock file for this folder.

## Success conditions

- A config file is written at the selected path.
- A lock file is created.
- The config points at an existing mods folder.
- The selected game version is valid.

## State model

States:
- INIT-01: start
- INIT-02: config exists check
- INIT-03: loader selection list
- INIT-04: Minecraft version text input
- INIT-ERR-VERSION: invalid Minecraft version (inline)
- INIT-ERR-LATEST-OFFLINE: latest cannot be resolved (inline)
- INIT-05: release types multi-select list
- INIT-06: mods folder text input
- INIT-ERR-MODS-FOLDER: invalid mods folder path (inline)
- INIT-07: confirm write
- INIT-08: write files
- INIT-09: success
- INIT-CANCEL: user cancels (`ctrl+c`, `esc`, or `q` as applicable)

## Frame snapshots

### INIT-03 Loader selection list (waiting for input)

#### Command used
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

### INIT-03 Loader chosen (collapsed into answered prompt)

#### Command used
`init`

```
? Which loader would you like to use? fabric
? What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1) 1.21.11

tab complete • enter accept • ctrl+c/esc quit
```

### INIT-ERR-VERSION Invalid Minecraft version (inline)

#### Command used
`init`

```
? Which loader would you like to use? fabric
? What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1) 1.21.12     <- That Minecraft version does not exist

tab complete • enter accept • ctrl+c/esc quit
```

### INIT-ERR-LATEST-OFFLINE latest cannot be resolved (inline)

#### Command used
`init`

```
? Which loader would you like to use? fabric
? What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1) latest     <- Could not resolve latest, enter an explicit version

tab complete • enter accept • ctrl+c/esc quit
```

### INIT-05 Release types selection list (waiting for input)

#### Command used
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

### INIT-05 Release types chosen (collapsed into answered prompt)

#### Command used
`init`

```
? Which loader would you like to use? fabric
? What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1) 1.21.11
? Which types of releases would you like to consider to download? beta,release
? Which folder should MMM download mods into? ./mods

tab complete • enter accept • ctrl+c/esc quit
```

### INIT-ERR-MODS-FOLDER Invalid mods folder (inline)

#### Command used
`init`

```
? Which loader would you like to use? fabric
? What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1) 1.21.11
? Which types of releases would you like to consider to download? release
? Which folder should MMM download mods into? ./mods-does-not-exist     <- Folder does not exist

tab complete • enter accept • ctrl+c/esc quit
```

### INIT-07 Confirm write (waiting for input)

#### Command used
`init`

```
? Which loader would you like to use? fabric
? What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1) 1.21.11
? Which types of releases would you like to consider to download? release
? Which folder should MMM download mods into? ./mods
? Write configuration now? (y/N): y

enter accept • ctrl+c/esc quit
```

### INIT-09 Success

#### Command used
`init`

```
Wrote ./modlist.json
Created ./modlist-lock.json
```

## Existing config behavior

### INIT-02 Config exists prompt

#### Command used
`init`

```
? ./modlist.json already exists. Overwrite it? (y/N):

enter accept • ctrl+c/esc quit
```

### INIT-02 Overwrite accepted (continues to loader selection)

#### Command used
`init`

```
? ./modlist.json already exists. Overwrite it? (y/N): y
? Which loader would you like to use?
❯ fabric
  forge

  ↑/k up • ↓/j down • / filter • esc clear filter • enter apply filter • esc cancel • q quit • ? more
```

### INIT-02 Overwrite declined and new config path entered

#### Command used
`init`

```
? ./modlist.json already exists. Overwrite it? (y/N): n
? Enter a new config path: ./modlist2.json

tab complete • enter accept • ctrl+c/esc quit
```

After this, MMM continues with INIT-03 using the new config path.

## Cancellation behavior

### INIT-CANCEL Cancel from a selection list (`q`)

#### Command used
`init`

```
? Which loader would you like to use?
❯ fabric
  forge

  ↑/k up • ↓/j down • / filter • esc clear filter • enter apply filter • esc cancel • q quit • ? more
q
Cancelled.
```

### INIT-CANCEL Cancel from a text input (`ctrl+c`)

#### Command used
`init`

```
? Which loader would you like to use? fabric
? What exact Minecraft version are you using? (eg: 1.18.2, 1.19, 1.19.1)

tab complete • enter accept • ctrl+c/esc quit
^C
Cancelled.
```

## Non-interactive behavior

Non-interactive mode MUST NOT prompt.

### Non-interactive success

#### Command used
`--non-interactive init <flags...>`

```
Wrote ./modlist.json
Created ./modlist-lock.json
```

### Non-interactive missing required input

#### Command used
`--non-interactive init`

```
Error: missing required values for init in non-interactive mode.
Next: run `mmm init` interactively or provide `--loader`, `--game-version`, and `--mods-folder`.
```

## Quiet flag

### `--quiet` success

#### Command used
`init --quiet`

```
Output: none
Exit code: 0
```

`--quiet` failure:
- Errors still print.

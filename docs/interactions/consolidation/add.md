# `add` frame snapshots

This document specifies `add` as state-by-state terminal frame snapshots.

All requirements in `docs/interactions/consolidation/shared.md` apply.

## User goal

- Add a mod and download its jar into the mods folder.

## Success conditions

- Config and lock are present.
- The mod is added to config.
- A matching jar is downloaded into the mods folder.

## State model

States:
- ADD-01: config present check (may delegate to init)
- ADD-02: platform selection (when args missing or platform invalid)
- ADD-03: id text input
- ADD-04: success
- ADD-ERR-NOT-FOUND: project not found
- ADD-ERR-NO-COMPAT: no compatible file
- ADD-ERR-DOWNLOAD: download failed
- ADD-CANCEL: user cancels (`ctrl+c`, `esc`, or `q` as applicable)

## Frame snapshots

### ADD-01 Missing config gate (delegates to init)

#### Command used
`add modrinth inventory-sorting`

```
No configuration found.
MMM looked for a config file at: ./modlist.json
This command needs config to continue.
? Initialize now? (y/N):

enter accept • ctrl+c/esc quit
```

If the user answers Yes, MMM MUST run the init flow exactly as specified in:
- `docs/interactions/consolidation/init.md`

After init success, MMM resumes add at ADD-02 or ADD-03 as needed.

### ADD-02 Platform selection list (waiting for input)

#### Command used
`add`

```
? Which platform should MMM use?
❯ modrinth
  curseforge

  ↑/k up • ↓/j down • / filter • esc clear filter • enter apply filter • esc cancel • q quit • ? more
```

### ADD-02 Platform selected -> id prompt

#### Command used
`add`

```
? Which platform should MMM use? modrinth
? What is the mod id? inventory-sorting

tab complete • enter accept • ctrl+c/esc quit
```

### ADD-04 Success

#### Command used
`add modrinth inventory-sorting`

```
Added Inventory Sorting.
```

## Error and recovery frames

### ADD-ERR-NOT-FOUND Project not found, retry id

#### Command used
`add modrinth not-a-real-id`

```
MMM could not find a project for modrinth not-a-real-id.
? Enter a different id? (y/N): y

enter accept • ctrl+c/esc quit
```

Then MMM returns to ADD-03:

#### Command used
`add modrinth not-a-real-id`

```
MMM could not find a project for modrinth not-a-real-id.
? Enter a different id? (y/N): y
? What is the mod id? inventory-sorting

tab complete • enter accept • ctrl+c/esc quit
```

### ADD-ERR-NO-COMPAT No compatible file, try alternate platform

#### Command used
`add modrinth <id>`

```
No file matched your loader and Minecraft version.
? Try the alternate platform? (y/N):

enter accept • ctrl+c/esc quit
```

If Yes:

#### Command used
`add modrinth <id>`

```
No file matched your loader and Minecraft version.
? Try the alternate platform? (y/N): y
? What is the mod id on the alternate platform? 325471

tab complete • enter accept • ctrl+c/esc quit
```

### ADD-ERR-DOWNLOAD Download failed, retry

#### Command used
`add <platform> <id>`

```
Download failed: <short reason>
? Retry? (y/N):

enter accept • ctrl+c/esc quit
```

## Cancellation frames

### ADD-CANCEL Cancel from platform selection (`q`)

#### Command used
`add`

```
? Which platform should MMM use?
❯ modrinth
  curseforge

  ↑/k up • ↓/j down • / filter • esc clear filter • enter apply filter • esc cancel • q quit • ? more
q
Cancelled.
```

### ADD-CANCEL Cancel from id input (`ctrl+c`)

#### Command used
`add`

```
? Which platform should MMM use? modrinth
? What is the mod id?

tab complete • enter accept • ctrl+c/esc quit
^C
Cancelled.
```

## Non-interactive behavior

Non-interactive mode MUST NOT prompt.

### Non-interactive success

#### Command used
`--non-interactive add modrinth inventory-sorting`

```
Added Inventory Sorting.
```

### Non-interactive missing config

#### Command used
`--non-interactive add modrinth inventory-sorting`

```
Error: no configuration file found at ./modlist.json.
Next: run `mmm init` to create one.
```

### Non-interactive unknown platform

#### Command used
`--non-interactive add not-a-platform inventory-sorting`

```
Error: unknown platform "not-a-platform". Use one of: curseforge, modrinth.
```

### Non-interactive project not found

#### Command used
`--non-interactive add modrinth not-a-real-id`

```
Error: could not find project for modrinth not-a-real-id.
Next: verify the id on the platform and try again.
```

### Non-interactive no compatible file

#### Command used
`--non-interactive add modrinth <id>`

```
Error: no compatible file found for modrinth <id> on loader <loader> for Minecraft <version>.
Next: try another mod version, change your Minecraft version, or use a different platform id.
```

### Non-interactive download failed

#### Command used
`--non-interactive add modrinth <id>`

```
Error: download failed for modrinth <id>.
Next: check your network and try again.
```

## Quiet flag

### `--quiet` success

#### Command used
`add --quiet modrinth inventory-sorting`

```
Output: none
Exit code: 0
```

`--quiet` failure:
- Errors still print.

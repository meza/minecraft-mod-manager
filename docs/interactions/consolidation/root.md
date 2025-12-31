# Root invocation frame snapshots

This covers:
- `mmm`
- `mmm --help`
- `mmm help`

The full root-level TUI is out of scope.

All requirements in `docs/interactions/consolidation/shared.md` apply.

## State model

States:
- ROOT-01: print root help
- ROOT-02: exit 0

## Frame snapshots

### ROOT-01 Root help output (no args)

#### Command used
`<no args>`

```
Minecraft Mod Manager (MMM)

Usage:
  mmm <command> [flags]

Commands:
  init, add, install, update, list, scan, prune, test, change, remove

Next:
  Run `mmm init` to create a configuration.
```

### ROOT-01 Root help output (`--help`)

#### Command used
`--help`

```
Minecraft Mod Manager (MMM)
...
```

### ROOT-01 Root help output (`help`)

#### Command used
`help`

```
Minecraft Mod Manager (MMM)
...
```

## Quiet flag

Help is the primary output and MUST still print.

#### Command used
`--quiet`

```
Minecraft Mod Manager (MMM)
...
```

## Non-interactive mode

Root help MUST still print.

#### Command used
`--non-interactive`

```
Minecraft Mod Manager (MMM)
...
```

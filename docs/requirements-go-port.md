# Requirements for Go Port

This document summarizes the features of the current TypeScript implementation and lists expectations for the upcoming Go port.

## Current Features

The CLI offers the following commands:

- `init` – interactively creates `modlist.json` with loader, game version, release types and mods folder settings.
- `add` – fetches a mod from CurseForge or Modrinth, downloads it and updates both configuration files. Supports version pinning and version fallback.
- `install` – ensures every mod listed in `modlist.json` is downloaded according to `modlist-lock.json`, reusing hashed versions when present.
- `update` – checks for new releases of configured mods and updates both the local files and lock file.
- `list` – prints the configured mods and whether they are currently installed.
- `change` – verifies if a different Minecraft version is supported, rewrites configuration and reinstalls mods.
- `test` – tests if a target game version is viable without altering the configuration. Exit codes signal success or failure.
- `prune` – removes unmanaged files in the mods folder. Honours `.mmmignore` patterns and supports a force option.
- `scan` – searches the mods directory for manually added files, matches them against the supported platforms and optionally updates the config.
- `remove` – deletes one or more mods from the config and filesystem. Supports glob patterns and a dry‑run mode.

Detailed information about what each command does and how it behaves is captured in
[the command reference](commands/README.md).

Configuration lives in `modlist.json` and `modlist-lock.json`. The lock file is entirely managed by the tool and should be committed alongside the main config. Ignored files can be listed in `.mmmignore`.

Global CLI flags include `--config` for selecting an alternative config file, `--quiet` to suppress prompts and regular logging, and `--debug` for verbose output. The Node version loads environment variables via `dotenv`, allowing values to be stored in a local `.env` file.

Environment variables include `CURSEFORGE_API_KEY`, `MODRINTH_API_KEY`, `POSTHOG_API_KEY` and `HELP_URL`. Defaults are provided in `src/env.ts`.

## Telemetry and Networking

Telemetry events are sent through PostHog (`src/telemetry/telemetry.ts`). Network requests such as GitHub release checks and Minecraft version verification run through a custom rate‑limited fetch utility (`src/lib/rateLimiter`).

## Platform APIs

The repositories handling CurseForge and Modrinth requests are explained in [platform-apis.md](platform-apis.md). These notes cover authentication, endpoints and the fallback behaviour used when versions are missing.

## Quality Guarantees

The project enforces 100% unit test coverage via Vitest (`vitest.config.ts`). An ADR defines that `console.log` and `console.error` must only be used from the `actions` folder (`doc/adr/0002-console-log-only-in-actions.md`).

## Bubbletea/Charm TUI Expectations

The Go port will rely on the Bubbletea ecosystem for interactive consoles. Implementations should follow Charm’s recommended patterns: models must be testable, pure functions should return commands, and views should avoid side effects. Error messages and user prompts must match the current CLI behaviour.


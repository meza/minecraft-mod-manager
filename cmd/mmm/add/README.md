# cmd/mmm/add

This package implements the `mmm add` command: take a platform + project ID, resolve a concrete mod file for the current config (loader, game version, release types), download it into the mods folder, and then update both `modlist.json` and `modlist-lock.json`.

## Start with the behavior docs

These files describe what the command must do. If you change behavior, update the docs first (or alongside the code) so reviewers have one source of truth.

- User guide: `docs/commands/add.md` (what users see and copy/paste)
- Command spec: `docs/specs/add.md` (behavior rules and edge cases)

## Code map

- `cmd/mmm/add/add.go`: cobra wiring + `runAdd` implementation
- `cmd/mmm/add/messages.go`: i18n-backed error message helpers used in non-interactive paths
- `cmd/mmm/add/tui.go`: Bubble Tea state machine used to recover from expected errors interactively
- `cmd/mmm/add/tui_test.go`: black-box TUI state snapshots (teatest + go-snaps)
- `cmd/mmm/add/add_test.go`: command behavior tests (config/lock writes, quiet vs interactive, telemetry)

## Execution flow (what happens at runtime)

At a high level, `runAdd` does:

1. Load or initialize config and lock:
   - if `modlist.json` is missing and `--quiet` is not set, create it via `internal/config.InitConfig` (calls `internal/minecraft.GetLatestVersion`)
   - ensure the lock exists via `internal/config.EnsureLock` (creates an empty lock file if missing)
2. Refuse to add duplicates (same platform + ID already in `modlist.json`).
3. Resolve a `platform.RemoteMod` via `internal/platform.FetchMod`:
   - expected typed errors map to UX flows:
     - `platform.UnknownPlatformError`
     - `platform.ModNotFoundError`
     - `platform.NoCompatibleFileError`
4. Download the resolved jar into the mods directory via `internal/httpClient.DownloadFile`.
5. Append:
   - a `models.Mod` entry to `modlist.json`
   - a `models.ModInstall` entry to `modlist-lock.json`
6. Record telemetry via `internal/telemetry.RecordCommand` (emitted once per session at shutdown).

## Interactive vs non-interactive behavior

The command only launches the interactive recovery flow when all of these are true:

- `--quiet` is not set
- stdin and stdout are terminals (checked via `internal/tui.ShouldUseTUI`)

If the user is piping/redirecting output, or running in CI, we intentionally stay non-interactive even if `--quiet` is false.

### Add TUI state machine

`tui.go` is a small finite state machine that exists to recover from the expected typed errors. A few rules are important when editing it:

- `esc` goes back one step; at the top-level it aborts.
- `ctrl+c` always aborts.
- When the state machine finishes successfully it returns the selected `platform.RemoteMod` plus the resolved platform + project ID.

## Testing and snapshots

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.

Snapshots for this command live at `cmd/mmm/add/__snapshots__/tui_test.snap`. Tests set `MMM_TEST=true` so i18n renders stable translation keys in snapshots.

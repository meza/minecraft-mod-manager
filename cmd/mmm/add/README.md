# cmd/mmm/add

This package implements the `mmm add` command: take a platform + project ID, resolve a concrete artifact using the current modlist settings (loader, game version, release types), download it into the mods folder, and then update both `modlist.json` and `modlist-lock.json`.

## Start with the behavior docs

Product intent defines what the command must do. Keep the user guide accurate when behavior changes.

- Product intent: [`docs/intent.md`](../../../docs/intent.md)
- User guide: [`docs/commands/add.md`](../../../docs/commands/add.md) (what users see and copy/paste)
- Shared terminal implementation and presentation: [terminal corpus](../../../docs/interactions/README.md)

## Code map

- `cmd/mmm/add/add.go`: cobra wiring + `runAdd` implementation
- `cmd/mmm/add/messages.go`: i18n-backed error message helpers used in unattended paths
- `cmd/mmm/add/interactive_flow.go`: Bubble Tea recovery flow for expected errors
- `cmd/mmm/add/prompt_models.go`: confirmation and text input prompt models used in the flow
- `cmd/mmm/add/config_prompt.go`: Bubble Tea prompt for missing-modlist recovery
- `cmd/mmm/add/output_model.go`: Bubble Tea output-only renderer for transcript output
- `cmd/mmm/add/progress_model.go`: Bubble Tea progress model for downloads
- `cmd/mmm/add/interactive_flow_test.go`: recovery flow snapshot tests (go-snaps)
- `cmd/mmm/add/add_test.go`: command behavior tests (modlist/lockfile writes, unattended vs interactive, telemetry)

## Execution flow (what happens at runtime)

At a high level, `runAdd` does:

1. Load or initialize the modlist and lockfile:
   - when `modlist.json` is missing, follow the shared missing-modlist gate and run the interactive init flow on confirmation
   - ensure the lockfile exists via `internal/config.EnsureLock` (creates an empty lockfile if missing)
2. Refuse to add duplicate mod configs (same platform + ID already in `modlist.json`).
3. Resolve a `platform.RemoteMod` via `internal/platform.FetchMod`.
4. Download the resolved artifact to a local file in the mods directory via `internal/httpclient.DownloadFile`.
5. Append:
   - a `models.Mod` mod config to `modlist.json`
   - a `models.ModInstall` lock entry to `modlist-lock.json`
6. Record telemetry via `internal/telemetry.RecordCommand` (emitted once per session at shutdown).

## Current prompt routing

The command only launches the interactive recovery flow when all of these are true:

- `--unattended` is not set
- stdin and stdout are terminals (checked via `internal/view.SupportsPrompting`)

Piped or redirected execution, including typical CI use, stays non-interactive even when `--unattended` is false. `--quiet` only suppresses non-essential output.

This is the current prompt gate. The target [execution profiles](../../../docs/intent.md#execution-modes-and-operator-intent) require a plain line-oriented recovery flow when interaction is available without control sequences, and plain append-only output whenever `--unattended` is set, including on a TTY. Those paths are not fully wired. Supplying complete arguments does not select unattended execution.

### Add recovery flow

`interactive_flow.go` is a small finite state machine that exists to recover from the expected typed errors. A few rules are important when editing it:

- `esc` goes back one step; at the top-level it aborts.
- `ctrl+c` always aborts.
- When the state machine finishes successfully it returns the selected platform + project ID.

## Testing and snapshots

See the root [verification guidance](../../../CONTRIBUTING.md#verification) for required checks and snapshot update instructions.

Snapshots for this command live at `cmd/mmm/add/__snapshots__/interactive_flow_test.snap`. Tests set `MMM_TEST=true` so i18n renders stable translation keys in snapshots.

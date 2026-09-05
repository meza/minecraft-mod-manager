# Command guides

These guides describe the Go-port target defined in [product intent](../intent.md), not a claim that every released build already implements it. Product intent is the authority when older documentation differs.

MMM helps server operators declare and maintain Minecraft Java Edition mods without a launcher. Players and modpack maintainers use the same commands. Each invocation finishes and returns to the shell.

## Choose a command

| Command | Use it to |
| --- | --- |
| [init](init.md) | Create configuration and an empty lockfile. |
| [add](add.md) | Declare a project, set its selection options and install it when resolvable. |
| [install](install.md) | Reconcile declared mods and managed files while preserving valid locked selections. |
| [update](update.md) | Reconcile using current settings and advance eligible unpinned selections. |
| [remove](remove.md) | Remove selected declarations, locks and managed files. |
| [list](list.md) | Inspect declared mods and their observed installation state. |
| [test](test.md) | Check platform-declared compatibility with a Minecraft target. |
| [change](change.md) | Change the configured Minecraft target and prepare its managed installation. |
| [scan](scan.md) | Recognize unmanaged jars and explicitly adopt selected artifacts. |
| [prune](prune.md) | Delete visible unmanaged jars with confirmation or force. |
| [help](help.md) | Read usage, options and defaults without setup. |
| [version](version.md) | Identify the running MMM release without setup. |
| [completion](completion.md) | Generate shell completion scripts and obtain supported suggestions without setup. |

## Configuration and setup

The default configuration is `./modlist.json`. Select another file with `--config` (`-c`):

```shell
mmm --config ./server.json list
```

Its lockfile uses the same basename: `server.json` uses `server-lock.json`. Relative mods-directory paths resolve against the configuration directory; absolute paths are supported. Use trusted configuration because it selects the directory MMM manages.

Every configuration-dependent command checks setup. If configuration is missing in a terminal, MMM offers [initialization](init.md). Accepting starts the shared init flow, then resumes the original command with the resulting configuration and its original operation inputs. Decline, cancellation or failure prevents the original operation from running. Without prompting, missing configuration produces setup guidance and a non-success result. Help, version and completion do not need setup.

An unreadable or invalid configuration is preserved and reported; it is not replaced by automatic initialization. Unknown properties are validation errors. Conflicting duplicate declarations can enter a separate interactive correction flow: choose which declaration to keep, save the accepted correction and resume. Without prompting, the conflict remains unchanged. Identical duplicates are deduplicated during modifying commands; inspection only reports them.

`list` and `test` do not rewrite metadata or repair files. An explicitly accepted initialization or correction is a separate flow, not part of their inspection work.

Invalid supplied values for supported fields receive consistent interactive correction, with other inputs preserved. Without prompting, the invalid request fails without changes. Malformed command syntax remains an invalid invocation.

## Execution modes

| Context | Behavior |
| --- | --- |
| Input and output are terminals | Ask for missing decisions and use interactive controls where helpful. |
| Terminal with `--unattended` | Never ask questions. Apply supplied policies and documented defaults, or fail when a required value or authorization is missing. Progress may remain dynamic. |
| Input or output is redirected | Never prompt. Emit plain, append-only output without animation or cursor-control sequences. |

Unattended or redirected execution grants no extra authority. `--force` has a specific meaning for each command:

| Command | Force authorizes |
| --- | --- |
| `init` | Reset existing configuration or an orphan lockfile without reset confirmation. |
| `add` | Save a verified but unresolved declaration, reporting incomplete installation. |
| `remove` | Remove explicitly selected mods without confirmation. |
| `prune` | Delete the identified unmanaged jars without confirmation. |
| `change` | Bypass platform-reported incompatibility and retain existing artifacts where eligible replacements are unavailable. |

For example, unattended removal still requires `--force`. Ordinary `install` and `update` already authorize their normal managed-file reconciliation. Force does not override exclusions, integrity checks or recovery requirements, or turn service failure into known incompatibility.

`--quiet` and `--non-interactive` are outside the target interface. Use `--unattended` to prohibit questions. Legacy `--lock-sync-add`, `--lock-sync-delete`, `--lock-sync-ignore` and `--lock-sync-skip` policies are replaced by declarative reconciliation. Change's legacy `--keep-config`, `--prune-config` and `--disable-skipped` options are also outside the target.

## Selection and lockfiles

`modlist.json` declares desired mods and their constraints. `modlist-lock.json` records exact resolved artifacts; the mods directory records what is actually installed. Keep configuration and lockfile together to reproduce a selection while its recorded artifacts remain available.

Selection matches platform metadata for Minecraft version, loader, release types and per-mod settings. Release-only is the default; a per-mod `allowedReleaseTypes` override takes precedence. Loader values do not introduce separate MMM modes or imply server-only filtering.

Pins remain fixed until explicitly changed. Mod version strings are not treated as reliable semantic versions. An unpinned update requires both a later publication date and different content within the applicable constraints. Display names may refresh from the platform; platform and project ID establish identity.

Per-mod `allowVersionFallback` defaults to false. When enabled, it searches nearest earlier Minecraft releases within the same release series and reports any fallback. It does not cross release series or apply snapshot fallback. If the newest matching artifact lacks required download or integrity information, report failure instead of selecting an older release to work around it.

Missing lock entries can be resolved by reconciliation. Corrupt or contradictory existing resolution evidence is preserved and reported with recovery guidance; it is not silently replaced by a new version. Inspection reports missing evidence without creating it.

`install` preserves valid locked selections. `update` selects newer eligible unpinned artifacts directly under the current declaration, without installing an intermediate old artifact first. Explicitly adopted or forced-retained compatibility exceptions remain specific to the artifact and target. See [the installation model](../intent.md#the-installation-model) and [version selection](../intent.md#sources-and-version-selection).

## File ownership and exclusions

Visible unmanaged jars are legitimate and reported without blocking unrelated operations. Discovering a jar does not adopt it. Adoption requires an explicit command option or accepted choice; repairing a managed file does not accept its modified content as the new locked selection.

`.mmmignore` is beside the configuration, with patterns relative to the mods directory. Matching files are excluded from scanning, unmanaged reporting, adoption, updates, removal and pruning. Invalid patterns stop mutation and identify the offending line. Files ending in `.disabled` receive the same exclusion treatment, even with force.

For the port, renaming a declared `example.jar` to `example.jar.disabled` does not satisfy the enabled declaration: installation may recreate `example.jar` alongside the untouched disabled file. First-class disabled-state management is deferred.

Discovery and pruning examine immediate files only, without recursing into subdirectories. An unrelated or protected destination collision fails the affected operation rather than overwriting that file. Independent work can continue when safe.

## Results and retry

The target distinguishes success or satisfied no-op, incomplete or failed execution, invalid invocation and interruption. A completed compatibility check with incompatibilities has its own non-success outcome; inconclusive checks are incomplete. Numeric assignments for these target categories are not defined by product intent; these guides do not invent new codes or preserve conflicting historical numbers.

Partial success is not overall success. Reports identify completed work, unresolved items, known reasons and useful next actions. Individual mods can fail independently during install, update and removal; completed work remains consistent and retries converge without duplicates. Version change instead has a coordinated preparation and switching boundary.

Incomplete downloads do not replace working artifacts. Metadata write failure is an operation failure. Harmless backup-cleanup failures receive bounded retries; if residue remains after the desired artifact and metadata are correct, report a visible warning and succeed. Uncertain metadata or an extra loadable jar remains a failure.

Concurrent modifying invocations against the same installation are unsupported in the port. MMM does not add process locking or waiting.

## Cancellation and terminal output

The first `Ctrl+C` requests safe cancellation. Stop new work and unfinished downloads, retain completed independent changes and finish necessary consistency or recovery work. Explain ongoing cleanup; required recovery has no automatic shutdown timeout. Warn that a second interruption forces termination and may leave recovery unfinished.

Settled results and answered decisions remain in the transcript exactly once, in completion order. The final summary adds counts and next steps without replaying the entire list. Normal completion, failure and safe cancellation restore terminal control and preserve pre-command shell history and durable results.

Scrolling away preserves the reading position while work continues. New results and pending prompts do not pull the reader to the bottom. Returning to the active end shows current state and resumes following output, including after resizing or list reorganization.

Messages and controls are localizable, with English fallback. Plain text and ASCII alternatives preserve meaning without depending on color or animation. A closed output pipe does not cancel an authorized operation; for example, updates continue if the reader in `mmm update | head` exits.

## Diagnostics and telemetry

`--debug` exposes additional diagnostics. `--perf` requests a local performance recording and `--perf-out-dir` controls its location. Diagnostic export failure does not turn a completed mod operation into failure.

Telemetry is enabled by default and can be disabled with `MMM_DISABLE_TELEMETRY`. Its permitted fields are bounded operational metrics and a documented stable machine identifier. It must not collect personal information, credentials, mod identities, private paths, arbitrary arguments or raw errors, including usernames embedded in other values. Telemetry failure does not fail the command. See the [collection boundary](../intent.md#operational-expectations).

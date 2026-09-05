# Minecraft Mod Manager product intent

## Authority and scope

This document defines the desired Minecraft Mod Manager (MMM) product. It is the authoritative baseline for product behaviour, user experience, architectural ownership and acceptance.

It describes the target, not a claim that the current implementation already satisfies it. Where older specifications, interaction frames, guides or implementation disagree, they must converge on this document. Existing tests establish what they exercise; they do not override product intent.

The baseline covers the Go port. The [next iteration](#next-iteration) section separates later capabilities from that release's promises. This document owns product expectations; contributor procedures and detailed implementation guidance remain in their respective documents.

## Purpose and audience

MMM helps Minecraft Java Edition server operators declare, install and maintain their mods without depending on a launcher. Modpack maintainers and players can use the same capabilities.

Operators come first. Most use MMM directly in a terminal, so interactive guidance is the default. Scripts are supported without making automation the default experience. Rich terminal controls improve clarity and convenience, but reliable operations and understandable results take priority over visual polish.

MMM is a command-oriented application. An invocation completes and returns control to the shell. A persistent full-screen manager, launching Minecraft and diagnosing Minecraft's runtime failures are outside the baseline.

The operator remains in control of the installation. MMM does not assume every jar comes from a supported platform or that platform compatibility metadata proves runtime behaviour.

## The installation model

MMM adopts npm's declarative installation model for the responsibilities below. This does not imply npm-compatible version syntax or dependency resolution in the Go port.

| Surface | Responsibility |
| --- | --- |
| `modlist.json` | Declares the mods the operator wants and the constraints used to select files. |
| `modlist-lock.json` | Records exact resolved artifacts so the declared selection can be reproduced. |
| Mods directory | Contains the actual installed files, which may differ from the declaration and lockfile. |

Configuration declares the Minecraft version, loader, mods directory, default allowed release types and desired mod entries. Each mod has a platform identity and project ID. It can also have an explicit version pin, release-type override and `allowVersionFallback` setting.

Stored mod names are platform display metadata and can refresh during installation and updates. A manually edited name does not become a protected operator label. Platform and project ID establish identity.

The lockfile records the selected artifacts, including their filenames, hashes and source information. It is resolved desired state, not an inventory that automatically adopts whatever files happen to exist.

`install` reconciles the declaration, lockfile and managed files:

- preserve locked resolutions that still satisfy the declaration
- resolve newly declared mods and entries whose constraints changed
- install missing locked files and repair drift in managed files
- remove lock entries and previously managed files for mods removed from the declaration
- update the lockfile to reflect the reconciled selection

For example, adding a mod to `modlist.json` does not require hand-editing the lockfile. The next `install` resolves and installs it. Removing that declaration makes the next `install` remove its previously managed file and lock entry.

A missing or incomplete lockfile is reconciled through the same operation. Existing valid resolutions are not discarded merely because other entries need resolution. A missing lockfile cannot provide the previous exact resolutions; MMM resolves the declared constraints and reports the outcome.

Missing entries differ from corrupt resolution evidence. Malformed lockfile data, contradictory resolutions or an existing artifact entry without required integrity information must be preserved and reported with recovery guidance. MMM must not silently reconstruct such evidence by selecting a potentially different version. Inspection reports what can be established without trusting corrupt entries.

`update` deliberately advances eligible resolutions. `install` does not upgrade an already-satisfied entry merely because a newer release exists.

Both operations share reconciliation: read the current declaration and existing resolution evidence, select the desired artifacts under the command's policy, then apply changes and persist consistent lock information. Changed configuration settings take effect during selection. Updating does not require installing an intermediate locked artifact before selecting its replacement.

Keeping configuration and lockfile together allows operators to reproduce the managed selection, provided the recorded artifacts remain available. Platform removal or download failure must be reported; MMM must not silently substitute another artifact for a locked one.

### Paths and installation boundaries

The default configuration is `./modlist.json`. `--config` selects an alternative file. The lockfile follows that file's basename: `server.json` uses `server-lock.json`.

Relative mods-directory paths resolve against the configuration directory. Absolute paths are supported. The operator chooses the installation boundary through trusted configuration; MMM confines each file operation to its intended target and does not follow unsafe paths outside that boundary.

A collision with an unrelated or protected file must not result in silent replacement. Fail the affected operation clearly while allowing independent operations to complete safely.

## Sources and version selection

The baseline supports CurseForge and Modrinth. Loader and release eligibility depend on the selected platform's capabilities. An unsupported combination must produce a clear explanation.

Loader values are platform metadata used for matching. They do not define separate MMM product modes or installation workflows. MMM does not infer a server-only eligibility filter from its primary audience.

Explicit Minecraft targets may be any version listed in Mojang's manifest, including snapshots and prereleases. The selected platform must still supply eligible artifacts; accepting a Minecraft version does not promise artifact availability. Defaults that select the latest Minecraft version select the latest stable release.

Version selection uses the configured Minecraft version, loader, allowed release types and per-mod settings. The default release policy is release-only. A per-mod release-type override takes precedence over the configuration default.

An explicit pin remains unchanged during ordinary updates. Changing a pin requires operator intent. MMM must not treat arbitrary mod version strings as reliable semantic versions.

For an unpinned mod, an update requires a later publication date and different file content, within the applicable selection constraints. A different display name or version string alone does not establish an update.

`allowVersionFallback` is an explicit per-mod choice to allow an older Minecraft release's artifact when a suitable target release is unavailable. It defaults to disabled. Fallback searches nearest earlier releases within the same release series, such as 1.20.2 to 1.20.1, without crossing to 1.19 or applying snapshot fallback. MMM reports when a fallback is selected. This permission does not enable unrelated overrides.

If the newest matching release lacks required download or integrity information, resolution fails with an explanation. It does not silently select an older release to work around that failure.

Platform metadata establishes declared compatibility, not whether Minecraft will successfully run the mod. MMM must distinguish an unavailable eligible artifact from a failed lookup or unavailable service.

Once adopted, a mod retains its platform association unless the operator explicitly changes it. MMM does not silently migrate it to another source.

## File ownership and adoption

Unmanaged mods are legitimate. Operators may install jars from unsupported sources or deliberately keep some mods outside MMM's management.

MMM reports visible unmanaged jars without blocking unrelated managed operations. It leaves those jars untouched during `install`, `update`, `remove` and `change`. Only a concrete collision or other failure affecting safe execution stops an affected operation.

Discovery never grants ownership. Adopting an unrecognised artifact, or accepting a modified artifact as the new locked selection, requires explicit intent. That intent may be an adoption command or flag, or an interactive choice.

Repairing an already-managed artifact during `install` is different from adopting its changed contents. A local modification does not silently redefine the locked version.

### Excluded files

`.mmmignore`, beside the configuration, makes matching files virtually invisible to MMM. Patterns apply relative to the mods directory. Excluded files are not scanned, reported as unmanaged, adopted, updated, removed or pruned.

Invalid ignore patterns stop mutation with an error identifying the offending line. A malformed pattern must not silently match nothing and expose intended exclusions to modification.

For the Go-port baseline, files ending in `.disabled` receive the same exclusion treatment. First-class disabled-mod management belongs to the next iteration.

This exclusion does not recognize a disabled counterpart as satisfying a declared enabled file. If `example.jar` is renamed to `example.jar.disabled`, `install` may recreate `example.jar` alongside it. Preventing that re-enabling behavior is outside the port baseline.

`--force` does not override either exclusion. Exclusion also does not grant permission to overwrite an existing file at a destination path. MMM must preserve that file if an operation would collide with it.

## Execution modes and operator intent

All execution modes use the same business operations, selection rules and safety guarantees. Presentation and the ability to ask questions differ.

| Context | Behaviour |
| --- | --- |
| Interactive terminal | When both input and output are terminals, ask for missing decisions and use interactive controls where helpful. |
| Terminal with `--unattended` | Never ask questions. Use documented defaults and supplied policies; fail clearly when required information or authorization is missing. Progress may still render dynamically. |
| Redirected input or output | Never prompt or depend on terminal interaction. Emit a plain, append-only transcript without animation or cursor-control sequences. |

An invocation does not acquire extra authority because it is unattended or redirected. `--unattended` and `--force` address different concerns and can be combined.

`--quiet` is outside the desired interface. Ordinary output must be concise enough for routine use while preserving requested results, failures and useful next steps. `--non-interactive` is not a separate mode flag; `--unattended` is the explicit no-prompt control.

Legacy `--lock-sync-add`, `--lock-sync-delete`, `--lock-sync-ignore` and `--lock-sync-skip` policies are outside this interface; declarative reconciliation owns those outcomes. Change's legacy `--keep-config`, `--prune-config` and `--disable-skipped` flags are also outside the baseline; forced retention has the single meaning below.

### Force has a documented meaning for each command

| Operation | What `--force` authorizes |
| --- | --- |
| `init` | Reset existing configuration or an orphan lockfile through initialization without an overwrite confirmation. |
| `add` | Save a verified project's requested declaration when no eligible artifact resolves, without claiming installation succeeded. |
| `remove` | Remove the explicitly selected mods without confirmation. |
| `prune` | Delete the identified unmanaged jars without confirmation. |
| `change` | Proceed despite platform-reported incompatibilities, retaining existing files when target replacements are unavailable. |

Force does not bypass file-integrity requirements, excluded-file protection or consistency and recovery guarantees. It does not turn an unknown lookup result into a known compatibility result.

For example, unattended removal still needs explicit authorization through `--force`. An explicitly requested `install` or `update` authorizes its normal reconciliation work; it does not need force merely because managed files will change.

## Shared setup and recovery

Every configuration-dependent command checks that the selected setup is usable before performing its operation. Help, version information and shell completion remain available without configuration.

If configuration is missing in an interactive invocation, the shared initialization flow takes over after the operator accepts the offer. On successful initialization, the original command resumes with the resulting configuration. If initialization is declined, cancelled or fails, the original operation does not proceed.

In unattended or redirected execution, missing configuration produces actionable setup guidance and a non-success result. It never leads to a hidden prompt or implicit initialization.

An existing but invalid or unreadable configuration is a different problem. MMM reports it and preserves the file unless the operator explicitly accepts a supported correction. It must not replace a broken declaration merely because initialization could produce a new one.

Unknown configuration properties are validation errors, not content that may silently disappear on rewrite. Validate required settings and supported values before mutation. Conflicting declarations for the same platform and project ID must not be resolved by choosing whichever entry is encountered first.

Interactive execution can offer a shared correction flow for conflicting duplicate declarations. The operator chooses which declaration to keep; MMM saves that accepted correction and resumes the original command. Without prompting, preserve the conflict and report how to correct it. Identical duplicate declarations are deduplicated automatically during modifying commands; inspection reports them without rewriting metadata.

A missing lockfile is not itself an uninitialized configuration. Commands handle it according to their responsibilities: reconciliation commands update it; inspection commands report the missing evidence without creating it.

`list` and `test` remain read-only operations. Offering a separate, explicitly accepted initialization or correction flow does not change that promise. Their own work does not repair files, rewrite declarations or reconcile lock entries.

Recovery is a shared capability. Each command must not independently implement its own version of setup or correction prompts, defaults, cancellation or return-to-command behaviour. A declined, cancelled or failed correction does not run the original operation against the unresolved invalid declaration.

Invalid supplied field values receive consistent treatment across commands. Interactive execution explains the invalid value and opens the corresponding field for correction, preserving other inputs. Unattended or redirected execution reports the error and makes no changes for that invalid request. This applies to supported field values such as loader, release types, Minecraft target and folder location; it does not turn malformed command syntax into a valid invocation.

Both `test` and `change` offer interactive correction of an invalid Minecraft target. If the latest-version lookup fails, they can instead collect an explicit target, provided it can be validated. Without prompting, report the lookup failure. An unavailable validation service must not be presented as proof that a supplied version is invalid.

## Command capabilities

### Initialize

`init` collects the loader, Minecraft version, allowed release types and mods location.

| Input | Meaning and default |
| --- | --- |
| `--loader`, `-l` | Platform loader value. There is no configuration default; collect it interactively or require it without prompting. |
| `--game-version`, `-g` | Minecraft target. Defaults to `latest`, meaning the latest stable release; explicit manifest-listed versions are supported. |
| `--release-types`, `-r` | Comma-separated allowed release types: `alpha`, `beta`, `release`. Defaults to `release`; at least one valid type is required. |
| `--mods-folder`, `-m` | Existing mods directory. Defaults to `mods`, relative to the selected configuration directory. Absolute paths are supported. |
| `--config`, `-c` | Shared option selecting the configuration path and its associated lockfile. |
| `--force`, `-f` | Skip confirmation of an existing metadata reset. It does not supply missing values or authorize invalid settings. |

In interactive execution, skip explicitly supplied valid fields and collect omitted fields with their defaults offered. For example, `init --loader fabric` still offers Minecraft version, release types and mods location. Defaulted values do not count as explicitly supplied values. Without prompting, use defaults directly and require the loader.

An invalid supplied value opens that field for correction in interactive execution, even if every field was supplied. Preserve other inputs. Empty or otherwise invalid explicit values must not bypass validation because their flags were present. Without prompting, invalid inputs fail before initialization writes metadata.

The collection order is loader, Minecraft version, release types and mods location, skipping valid supplied fields. Minecraft input supports version suggestions; release types use a selection requiring at least one choice. Text fields offer their defaults without discarding an invalid value that the operator needs to correct.

Initialization writes the configuration and an empty lockfile. It does not download or adopt mods. The selected mods directory must already exist and be usable; MMM does not offer to create it. Creating parent directories for a newly selected configuration file is separate from creating the mods directory.

Replacing existing configuration or an orphan lockfile requires explicit reset confirmation or `--force`. Explain that metadata will be reset and jars will neither be adopted nor deleted. Declining the reset offers another configuration path, with that consequence clearly labelled; it does not silently cancel initialization. Changing the configuration path also changes the base for relative mods paths. Revalidate the supplied location and preserve its text if correction is needed.

When init uses its interactive flow, it ends with a final confirmation before writing. This confirmation remains even with `--force`; force skips only the existing-metadata reset confirmation. Complete valid command-line inputs write directly when no reset confirmation is needed. Declining final confirmation cancels without saving.

For the port, init has no step-back navigation. Escape cancels initialization, except when a list control uses it to leave filtering. The shared flow uses these same controls when another command offers initialization. On success, that command resumes with the resulting configuration path and retains its original operation inputs.

### Add

`add <platform> <id>` adds an explicitly selected project to the desired set and reconciles its installation.

| Option | Effect |
| --- | --- |
| `--version` | Request an exact platform-specific pin. Modrinth uses its version number; CurseForge uses the artifact filename. |
| `--unpin` | Explicitly clear an existing pin. |
| `--allow-version-fallback` | Set the per-mod permission to use an older Minecraft release under the fallback rules. Explicit `false` removes that permission. |
| `--release-types` | Set the mod's `allowedReleaseTypes` override from a nonempty comma-separated list of `alpha`, `beta`, `release`. It does not change the installation-wide default. |
| `--force` | Persist a verified project's requested declaration when no eligible artifact resolves. It does not relax selection constraints or manufacture an installation. |

For an existing declaration, explicitly supplied settings replace the corresponding settings when the request resolves or an unresolved declaration is authorized through force; omitted settings preserve them. Without an existing per-mod release override, omission inherits the configuration default. `--unpin` explicitly clears the pin and reconciles to the newest eligible artifact. Combining `--unpin` with `--version` is invalid input and makes no changes.

For example, `add modrinth ID --release-types release,beta` can install an eligible beta while leaving the installation-wide release-only policy unchanged. The override becomes part of that mod's declaration.

A requested pin change remains in the declaration if downloading the selected artifact fails. Preserve the previous working jar and enough lock information for safe retry, and report that the installed state does not satisfy the new declaration. For example, pin B remains requested while jar A is still installed; a later `install` can complete the change. Retaining A does not make it a valid resolution of pin B.

For a first-time addition, a confirmed valid project with a resolved artifact remains declared if its download fails. Report the unresolved installation so a later `install` can finish. Invalid or unconfirmed project identities are not saved merely because they were supplied as input.

#### Lookup failure and search correction

Project-not-found results, and no-eligible-artifact results without force, offer the existing interactive search-correction sequence: show the failure, ask whether to modify the search with No as the default, select a platform, edit the failed ID and retry. Accepting the offer starts the selector on the other platform: Modrinth to CurseForge or CurseForge to Modrinth. The operator can still choose a platform. Prefill the ID with the failed value and retain the other supplied constraints. A further not-found or no-eligible result can repeat this flow.

For the port, this interaction edits platform and ID only. It does not loosen release types, pins, Minecraft target, loader or fallback permission. The richer interaction that explains alternative eligible files and offers constraint changes belongs to the next iteration. An invalid platform value uses the shared field-correction rule rather than being presented as a confirmed project-not-found result.

Authentication errors, timeouts and service failures remaining after any applicable automatic retries are reported and stop the addition. They do not trigger modify-search recovery: an unavailable service has not established that the project is absent. Without prompting, report unresolved lookups without opening correction controls.

If no eligible artifact resolves and search correction is declined, an ordinary addition saves no new declaration. A requested change from pin A to an unresolvable pin B likewise preserves A unless force authorizes the unresolved change. These are resolution failures, distinct from downloading an already-resolved artifact.

With `add --force`, a verified project can enter the declaration even when no eligible artifact resolves. An existing declaration can likewise record an explicitly requested unresolved pin. Preserve the requested constraints and any previous working artifact and recovery evidence. Do not invent an artifact or an incomplete lock entry as proof of installation. Force does not authorize saving an invalid or unverified project identity.

A forced declaration-only addition reports that configuration was saved and installation remains unresolved, and returns the non-success incomplete outcome. The operator can adjust the declaration, including its allowed release types, and retry `install`. No compatibility constraint is silently bypassed. Repeating an already-satisfied addition does not create duplicate declarations or lock entries.

### Install

`install` makes the managed installation match the declaration and its valid locked resolutions, as defined in the [installation model](#the-installation-model).

It repairs missing or mismatched managed artifacts, resolves changed declarations, and removes artifacts no longer declared. It neither adopts unrelated jars nor upgrades valid locked entries opportunistically.

Each mod can succeed or fail independently. The final report makes incomplete work clear and supports a safe retry.

### Update

`update` uses the shared reconciliation capability with an update selection policy. It incorporates added or changed declaration settings and selects newer eligible artifacts for unpinned mods before applying the resulting installation. It does not invoke an ordinary `install` as a prerequisite or reconstruct another command's runtime.

A missing old jar or unavailable old download does not prevent selecting and installing an eligible newer artifact directly. Explicit pins remain binding, including a pin changed in the declaration since the lockfile was written.

Each replacement is prepared before the previous working file is removed. A failed replacement preserves the previous working artifact. Successful updates remain installed even when another mod fails.

Pinned mods remain pinned. A run is a successful no-op when the declared installation is already satisfied and there are no eligible updates.

### Remove

`remove` accepts selected mod IDs or names, including supported case-insensitive glob patterns. It removes the matched declarations, lock entries and managed artifacts. Matching the same mod more than once does not repeat the operation.

Interactive removal confirms the intended selection unless force is supplied. Without prompting, force is required. An already-absent target is a successful no-op.

A failed file removal must not be presented as a completed removal. Preserve enough consistent state to identify and retry the remaining work. Unrelated and excluded files remain untouched.

### List

`list` reports the declared mods and their observed installation state. It distinguishes installed artifacts from missing files, missing resolution evidence and content mismatches. A matching filename alone does not prove installation correctness.

Inspection lists use stable, case-insensitive name ordering, with platform and project ID as tie-breakers. Visible unmanaged files are reported without preventing the managed list from being shown.

The operation makes no changes to configuration, lockfile or mod files.

### Test compatibility

`test` checks platform-declared eligibility for a target Minecraft version. It reports compatible, incompatible and inconclusive outcomes distinctly. If no target is supplied, the latest Minecraft release is the default.

The report explains what could not be determined when an API or lookup fails. It does not claim to have tested Minecraft itself. An incompatible result can act as a scriptable compatibility gate; an inconclusive result must not masquerade as success.

If a run finds both known incompatibilities and inconclusive checks, its overall outcome is incomplete. Preserve both kinds of findings in the report.

The operation makes no changes to configuration, lockfile or mod files.

### Change Minecraft version

`change` changes the configured Minecraft version and prepares the corresponding managed installation. The default target is the latest Minecraft release. When the requested target already equals the configured Minecraft version, the command succeeds as a no-op even if managed files need repair. Same-target change does not perform installation; `install` owns that repair.

Ordinary change requires the target selection to pass the platform compatibility checks. Prepare the target artifacts before switching the working installation. If preparation fails, preserve the original installation and configuration.

`change --force` bypasses the platform-reported incompatibility gate. Install available eligible replacements and retain the currently installed file where no target replacement is available. Retained files remain declared and locked, and the configured Minecraft version changes to the requested target.

An explicitly retained artifact is a valid locked selection for that target despite the missing platform compatibility declaration. A later `install` preserves or reproduces it without requiring force again. `update` can replace it with a newer eligible artifact; the absence of such an artifact does not invalidate the retained selection.

This authorization belongs to the retained artifact and target, not to arbitrary future resolutions. A replacement selection or a change to the declaration's resolution constraints requires reassessment under the applicable selection and force rules.

Force does not disable retained mods, remove their declarations or choose a skipped-mod disposal policy. The report identifies retained artifacts and their declared compatibility limitations. The operator can launch Minecraft and decide what to remove based on actual behaviour.

If no replacement and no existing file are available, MMM cannot manufacture a satisfied installation. Report that unresolved artifact and treat preparation as incomplete. Similarly, a failed request or download is not permission to silently claim a forced change succeeded.

Once switching starts, finish the necessary consistency work or attempt recovery to the previous installation. If recovery is incomplete, report the actual remaining state and next action. Never claim that nothing changed unless that is established.

### Scan and adopt

`scan` identifies visible candidate jars in the mods directory and reports known, unknown and uncertain results. It prefers Modrinth by default, accepts an explicit platform preference, and uses the other platform as fallback.

Discovery examines only immediate files in the configured directory. It does not recurse into subdirectories. If retries against the preferred platform are exhausted after timeout or connection failure, try the other platform. A conclusive match can still succeed; unresolved lookup failures remain uncertain.

An explicit `--add` request or accepted interactive adoption choice allows recognized artifacts to enter the declaration and lockfile. Adoption records the actual discovered artifact. It does not download a newer version as part of adopting it.

Unknown or uncertain matches must not be silently adopted. Declining adoption preserves the files and metadata. Already-managed artifacts are not duplicated.

When multiple discovered jars identify the same project, keep the existing valid locked selection. If there is no locked selection, interactive adoption asks which artifact to adopt. Without prompting, leave that project unresolved while adopting independent unambiguous projects. Other jars remain unmanaged; processing order must not decide ownership.

Adoption does not silently override an existing pin. Report a conflicting discovered version and require explicit consent to change the pin; without prompting, leave that project unchanged. Explicit adoption of an otherwise undeclared artifact may authorize that exact artifact despite platform-reported incompatibility with the configured target. Report the limitation and preserve or reproduce the adopted selection on later installs. This authorization is specific to the artifact and target, as with forced retention, and does not waive integrity requirements or authorize arbitrary future selections.

### Prune

`prune` explicitly removes visible unmanaged `.jar` files. It shows the intended deletion set and requires confirmation or `--force`.

Ignored files, `.disabled` files, managed artifacts and unrelated directory contents remain untouched. Pruning does not recurse into subdirectories. Without a lockfile, it must refuse deletion and explain that `install` is needed to establish resolution evidence. An empty deletion set is a successful no-op. Individual failures are reported without undoing independent successful deletions.

### Help and version

Help explains commands, defaults, constraints and force semantics without requiring setup. Bare `mmm` shows help. Version output identifies the running MMM release. Both remain usable when an installation is missing or broken.

### Shell completion

`completion bash`, `completion zsh`, `completion fish` and `completion powershell` emit shell integration scripts. `--no-descriptions` omits completion descriptions. Script generation and command, flag and supported value completion are utilities independent of initialization; they must not prompt for setup or modify installation metadata.

Init supplies loader and release-type value suggestions through static shell-completion callbacks, without network or configuration access. These differ from Minecraft suggestions inside the interactive init flow, which can use Minecraft metadata. Failure to register optional completion support must not prevent ordinary init use.

## Failure, retry and cancellation promises

In the baseline, individual mods are independent artifacts. Install, update and removal can retain successful work when another mod fails. A Minecraft version change has a coordinated preparation and switching boundary because it changes the installation's common target.

Repeated operations must be safe:

- unchanged inputs and upstream state produce no additional material changes once the requested outcome is satisfied
- retries after partial failure converge on the requested state without duplicate entries
- already-satisfied work succeeds as a no-op
- metadata and files remain consistent with completed work

An update may find a newer release on a later invocation. Retry safety does not freeze upstream state or promise byte-identical logs and timestamps.

MMM verifies artifact integrity and does not replace a working file with an incomplete download. A metadata write failure is an operation failure, not an optional logging problem. Stop work that cannot safely continue and report recovery needs accurately.

Retry cleanup failures before involving the operator. If bounded retries cannot remove a non-loadable backup after the desired artifact and metadata are correct, the operation succeeds with a visible warning identifying the residue and required cleanup. Uncertain metadata or an extra loadable jar is still an operation failure.

Concurrent modifying invocations against the same installation are unsupported in the port. The baseline does not add process locking or waiting; consistency and retry promises assume operations are not competing with another writer.

A broken output pipe, such as a reader exiting during `mmm update | head`, does not cancel the authorized operation. Continue the operation and required consistency work even though the reader no longer receives output.

Cancellation stops scheduling new work and cancels unfinished downloads. Retain completed independent changes. Complete necessary metadata consistency or version-switch recovery before returning control to the shell.

The operator may have to wait for safe cancellation. MMM explains ongoing cleanup instead of claiming to have stopped while still mutating the installation. Required consistency or recovery work has no automatic shutdown timeout. Normal cancellation and failure restore terminal control as well as installation consistency.

After the first `Ctrl+C`, explain that pressing it again forces termination and may leave recovery work unfinished. A second interruption is an explicit emergency exit, outside the safe-cancellation guarantee. MMM must not claim successful recovery merely because the process exited.

### Results and exit status

Exit status is a stable part of the interface. It distinguishes successful satisfaction, incomplete or failed execution, invalid invocation and interruption. A completed compatibility check with incompatible results has a distinct non-success outcome. Inconclusive checks must also be identifiable as incomplete.

Partial success is not overall success. Reports identify what completed, what failed or remained unresolved, and what the operator can do next. Detailed numeric codes belong in command reference documentation and acceptance scenarios; these outcome distinctions are mandatory.

An error explains the affected mod or file, the reason when known, and the next useful action. Do not label network failure as incompatibility, duplicate a handled error through multiple layers, or obscure the result with unrelated usage output.

## Terminal experience

### Active display and permanent transcript

The terminal experience has two distinct lifetimes. Active work can repaint and reorganize. Committed history cannot.

Pending items, progress indicators, selection controls and other temporary state may change in place. When a result settles, emit it once into the permanent transcript in completion order. Use the same result meaning and format as non-interactive output.

Later repaints must not rewrite, reorder or erase committed results. A corrected or additional outcome is a new explicit event, not a silent rewrite of history. The final summary adds counts, failures and next steps without replaying the entire result list.

Answered prompts leave a concise record of the decision. Temporary option lists and animation do not need to remain. Decisions that matter to understanding the operation must survive completion and cancellation.

The normal terminal history must preserve pre-command shell output and the command's durable results after exit. Temporary alternate screens are allowed, but a separate transcript viewer is not required to recover that history.

Long lists must remain accessible. They do not have to fit on one screen. The active display can group or sort work differently from the transcript; list reorganization must not move already-committed output.

### Scrolling and returning to active work

Follow new output while the operator is at the active end of the transcript. Once they scroll away, preserve their reading position while work continues.

Settled results continue to commit exactly once. Progress updates, completed items and newly required prompts must not pull the operator back to the bottom.

Returning to the active segment shows the current state, including any pending prompt. It must not reveal stale frames, duplicate results, missing lines or misplaced controls. Reaching the bottom resumes following new output.

These transitions must survive resizing, changes in active-list grouping, and items moving from pending to completed. A prompt can wait while the operator reads history; the application does not own that reading position.

### Controls, language and accessibility

Related capabilities use the same input semantics, defaults, confirmation behaviour and help presentation wherever they appear. Explicit inputs are preserved when a recovery flow asks for another value.

The first `Ctrl+C` requests safe cancellation; a second can force termination as described in the cancellation contract. Escape provides consistent back or cancel behaviour appropriate to the active control. Ordinary text input must not unexpectedly treat a letter as cancellation. Progress-only controls may expose a documented quit shortcut.

User-facing messages, choices and input hints are localizable. Missing translations fall back to English. Localized confirmation tokens remain unambiguous and usable; translation must not change the underlying action.

Colour and Unicode improve readability where supported. Plain text and ASCII alternatives must preserve meaning. Success and failure must not be distinguishable only through colour, animation or an icon.

## Component ownership and architecture

MMM requires a strict component hierarchy. Each shared capability has one owner for its state, behaviour, interaction and output. Sharing styles alone does not satisfy this requirement.

| Layer | Owns |
| --- | --- |
| Terminal session | Input routing, focus, scrolling, active rendering, transcript commits and terminal restoration. |
| Command flow | The requested operation's sequence and composition of capabilities, including shared recovery and return to the original command. |
| Capability component | A reusable interaction end to end, such as confirmation, selection, initialization or mod-operation progress and results. |
| Visual primitives | Consistent controls, styling, icons and presentation building blocks. |

Business operations are independent of terminal rendering. They resolve artifacts and perform filesystem and metadata work through explicit boundaries. Interactive and unattended flows invoke the same operations and consume their outcomes.

Command flows supply domain data and decisions to components. Components report user choices and operation outcomes through explicit contracts. The terminal session is the authority for committing permanent output and coordinating presentation; child components do not independently seize the terminal.

Command-specific orchestration can remain in command packages. Reusable MMM capabilities have shared ownership even when their first use appears in one command. This is not a requirement to build a general-purpose terminal framework.

A command must not duplicate a shared capability, reconstruct another command's private runtime, or add a local workaround for a shared behaviour defect. Improvements to confirmation, scrolling, progress or transcript handling must reach all consumers through the owning component.

Bubble Tea and its existing controls support composition. Their presence alone does not prove this hierarchy exists. Acceptance must demonstrate that components behave consistently within real command flows.

## Operational expectations

Windows, macOS and Linux are first-class environments. Paths, terminal handling, cancellation and filesystem behaviour must work on each supported system. The Go distribution must not require Node.js, a launcher or a persistent service to run ordinary commands.

Network access respects shared service limits and standard proxy configuration. `HTTP_PROXY`, `HTTPS_PROXY` and `NO_PROXY` apply to outbound HTTP requests, including API calls and downloads, and user documentation explains their use. Avoid unnecessary requests and distinguish service unavailability from invalid operator input.

When an invocation first needs the Minecraft version manifest, attempt to refresh it using the normal bounded retry policy. Reuse the resulting metadata throughout that invocation. Persist successfully validated manifests locally in a system-appropriate location, together with their fetch time, for reuse across invocations.

If refresh fails, use a valid cached manifest regardless of its age. There is no hard expiry. Warn that cached metadata is being used and identify when it was fetched. A request for `latest` may use the cached latest stable release; name the selected version and warn that a newer release may exist.

An explicit version present in the cached manifest can be validated. If it is absent and refresh failed, report that MMM cannot validate the version, not that the version is invalid. If refresh fails and the cache is missing or corrupt, report an actionable failure rather than inventing validation evidence.

These rules apply equally to interactive, unattended and redirected execution. Cache fallback alone does not make an operation unsuccessful. This does not promise full offline operation or permit invented version validation.

Official distribution access defaults should allow ordinary use without obtaining personal API credentials. Explicit runtime overrides remain available. Credentials must not appear in user output, diagnostic artifacts or telemetry.

Diagnostics help operators explain failures without overwhelming routine output. `--debug` exposes additional diagnostic detail. `--perf` explicitly requests a local performance recording, with `--perf-out-dir` controlling its location. A diagnostic export failure must not change a successfully completed mod operation into a failure.

Telemetry remains enabled by default with an explicit `MMM_DISABLE_TELEMETRY` opt-out. Its collection policy must document bounded operational metrics and the stable machine identifier; do not describe that identifier as absent or imply unqualified anonymity.

Permitted telemetry consists of app version, OS, execution mode, command outcomes and timings, Minecraft version, loader, mod count, request/download counts, bounded error categories and the documented stable machine identifier. Emit only explicitly permitted fields.

Telemetry must not collect personal information, credentials, mod identities, private filesystem paths, arbitrary command arguments or raw error text. Usernames must not leak indirectly through paths, URLs, error messages or other field values. An allowed field name is not permission to include personal information in its value. Telemetry failure must never fail the requested operation or prevent the operator from regaining control. Detailed local performance recordings remain separately requested.

## Acceptance and evidence

Product acceptance describes what an operator does and can observe. Third-person BDD scenarios exercise the real MMM process. tui-test owns terminal input, waiting, screen state, scrolling, resizing and process lifecycle.

Use semantic outcomes for messages, exit status and filesystem effects. Use rendered terminal snapshots when the requirement depends on complete layout or styling. Stable localization keys verify message selection; real-locale checks verify translated layout and input behaviour.

Network-dependent scenarios require deterministic fixtures that work across the process boundary. An in-process HTTP fixture cannot establish that a separately launched command is deterministic or network-free.

The baseline must have evidence for these journeys:

| Journey | Required observation |
| --- | --- |
| Missing setup | A command hands control to initialization on acceptance, then resumes. Decline, cancellation and failure do not run the original operation. |
| Init parameters and correction | Valid supplied fields skip prompts; omitted fields offer defaults interactively and use them directly without prompting. Invalid explicit values enter consistent correction or fail without changes when prompting is unavailable. Other values survive correction. |
| Init reset and confirmation | Existing configuration and orphan lockfiles require reset authority. Declining reset selects another path and revalidates relative locations. A collected setup needs final confirmation even with force; cancellation writes nothing. |
| Declaration validation and correction | Unknown properties fail before mutation; conflicting duplicates require an accepted correction; identical duplicates are deduplicated only by modifying operations. Accepted shared correction resumes the original command, including inspection. |
| Inspection | `list` and `test` leave existing installation files and metadata unchanged. Explicit initialization or correction is observed as a separate flow. |
| Declarative install | Added and removed declarations reconcile correctly; valid resolutions stay fixed; an incomplete lockfile is updated without losing unrelated resolutions. |
| Resolution evidence | Missing entries resolve normally; corrupt or contradictory existing evidence is preserved and reported instead of silently replaced. |
| Explicit selection changes | Supplied add settings replace existing settings, omitted settings preserve them, and unpin clears the pin. A failed pin-change download retains the requested pin and previous working jar with an unsatisfied-state report. A confirmed first-time addition remains declared after download failure; invalid or unconfirmed identities are not saved. |
| Add lookup recovery | Not-found results, and no-eligible results without force, offer modify-search, start on the other platform and retain the ID and constraints. Service failure after retries ends the request without treating it as absence. Unresolved ordinary additions do not save new declarations. |
| Add release policy and force | A supplied release-type list changes only the mod's override. Force can persist a verified unresolved declaration or pin change, retaining existing working files and reporting incomplete installation without inventing a locked artifact. |
| Update selection | Changed declarations participate directly in update resolution; no intermediate artifact is required. An unusable newest matching release fails rather than silently selecting an older release. |
| Ownership | Unmanaged jars coexist without blocking work; adoption requires intent; ignored and `.disabled` files remain protected. |
| Disabled-file baseline | A disabled jar remains untouched even when installation recreates its declared enabled counterpart. First-class disabled-state preservation is not inferred. |
| Scan ambiguity and fallback | Provider failure after retries permits fallback; multiple artifacts preserve the lock selection or require a choice; pin conflicts need consent; explicitly adopted incompatible artifacts remain reproducible. |
| Partial failure and retry | Successful independent work remains consistent; failed replacements preserve old files; retry completes remaining work without duplicates. |
| Forced version change | Available replacements install, unavailable replacements retain existing files, and retained mods are neither disabled nor removed. |
| Interrupted version change | Preparation preserves the original state; interruption during switching completes consistency or reports the actual recovery outcome. |
| Cancellation and cleanup | Safe recovery has no automatic timeout; a second interruption follows an explicit force-exit warning. Cleanup retries precede warnings about harmless residue. Broken output pipes do not cancel authorized operations. |
| Compatibility outcomes | Explicit manifest-listed targets are accepted; mixed incompatible and inconclusive findings produce an incomplete outcome. Same-target change is a no-op even when files need repair. |
| Target recovery | Test and change offer the same correction for invalid targets and allow an explicit validated target when latest lookup fails. Without prompting, they report the failure. |
| Modes and authority | No-prompt runs never wait for input or imply force; redirected runs contain no terminal-control output. |
| Transcript lifetime | Existing shell history and settled results survive exit exactly once, including failure and cancellation. |
| Scroll round trip | While work continues, scroll away, receive more results and a pending prompt, resize, then return. Position remains stable and the current active state returns correctly. |
| Shared capabilities | The same confirmation, selection, progress and recovery behaviours hold across consuming commands. |
| Shell completion | Supported shell scripts and init value suggestions work without setup prompts or metadata changes; static value completion needs no network. |
| Telemetry boundary | Only permitted operational fields are emitted; personal information, including usernames embedded in values, never enters the payload. |

Test normalization must not manufacture the required screen by reordering results, injecting missing headers, removing duplicates or substituting model output for terminal observations. Historical snapshots and statement coverage cannot replace evidence of these journeys.

## Next iteration

### Guided constraint resolution

When a project exists but no artifact satisfies the current constraints, a richer interaction should explain which constraints exclude available files and offer explicit per-mod changes. For example, a Minecraft-compatible beta may be available while the installation permits releases only. The operator can choose a per-mod `allowedReleaseTypes` override without changing the installation-wide default.

The port provides explicit add release-type input and forced declaration-only addition. It does not promise interactive comparison of alternative constraints or silently select an excluded artifact.

### Dependency management

Automatic dependency resolution and installation are intended for the next iteration. They are outside the Go-port baseline because dependency relationships introduce additional selection, ordering and failure semantics.

For the port, operators explicitly declare each mod. MMM must not silently introduce dependencies. Later dependency management must define how dependencies affect pins, removal, updates and the current independent-failure model.

### Managed enabled and disabled state

The next iteration treats disabled mods as managed members of the installation. The lockfile records enabled state and enough artifact identity to locate enabled and `.disabled` filenames accurately.

Listing exposes that state. Installation preserves it, updates replace a disabled artifact without enabling it, and explicit removal handles either state. Enable and disable operations expose the capability directly.

Operators may toggle a mod through another tool or a manual rename. The design must recognize such changes rather than blindly restoring a stale lockfile state. `.mmmignore` remains a separate exclusion mechanism.

### Longer-term directions

Richer version constraints are desirable when upstream version information can support them reliably. The port does not promise semantic-version ranges.

Runtime compatibility diagnosis and assisted remediation may follow later. For now, the operator determines whether retained mods work after a forced version change.

A dedicated transcript viewer, new artifact sources and self-update are not baseline requirements. They need separate product decisions and must not become hidden prerequisites for the promises in this document.

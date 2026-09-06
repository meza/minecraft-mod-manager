# Shared terminal component examples

These annotated examples illustrate the Go-port target defined by [product intent](../../intent.md). They are conceptual examples for component design, implementation, and acceptance work. They are not screenshots of the current application, prescribed English copy, or an exact output-format contract.

Product intent owns behavior and outcomes. The [command guides](../../commands/README.md) own operator workflows. This page shows how shared terminal capabilities can present those outcomes without giving commands different business rules.

## How to read the examples

Each frame has one of three roles:

| Frame role | Lifetime |
| --- | --- |
| **Active display** | Temporary state that may repaint while work or input is pending. |
| **New durable events** | Settled results or relevant resolved decisions appended once to the permanent transcript. |
| **Final transcript** | The durable history left in the ordinary terminal when the command returns to the shell. |

Examples often show only the new lines relevant to a transition. Earlier durable events remain above them in the terminal even when they are omitted from the next code block. Removing a completed row from the active display never erases its durable event.

Temporary controls can adapt to terminal capabilities. Durable records use the same text and formatting across execution modes under the same locale and character capabilities. Text must preserve meaning when Unicode or color is unavailable. Placeholders such as `<confirm-short>` represent localized content, not literal output. The examples illustrate that shared contract without prescribing exact English copy.

Rich frames below apply to TUI presentation. ASCII UI symbols retain equivalent controls when Unicode is unsupported. Plain interactive execution collects equivalent choices through line-based questions; unattended and non-interactive execution never ask questions. All profiles emit the shared durable records. See the [execution-mode matrix](../../intent.md#execution-modes-and-operator-intent).

## Selection focus, selected values, and answered prompts

A selection control must distinguish the row under the cursor from values already selected. The pointer marks focus; the check mark marks selection. Color can reinforce these states but cannot be their only distinction.

**Active display — a multiple-selection control is waiting for input:**

```text
? Which release types should be allowed?
    alpha
❯   beta
  ✓ release

  ↑/↓ move • space toggle • enter accept • esc cancel
```

Here, `beta` has focus and `release` is selected. Moving focus does not change the selection. Toggling `beta` selects it; accepting the control settles the decision.

**New durable event — the accepted control produces one decision record:**

```text
Allowed release types: beta, release
```

The temporary list disappears from the active display, while the decision stays in the transcript. Supplying the same release types as arguments produces that same durable line without an invented question. A single-choice control follows the same pattern:

**Active display:**

```text
? Which platform should MMM use?
❯ Modrinth
  CurseForge
```

**New durable event:**

```text
Platform: Modrinth
```

Resolving Modrinth from a supplied argument or documented default produces the same `Platform: Modrinth` decision record where that decision is relevant. The [init guide](../../commands/init.md) defines field collection and defaults. The [add guide](../../commands/add.md#lookup-and-eligibility-results) defines when platform and project-ID correction is offered.

## Destructive confirmation with a localized preview

A destructive confirmation first previews the exact selection. The localized option labels and short tokens come from one option set, while the safe default is owned by the behavior.

**New durable events — preview committed before asking:**

```text
Files proposed for deletion:
  old-library.jar
  unused-addon.jar
```

**Active display — pending confirmation:**

```text
? Delete these 2 files? (<confirm-short>/<cancel-short>) [default: <cancel-label>]
```

Empty input chooses the displayed safe default. The component accepts localized short tokens and full labels without assuming English initials.

**New durable event — accepted decision collapses to a record:**

```text
Deletion of these 2 files: authorized
```

After that event, deletion progress may begin. If the operator declines, the declined decision and the fact that nothing was deleted remain in the final transcript. When command-specific force supplies equivalent authority for the same selection, the prompt is omitted, but the preview, resolved authorization record and settled results remain the same. Plain execution emits those durable records without repainted progress. See the [remove](../../commands/remove.md) and [prune](../../commands/prune.md) guides for their authorization rules.

## Bounded and numeric progress

In the TUI, when a byte total is known, a progress bar shows filled and unfilled cells and is paired with numeric progress. Item counts can orient the operator across a batch. These temporary frames do not become additional progress lines in pure CLI output.

**Active display — two items are still running:**

```text
Installing mods (5/12 settled)

⬇ Fabric API (fabric-api) [modrinth]
[#####-----] 50% (512 KiB / 1 MiB)

⏳ Mod Menu (modmenu) [modrinth]
Waiting for download metadata
```

An indeterminate spinner or status label is appropriate when no trustworthy total exists. The component must not invent a percentage.

As each item settles, it leaves the active display and emits one result:

**New durable events:**

```text
✅ Fabric API (fabric-api) [modrinth] installed
❌ Mod Menu (modmenu) [modrinth] download failed: connection reset; retry with mmm install
```

**Final transcript — summary added after the already committed events:**

```text
Installation incomplete: 11 satisfied, 1 failed.
```

The final summary does not print the full completed list again. The [install](../../commands/install.md) and [update](../../commands/update.md) guides define selection, partial success, and retry behavior.

## Recognition from filename to identity

Recognition begins with a filename and may settle as recognized, unknown, or uncertain. Unknown means both supported platforms conclusively produced no match. Uncertain means a service or lookup failure prevented a conclusion.

**Active display — lookup is underway:**

```text
Recognizing files
⠋ inventorysorter.jar
⏳ private-addon.jar
⏳ flaky-service-result.jar
```

**New durable events — each filename retains its relationship to the result:**

```text
RECOGNIZED  inventorysorter.jar -> Inventory Sorting (inventory-sorting) [modrinth]
UNKNOWN     private-addon.jar -> no match on Modrinth or CurseForge
UNCERTAIN   flaky-service-result.jar -> CurseForge lookup unavailable after retries
```

Unknown and uncertain must remain visibly different in Unicode, ASCII, and plain text. Recognition does not itself grant ownership, and unknown or uncertain files are not silently adopted. Multiple candidates and pin conflicts remain business decisions described by the [scan guide](../../commands/scan.md).

## Compatibility: incompatible and inconclusive

Compatibility presentation distinguishes a known absence of an eligible artifact from an inability to complete the check.

**Active display — remaining checks continue:**

```text
Checking eligibility for Minecraft 1.21.1
⠋ Inventory Sorting (inventory-sorting) [modrinth]
⏳ Fabric API (fabric-api) [modrinth]
```

**New durable events:**

```text
COMPATIBLE    Inventory Sorting (inventory-sorting) [modrinth]
INCOMPATIBLE  Legacy Map (legacy-map) [curseforge] -> no eligible artifact for 1.21.1
INCONCLUSIVE  Fabric API (fabric-api) [modrinth] -> service unavailable after retries
```

**Final transcript — a mixed result remains incomplete:**

```text
Compatibility check incomplete: 1 incompatible, 1 inconclusive.
Review both sections before changing the installation.
```

The wording does not claim that Minecraft was launched or that runtime compatibility was tested. See the [test guide](../../commands/test.md#reported-outcomes) for result meanings and the [change guide](../../commands/change.md) for how compatibility affects version changes.

## Inspection with a partial warning

Inspection can emit useful observations before a later read fails. Those observations remain valid durable events, while the report clearly states that it is incomplete.

**New durable events:**

```text
INSTALLED  Inventory Sorting (inventory-sorting) [modrinth]
MISSING    Some Mod (some-mod) [curseforge] -> managed file not found
```

**Final transcript — warning appended after the observations:**

```text
‼ Could not finish inspecting <path>: permission denied.
The report may be incomplete. Fix access and rerun mmm list.
```

The final warning does not retract or replay earlier rows. Visible unmanaged jars can be reported separately without preventing managed observations from being shown. The [list guide](../../commands/list.md) defines the states inspection distinguishes and its read-only boundary.

## Coordinated preparation and switching

A Minecraft version change has a coordinated boundary. Preparation happens before the working installation is switched.

**Active display — preparation is incomplete, so switching waits:**

```text
Preparing Minecraft 1.21.1

Compatibility
✅ Inventory Sorting (inventory-sorting) [modrinth]
⠋ Fabric API (fabric-api) [modrinth]

Downloads
⬇ Better Clouds (better-clouds) [modrinth]
[###-------] 30% (3 MiB / 10 MiB)

Switching
⏳ Waiting for preparation to finish
```

Settled compatibility and download results become durable events as they complete. A preparation failure leaves the original modlist and installation in place when that preservation is established.

**Active display — switching has begun:**

```text
Switching to Minecraft 1.21.1
✅ Target artifacts prepared
⠋ Replacing managed files
⏳ Writing consistent metadata
```

Once switching begins, the operation completes consistency work or attempts recovery. A failure reports the actual recovered or remaining state rather than assuming nothing changed.

**Final transcript — successful switch summary:**

```text
Now targeting Minecraft 1.21.1.
```

**Final transcript — incomplete recovery example:**

```text
‼ Switching failed and recovery is incomplete.
Working files were restored, but lock metadata still needs repair: <action>.
```

Forced version changes use the same preparation and switching presentation. Available replacements install, while authorized retention results identify the existing artifacts kept for the target. The [change guide](../../commands/change.md#forced-retention) owns the exact retention rules.

## Safe cancellation and cleanup

The first interruption requests safe cancellation. The component stops scheduling new work and shows consistency or recovery activity for work already authorized.

**New durable event:**

```text
Cancellation requested. Finishing installation consistency before exit.
```

**Active display — cleanup continues without a shutdown timeout:**

```text
Safe cleanup
✅ Unfinished downloads cancelled
⠋ Recording completed artifact results
⏳ Removing non-loadable backup

Press Ctrl+C again to force exit; recovery may remain unfinished.
```

Cleanup retries bounded operations before involving the operator. If the desired artifact and metadata are already correct, harmless non-loadable residue can settle as a warning with its path and cleanup action. Uncertain metadata or an extra loadable jar remains a failure.

**New durable events:**

```text
✅ Completed artifacts recorded consistently
WARNING  Could not remove non-loadable backup <path>; remove it manually
```

**Final transcript:**

```text
Cancelled safely with 4 completed items retained and 3 unfinished items cancelled.
```

A second interruption after the warning is an emergency exit. The transcript must not claim safe recovery merely because the process ended. See [failure, retry, and cancellation promises](../../intent.md#failure-retry-and-cancellation-promises).

## Scrolling, a pending prompt, resize, and return

The terminal session owns scroll position across child components. The following sequence is one continuing command, not four separate screens.

### 1. Following active work

**Active display:** the operator is at the bottom, so new progress remains visible. Settled items emit durable events above the active segment.

```text
... durable transcript ...
✅ Alpha Mod installed

Active work
⠋ Beta Mod
⏳ Gamma Mod
```

### 2. Scrolled away while work continues

**Active display viewed through history:** the operator scrolls upward. New durable events still append at the end, but the viewport remains anchored on the lines being read.

```text
Earlier transcript being read
...

[more output below; viewport does not jump]
```

### 3. A prompt becomes pending and the terminal is resized

**Active display at the unseen end:** a confirmation or correction prompt can wait while the operator reads history. Resizing reflows or redraws the viewport without changing its logical anchor. It must not duplicate durable events or move the reader to the prompt.

```text
... new durable events appended once ...

? Apply the selected correction? <pending>
```

The pending prompt is temporary state until answered. It is not committed as an answered decision.

### 4. Return to active work

**Active display:** when the operator returns to the bottom, the current prompt and current work state appear. Stale frames do not reappear, and completed items are not emitted again.

```text
... latest durable event ...

? Apply the selected correction? <pending>
```

After the prompt is answered, it produces a new durable decision event using the same record as an equivalent supplied decision. Reaching the bottom resumes following new output. This rich-TUI journey is defined by [scrolling and returning to active work](../../intent.md#scrolling-and-returning-to-active-work); plain execution leaves scrolling to the terminal.

## Redirected output and the final transcript

Explicit `--unattended` emits plain append-only output even in a capable terminal. Non-interactive execution, including redirected input or output, does the same. There are no spinners, cursor movement, active selection controls, or repainted progress bars. Results appear once as they settle, followed by a summary that does not replay the list.

Interactive completion, failure, and safe cancellation leave the same durable text and formatting in ordinary shell history as equivalent pure CLI work under the same locale and character capabilities. Relevant resolved decisions use the same records whether supplied by answers, arguments or defaults. See [active display and permanent transcript](../../intent.md#active-display-and-permanent-transcript) and [shared command behavior](../../commands/README.md#cancellation-and-terminal-output).

For example, given equivalent installation outcomes in the same completion order, the TUI may show the temporary download frames above, while pure CLI execution shows none. Both emit these durable lines once (illustrated with ASCII text):

```text
Fabric API (fabric-api) [modrinth] installed
Mod Menu (modmenu) [modrinth] download failed: connection reset; retry with mmm install
Installation incomplete: 1 satisfied, 1 failed.
```

This comparison concerns product records, not raw control bytes, input echoes or temporary frames. Plain interactive questions may remain in terminal history alongside the shared records. A different choice, outcome or completion order is a real difference; do not hide it by rewriting or sorting a captured transcript.

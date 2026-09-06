# Guide to working with the terminal

This guide helps contributors implement the terminal architecture required by [product intent](intent.md#component-ownership-and-architecture). It describes the target design and evidence needed to demonstrate it, not a claim that existing commands conform. [Command guides](commands/README.md) own inputs and outcomes. [Interaction conventions](interactions/interaction-guidelines.md) and [component examples](interactions/component-examples.md) support presentation decisions.

## Start with ownership

| Owner | Responsibility | Boundary |
| --- | --- | --- |
| Terminal session | Input routing, focus, scrolling, active rendering, transcript commits and restoration | Coordinates terminal access for all components. |
| Command flow | Operation sequence and composition of capabilities | Supplies domain decisions; resumes after accepted shared recovery. |
| Capability component | Reusable interaction, including state, controls, help and output | Returns choices and outcomes through explicit contracts. |
| Visual primitive | Controls, styles, icons and layout | Does not decide policy or acquire terminal ownership. |

Business operations are independent of rendering. Every execution profile consumes the same operation outcomes. Capability owners expose relevant resolved decisions from interactive input, arguments or defaults; presentation produces the shared durable records and the session coordinates their emission. Components must not reconstruct what happened by running a second version of an operation or formatting a separate permanent transcript for each mode.

Command orchestration can remain in command packages. Confirmation, selection, setup recovery, progress and results need shared owners even when their first consumer is one command. Sharing styles while copying state transitions and key handling does not establish reusable capabilities.

Use Bubble Tea composition for component state and messages. Make transitions explicit: the flow supplies a request, the capability collects a decision or observes work, and it returns a typed outcome. Children ask the session to present or commit output; they do not independently start terminal programs or print around the renderer. Concrete packages and message types belong to implementation, following these boundaries.

Existing capability helpers are described in [internal/view](../internal/view/README.md). Their presence is not evidence of a complete session architecture. Check an existing command's lifecycle before treating it as a reusable example.

## Separate active state from durable events

The [terminal contract](intent.md#active-display-and-permanent-transcript) gives active rendering and history different lifetimes. In the TUI, pending items, progress and unanswered prompts can repaint, sort and regroup. In every profile, settled results and relevant resolved decisions append once and cannot change in a later render.

The session coordinates these transitions:

1. Receive a settled operation outcome. A completed download alone does not establish successful installation and metadata persistence.
2. Produce the same durable text and formatting in the TUI, pure CLI and other profiles for equivalent outcomes under the same locale and character capabilities.
3. Commit it once and remove its transient representation. Repaint or repeated observation must not emit it again.
4. Render remaining active work. A later correction becomes a new explicit event.
5. Add the final summary, failures and next steps without replaying the item list.

Data structures and event identification are implementation choices. The observable invariant is one durable record per settled outcome, independent of repaint frequency, grouping, size or component reuse. Reprinting the whole model at exit is not a substitute.

Relevant resolved decisions leave the same concise records whether collected by a prompt, argument or default. Describe the choice, not the input mechanism; do not invent an answered question for unattended execution. Collapsing a temporary option list must not erase earlier history. A pending TUI prompt stays active until answered or cancelled. Plain questions and input echoes may remain in terminal history, but do not replace the shared decision record.

Compare durable records, not raw control bytes, input exchanges or temporary progress frames. Actual differences in choices, outcomes and completion order remain visible. Use controlled equivalent work for comparisons; do not repair mismatches by sorting records, changing their wording or deleting duplicate output.

## Coordinate primary and alternate screens

Normal terminal history must retain pre-command shell output and MMM's durable results after exit. The screen coordination below applies to rich TUI presentation; plain execution appends records without acquiring these screen modes. Temporary alternate screens may host active controls, but cannot be the only home of committed history. An optional transcript viewer cannot be required to recover normal history.

The session coordinates screen transitions with its renderer. Before adopting an alternate screen, establish how durable events reach primary-screen history, how active work returns after a commit, and how input and focus remain attached to the current component. Child components must not toggle screens or write directly while another renderer owns the terminal.

This guide does not prescribe an unverified Bubble Tea call sequence. A repaintable view, in-memory event list, or final dump after leaving the alternate screen does not by itself establish the required lifecycle. Demonstrate the renderer integration in a real terminal before recommending it as a shared pattern. An inline active region and temporary alternate screens must satisfy the same history observations.

On completion, failure and safe cancellation, restore terminal state acquired by the session, including screen mode, input mode and cursor visibility. Restoration must not clear pre-command output or duplicate records. Coordinate it with operation recovery so the returning shell prompt does not falsely imply that installation changes have stopped.

## Preserve reading position

In the TUI, following new output and reading older history are distinct presentation states. Once the operator scrolls away, preserve their position as work progresses and durable events arrive. A newly required prompt waits without taking that position away. Plain execution leaves scrolling to the terminal and must not introduce a repaintable viewport to emulate these controls.

Returning to the active end shows current work and any pending prompt, then resumes following. Resize and regrouping must not restore stale frames, duplicate history or misplace controls. Long lists need accessible content, not every row rendered simultaneously.

Distinguish terminal-emulator scrollback from an MMM-owned viewport. A component's viewport offset alone does not prove that the terminal's reading position survives. Verify the selected rendering approach in supported terminals; do not assume a library option owns behavior that has not been demonstrated.

## Select presentation from capabilities and policy

Follow the [execution-mode matrix](intent.md#execution-modes-and-operator-intent). Explicit `--unattended` always selects plain append-only output without questions, animation or styling/control sequences. Complete arguments alone do not select that policy. Otherwise, interactive input and output use rich controls where control sequences are supported, with Unicode or ASCII UI symbols as supported. Without control-sequence support, interactive execution collects equivalent decisions through line-based questions. Non-interactive execution, including redirection, never prompts and emits plain records.

Detect interaction, control-sequence and character capabilities at the session boundary and pass them consistently to consumers alongside invocation policy. TTY detection alone does not establish control-sequence or Unicode support. Terminal detection does not grant mutation authority, and unattended execution does not imply force. Use existing helpers where applicable, but verify both absence of control sequences and presence of required results. Disabling a renderer alone does not prove result delivery.

Error and diagnostic paths must cooperate with session output ownership, avoid corrupting active frames and avoid duplicating handled errors. A broken output pipe does not cancel authorized work or required consistency operations.

## Compose setup and correction

Invoke shared recovery according to [intent](intent.md#shared-setup-and-recovery). Preserve valid inputs while correcting another field. Return an explicit accepted, declined, cancelled or failed outcome; resume the command only after successful accepted recovery.

Malformed syntax differs from a correctable supplied value. Do not reject every invalid field in a parser hook if that prevents required interactive correction. Without prompting, invalid requests fail without mutation.

Shared recovery must not reconstruct another command's private runtime. Inspection stays read-only; explicitly accepted initialization or correction is a separate preceding operation. Command guides own defaults, reset authority and final confirmation.

## Cancel safely

Route the first interruption to safe cancellation: stop scheduling, cancel unfinished downloads and finish necessary consistency or recovery. Show ongoing cleanup and retain completed independent outcomes and decisions.

Explain that a second interruption forces termination and may leave recovery unfinished. Required recovery has no automatic shutdown timeout. Report established results; attempting rollback is not proof that nothing changed. Restore terminal control as part of safe completion.

Escape and documented quit shortcuts follow the active capability's [controls](interactions/interaction-guidelines.md#controls-and-help) and reach the same safe cancellation boundary when work is running. Free text remains free text.

## Prove the integration

Start with the [E2E testing guide](testing/README.md) for tooling, scenarios and presentation checks. [Intent's acceptance journeys](intent.md#acceptance-and-evidence) govern required observations.

Exercise real consuming commands across the mode matrix, including unattended execution in a capable terminal and with redirected I/O, rich ASCII interaction and plain line-based questions. Compare permanent records for equivalent prompted and supplied decisions, successful work, partial failure and cancellation. Model tests can verify transitions, but cannot establish shell history, scroll round trips, restoration or cross-command consistency. Record the terminal and platform used to demonstrate renderer behavior. Do not normalize, reorder or reconstruct missing output to manufacture an expected screen.

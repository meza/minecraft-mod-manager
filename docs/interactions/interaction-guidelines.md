# Terminal interaction conventions

These conventions help contributors present shared capabilities consistently. [Product intent](../intent.md) owns product behavior; [command guides](../commands/README.md) own workflows and outcomes. This document refines presentation without defining additional modes or authorization policies.

Use the [implementation guide](../guide-to-working-with-the-terminal.md) for ownership and lifetimes, and [component examples](component-examples.md) for annotated target illustrations. Examples do not claim current implementation conformance.

## Execution contexts

Interactive terminals can collect decisions. `--unattended` suppresses questions without granting authorization; progress may remain dynamic. Redirected input or output requires plain append-only output. See [execution modes](../intent.md#execution-modes-and-operator-intent).

### Redirected input or output

Preserve requested results, failures and useful next steps without prompts, spinners, cursor movement or styling control sequences. Presentation must not change selection, ownership or operation authority.

## Questions and recorded answers

Show what is being asked and make defaults explicit. Preview affected items before destructive confirmation. Command contracts determine when consent is required and what force authorizes; the shared confirmation component owns presentation and input.

Destructive confirmations default to No. Enter selects the displayed default. EOF while waiting for a decision aborts safely. Confirmed choices leave concise durable records; temporary option lists collapse out of active presentation.

### Localized option tokens

Each option has a stable action identity, localized label and localized short token. Defaults belong to behavior, not translations. Trim input and accept localized short tokens and full labels case-insensitively. Empty input selects the declared default. Invalid input keeps an interactive question open with a localized explanation.

Short tokens must be non-empty and unique within a locale's option set; labels must be non-empty. Validate these constraints when maintaining translations. Do not hardcode English tokens or rely on capitalization to communicate defaults.

Illustrative English rendering:

```text
? Delete the selected files? (y/n) [default: n]
```

Letters and labels change with locale; actions do not. No-prompt execution never waits in this parser: command policy determines the outcome when a required decision is absent.

## Selection

Distinguish focus from selection: a pointer such as `❯` identifies focus, while `✓` identifies selection. Supply ASCII equivalents and labels; colour alone is insufficient.

Show supported navigation and selection actions. On confirmation, record all chosen values in a concise answered-question line. Long lists remain accessible through navigation and scrolling rather than having to fit on one screen. Active sorting must not reorder committed records.

## Controls and help

Use the same semantics wherever a capability appears. Localizable help must describe the active control's actual actions. Reuse suitable Bubble Tea controls without treating default keymaps or untranslated help as product authority.

The first `Ctrl+C` requests safe cancellation; a second can force termination after the warning defined in [intent](../intent.md#failure-retry-and-cancellation-promises). Escape follows the control's back or cancel semantics. [Initialization](../commands/init.md) has no step-back navigation: Escape leaves active filtering or otherwise cancels.

Ordinary letters must not cancel free text. A progress-only control may expose a documented quit shortcut. New prompts must not pull an operator away from history.

## Progress bars

Determinate bars show filled and empty portions with numeric progress:

```text
[#####-----] 50% (512 KB / 1 MB)
```

Use an indeterminate indicator when no meaningful total is known. Do not invent percentages. Keep the operation identifiable without animation. Redirected output omits transient animation and reports durable outcomes.

Completed transfer does not necessarily mean completed installation. Reserve success for settled outcomes, including required persistence. Preparation, downloading, switching and cleanup labels can explain what remains in progress.

## Results and summaries

Use consistent terms: `modlist.json`, `modlist-lock.json`, mods folder, managed file, unmanaged file and `.mmmignore`. Identify affected items, explain what happened and why when known, and give the next useful action.

Distinguish failure from uncertainty. Unmanaged files can legitimately coexist; do not use error styling merely because they are unmanaged. Separate conclusive incompatibility from inconclusive lookup, and missing files from missing or contradictory resolution evidence. Command guides define these outcomes and recovery actions.

Use text alongside icons. Distinguish questions, focus, selection, pending work, pins, success, warnings and failure in ASCII as well as Unicode. Keep secondary metadata subordinate but legible; error and follow-up lines must remain recognizable without styles.

Commit settled results once in completion order. Summaries add counts and next steps without replaying each item. Do not claim unchanged files or successful recovery without established evidence. Numeric exit assignments belong in command references and acceptance scenarios, not visual examples.

## Accessibility and language

Every interactive path must support keyboard use. Keep prompts readable in narrow terminals, allow access to long content and avoid dense decoration. Colour, icons and animation must not be the sole carriers of meaning.

Respect capabilities rather than assuming every TTY supports the same features. Translate messages, choices and hints, with English fallback for missing translations. Verify real localized layouts and input tokens as well as stable localization keys.

## Validation

Follow the [terminal E2E guide](../testing/terminal-harness.md). Verify shared conventions within real consuming commands. Use [examples](component-examples.md) to discuss visual transitions and [intent's acceptance journeys](../intent.md#acceptance-and-evidence) to establish required behavior.

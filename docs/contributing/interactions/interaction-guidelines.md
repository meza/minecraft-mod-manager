# Terminal interaction conventions

These conventions help contributors present shared capabilities consistently. [Product intent](../../intent.md) owns product behavior; [command guides](../../commands/README.md) own workflows and outcomes. This document refines presentation without defining additional modes or authorization policies.

Use the [implementation guide](../guide-to-working-with-the-terminal.md) for ownership and lifetimes, and [component examples](component-examples.md) for annotated target illustrations. Examples do not claim current implementation conformance.

## Execution contexts

Follow the [execution-mode matrix](../../intent.md#execution-modes-and-operator-intent). Interactive terminals use rich controls with Unicode or ASCII UI symbols where supported, or line-based questions without control-sequence support. Explicit `--unattended` always selects plain append-only output and prohibits questions, even in a capable terminal. Complete arguments alone do not select unattended presentation. Non-interactive execution, including redirection, is also plain and never prompts.

<a id="redirected-input-or-output"></a>

### Plain output

Preserve requested results, failures and useful next steps without spinners, cursor movement or styling control sequences. Unattended and non-interactive execution never prompt; plain interactive execution can ask line-based questions. Unicode support is independent of these constraints. Presentation must not change selection, ownership or operation authority.

## Questions and recorded answers

Show what is being asked and make defaults explicit. Preview affected items before destructive confirmation. Command contracts determine when consent is required and what force authorizes; the shared confirmation component owns presentation and input.

Destructive confirmations default to No. Enter selects the displayed default. EOF while waiting for a decision aborts safely. Relevant resolved choices leave the same concise durable records whether supplied by answers, arguments or documented defaults. Records describe choices independently of their input mechanism. Temporary TUI option lists collapse out of active presentation; plain input exchanges may remain in history but do not replace the shared records.

### Localized option tokens

Each option has a stable action identity, localized label and localized short token. Defaults belong to behavior, not translations. Trim input and accept localized short tokens and full labels case-insensitively. Empty input selects the declared default. Invalid input keeps an interactive question open with a localized explanation.

Short tokens must be non-empty and unique within a locale's option set; labels must be non-empty. Validate these constraints when maintaining translations. Do not hardcode English tokens or rely on capitalization to communicate defaults.

Illustrative English rendering:

```text
? Delete the selected files? (y/n) [default: n]
```

Letters and labels change with locale; actions do not. No-prompt execution never waits in this parser: command policy determines the outcome when a required decision is absent.

## Selection

In rich controls, distinguish focus from selection: a pointer such as `❯` identifies focus, while `✓` identifies selection. Supply ASCII equivalents and labels; colour alone is insufficient. Plain interactive selection exposes the same choices through line-based input without requiring cursor navigation.

Show the actions supported by the current presentation. On confirmation, record all chosen values in a concise decision record that also works for supplied arguments or defaults. Long lists remain accessible without having to fit on one screen; rich controls may use navigation and scrolling, while plain interaction uses readable append-only choices. Active sorting must not reorder committed records.

## Controls and help

Use the same semantics wherever a capability appears. Localizable help must describe the active control's actual actions. Reuse suitable Bubble Tea controls without treating default keymaps or untranslated help as product authority.

The first `Ctrl+C` requests safe cancellation; a second can force termination after the warning defined in [intent](../../intent.md#failure-retry-and-cancellation-promises). In rich controls, Escape follows the control's back or cancel semantics. [Initialization](../../commands/init.md) has no step-back navigation: Escape leaves active filtering or otherwise cancels. Plain interactive questions expose equivalent decisions without requiring rich-control key sequences.

Ordinary letters must not cancel free text. A progress-only control may expose a documented quit shortcut. New prompts must not pull an operator away from history.

## Progress bars

In rich TUI presentation, determinate bars show filled and empty portions with numeric progress:

```text
[#####-----] 50% (512 KB / 1 MB)
```

Use an indeterminate indicator when no meaningful total is known. Do not invent percentages. Keep the operation identifiable without animation. Plain output, including unattended execution in a capable terminal, omits transient animation and reports durable outcomes.

Completed transfer does not necessarily mean completed installation. Reserve success for settled outcomes, including required persistence. Preparation, downloading, switching and cleanup labels can explain what remains in progress.

## Results and summaries

Use consistent terms: `modlist.json`, `modlist-lock.json`, mods folder, managed file, unmanaged file and `.mmmignore`. Identify affected items, explain what happened and why when known, and give the next useful action.

Distinguish failure from uncertainty. Unmanaged files can legitimately coexist; do not use error styling merely because they are unmanaged. Separate conclusive incompatibility from inconclusive lookup, and missing files from missing or contradictory resolution evidence. Command guides define these outcomes and recovery actions.

Use text alongside icons. Distinguish questions, focus, selection, pending work, pins, success, warnings and failure in ASCII as well as Unicode. Keep secondary metadata subordinate but legible; error and follow-up lines must remain recognizable without styles.

Use the same durable text and formatting for equivalent records in every mode under the same locale and character capabilities, as required by the [transcript contract](../../intent.md#active-display-and-permanent-transcript). This includes resolved decisions, failures, warnings and summaries, not only successful results. Commit settled results once in completion order. Summaries add counts and next steps without replaying each item. Do not claim unchanged files or successful recovery without established evidence. Numeric exit assignments belong in command references and acceptance scenarios, not visual examples.

## Accessibility and language

Every interactive path must support keyboard use. Keep prompts readable in narrow terminals, allow access to long content and avoid dense decoration. Colour, icons and animation must not be the sole carriers of meaning.

Respect capabilities rather than assuming every TTY supports the same features. Translate messages, choices and hints, with English fallback for missing translations. Verify real localized layouts and input tokens as well as stable localization keys.

## Validation

Follow the [E2E testing guide](../testing/README.md). Verify shared conventions within real consuming commands. Use [examples](component-examples.md) to discuss visual transitions and [intent's acceptance journeys](../../intent.md#acceptance-and-evidence) to establish required behavior.

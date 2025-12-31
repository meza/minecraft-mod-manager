# Language-dependent prompts (option initials)

Some interactive prompts ask the user to respond with the initial (or short form) of one of two or more options.
These prompts are language-dependent and must not be hardcoded to English.

This document defines the contract for how MMM renders and parses language-dependent prompts.

Primary audience: maintainers and developers implementing prompts.

Related docs:
- `docs/interactions/patterns/confirmations.md`
- `docs/interactions/patterns/accessibility.md`
- `internal/i18n/README.md`
- `docs/i18n.md`

## Problem

Using `y/n` (or printing `(y/N)`) assumes English.
In many locales the initials are different, and showing `ja/nein (y/N)` is confusing and unusable.

This repo currently has several line prompts that print `(y/N)` and only accept `y` or `yes`.
That behavior is a footgun for non-English locales.

## Scope

This applies to prompts where the user is expected to answer using:
- a short token (typically a single character), or
- the full option label

Examples:
- Yes/No
- Overwrite/Cancel
- Retry/Cancel

This does not apply to Bubble Tea navigation keys (j/k/h/l/q/?/esc/arrows/enter/tab/ctrl+c).
Those are treated as universal interaction conventions, not language abbreviations.

## Contract

### Option sets

Every language-dependent prompt uses an explicit option set.
Options have stable IDs in code, plus localized display and input tokens from i18n.

Each option must have:
- `label`: what MMM prints to describe the option (and what MMM accepts as full input)
- `short`: the short token MMM prints and accepts (typically one character)

Defaults are behavior and MUST be owned by code, not translations.

### Rendering

MMM SHOULD render a compact suffix that shows the short tokens for the available options.
The suffix MUST make the default clear, but default highlighting is a UI concern in code.

Recommended style:

```
? <question> (o/c) [default: c]
```

Notes:
- Do not rely on case to communicate defaults. Many scripts do not have case.
- Keep the suffix stable and easy to scan in monospaced terminals.

### Parsing

Given user input:
- Trim whitespace.
- If the input is empty, select the default option (owned by code).
- Otherwise match case-insensitively against:
  - the option `short`, then
  - the option `label`
- If no option matches:
  - interactive mode: re-prompt with a localized "invalid choice" message and re-render the options
  - non-interactive mode: do not prompt; use the command's defined safe fallback (often cancel/no-op) or require an explicit flag such as `--force`

### Validation requirements

Within a single option set and locale:
- `short` MUST be non-empty.
- `short` MUST be unique across options.
- `label` MUST be non-empty.

Violations should fail loudly in developer/CI contexts so mistakes in Crowdin do not ship silently.

## i18n key guidance

The i18n keys should keep translation authoring simple.
Avoid mini-languages (like `c|cancel`) in translation files.

Suggested naming pattern:

- `<area>.<thing>.question`
- `<area>.<thing>.option.<id>.label`
- `<area>.<thing>.option.<id>.short`

Where `<id>` is the stable option ID in code (example: `overwrite`, `cancel`).

## Examples

### Overwrite / Cancel

```
? Configuration file already exists at config.json. Overwrite? (o/c) [default: c]
```

Accepted inputs:
- empty input selects default `cancel`
- `o` or `overwrite`
- `c` or `cancel`

### Yes / No

Do not hardcode `(y/N)`.
Define localized `label` and `short` values and render the same suffix style:

```
? Delete these files? (<yesShort>/<noShort>) [default: <noShort>]
```

The parser accepts both the localized short token and the localized full label.


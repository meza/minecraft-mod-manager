# i18n

This package provides one job for the wider project: return the right user-facing string for the current environment.

It is intentionally small and boring. There is one public entry point: `T()`.

## When to use this

**Any user-facing text must be translated.**

Use this module anywhere a human user will see text:

* CLI output
* TUI prompts and labels
* errors shown directly to users

**Logs must be English-only.**

Do not use translation keys in:

* log files / debug logs
* telemetry labels
* internal error wrapping

The reason is practical: when a user asks for help and shares logs, we need a common language so anyone on the team can assist.

## Quick start

Import the package and call `T()` with a translation key.

```go
// example
fmt.Println(i18n.T("app.description"))
```

There is no explicit initialization step. The first call to `T()` lazily loads the embedded locale files and selects a locale.

## API

### `T(key string, args ...Tvars) string`

`T()` resolves a translation key and returns a string.

* If the key is missing, the key itself is returned (this is a deliberate, visible failure mode).
* Only zero or one `Tvars` argument is supported. Passing more will panic.

Simple usage:

```go
s := i18n.T("cmd.version.short")
```

With variables and plurals:

```go
s := i18n.T(
  "cmd.help.error",
  i18n.Tvars{Data: &i18n.TData{"topic": "init"}},
)
```

Plural example (ICU MessageFormat):

```go
s := i18n.T(
  "downloads.count",
  i18n.Tvars{Count: 2, Data: &i18n.TData{"appName": "mmm"}},
)
```

Behavior notes:

* `Count` is always injected as the variable `count`.
* If `Data` is nil, only `count` is injected.

## Locale selection

The module selects locales automatically. The intent is "do what the system says" with predictable fallbacks.

Resolution order:

### 1) `LANG`

If `LANG` is set, it is used as the only input locale.

Example:

```bash
LANG=de_DE ./mmm
```

### 2) OS locale detection

If `LANG` is not set, the module asks the OS for preferred locales.

### 3) Fallback

If locale detection fails, the module falls back to English.

### Locale normalization

Detected locale strings are normalized and expanded:

* canonical tag first (example: `fr_FR` -> `fr-FR`)
* then the base language (example: `fr`)

This allows region-specific translations when available, and graceful degradation when not.

## Translation files

Locale files are embedded into the binary from `lang/*.json`.

Requirements:

* `lang/en-GB.json` must exist (it is the default locale).
* filenames must match the locale code (example: `de-DE.json`).
* files must be valid JSON.

Fail-fast behavior:

* missing `lang/` directory panics
* invalid locale JSON panics

These are treated as developer/CI failures, not user-facing errors.

## Test mode

Set `MMM_TEST=true` to make `T()` deterministic.

In test mode, `T()` returns the key (and argument details) instead of translating.

Why this exists:

* unit tests can assert against stable output
* tests do not depend on translation file contents
* tests do not depend on the machine locale

Example:

```go
t.Setenv("MMM_TEST", "true")
assert.Equal(t, "test.simple", i18n.T("test.simple"))
```

## Translation key naming guidelines

Consistency here matters more than perfection. The goal is that developers can:

* guess the key name before searching
* add new strings without inventing new conventions
* keep keys stable across refactors

### Terminology

* namespace: the dot-separated path segments (example: `cmd.init.tui.loader.question`)
* leaf: the final segment that describes the message variant (example: `question`)
* variant: a leaf that distinguishes multiple messages for the same concept (example: `short` vs `usage.long`)

### Key format

Use lower-case dot-separated namespaces:

```
<area>.<feature>.<surface>.<concept>.<variant>
```

Examples from this project:

* `app.description`
* `cmd.help.error`
* `cmd.init.tui.mods-folder.question`
* `cmd.init.usage.release-types`
* `key.help.page_next`

### Namespaces

Prefer these top-level areas:

#### `app.*`

High-level app labels and descriptions.

#### `cmd.<command>.*`

Command text.

Use sub-namespaces to keep concerns separated:

* `cmd.<command>.short` for the one-line summary (help listing)
* `cmd.<command>.usage.*` for usage strings
* `cmd.<command>.tui.*` for interactive prompts and validation messages

#### `key.*`

Key labels shown to users (especially in the TUI).

Use `key.help.*` for the "help bar" / legend verbs.

### Variants and suffixes

Use these variant suffixes consistently:

* `short` - one-line description for listings
* `usage.<name>` - usage strings where `<name>` matches a CLI arg (example: `usage.loader`)
* `question` - prompt text in interactive flows
* `invalid` - validation error messages for a single field
* `template` - reusable message template (only when it is actually used in multiple places)

Avoid inventing synonyms (`prompt` vs `question`, `bad` vs `invalid`). Pick one.

### Variables

Variable names are part of the API. Treat them as stable.

Rules:

* use lowerCamelCase (example: `appName`, `helpUrl`, `releaseTypes`)
* keep variable names consistent across all locales
* keep variable names semantic (prefer `gameVersion` over `version`)
* never reuse a variable name for different meaning across keys

The plural selector variable is always `count`.

### Plural keys

If a string depends on quantity, make it explicit in the message.

* use ICU plurals with `count`
* do not bake numbers into strings

Example:

```json
{
  "mods.selected": "{count, plural, one {Selected 1 mod} other {Selected {count} mods}}"
}
```

### Key stability

Keys are a contract between code and translations.

* do not rename keys casually
* if you must rename, do it as a deliberate change with a search-and-update across locales
* prefer moving code without changing keys

### What not to do

* Do not include language in the key name (`cmd.init.short.en`).
* Do not use spaces, slashes, or mixed casing in keys.
* Do not use unstable identifiers like filenames or numeric IDs in keys.

## Where to add new strings

Add the key to `lang/en-GB.json` first, then propagate it to other locales.

Practical workflow:

1. add the new key in English
2. wire it up in code
3. run tests in `MMM_TEST=true` mode
4. fill in other locales (or leave the key missing intentionally for a visible fallback)

## Maintainer notes

This module uses package-level state and lazy initialization.

* It is designed for CLI startup patterns (initialize once, then read many times).
* If we ever need stronger concurrency guarantees, we should wrap initialization in `sync.Once` and add a race test.

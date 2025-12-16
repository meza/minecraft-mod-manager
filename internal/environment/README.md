# internal/environment

This package is the single place where we read environment variables that affect platform APIs and telemetry.

Commands and platform clients call these helpers so that:

- the env var names stay consistent
- tests can override behavior with `t.Setenv`
- release builds can inject values without changing call sites

## Public API

API keys:

- `ModrinthApiKey()` reads `MODRINTH_API_KEY`
- `CurseforgeApiKey()` reads `CURSEFORGE_API_KEY`
- `PosthogApiKey()` reads `POSTHOG_API_KEY`

Build-time values:

- `AppVersion()` (used in User-Agent strings and telemetry)
- `HelpURL()` (used to link to docs/help)

When the relevant env var is missing, these functions return placeholder strings (for example `REPL_MODRINTH_API_KEY`). In release builds, those placeholders are replaced during the build.

## Related docs

The root `README.md` documents how users set env vars like `MMM_DISABLE_TELEMETRY`. This package documents the ones used for external APIs.

## Tests

Run from the repo root:

```bash
make test
```


# internal/environment

This package is the single place where we read environment variables that affect platform APIs and telemetry.

Commands and platform clients call these helpers so that:

- the env var names stay consistent
- tests can override behavior with `t.Setenv`
- release builds can inject values without changing call sites

## Public API

API keys:

- `ModrinthAPIKey()` reads `MODRINTH_API_KEY`
- `CurseforgeAPIKey()` reads `CURSEFORGE_API_KEY`
- `PosthogAPIKey()` reads `POSTHOG_API_KEY`

Build-time values:

- `AppVersion()` (used in User-Agent strings and telemetry)
- `HelpURL()` (used to link to docs/help)

When the relevant env var is missing, these functions return embedded defaults.

## Build-time injection

For release and CI builds, token defaults are embedded at link time using Go `-ldflags -X` (not by rewriting source files).

For local builds, `make build` loads token values from:

- environment variables set in your shell, then
- `./.env` in the repo root (if present), without overwriting any values already set in your shell

If any token is still missing, `make build` fails fast so we do not silently compile placeholder values into the binary.

If you need a contributor build that does not require tokens, use `make build-dev` instead.

The `make build` token-injection logic is implemented for both Unix-like hosts (POSIX shell) and Windows hosts (PowerShell).

## Runtime precedence

At runtime, values are resolved in this order:

1. environment variables (including ones loaded from a runtime `.env` file via `github.com/joho/godotenv/autoload`)
2. embedded defaults compiled into the `mmm` binary

Note: if you set an env var to an empty value, it still counts as "set" and overrides the embedded default (for example, setting `POSTHOG_API_KEY=` disables telemetry).

## Related docs

The root `README.md` documents how users set env vars like `MMM_DISABLE_TELEMETRY`. This package documents the ones used for external APIs.

## Tests

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.

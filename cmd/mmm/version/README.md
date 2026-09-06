# cmd/mmm/version

This package implements `mmm version`: print the current application version and exit.

The version string comes from `internal/environment.AppVersion()` so release builds can inject the real version at build time.

## Code map

- `cmd/mmm/version/version.go`: cobra command definition
- `cmd/mmm/version/version_test.go`: basic behavior test

## Related docs

See [`docs/commands/version.md`](../../../docs/commands/version.md) for the user-facing command guide and [`docs/intent.md`](../../../docs/intent.md) for authoritative product behavior. Keep the guide accurate if the command grows.

## Tests

See the root [verification guidance](../../../CONTRIBUTING.md#verification) for required checks and snapshot update instructions.

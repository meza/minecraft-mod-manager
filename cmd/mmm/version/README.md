# cmd/mmm/version

This package implements `mmm version`: print the current application version and exit.

The version string comes from `internal/environment.AppVersion()` so release builds can inject the real version at build time.

## Code map

- `cmd/mmm/version/version.go`: cobra command definition
- `cmd/mmm/version/version_test.go`: basic behavior test

## Related docs

There is no dedicated `docs/commands/version.md` page today because the behavior is intentionally tiny. If the command grows (flags, additional output), add a user-facing doc page under `docs/commands/` and a spec under `docs/specs/`.

## Tests

Run from the repo root:

```bash
make test
```


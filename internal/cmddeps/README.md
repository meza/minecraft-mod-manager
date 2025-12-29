# internal/cmddeps

This package standardizes shared command dependencies so every command starts from the same, explicit defaults.
Use it to build per-command deps without reimplementing filesystem, output, logger, or client wiring.

## Usage

Build common deps in the command entrypoint, then derive command-specific deps.

```go
common := cmddeps.NewCommonDeps(cmd, cmddeps.CommonDepsOptions{
	Quiet: opts.Quiet,
	Debug: opts.Debug,
})

deps := newRemoveDeps(common)
```

## Discipline

Keep `CommonDeps` small and stable.
Only include dependencies that are truly shared across multiple commands.
If a dependency is command-specific (or only shared by two commands), keep it in that command’s deps instead of adding it here.

## Testing

Prefer overriding `CommonDepsOptions` in tests instead of hand-wiring shared fields.
Use in-memory filesystems (`afero.NewMemMapFs`) and stub clients when you need deterministic behavior.

See `CONTRIBUTING.md` for required test and coverage checks.

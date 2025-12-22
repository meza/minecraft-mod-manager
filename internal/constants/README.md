# internal/constants

This package is where we keep a tiny set of stable string constants that are shared across commands.

Right now it exists to avoid "stringly-typed" drift when code needs the application name or command name.

## What is here

- `APP_NAME`: the project identifier (`minecraft-mod-manager`)
- `COMMAND_NAME`: the CLI command (`mmm`)

## When to add something here

Add a constant only if:

- it is used in multiple packages, and
- changing it would be risky or annoying to review, and
- it is not better represented as a config/env value.

If it is only used by one command or module, prefer keeping it close to that code.

## Tests

See `CONTRIBUTING.md` for required test/coverage checks and snapshot update instructions.

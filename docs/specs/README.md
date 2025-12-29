# Command Reference

This section documents how each CLI command operates. These guides are intended for the Go port and describe the logic without referring to implementation details of the original TypeScript code.

- [init](init.md)
- [add](add.md)
- [install](install.md)
- [update](update.md)
- [list](list.md)
- [change](change.md)
- [test](test.md)
- [prune](prune.md)
- [scan](scan.md)
- [remove](remove.md)

## Global Options

Every command supports a few shared flags provided by the CLI parser:

- `-c, --config <file>` - path to `modlist.json`. Defaults to `./modlist.json`.
- `--non-interactive` - disables prompts and fails fast when required inputs are missing.
- `-q, --quiet` - suppresses non-essential output; errors and required results still print.
- `-d, --debug` - prints additional debug messages.

Non-TTY usage is treated as non-interactive by default.

These options must appear before the command name, e.g. `mmm --quiet install`.

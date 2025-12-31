# Flow: help

## User goal and success conditions

You want to discover MMM commands and understand how to use a specific command.

Success looks like:
- You can find the command you need
- You can see required arguments and available flags
- You can find examples or next steps for learning more

## Entry points

- Root help: `mmm --help` and `mmm -h`
- Help command: `mmm help` and `mmm help <command>`
- User guide: `README.md`

## Primary flow

1. You run `mmm --help` or `mmm help`.
2. MMM prints usage and a list of available commands.
3. If you request help for a specific command, MMM prints that command help.

## Key alternate paths

- If you request help for an unknown topic, MMM prints an error and then prints root usage.

## Non-interactive behavior

Help output must always work in non-interactive environments and when stdout is redirected.

## References in code

- Root command and help wiring: `cmd/mmm/root.go`


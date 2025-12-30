# Flow: version

## User goal and success conditions

You want to know which version of MMM you are running.

Success looks like:
- MMM prints a version string and exits

## Entry points

- Command: `mmm version`
- Code: `cmd/mmm/version/version.go`

## Notes

This command is designed to be non-interactive.

At the time of writing, there is no dedicated behavior spec file for `version` under `docs/specs/`.

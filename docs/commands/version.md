# Version

This guide describes the Go-port target defined in [product intent](../intent.md), not a claim that every released build already implements it.

Use `version` to identify the running MMM release, for example when reporting a problem.

## Usage

```shell
mmm version
```

Version output works without a modlist, including in a missing or broken installation. It does not offer initialization, change the installation or update MMM itself. Self-update is outside the port baseline.

See [help](help.md) for command usage and [shared command behavior](README.md) for diagnostics and execution modes.

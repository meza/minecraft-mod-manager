# Configuration documentation

Configuration is where users succeed or give up. Document it as a first-class topic, not an appendix. Show people how to go from zero to a correct, safe configuration, then how to grow into advanced setups.

## Goals

Explain what can be configured, why the options exist, and how choices change behavior. Avoid option dumps without context.

## Placement

Keep a "Configuration" section in the nearest relevant README. If the catalog is long, place a dedicated `docs/configuration.md` and link to it. Co-locate example files next to the code that consumes them.

## Quick start configuration

Provide a minimal, copyable example that works out of the box. Follow with a short paragraph explaining what it does and when to use it.

## Common scenarios

Document a few real setups that map to user intent, for example "single-node dev," "replicated prod," or "behind a reverse proxy." Show the smallest viable configuration for each scenario.

## Option catalog

Group options by feature or component, not alphabetically. For each option, provide the same fields in the same order so the catalog is scannable.

### Name

Exact key as it appears in config files, CLI flags, or environment variables.

### Purpose

One sentence on what behavior this option controls and why someone would change it.

### Type and allowed values

State the type clearly. If enumerated, list allowed values. Note units for numbers and duration syntax.

### Default

Show the default as actually applied by the program. If the default is dynamic, explain the rule.

### Required

Say whether the option is required. If it becomes required only in certain modes, say when.

### Sources and precedence

List supported sources for this option (config file path, environment variable, CLI flag, service discovery). Document the precedence order exactly. If the program merges values, explain how.

### Reload behavior

State whether changes take effect on reload, on restart, or immediately. If hot reload is supported, describe the trigger.

### Security notes

Call out sensitive values. Show how to provide secrets safely (env vars, secret files, vault references). Never show real secrets; use placeholders and mark them clearly.

### Interactions

Note important relationships with other options, feature gates, or modes. If an option is ignored under certain conditions, say so.

### Versioning

Indicate when the option was added, deprecated, or changed. Link to ADRs or release notes if relevant.

### Example

Provide a minimal example that demonstrates the option in context.

## File formats and locations

Document supported config formats and their exact file locations or search order per platform. If the program reads multiple files, show the merge rules.

## Environment variables

List environment variables in a dedicated subsection that mirrors the catalog fields above. Show platform-specific notes for systemd, containers, or CI usage when relevant.

## CLI flags

Provide the mapping between flags, environment variables, and config keys when they exist. Clarify which flags override file values.

## Validation and failure modes

Describe what the program validates at startup and typical error messages for misconfiguration. Offer troubleshooting steps and links to logs or debug commands.

## Migration guidance

When configuration changes between versions, provide a short migration section with a before and after example and a link to the ADR or release notes.

## Example library

Include a small `examples/` directory with runnable samples referenced from the docs. Keep them pinned to known-good versions where possible.

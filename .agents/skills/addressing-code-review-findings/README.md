# Addressing Code-Review Findings

This skill helps an agent assess code-review feedback before presenting or acting on it. It treats reviewer comments as evidence to verify, not as an automatic work queue or permission to change code.

The skill requires the agent to:

- separate each reported observation from its proposed remedy;
- verify the observation against the implementation, tests, requirements, and architecture;
- classify every finding and preserve the reason for its disposition;
- recommend a proportionate remedy independently of whether the observation is valid; and
- act only when the active task authorizes the exact change.

## Install with `npx skills`

Install the skill into the current project:

```console
npx skills add meza/skills --skill addressing-code-review-findings
```

Install it globally for Codex:

```console
npx skills add meza/skills --skill addressing-code-review-findings --agent codex --global
```

Install it globally for Claude Code:

```console
npx skills add meza/skills --skill addressing-code-review-findings --agent claude-code --global
```

After installation, ask your agent to assess or address a set of code-review findings.
